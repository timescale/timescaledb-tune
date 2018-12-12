// Package tstune provides the needed resources and interfaces to create and run
// a tuning program for TimescaleDB.
package tstune

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/pbnjay/memory"
	"github.com/timescale/timescaledb-tune/internal/parse"
	"github.com/timescale/timescaledb-tune/pkg/pgtune"
)

const (
	errCouldNotExecuteFmt  = "could not execute `%s --version`: %v"
	errUnsupportedMajorFmt = "unsupported major PG version: %s"

	currentLabel   = "Current:"
	recommendLabel = "Recommended:"

	promptOkay    = "Is this okay? "
	promptCorrect = "Is this correct? "
	promptYesNo   = "[(y)es/(n)o]: "
	promptSkip    = "[(y)es/(s)kip/(q)uit]: "

	statementConfFileCheck = "Using postgresql.conf at this path:"
	errConfFileCheckNo     = "please pass in the correct path to postgresql.conf using the --conf-path flag"
	errConfFileMismatchFmt = "ambiguous conf file path: got both %s and %s"

	errSharedLibNeeded             = "`timescaledb` needs to be added to shared_preload_libraries in order for it to work"
	successSharedLibCorrect        = "shared_preload_libraries is set correctly"
	successSharedLibUpdated        = "shared_preload_libraries will be updated"
	statementSharedLibNotFound     = "Unable to find shared_preload_libraries in configuration file"
	plainSharedLibLine             = "shared_preload_libraries = 'timescaledb'"
	plainSharedLibLineWithComments = plainSharedLibLine + "	# (change requires restart)"

	statementTunableIntro = "Recommendations based on %s of available memory and %d CPUs for PostgreSQL %s"
	promptTune            = "Tune memory/parallelism/WAL and other settings?"

	successQuiet = "all settings tuned, no changes needed"

	fmtTunableParam = "%s = %s%s\n"
	fmtLastTuned    = "timescaledb.last_tuned = '%s'"

	lastTunedDateFmt = "2006-01-02 15:04"
	fudgeFactor      = 0.05

	pgMajor96 = "9.6"
	pgMajor10 = "10"
	pgMajor11 = "11"
)

var (
	// allows us to substitute mock versions in tests
	getPGConfigVersionFn = getPGConfigVersion

	pgVersionRegex = regexp.MustCompile("^PostgreSQL ([0-9]+?).([0-9]+?).*")
	pgVersions     = []string{pgMajor11, pgMajor10, pgMajor96}
)

func getPGMajorVersion(binPath string) (string, error) {
	version, err := getPGConfigVersionFn(binPath)
	if err != nil {
		return "", fmt.Errorf(errCouldNotExecuteFmt, binPath, err)
	}
	majorVersion, err := parse.ToPGMajorVersion(string(version))
	if err != nil {
		return "", err
	}
	if !isIn(majorVersion, pgVersions) {
		return "", fmt.Errorf(errUnsupportedMajorFmt, majorVersion)
	}
	return majorVersion, nil
}

// TunerFlags are the flags that control how a Tuner object behaves when it is run.
type TunerFlags struct {
	Memory    string // amount of memory to base recommendations on
	NumCPUs   uint   // number of CPUs to base recommendations on
	PGConfig  string // path to pg_config binary
	ConfPath  string // path to the postgresql.conf file
	DestPath  string // path to output file
	YesAlways bool   // always respond yes to prompts
	Quiet     bool   // show only the bare necessities
	UseColor  bool   // use color in output
	DryRun    bool   // whether to actual persist changes to disk
}

// Tuner represents the tuning program for TimescaleDB.
type Tuner struct {
	handler *ioHandler
	cfs     *configFileState
	flags   *TunerFlags
}

