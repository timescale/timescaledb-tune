// timescaledb-tune analyzes a user's postgresql.conf file to make sure it is
// ready and tuned to use TimescaleDB. It checks that the library is properly
// listed as a shared preload library and analyzes the memory settings to make
// sure they are reasonably set for the machine's resources.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/pbnjay/memory"
)

const (
	errConfigNotFoundFmt = "could not find postgresql.conf at any of these locations:\n%v"

	extName = "timescaledb"

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

type flags struct {
	confPath  string // path to the postgresql.conf file
	destPath  string // path to output file
	pgconfig  string // path to the pg_config binary
	yesAlways bool   // always respond yes to prompts
	quiet     bool   // show only the bare necessities
	useColor  bool   // use color in output
	dryRun    bool   // whether to actual persist changes to disk
}

// sharedLibResult holds the results of extracting/parsing the shared_preload_libraries
// line of a postgresql.conf file.
type sharedLibResult struct {
	idx          int    // the line index where this result was parsed
	commented    bool   // whether the line is currently commented out (i.e., prepended by #)
	hasTimescale bool   // whether 'timescaledb' appears in the list of libraries
	commentGroup string // the combination of # + spaces that appear before the key / value
	libs         string // the string value of the libraries currently set in the config file
}

// ioHandler manages the reading and writing of the application
type ioHandler struct {
	p  printer       // handles output
	br *bufio.Reader // handles input
}

func (h *ioHandler) errorExit(err error) {
	h.exit(1, err.Error())
}

func (h *ioHandler) exit(errCode int, format string, args ...interface{}) {
	h.p.Error("exit", format, args...)
	os.Exit(errCode)
}

// Flag vars
var (
	f           flags
	sharedRegex = regexp.MustCompile("(#+?\\s*)?shared_preload_libraries = '(.*?)'.*")
	errNeedEdit = fmt.Errorf("need to edit")

	printFn    = fmt.Printf
	pgVersions = []string{"10", "9.6"}
)

// Parse args
func init() {
	flag.StringVar(&f.confPath, "conf-path", "", "Path to postgresql.conf. If blank, heuristics will be used to find it")
	flag.StringVar(&f.pgconfig, "pgconfig", "pg_config", "Path to pg_config binary")
	flag.StringVar(&f.destPath, "out-path", "", "Path to write the new configuration file. If blank, will use the same file that is read from")
	flag.BoolVar(&f.yesAlways, "yes", false, "Answer 'yes' to every prompt")
	flag.BoolVar(&f.quiet, "quiet", false, "Show only the total recommendations at the end")
	flag.BoolVar(&f.useColor, "color", true, "Use color in output (works best on dark terminals)")
	flag.BoolVar(&f.dryRun, "dry-run", false, "Whether to just show the changes without overwriting the configuration file")
	flag.Parse()
}

func fileExists(name string) bool {
	// for our purposes, any error is a problem, so assume it does not exist
	if _, err := os.Stat(name); err != nil {
		return false
	}
	return true
}

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
	case os == "darwin":
		fileName := try("/usr/local/var/postgres/postgresql.conf")
		if fileName != "" {
			return fileName, nil
		}
	case os == "linux":
		for _, v := range pgVersions {
			fileName := try("/etc/postgresql/%s/main/postgresql.conf", v)
			if fileName != "" {
				return fileName, nil
			}
		}
		for _, v := range pgVersions {
			fileName := try("/var/lib/pgsql/%s/data/postgresql.conf", v)
			if fileName != "" {
				return fileName, nil
			}
		}
	}
	return "", fmt.Errorf(errConfigNotFoundFmt, strings.Join(tried, "\n"))
}

