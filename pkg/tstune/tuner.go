// Package tstune provides the needed resources and interfaces to create and run
// a tuning program for TimescaleDB.
package tstune

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/pbnjay/memory"
	"github.com/timescale/timescaledb-tune/internal/parse"
	"github.com/timescale/timescaledb-tune/pkg/pgtune"
)

const (
	// Version is the version of this library
	Version = "0.18.1"

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

	errCouldNotGetBackupsFmt = "could not get list of backup files: %v"
	errNoBackupsFound        = "no backup files found"
	errNoBackupRestored      = "no backup restored"
	errCouldNotRestoreFmt    = "could not restore %s: %v"
	backupListFmt            = "%d) %s (%v ago)\n"
	promptBackupNumber       = "Use which backup? Number or (q)uit: "
	successRestore           = "restored successfully"

	errSharedLibNeeded             = "`timescaledb` needs to be added to shared_preload_libraries in order for it to work"
	successSharedLibCorrect        = "shared_preload_libraries is set correctly"
	successSharedLibUpdated        = "shared_preload_libraries will be updated"
	statementSharedLibNotFound     = "Unable to find shared_preload_libraries in configuration file"
	plainSharedLibLine             = "shared_preload_libraries = 'timescaledb'"
	plainSharedLibLineWithComments = plainSharedLibLine + "	# (change requires restart)"

	statementTunableIntro = "Recommendations based on %s of available memory and %d CPUs for PostgreSQL %s"
	promptTune            = "Tune memory/parallelism/WAL and other settings? "

	successQuiet = "all settings tuned, no changes needed"

	errCouldNotWriteFmt = "could not open %s for writing: %v"

	fmtTunableParam = "%s = %s%s"

	fudgeFactor = 0.05
)

// allows us to substitute mock versions in tests
var filepathAbsFn = filepath.Abs