// initializeIOHandler sets up the printer to be used throughout the running of
// the Tuner based on the Tuner's TunerFlags, while also setting the proper
// io.Writers for basic output and error output.
func (t *Tuner) initializeIOHandler(in io.Reader, out io.Writer, outErr io.Writer) {
	var p printer
	if t.flags.UseColor {
		p = &colorPrinter{outErr}
	} else {
		p = &noColorPrinter{outErr}
	}
	t.handler = &ioHandler{
		p:      p,
		br:     bufio.NewReader(in),
		out:    out,
		outErr: outErr,
	}
}

// initializeSystemConfig creates the pgtune.SystemConfig to be used for recommendations
// based on the Tuner's TunerFlags (i.e., whether memory and/or number of CPU cores has
// been overridden).
func (t *Tuner) initializeSystemConfig() (*pgtune.SystemConfig, error) {
	// Some settings are not applicable in some versions,
	// e.g. max_parallel_workers is not available in 9.6
	pgVersion, err := getPGMajorVersion(t.flags.PGConfig)
	if err != nil {
		return nil, err
	}

	// Memory flag needs to be in PostgreSQL format, default is all memory
	var totalMemory uint64
	if t.flags.Memory != "" {
		temp, err := parse.PGFormatToBytes(t.flags.Memory)
		if err != nil {
			return nil, err
		}
		totalMemory = temp
	} else {
		totalMemory = memory.TotalMemory()
	}

	// Default to the number of cores
	cpus := int(t.flags.NumCPUs)
	if t.flags.NumCPUs == 0 {
		cpus = runtime.NumCPU()
	}

	return pgtune.NewSystemConfig(totalMemory, cpus, pgVersion), nil
}

