// Package tstune provides the needed resources and interfaces to create and run
// a tuning program for TimescaleDB.
package tstune

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pbnjay/memory"
	"github.com/timescale/timescaledb-tune/internal/parse"
	"github.com/timescale/timescaledb-tune/pkg/pgtune"
)

const (
	osMac                = "darwin"
	osLinux              = "linux"
	fileNameMac          = "/usr/local/var/postgres/postgresql.conf"
	fileNameDebianFmt    = "/etc/postgresql/%s/main/postgresql.conf"
	fileNameRPMFmt       = "/var/lib/pgsql/%s/data/postgresql.conf"
	fileNameArch         = "/var/lib/postgres/data/postgresql.conf"
	errConfigNotFoundFmt = "could not find postgresql.conf at any of these locations:\n%v"

	currentLabel   = "Current:"
	recommendLabel = "Recommended:"

	promptOkay  = "Is this okay? "
	promptYesNo = "[(y)es/(n)o]: "
	promptSkip  = "[(y)es/(s)kip/(q)uit]: "

	errSharedLibNeeded         = "`timescaledb` needs to be added to shared_preload_libraries in order for it to work"
	successSharedLibCorrect    = "shared_preload_libraries is set correctly"
	successSharedLibUpdated    = "shared_preload_libraries will be updated"
	statementSharedLibNotFound = "Unable to find shared_preload_libraries in configuration file"
	plainSharedLibLine         = "shared_preload_libraries = 'timescaledb'	# (change requires restart)"

	statementTunableIntro = "Recommendations based on %s of available memory and %d CPUs"
	promptTune            = "Tune memory/parallelism/WAL and other settings?"
	tunableMemory         = "memory"
	tunableWAL            = "WAL"
	tunableParallelism    = "parallelism"
	tunableOther          = "miscellaneous"

	fmtTunableParam = "%s = %s%s\n"

	fudgeFactor = 0.05
)

var (
	osStatFn = os.Stat

	pgVersions = []string{"10", "9.6"}
)

func isCloseEnough(actual, target, fudge float64) bool {
	return math.Abs((target-actual)/target) <= fudge
}