// TunerFlags are the flags that control how a Tuner object behaves when it is run.
type TunerFlags struct {
	Memory       string // amount of memory to base recommendations on
	NumCPUs      uint   // number of CPUs to base recommendations on
	WALDiskSize  string // disk size of WAL to base recommendations on
	PGVersion    string // major version of PostgreSQL to base recommendations on
	PGConfig     string // path to pg_config binary
	MaxConns     uint64 // max number of database connections
	MaxBGWorkers int    // max number of background workers
	ConfPath     string // path to the postgresql.conf file
	DestPath     string // path to output file
	YesAlways    bool   // always respond yes to prompts
	Quiet        bool   // show only the bare necessities
	UseColor     bool   // use color in output
	DryRun       bool   // whether to actually persist changes to disk
	Restore      bool   // whether to restore a backup
	Profile      string // a specific "mode" to provide recommendations tailored to a special workload type, e.g. "promscale"
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
	var err error

	// Some settings are not applicable in some versions,
	// e.g. max_parallel_workers is not available in 9.6
	var pgVersion string
	if t.flags.PGVersion != "" {
		if err = validatePGMajorVersion(t.flags.PGVersion); err != nil {
			return nil, err
		}
		pgVersion = t.flags.PGVersion
	} else {
		pgVersion, err = getPGMajorVersion(t.flags.PGConfig)
		if err != nil {
			return nil, err
		}
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

	// WAL Disk size needs to be in PostgreSQL format, default is 0
	var walDisk uint64 = 0
	if t.flags.WALDiskSize != "" {
		temp, err := parse.PGFormatToBytes(t.flags.WALDiskSize)
		if err != nil {
			return nil, err
		}
		walDisk = temp
	}

	// Default to the number of cores
	cpus := int(t.flags.NumCPUs)
	if t.flags.NumCPUs == 0 {
		cpus = runtime.NumCPU()
	}

	// Use default BG Workers if not provided
	maxBGWorkers := int(t.flags.MaxBGWorkers)
	if t.flags.MaxBGWorkers == 0 {
		maxBGWorkers = pgtune.MaxBackgroundWorkersDefault
	}

	return pgtune.NewSystemConfig(totalMemory, cpus, pgVersion, walDisk, t.flags.MaxConns, maxBGWorkers)
}

func (t *Tuner) restore(r restorer, filePath string) error {
	files, err := getBackups()
	if err != nil {
		return fmt.Errorf(errCouldNotGetBackupsFmt, err)
	}

	if len(files) == 0 {
		return fmt.Errorf(errNoBackupsFound)
	}

	// Reverse sort the list so most recent backups are first
	sort.Strings(files)
	for i := len(files)/2 - 1; i >= 0; i-- {
		opp := len(files) - 1 - i
		files[i], files[opp] = files[opp], files[i]
	}
	t.handler.p.Statement("Available backups (most recent first):")
	for i, file := range files {
		now := time.Now()
		name := path.Base(file)
		datePart := strings.Replace(name, backupFilePrefix, "", -1)
		// no need to check the error, as getBackups does that for us
		when, _ := time.ParseInLocation(backupDateFmt, datePart, now.Location())
		ago := now.Sub(when)
		fmt.Fprintf(t.handler.out, backupListFmt, i+1, name, parse.PrettyDuration(ago))
	}
	fmt.Fprintf(t.handler.out, "\n")
	checker := newNumberedListChecker(len(files), errNoBackupRestored)
	// call directly to forcePromptUntilValidInput since --yes should not apply here
	err = t.forcePromptUntilValidInput(promptBackupNumber, checker)
	if err != nil {
		return err
	}

	backupPath := files[checker.response-1]
	shortBackupName := path.Base(backupPath)

	t.handler.p.Statement("Restoring '%s'...", shortBackupName)
	err = r.Restore(backupPath, filePath)
	if err != nil {
		return fmt.Errorf(errCouldNotRestoreFmt, shortBackupName, err)
	}
	t.handler.p.Success(successRestore)

	return nil
}

// verifyTunerFlags evaluates the provided tuner flags for validity and augments
// values where needed, e.g. expanding dirnames into full paths.
func verifyTunerFlags(flags *TunerFlags) (*TunerFlags, error) {
	if flags == nil {
		flags = &TunerFlags{}
	}

	// As path is also used to mean directory, and most PostgreSQL tools
	// themselves work with pgdata/bindir as directories, we ensure these
	// paths can also be specified as a directory
	flags.PGConfig = dirPathToFile(flags.PGConfig, "pg_config")
	flags.ConfPath = dirPathToFile(flags.ConfPath, "postgresql.conf")
	flags.DestPath = dirPathToFile(flags.DestPath, "postgresql.conf")

	return flags, nil
}

// Run executes the tuning process given the provided flags and looks for input
// on the in io.Reader. Informational messages are written to outErr while
// actual recommendations are written to out.
func (t *Tuner) Run(flags *TunerFlags, in io.Reader, out io.Writer, outErr io.Writer) {
	t.flags, _ = verifyTunerFlags(flags)
	t.initializeIOHandler(in, out, outErr)

	ifErrHandle := func(err error) {
		if err != nil {
			t.handler.errorExit(err)
		}
	}

	profile, err := pgtune.ParseProfile(t.flags.Profile)
	ifErrHandle(err)
	if profile != pgtune.DefaultProfile {
		t.handler.p.Statement("Tuning with profile: %s", profile)
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

	// If restore flag, restore and that's it
	if t.flags.Restore {
		r := &fsRestorer{}
		err = t.restore(r, filePath)
		ifErrHandle(err)
		return // do nothing else!
	}

	// Generate current conf file state
	t.cfs, err = getConfigFileState(file)
	ifErrHandle(err)

	// Write backup
	if !t.flags.DryRun {
		backupPath, err := backup(t.cfs)
		t.handler.p.Statement("Writing backup to:")
		fmt.Fprintf(t.handler.outErr, backupPath+"\n\n")
		ifErrHandle(err)
	}

	// Process the tuning of settings
	if t.flags.Quiet {
		err = t.processQuiet(config, profile)
		ifErrHandle(err)
	} else {
		err = t.processSharedLibLine()
		ifErrHandle(err)

		fmt.Fprintf(t.handler.outErr, "\n")
		err = t.promptUntilValidInput(promptTune+promptYesNo, newYesNoChecker(""))
		if err == nil {
			err = t.processTunables(config, profile)
			ifErrHandle(err)
		} else if err.Error() != "" { // error msg of "" is response when user selects no to tuning
			t.handler.errorExit(err)
		}
	}

	// Add our params to the conf file, and cleanup because old versions of Tuner
	// were noisy and left these params each time.
	t.processOurParams()
	t.cfs.ProcessLines(getRemoveDuplicatesProcessors(ourParams)...)

	// Wrap up: Either write it out, or show success in --dry-run
	if !t.flags.DryRun {
		err = t.writeConfFile(filePath)
		ifErrHandle(err)
	} else {
		t.handler.p.Statement("Success, but not writing due to --dry-run flag")
	}
}

// promptUntilValidInput continually prompts the user via handler's output to
// answer a question provided in prompt until an acceptable answer is given, or
// returns immediately if the Yes flag is passed in.
func (t *Tuner) promptUntilValidInput(prompt string, checker promptChecker) error {
	if t.flags.YesAlways {
		return nil
	}
	return t.forcePromptUntilValidInput(prompt, checker)
}

// forcePromptUntilValidInput continually prompts the user to answer a question
// provided in prompt until an acceptable answer is given. It is not affected by
// the presence of the Yes flag.
func (t *Tuner) forcePromptUntilValidInput(prompt string, checker promptChecker) error {
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
	fmt.Fprintf(t.handler.outErr, filePath+"\n\n")
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

	t.cfs.lines = append(t.cfs.lines, &configLine{content: plainSharedLibLine})
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
	newLine := updateSharedLibLine(t.cfs.lines[idx].content, res)
	if newLine == t.cfs.lines[idx].content { // already valid, nothing to do
		t.handler.p.Success(successSharedLibCorrect)
	} else {
		t.handler.p.Statement("shared_preload_libraries needs to be updated")
		t.handler.p.Statement(currentLabel)
		// want to print without trailing comments to reduce clutter
		currWithoutComments := fmt.Sprintf("%sshared_preload_libraries = '%s'", res.commentGroup, res.libs)
		fmt.Fprintf(t.handler.out, currWithoutComments+"\n")

		t.handler.p.Statement(recommendLabel)
		// want to print without trailing comments to reduce clutter
		recWithoutComments := updateSharedLibLine(currWithoutComments, res)
		fmt.Fprintf(t.handler.out, recWithoutComments+"\n")

		checker := newYesNoChecker(errSharedLibNeeded)
		err := t.promptUntilValidInput(promptOkay+promptYesNo, checker)
		if err != nil {
			return err
		}
		t.cfs.lines[idx] = &configLine{content: newLine} // keep trailing comments when writing
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

		rv := pgtune.GetFloatParser(recommender)

		// get and parse our recommendation; fail if for we can't
		rec := recommender.Recommend(k)

		switch {
		case rec == pgtune.NoRecommendation:
			// don't bother adding it to the map. no recommendation
			continue
		case r.commented:
			show[k] = true
		case r.value == rec:
			// don't bother adding it to the map. no recommendation
			continue

		}

		// parse the value already there; if unparseable, should show our rec
		curr, err := rv.ParseFloat(k, r.value)
		if err != nil {
			show[k] = true
			continue
		}

		target, err := rv.ParseFloat(k, rec)
		if err != nil {
			return nil, fmt.Errorf("unexpected parsing problem: %v", err)
		}

		// only show if our recommendation is significantly different, or config is commented
		if !isCloseEnough(curr, target, fudgeFactor) {
			show[k] = true
		}
	}
	return show, nil
}

func (t *Tuner) processSettingsGroup(sg pgtune.SettingsGroup, profile pgtune.Profile) error {
	label := sg.Label()
	quiet := t.flags.Quiet
	if !quiet {
		fmt.Fprintf(t.handler.out, "\n")
		t.handler.p.Statement(fmt.Sprintf("%s%s settings recommendations", strings.ToUpper(label[:1]), label[1:]))
	}
	keys := sg.Keys()
	recommender := sg.GetRecommender(profile)

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
				// don't bother displaying current settings for keys for which we have no recommendation
				rec := recommender.Recommend(r.key)
				if rec == pgtune.NoRecommendation {
					return
				}
				if r.idx == -1 {
					t.handler.p.Error("missing", r.key)
					return
				}
				format := fmtTunableParam + "\n"
				if r.commented {
					format = "#" + format
				}
				fmt.Fprintf(t.handler.out, format, r.key, r.value, "") // don't print comment, too cluttered
			})

			// Now display recommendations, but only those with new recommendations
			t.handler.p.Statement(recommendLabel)
		}
		// Recommendations are always displayed, but the label above may not be
		doWithVisibile(func(r *tunableParseResult) {
			rec := recommender.Recommend(r.key)
			// skip keys for which we have no recommendation
			if rec == pgtune.NoRecommendation {
				return
			}
			fmt.Fprintf(t.handler.out, fmtTunableParam+"\n", r.key, rec, "") // don't print comment, too cluttered
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
			rec := recommender.Recommend(r.key)
			if rec == pgtune.NoRecommendation {
				return
			}
			newLine := &configLine{content: fmt.Sprintf(fmtTunableParam, r.key, rec, r.extra)} // do write comment into file
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
func (t *Tuner) processTunables(config *pgtune.SystemConfig, profile pgtune.Profile) error {
	quiet := t.flags.Quiet
	if !quiet {
		t.handler.p.Statement(statementTunableIntro, parse.BytesToDecimalFormat(config.Memory), config.CPUs, config.PGMajorVersion)
	}
	tunables := []string{
		pgtune.MemoryLabel,
		pgtune.ParallelLabel,
		pgtune.WALLabel,
		pgtune.BgwriterLabel,
		pgtune.MiscLabel,
	}

	for _, label := range tunables {
		sg := pgtune.GetSettingsGroup(label, config)
		r := sg.GetRecommender(profile)
		if !r.IsAvailable() {
			continue
		}
		err := t.processSettingsGroup(sg, profile)
		if err != nil {
			return err
		}
	}
	return nil
}

// processOurParams manages the parameters that the Tuner generates, such as
// the timescaledb.last_tuned parameter.
func (t *Tuner) processOurParams() {
	findRegexes := map[string]*regexp.Regexp{}
	for _, param := range ourParams {
		findRegexes[param] = keyToRegexQuoted(param)
	}
	foundLines := map[string]int{}

	// Since we usually append our settings to the end, it is more efficient
	// to work backwards. The basic idea is to check each line against our
	// map of regexes and if it matches we've found the latest occurrence of
	// that parameter, so we can 1) skip testing other regexes and 2) move
	// the parameter from findRegexes map to foundLines. Once each has been
	// found, we can quit searching (or go until the end).
	for i := len(t.cfs.lines) - 1; i >= 0; i-- {
		if len(findRegexes) == 0 {
			break
		}
		for param, regex := range findRegexes {
			if found := parseWithRegex(t.cfs.lines[i].content, regex); found != nil {
				foundLines[param] = i
				delete(findRegexes, param)
				continue
			}
		}
	}

	// For each one we found, replace in place.
	for param, idx := range foundLines {
		t.cfs.lines[idx] = &configLine{content: ourParamString(param)}
	}

	// For each one we did NOT find, append to the end. Use our params so they
	// are always added in the same order (easier to test)
	for _, param := range ourParams {
		if _, ok := findRegexes[param]; ok {
			line := &configLine{content: ourParamString(param)}
			t.cfs.lines = append(t.cfs.lines, line)
		}
	}
}

// counterWriter is used to count how many writes are done, to determine whether
// to show additional dialog during the CLI
type counterWriter struct {
	count uint64
	w     io.Writer
}

func (w *counterWriter) Write(p []byte) (int, error) {
	w.count++
	return w.w.Write(p)
}

// processQuiet handles the iteractions when the user wants "quiet" output.
func (t *Tuner) processQuiet(config *pgtune.SystemConfig, profile pgtune.Profile) error {
	t.handler.p.Statement(statementTunableIntro, parse.BytesToDecimalFormat(config.Memory), config.CPUs, config.PGMajorVersion)

	// Replace the print function with a version that counts how many times it
	// is invoked so we can know whether to prompt the user or not. It doesn't
	// make sense to ask for a yes/no if nothing would change.
	newWriter := &counterWriter{0, t.handler.out}
	t.handler.out = newWriter
	defer func() {
		t.handler.out = newWriter.w
	}()

	if t.cfs.sharedLibResult == nil { // shared lib line is missing completely
		fmt.Fprintf(t.handler.out, plainSharedLibLine+"\n")
		t.cfs.lines = append(t.cfs.lines, &configLine{content: plainSharedLibLine})
		t.cfs.sharedLibResult = parseLineForSharedLibResult(plainSharedLibLineWithComments)
		t.cfs.sharedLibResult.idx = len(t.cfs.lines) - 1
	} else { // exists, but may need to be updated
		sharedIdx := t.cfs.sharedLibResult.idx
		newLine := updateSharedLibLine(t.cfs.lines[sharedIdx].content, t.cfs.sharedLibResult)
		if newLine != t.cfs.lines[sharedIdx].content {
			fmt.Fprintf(t.handler.out, newLine+"\n")
			t.cfs.lines[sharedIdx] = &configLine{content: newLine}
		}
	}

	// print out all tunables that need to be changed
	err := t.processTunables(config, profile)
	if err != nil {
		return err
	}
	if newWriter.count > 0 {
		for _, param := range ourParams {
			fmt.Fprintf(t.handler.out, ourParamString(param)+"\n")
		}
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

func (t *Tuner) writeConfFile(confPath string) error {
	var err error
	outPath := t.flags.DestPath
	if len(outPath) == 0 {
		outPath, err = filepathAbsFn(confPath)
		if err != nil {
			return fmt.Errorf(errCouldNotWriteFmt, confPath, err)
		}
	}

	t.handler.p.Statement("Saving changes to: " + outPath)
	f, err := osCreateFn(outPath)
	if err != nil {
		return fmt.Errorf(errCouldNotWriteFmt, outPath, err)
	}
	defer f.Close()

	_, err = t.cfs.WriteTo(f)
	if err != nil {
		return fmt.Errorf(errCouldNotWriteFmt, outPath, err)
	}
	return nil
}