// Run executes the tuning process given the provided flags and looks for input
// on the in io.Reader. Informational messages are written to outErr while
// actual recommendations are written to out.
func (t *Tuner) Run(flags *TunerFlags, in io.Reader, out io.Writer, outErr io.Writer) {
	t.flags = flags
	if t.flags == nil {
		t.flags = &TunerFlags{}
	}
	t.initializeIOHandler(in, out, outErr)

	ifErrHandle := func(err error) {
		if err != nil {
			t.handler.errorExit(err)
		}
	}

	// Before proceeding, make sure we have a valid system config
	config, err := t.initializeSystemConfig()
	ifErrHandle(err)

	// Attempt to find the config file and open it for reading
	filePath := t.flags.ConfPath
	if len(filePath) == 0 {
		filePath, err = getConfigFilePath(runtime.GOOS, config.PGMajorVersion)
		ifErrHandle(err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		t.handler.errorExit(fmt.Errorf("could not open config file for reading: %v", err))
	}
	defer file.Close()

	// Do user verification of the found conf file (if not provided via a flag)
	err = t.processConfFileCheck(filePath)
	ifErrHandle(err)

	// Generate current conf file state
	cfs, err := getConfigFileState(file)
	ifErrHandle(err)
	t.cfs = cfs

	// Write backup
	backupPath, err := cfs.Backup()
	t.handler.p.Statement("Writing backup to:")
	printFn(os.Stderr, backupPath+"\n\n")
	ifErrHandle(err)

	// Process the tuning of settings
	if t.flags.Quiet {
		err = t.processQuiet(config)
		ifErrHandle(err)
	} else {
		err = t.processSharedLibLine()
		ifErrHandle(err)

		printFn(os.Stderr, "\n")
		err = t.promptUntilValidInput(promptTune+promptYesNo, newYesNoChecker(""))
		if err == nil {
			err = t.processTunables(config)
			ifErrHandle(err)
		} else if err.Error() != "" { // error msg of "" is response when user selects no to tuning
			t.handler.errorExit(err)
		}
	}

	// Append the current time to mark when database was last tuned
	lastTunedLine := fmt.Sprintf(fmtLastTuned, time.Now().Format(lastTunedDateFmt))
	cfs.lines = append(cfs.lines, lastTunedLine)

	// Wrap up: Either write it out, or show success in --dry-run
	if !t.flags.DryRun {
		outPath := t.flags.DestPath
		if len(outPath) == 0 {
			outPath, err = filepath.Abs(filePath)
			if err != nil {
				t.handler.exit(1, "could not open %s for writing: %v", filePath, err)
			}
		}

		t.handler.p.Statement("Saving changes to: " + outPath)
		f, err := os.Create(outPath)
		if err != nil {
			t.handler.exit(1, "could not open %s for writing: %v", outPath, err)
		}
		_, err = t.cfs.WriteTo(f)
		ifErrHandle(err)
	} else {
		t.handler.p.Statement("Success, but not writing due to --dry-run flag")
	}
}

// promptUntilValidInput continually prompts the user via handler's output to
// answer a question provided in prompt until an acceptable answer is given.
func (t *Tuner) promptUntilValidInput(prompt string, checker promptChecker) error {
	if t.flags.YesAlways {
		return nil
	}
	for {
		t.handler.p.Prompt(prompt)
		resp, err := t.handler.br.ReadString('\n')
		if err != nil {
			return fmt.Errorf("could not parse response: %v", err)
		}
		r := strings.ToLower(strings.TrimSpace(resp))
		ok, err := checker.Check(r)
		if ok || err != nil {
			return err
		}
	}
}

// processConfFileCheck handles the interactions for checking whether Tuner is
// using the correct conf file. If provided by a flag, it should skip prompting
// error if somehow the provided filePath differs from the flag value. Otherwise,
// it prompts the user for input on whether the provided path is correct.
func (t *Tuner) processConfFileCheck(filePath string) error {
	t.handler.p.Statement(statementConfFileCheck)
	printFn(os.Stderr, filePath+"\n\n")
	if len(t.flags.ConfPath) == 0 {
		checker := newYesNoChecker(errConfFileCheckNo)
		err := t.promptUntilValidInput(promptCorrect+promptYesNo, checker)
		if err != nil {
			return err
		}
	} else if t.flags.ConfPath != filePath {
		return fmt.Errorf(errConfFileMismatchFmt, t.flags.ConfPath, filePath)
	}
	return nil
}

// processNoSharedLibLine goes through interactions with the user if the
// shared_preload_libraries line is completely missing from the conf file.
func (t *Tuner) processNoSharedLibLine() error {
	t.handler.p.Statement(statementSharedLibNotFound)
	checker := newYesNoChecker(errSharedLibNeeded)
	err := t.promptUntilValidInput("Append to end? "+promptYesNo, checker)
	if err != nil {
		return err
	}

	t.cfs.lines = append(t.cfs.lines, plainSharedLibLine)
	t.handler.p.Success("appending shared_preload_libraries = 'timescaledb' to end of configuration file")

	return nil
}

// processSharedLibLine goes through the interactions to handle updating the
// conf file to correctly support timescaledb in the shared_preload_libraries config param.
func (t *Tuner) processSharedLibLine() error {
	if t.cfs.sharedLibResult == nil {
		return t.processNoSharedLibLine()
	}

	res := t.cfs.sharedLibResult
	idx := res.idx
	newLine := updateSharedLibLine(t.cfs.lines[idx], res)
	if newLine == t.cfs.lines[idx] { // already valid, nothing to do
		t.handler.p.Success(successSharedLibCorrect)
	} else {
		t.handler.p.Statement("shared_preload_libraries needs to be updated")
		t.handler.p.Statement(currentLabel)
		// want to print without trailing comments to reduce clutter
		currWithoutComments := fmt.Sprintf("%sshared_preload_libraries = '%s'", res.commentGroup, res.libs)
		printFn(t.handler.out, currWithoutComments+"\n")

		t.handler.p.Statement(recommendLabel)
		// want to print without trailing comments to reduce clutter
		recWithoutComments := updateSharedLibLine(currWithoutComments, res)
		printFn(t.handler.out, recWithoutComments+"\n")

		checker := newYesNoChecker(errSharedLibNeeded)
		err := t.promptUntilValidInput(promptOkay+promptYesNo, checker)
		if err != nil {
			return err
		}
		t.cfs.lines[idx] = newLine // keep trailing comments when writing
		t.handler.p.Success(successSharedLibUpdated)
	}
	return nil
}

// checkIfShouldShowSetting iterates through a group of settings defined by keys
// and checks whether the setting should be shown to the user for modification.
// The criteria for being shown is either:
// (a) the setting is missing altogether,
// (b) the setting is currently commented out,
// (c) OR the setting's recommended value is far enough away from its current value.
func checkIfShouldShowSetting(keys []string, parseResults map[string]*tunableParseResult, recommender pgtune.Recommender) (map[string]bool, error) {
	show := make(map[string]bool)
	for _, k := range keys {
		r := parseResults[k]

		// if the setting was not found on pass through, should show our rec
		if r == nil {
			show[k] = true
			continue
		}

		rv := getFloatParser(recommender)

		// parse the value already there; if unparseable, should show our rec
		curr, err := rv.ParseFloat(r.value)
		if err != nil {
			show[k] = true
			continue
		}

		// get and parse our recommendation; fail if for we can't
		rec := recommender.Recommend(k)
		target, err := rv.ParseFloat(rec)
		if err != nil {
			return nil, fmt.Errorf("unexpected parsing problem: %v", err)
		}

		// only show if our recommendation is significantly different, or config is commented
		if !isCloseEnough(curr, target, fudgeFactor) || r.commented {
			show[k] = true
		}
	}
	return show, nil
}

func (t *Tuner) processSettingsGroup(sg pgtune.SettingsGroup) error {
	label := sg.Label()
	quiet := t.flags.Quiet
	if !quiet {
		printFn(os.Stdout, "\n")
		t.handler.p.Statement(fmt.Sprintf("%s%s settings recommendations", strings.ToUpper(label[:1]), label[1:]))
	}
	keys := sg.Keys()
	recommender := sg.GetRecommender()

	// Get a map of only the settings that are missing, commented out, or not "close enough" to our recommendation.
	show, err := checkIfShouldShowSetting(keys, t.cfs.tuneParseResults, recommender)
	if err != nil {
		return err
	}

	// Settings that need to be changed exist...
	if len(show) > 0 {
		// Decorator for a function fn, where only the lines that need to be updated
		// are processed
		doWithVisibile := func(fn func(r *tunableParseResult)) {
			for _, k := range keys {
				if _, ok := show[k]; !ok {
					continue
				}
				r, ok := t.cfs.tuneParseResults[k]
				if !ok {
					r = &tunableParseResult{idx: -1, missing: true, key: k}
				}
				fn(r)
			}
		}

		// Display extra helpful info in non-quiet mode
		if !quiet {
			// Display current settings, but only those with new recommendations
			t.handler.p.Statement(currentLabel)
			doWithVisibile(func(r *tunableParseResult) {
				if r.idx == -1 {
					t.handler.p.Error("missing", r.key)
					return
				}
				format := fmtTunableParam
				if r.commented {
					format = "#" + format
				}
				printFn(os.Stdout, format, r.key, r.value, "") // don't print comment, too cluttered
			})

			// Now display recommendations, but only those with new recommendations
			t.handler.p.Statement(recommendLabel)
		}
		// Recommendations are always displayed, but the label above may not be
		doWithVisibile(func(r *tunableParseResult) {
			printFn(os.Stdout, fmtTunableParam, r.key, recommender.Recommend(r.key), "") // don't print comment, too cluttered
		})

		// Prompt the user for input (only in non-quiet mode)
		if !quiet {
			checker := newSkipChecker(label + " settings still need to be tuned, please re-run or do so manually")
			err := t.promptUntilValidInput(promptOkay+promptSkip, checker)
			if err == errSkip {
				t.handler.p.Error("warning", label+" settings left alone, but still need tuning")
				return nil
			} else if err != nil {
				return err
			}
			t.handler.p.Success(label + " settings will be updated")
		}

		// If we reach here, it means the user accepted our recommendations, so update the lines
		doWithVisibile(func(r *tunableParseResult) {
			newLine := fmt.Sprintf(fmtTunableParam, r.key, recommender.Recommend(r.key), r.extra) // do write comment into file
			if r.idx == -1 {
				t.cfs.lines = append(t.cfs.lines, newLine)
			} else {
				t.cfs.lines[r.idx] = newLine
			}
		})
	} else if !quiet { // nothing to tune
		t.handler.p.Success(label + " settings are already tuned")
	}

	return nil
}

// processTunables handles user interactions for updating the conf file when it comes
// to parameters than be tuned, e.g. memory.
func (t *Tuner) processTunables(config *pgtune.SystemConfig) error {
	quiet := t.flags.Quiet
	if !quiet {
		t.handler.p.Statement(statementTunableIntro, parse.BytesToDecimalFormat(config.Memory), config.CPUs, config.PGMajorVersion)
	}
	tunables := []string{
		pgtune.MemoryLabel,
		pgtune.ParallelLabel,
		pgtune.WALLabel,
		pgtune.MiscLabel,
	}

	for _, label := range tunables {
		sg := pgtune.GetSettingsGroup(label, config)
		r := sg.GetRecommender()
		if !r.IsAvailable() {
			continue
		}
		err := t.processSettingsGroup(sg)
		if err != nil {
			return err
		}
	}
	return nil
}

// processQuiet handles the iteractions when the user wants "quiet" output.
func (t *Tuner) processQuiet(config *pgtune.SystemConfig) error {
	t.handler.p.Statement(statementTunableIntro, parse.BytesToDecimalFormat(config.Memory), config.CPUs, config.PGMajorVersion)

	// Replace the print function with a version that counts how many times it
	// is invoked so we can know whether to prompt the user or not. It doesn't
	// make sense to ask for a yes/no if nothing would change.
	changedSettings := uint64(0)
	oldPrintFn := printFn
	printFn = func(w io.Writer, format string, args ...interface{}) (int, error) {
		changedSettings++
		return oldPrintFn(w, format, args...)
	}
	// Need to restore the old printFn whenever this returns
	defer func() {
		printFn = oldPrintFn
	}()

	if t.cfs.sharedLibResult == nil { // shared lib line is missing completely
		printFn(os.Stdout, plainSharedLibLine+"\n")
		t.cfs.lines = append(t.cfs.lines, plainSharedLibLine)
		t.cfs.sharedLibResult = parseLineForSharedLibResult(plainSharedLibLineWithComments)
		t.cfs.sharedLibResult.idx = len(t.cfs.lines) - 1
	} else { // exists, but may need to be updated
		sharedIdx := t.cfs.sharedLibResult.idx
		newLine := updateSharedLibLine(t.cfs.lines[sharedIdx], t.cfs.sharedLibResult)
		if newLine != t.cfs.lines[sharedIdx] {
			printFn(os.Stdout, newLine+"\n")
			t.cfs.lines[sharedIdx] = newLine
		}
	}

	// print out all tunables that need to be changed
	err := t.processTunables(config)
	if err != nil {
		return err
	}
	if changedSettings > 0 {
		printFn(os.Stdout, fmtLastTuned+"\n", time.Now().Format(lastTunedDateFmt))
		checker := newYesNoChecker("not using these settings could lead to suboptimal performance")
		err = t.promptUntilValidInput("Use these recommendations? "+promptYesNo, checker)
		if err != nil {
			return err
		}
	} else {
		t.handler.p.Success(successQuiet)
	}

	return nil
}