// TunerFlags are the flags that control how a Tuner object behaves when it is run.
type TunerFlags struct {
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

func (t *Tuner) initializeIOHandler(out io.Writer, outErr io.Writer) {
	var p printer
	if t.flags.UseColor {
		p = &colorPrinter{outErr}
	} else {
		p = &noColorPrinter{outErr}
	}
	t.handler = &ioHandler{p: p, out: out, outErr: outErr}
}

// Run executes the tuning process given the provided flags and looks for input
// on the in io.Reader. Informational messages are written to outErr while
// actual recommendations are written to out.
func (t *Tuner) Run(flags *TunerFlags, in io.Reader, out io.Writer, outErr io.Writer) {
	t.flags = flags
	if t.flags == nil {
		t.flags = &TunerFlags{}
	}
	t.initializeIOHandler(out, outErr)

	ifErrHandle := func(err error) {
		if err != nil {
			t.handler.errorExit(err)
		}
	}
	var err error

	// attempt to find the config file and open it for reading
	fileName := t.flags.ConfPath
	if len(fileName) == 0 {
		fileName, err = getConfigFilePath(runtime.GOOS)
		ifErrHandle(err)
	}

	file, err := os.Open(fileName)
	if err != nil {
		t.handler.errorExit(fmt.Errorf("could not open config file for reading: %v", err))
	}
	defer file.Close()

	br := bufio.NewReader(in)
	t.handler.br = br

	t.handler.p.Statement("Using postgresql.conf at this path:")
	printFn(os.Stderr, fileName+"\n\n")
	if len(t.flags.ConfPath) == 0 {
		checker := newYesNoChecker("please pass in the correct path to postgresql.conf using the --conf-path flag")
		err = t.promptUntilValidInput("Is this the correct path? "+promptYesNo, checker)
		if err != nil {
			t.handler.exit(0, err.Error())
		}
	}

	// write backup

	cfs, err := getConfigFileState(file)
	ifErrHandle(err)
	t.cfs = cfs

	totalMemory := memory.TotalMemory()
	cpus := runtime.NumCPU()

	if t.flags.Quiet {
		err = t.processQuiet(totalMemory, cpus)
		ifErrHandle(err)
	} else {
		err = t.processSharedLibLine()
		ifErrHandle(err)

		printFn(os.Stderr, "\n")
		err = t.promptUntilValidInput(promptTune+promptYesNo, newYesNoChecker(""))
		if err == nil {
			err = t.processTunables(totalMemory, cpus, false /* quiet */)
			ifErrHandle(err)
		} else if err.Error() != "" { // error msg of "" is response when user selects no to tuning
			t.handler.errorExit(err)
		}
	}

	if !t.flags.DryRun {
		outPath := t.flags.DestPath
		if len(outPath) == 0 {
			outPath, err = filepath.Abs(fileName)
			if err != nil {
				t.handler.exit(1, "could not open %s for writing: %v", fileName, err)
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

// fileExists is a simple check for stating if a file exists and if any error
// occurs it returns false.
func fileExists(name string) bool {
	// for our purposes, any error is a problem, so assume it does not exist
	if _, err := osStatFn(name); err != nil {
		return false
	}
	return true
}

// getConfigFilePath attempts to find the postgresql.conf file using path heuristics
// for different operating systems. If successful it returns the full path to
// the file; otherwise, it returns with an empty path and error.
func getConfigFilePath(os string) (string, error) {
	tried := []string{}
	try := func(format string, args ...interface{}) string {
		fileName := fmt.Sprintf(format, args...)
		tried = append(tried, fileName)
		if fileExists(fileName) {
			return fileName
		}
		return ""
	}
	switch {
	case os == osMac:
		fileName := try(fileNameMac)
		if fileName != "" {
			return fileName, nil
		}
	case os == osLinux:
		for _, v := range pgVersions {
			fileName := try(fileNameDebianFmt, v)
			if fileName != "" {
				return fileName, nil
			}
		}
		for _, v := range pgVersions {
			fileName := try(fileNameRPMFmt, v)
			if fileName != "" {
				return fileName, nil
			}
		}

		fileName := try(fileNameArch)
		if fileName != "" {
			return fileName, nil
		}
	}
	return "", fmt.Errorf(errConfigNotFoundFmt, strings.Join(tried, "\n"))
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

// updateSharedLibLine takes a given line that matched the shared_preload_libraries
// regex and updates it to validly include the 'timescaledb' extension.
func updateSharedLibLine(line string, parseResult *sharedLibResult) string {
	res := line
	if parseResult.commented {
		res = strings.Replace(res, parseResult.commentGroup, "", 1)
	}

	if parseResult.hasTimescale {
		return res
	}
	newLibsVal := "= '"
	if len(parseResult.libs) > 0 {
		newLibsVal += parseResult.libs + ","
	}
	newLibsVal += extName + "'"
	replaceVal := "= '" + parseResult.libs + "'"
	res = strings.Replace(res, replaceVal, newLibsVal, 1)

	return res
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

func getFloatParser(r pgtune.Recommender) floatParser {
	switch r.(type) {
	case *pgtune.MemoryRecommender:
		return &bytesFloatParser{}
	case *pgtune.WALRecommender:
		return &bytesFloatParser{}
	default:
		return &numericFloatParser{}
	}
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

func (t *Tuner) processSettingsGroup(sg pgtune.SettingsGroup, quiet bool) error {
	label := sg.Label()
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
func (t *Tuner) processTunables(totalMemory uint64, cpus int, quiet bool) error {
	if !quiet {
		t.handler.p.Statement(statementTunableIntro, parse.BytesToDecimalFormat(totalMemory), cpus)
	}
	tunables := []string{
		pgtune.MemoryLabel,
		pgtune.ParallelLabel,
		pgtune.WALLabel,
		pgtune.MiscLabel,
	}

	for _, label := range tunables {
		sg := pgtune.GetSettingsGroup(label, totalMemory, cpus)
		r := sg.GetRecommender()
		if !r.IsAvailable() {
			continue
		}
		err := t.processSettingsGroup(sg, quiet)
		if err != nil {
			return err
		}
	}
	return nil
}

// processQuiet handles the iteractions when the user wants "quiet" output.
func (t *Tuner) processQuiet(totalMemory uint64, cpus int) error {
	t.handler.p.Statement(statementTunableIntro, parse.BytesToDecimalFormat(totalMemory), cpus)
	if t.cfs.sharedLibResult == nil {
		printFn(os.Stdout, plainSharedLibLine+"\n")
		t.cfs.lines = append(t.cfs.lines, plainSharedLibLine)
		t.cfs.sharedLibResult = parseLineForSharedLibResult(plainSharedLibLine)
		t.cfs.sharedLibResult.idx = len(t.cfs.lines) - 1
	} else {
		sharedIdx := t.cfs.sharedLibResult.idx
		newLine := updateSharedLibLine(t.cfs.lines[sharedIdx], t.cfs.sharedLibResult)
		if newLine != t.cfs.lines[sharedIdx] {
			printFn(os.Stdout, newLine+"\n")
			t.cfs.lines[sharedIdx] = newLine
		}
	}

	_ = t.processTunables(totalMemory, cpus, true /* quiet */)
	checker := newYesNoChecker("not using these settings could lead to suboptimal performance")
	err := t.promptUntilValidInput("Use these recommendations? "+promptYesNo, checker)
	if err != nil {
		return err
	}

	return nil
}