// promptUntilValidInput continually prompts the user via handler's output to
// answer a question provided in prompt until an acceptable answer is given.
func promptUntilValidInput(handler *ioHandler, prompt string, checker promptChecker) error {
	if f.yesAlways {
		return nil
	}
	for {
		handler.p.Prompt(prompt)
		resp, err := handler.br.ReadString('\n')
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

func parseLineForSharedLibResult(line string) *sharedLibResult {
	res := sharedRegex.FindStringSubmatch(line)
	if len(res) > 0 {
		return &sharedLibResult{
			commented:    len(res[1]) > 0,
			hasTimescale: strings.Contains(res[2], extName),
			commentGroup: res[1],
			libs:         res[2],
		}
	}
	return nil
}

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

func processNoSharedLibLine(handler *ioHandler, cfs *configFileState) error {
	handler.p.Statement(statementSharedLibNotFound)
	checker := newYesNoChecker(errSharedLibNeeded)
	err := promptUntilValidInput(handler, "Append to end? "+promptYesNo, checker)
	if err != nil {
		return err
	}

	cfs.lines = append(cfs.lines, plainSharedLibLine)
	handler.p.Success("appending shared_preload_libraries = 'timescaledb' to end of configuration file")

	return nil
}

func processSharedLibLine(handler *ioHandler, cfs *configFileState) error {
	if cfs.sharedLibResult == nil {
		return processNoSharedLibLine(handler, cfs)
	}

	sharedIdx := cfs.sharedLibResult.idx
	newLine := updateSharedLibLine(cfs.lines[sharedIdx], cfs.sharedLibResult)
	if newLine == cfs.lines[sharedIdx] {
		handler.p.Success(successSharedLibCorrect)
	} else {
		handler.p.Statement("shared_preload_libraries needs to be updated")
		handler.p.Statement(currentLabel)
		printFn(cfs.lines[sharedIdx] + "\n")
		handler.p.Statement(recommendLabel)
		printFn(newLine + "\n")
		checker := newYesNoChecker(errSharedLibNeeded)
		err := promptUntilValidInput(handler, promptOkay+promptYesNo, checker)
		if err != nil {
			return err
		}
		cfs.lines[sharedIdx] = newLine
		handler.p.Success(successSharedLibUpdated)
	}
	return nil
}

func checkIfShouldShowSetting(keys []string, parseResults map[string]*tunableParseResult, recommender recommender) (map[string]bool, error) {
	show := make(map[string]bool)
	for _, k := range keys {
		r := parseResults[k]

		// if the setting was not found on pass through, should show our rec
		if r == nil {
			show[k] = true
			continue
		}

		// parse the value already there; if unparseable, should show our rec
		curr, err := parsers[k](r.value)
		if err != nil {
			show[k] = true
			continue
		}

		// get and parse our recommendation; fail if for we can't
		rec := recommender.Recommend(k)
		target, err := parsers[k](rec)
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

type tuneSettings struct {
	label string
	rec   recommender
	keys  []string
}

func processSettingsGroup(handler *ioHandler, cfs *configFileState, ts *tuneSettings, quiet bool) error {
	if !quiet {
		printFn("\n")
		handler.p.Statement(fmt.Sprintf("%s%s settings recommendations", strings.ToUpper(ts.label[:1]), ts.label[1:]))
	}

	// Get a map of only the settings that are missing, commented out, or not "close enough" to our recommendation.
	show, err := checkIfShouldShowSetting(ts.keys, cfs.tuneParseResults, ts.rec)
	if err != nil {
		return err
	}

	// Settings that need to be changed exist...
	if len(show) > 0 {
		// Decorator for a function fn, where only the lines that need to be updated
		// are processed
		doWithVisibile := func(fn func(r *tunableParseResult)) {
			for _, k := range ts.keys {
				if _, ok := show[k]; !ok {
					continue
				}
				r, ok := cfs.tuneParseResults[k]
				if !ok {
					r = &tunableParseResult{idx: -1, missing: true, key: k}
				}
				fn(r)
			}
		}

		// Display extra helpful info in non-quiet mode
		if !quiet {
			// Display current settings, but only those with new recommendations
			handler.p.Statement(currentLabel)
			doWithVisibile(func(r *tunableParseResult) {
				if r.idx == -1 {
					handler.p.Error("missing", r.key)
					return
				}
				printFn(cfs.lines[r.idx] + "\n")
			})

			// Now display recommendations, but only those with new recommendations
			handler.p.Statement(recommendLabel)
		}
		// Recommendations are always displayed, but the label above may not be
		doWithVisibile(func(r *tunableParseResult) {
			printFn(fmtTunableParam, r.key, ts.rec.Recommend(r.key), r.extra)
		})

		// Prompt the user for input (only in non-quiet mode)
		if !quiet {
			checker := newSkipChecker(ts.label + " settings still need to be tuned, please re-run or do so manually")
			err := promptUntilValidInput(handler, promptOkay+promptSkip, checker)
			if err == errSkip {
				handler.p.Error("warning", ts.label+" settings left alone, but still need tuning")
				return nil
			} else if err != nil {
				return err
			}
			handler.p.Success(ts.label + " settings will be updated")
		}

		// If we reach here, it means the user accepted our recommendations, so update the lines
		doWithVisibile(func(r *tunableParseResult) {
			newLine := fmt.Sprintf(fmtTunableParam, r.key, ts.rec.Recommend(r.key), r.extra)
			if r.idx == -1 {
				cfs.lines = append(cfs.lines, newLine)
			} else {
				cfs.lines[r.idx] = newLine
			}
		})
	} else if !quiet { // nothing to tune
		handler.p.Success(ts.label + " settings are already tuned")
	}

	return nil
}

func processTunables(handler *ioHandler, cfs *configFileState, totalMemory uint64, cpus int, quiet bool) {
	tune := func(label string, r recommender, keys []string) {
		ts := tuneSettings{label, r, keys}
		err := processSettingsGroup(handler, cfs, &ts, quiet)
		if err != nil {
			handler.errorExit(err)
		}
	}
	if !quiet {
		handler.p.Statement(statementTunableIntro, bytesFormat(totalMemory), cpus)
	}

	mr := &memoryRecommender{totalMemory, cpus}
	tune(tunableMemory, mr, memoryKeys)

	pr := &parallelRecommender{cpus}
	tune(tunableParallelism, pr, parallelKeys)

	wr := &walRecommender{totalMemory}
	tune(tunableWAL, wr, walKeys)

	mir := &miscRecommender{}
	tune(tunableOther, mir, otherKeys)
}

func processQuiet(handler *ioHandler, cfs *configFileState, totalMemory uint64, cpus int) error {
	handler.p.Statement(statementTunableIntro, bytesFormat(totalMemory), cpus)
	if cfs.sharedLibResult == nil {
		printFn(plainSharedLibLine + "\n")
		cfs.lines = append(cfs.lines, plainSharedLibLine)
		cfs.sharedLibResult = parseLineForSharedLibResult(plainSharedLibLine)
		cfs.sharedLibResult.idx = len(cfs.lines) - 1
	} else {
		sharedIdx := cfs.sharedLibResult.idx
		newLine := updateSharedLibLine(cfs.lines[sharedIdx], cfs.sharedLibResult)
		if newLine != cfs.lines[sharedIdx] {
			printFn(newLine + "\n")
			cfs.lines[sharedIdx] = newLine
		}
	}

	processTunables(handler, cfs, totalMemory, cpus, true /* quiet */)
	checker := newYesNoChecker("not using these settings could lead to suboptimal performance")
	err := promptUntilValidInput(handler, "Use these recommendations? "+promptYesNo, checker)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var err error
	// setup IO
	var p printer
	if f.useColor {
		p = &colorPrinter{}
	} else {
		p = &noColorPrinter{}
	}
	handler := &ioHandler{p: p}

	// attempt to find the config file and open it for reading
	fileName := f.confPath
	if len(fileName) == 0 {
		fileName, err = getConfigFilePath(runtime.GOOS)
		if err != nil {
			handler.errorExit(err)
		}
	}

	file, err := os.Open(fileName)
	if err != nil {
		handler.errorExit(fmt.Errorf("could not open config file for reading: %v", err))
	}
	defer file.Close()

	br := bufio.NewReader(os.Stdin)
	handler.br = br

	handler.p.Statement("Using postgresql.conf at this path:")
	printFn(fileName + "\n\n")
	if len(f.confPath) == 0 {
		checker := newYesNoChecker("please pass in the correct path to postgresql.conf using the --conf-path flag")
		err = promptUntilValidInput(handler, "Is this the correct path? "+promptYesNo, checker)
		if err != nil {
			handler.exit(0, err.Error())
		}
	}

	// write backup

	cfs, err := getConfigFileState(file)
	if err != nil {
		handler.errorExit(err)
	}

	totalMemory := memory.TotalMemory()
	cpus := runtime.NumCPU()

	if f.quiet {
		err = processQuiet(handler, cfs, totalMemory, cpus)
		if err != nil {
			handler.errorExit(err)
		}
	} else {
		err = processSharedLibLine(handler, cfs)
		if err != nil {
			handler.errorExit(err)
		}

		printFn("\n")
		err = promptUntilValidInput(handler, promptTune+promptYesNo, newYesNoChecker(""))
		if err == nil {
			processTunables(handler, cfs, totalMemory, cpus, false /* quiet */)
		}
	}

	if !f.dryRun {
		outPath := f.destPath
		if len(outPath) == 0 {
			outPath, err = filepath.Abs(fileName)
			if err != nil {
				handler.exit(1, "could not open %s for writing: %v", fileName, err)
			}
		}

		handler.p.Statement("Saving changes to: " + outPath)
		f, err := os.Create(outPath)
		if err != nil {
			handler.exit(1, "could not open %s for writing: %v", outPath, err)
		}
		_, err = cfs.WriteTo(f)
		if err != nil {
			handler.errorExit(err)
		}
	} else {
		handler.p.Statement("Success, but not writing due to --dry-run flag")
	}
}
