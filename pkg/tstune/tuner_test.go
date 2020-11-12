package tstune

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pbnjay/memory"
	"github.com/timescale/timescaledb-tune/internal/parse"
	"github.com/timescale/timescaledb-tune/pkg/pgtune"
	"github.com/timescale/timescaledb-tune/pkg/pgutils"
)

func newTunerWithDefaultFlags(handler *ioHandler, cfs *configFileState) *Tuner {
	return &Tuner{handler, cfs, &TunerFlags{}}
}

func TestVerifyTunerFlags(t *testing.T) {
	defaultPGConfig := "pg_config"

	cases := []struct {
		desc         string
		input        *TunerFlags
		flagPGConfig string
	}{
		{
			desc:         "nil should become default values",
			input:        nil,
			flagPGConfig: "",
		},
		{
			desc:         "pg_config should be appended to a directory",
			input:        &TunerFlags{PGConfig: "."},
			flagPGConfig: defaultPGConfig,
		},
		{
			desc:         "pg_config should not be appended to a regular file",
			input:        &TunerFlags{PGConfig: "ghost"},
			flagPGConfig: "ghost",
		},
	}

	for _, c := range cases {
		flags, _ := verifyTunerFlags(c.input)
		if flags.PGConfig != c.flagPGConfig {
			t.Errorf("%s: unexpected error (PGConfig): got %v, wanted: %v", c.desc, flags.PGConfig, c.flagPGConfig)
		}
	}
}

func TestTunerInitializeIOHandler(t *testing.T) {
	tuner := &Tuner{nil, nil, &TunerFlags{}}
	tuner.flags.UseColor = true
	tuner.initializeIOHandler(os.Stdin, os.Stdout, os.Stderr)

	switch x := tuner.handler.p.(type) {
	case *colorPrinter:
	default:
		t.Errorf("non-color printer for UseColor flag: got %T", x)
	}

	tuner.flags.UseColor = false
	tuner.initializeIOHandler(os.Stdin, os.Stdout, os.Stderr)

	switch x := tuner.handler.p.(type) {
	case *noColorPrinter:
	default:
		t.Errorf("color printer for UseColor=false flag: got %T", x)
	}
}

func TestTunerInitializeSystemConfig(t *testing.T) {
	totalMemory := memory.TotalMemory()
	okPGConfig := "pg_config"
	okPGVersion := pgutils.MajorVersion11
	cases := []struct {
		desc          string
		flagPGConfig  string
		flagMemory    string
		flagNumCPUs   uint
		flagPGVersion string
		flagWALDisk   string
		wantMemory    uint64
		wantCPUs      int
		wantPGVersion string
		wantWALDisk   uint64
		errMsg        string
	}{
		{
			desc:         "bad pgconfig flag",
			flagPGConfig: "foo",
			errMsg:       "could not execute `foo --version`: executable file not found in $PATH",
		},
		{
			desc:         "bad memory flag",
			flagPGConfig: okPGConfig,
			flagMemory:   "foo",
			errMsg:       "incorrect PostgreSQL bytes format: 'foo'",
		},
		{
			desc:          "bad pgversion flag",
			flagPGVersion: "9.5",
			errMsg:        fmt.Sprintf(errUnsupportedMajorFmt, "9.5"),
		},
		{
			desc:         "bad wal disk flag",
			flagPGConfig: okPGConfig,
			flagWALDisk:  "400 gigs",
			errMsg:       "incorrect PostgreSQL bytes format: '400 gigs'",
		},
		{
			desc:          "use mem flag only",
			flagPGConfig:  okPGConfig,
			flagMemory:    "1" + parse.GB,
			wantMemory:    1 * parse.Gigabyte,
			wantCPUs:      runtime.NumCPU(),
			wantPGVersion: okPGVersion,
		},
		{
			desc:          "use cpu flag only",
			flagPGConfig:  okPGConfig,
			flagNumCPUs:   2,
			wantMemory:    totalMemory,
			wantCPUs:      2,
			wantPGVersion: okPGVersion,
		},
		{
			desc:          "use pg-version flag only",
			flagPGVersion: pgutils.MajorVersion10,
			wantMemory:    totalMemory,
			wantCPUs:      runtime.NumCPU(),
			wantPGVersion: pgutils.MajorVersion10,
		},
		{
			desc:          "use wal-disk flag only",
			flagPGConfig:  okPGConfig,
			flagWALDisk:   "4GB",
			wantMemory:    totalMemory,
			wantCPUs:      runtime.NumCPU(),
			wantPGVersion: okPGVersion,
			wantWALDisk:   4 * parse.Gigabyte,
		},
		{
			desc:          "all flags",
			flagPGConfig:  okPGConfig,
			flagMemory:    "128" + parse.GB,
			flagNumCPUs:   1,
			flagPGVersion: pgutils.MajorVersion96,
			wantMemory:    128 * parse.Gigabyte,
			wantCPUs:      1,
			wantPGVersion: pgutils.MajorVersion96,
		},
		{
			desc:          "none flags",
			flagPGConfig:  okPGConfig,
			wantMemory:    totalMemory,
			wantCPUs:      runtime.NumCPU(),
			wantPGVersion: okPGVersion,
		},
	}

	oldVersionFn := getPGConfigVersionFn
	getPGConfigVersionFn = func(binPath string) (string, error) {
		if binPath == okPGConfig {
			return fmt.Sprintf("PostgreSQL %s.0", okPGVersion), nil
		}
		return "", exec.ErrNotFound
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			tuner := &Tuner{nil, nil, &TunerFlags{
				PGConfig:    c.flagPGConfig,
				PGVersion:   c.flagPGVersion,
				Memory:      c.flagMemory,
				NumCPUs:     c.flagNumCPUs,
				WALDiskSize: c.flagWALDisk,
			}}
			config, err := tuner.initializeSystemConfig()
			if len(c.errMsg) == 0 {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if got := config.Memory; got != c.wantMemory {
					t.Errorf("incorrect amount of memory: got %d want %d", got, c.wantMemory)
				}

				if got := config.CPUs; got != c.wantCPUs {
					t.Errorf("incorrect number of CPUs: got %d want %d", got, c.wantCPUs)
				}
				if got := config.PGMajorVersion; got != c.wantPGVersion {
					t.Errorf("incorrect pg version: got %s want %s", got, c.wantPGVersion)
				}
				if got := config.WALDiskSize; got != c.wantWALDisk {
					t.Errorf("incorrect WAL disk: got %d want %d", got, c.wantWALDisk)
				}
			} else {
				if err == nil {
					t.Errorf("unexpected lack of error")
				} else if got := err.Error(); got != c.errMsg {
					t.Errorf("incorrect error: got\n%s\nwant\n%s", got, c.errMsg)
				}
			}
		})
	}

	getPGConfigVersionFn = oldVersionFn
}

type testRestorer struct {
	errMsg string
}

func (r *testRestorer) Restore(backupPath, confPath string) error {
	if r.errMsg != "" {
		return fmt.Errorf(r.errMsg)
	}
	return nil
}

func setupDefaultTestIO(input string) *ioHandler {
	buf := bytes.NewBufferString(input)
	br := bufio.NewReader(buf)
	out := &testWriter{}
	return &ioHandler{
		p:      &testPrinter{},
		br:     br,
		out:    out,
		outErr: out,
	}
}

func TestRestore(t *testing.T) {
	errGlob := "glob error"
	now := time.Now()
	baseFile := path.Join(os.TempDir(), backupFilePrefix)
	time1 := now.Add(-5 * time.Minute).Format(backupDateFmt)
	time2 := now.Add(-3 * time.Hour).Format(backupDateFmt)
	correctFile1 := baseFile + time1
	correctFile2 := baseFile + time2
	shortFile1 := backupFilePrefix + time1
	wantPrint1 := fmt.Sprintf(backupListFmt, 1, backupFilePrefix+time1, parse.PrettyDuration(now.Sub(now.Add(-5*time.Minute))))
	wantPrint2 := fmt.Sprintf(backupListFmt, 2, backupFilePrefix+time2, parse.PrettyDuration(now.Sub(now.Add(-3*time.Hour))))

	cases := []struct {
		desc          string
		filePath      string
		onDiskFiles   []string
		input         string
		statements    uint64
		prompts       uint64
		successes     uint64
		wantPrints    []string
		globErr       bool
		errMsg        string
		restoreErrMsg string
	}{
		{
			desc:        "error in getBackups makes error",
			onDiskFiles: []string{"foo"},
			globErr:     true,
			errMsg:      fmt.Sprintf(errCouldNotGetBackupsFmt, errGlob),
		},
		{
			desc:        "no backups returned",
			onDiskFiles: []string{},
			errMsg:      errNoBackupsFound,
		},
		{
			desc:        "only one backup",
			onDiskFiles: []string{correctFile1},
			input:       "1\n",
			statements:  2,
			prompts:     1,
			successes:   1,
			wantPrints:  []string{wantPrint1},
		},
		{
			desc:        "two backups in order",
			onDiskFiles: []string{correctFile1, correctFile2},
			input:       "1\n",
			statements:  2,
			prompts:     1,
			successes:   1,
			wantPrints:  []string{wantPrint1, wantPrint2},
		},
		{
			desc:        "two backups wrong order",
			onDiskFiles: []string{correctFile1, correctFile2},
			input:       "1\n",
			statements:  2,
			prompts:     1,
			successes:   1,
			wantPrints:  []string{wantPrint1, wantPrint2},
		},
		{
			desc:        "quit after backups list",
			onDiskFiles: []string{correctFile1},
			input:       "q\n",
			statements:  1,
			prompts:     1,
			wantPrints:  []string{wantPrint1},
			errMsg:      errNoBackupRestored,
		},
		{
			desc:        "two backups, incorrect numbers",
			onDiskFiles: []string{correctFile1, correctFile2},
			input:       "0\n5\n2\n",
			statements:  2,
			prompts:     3,
			successes:   1,
			wantPrints:  []string{wantPrint1, wantPrint2},
		},
		{
			desc:          "one backup, failed restore",
			onDiskFiles:   []string{correctFile1},
			input:         "1\n",
			statements:    2,
			prompts:       1,
			wantPrints:    []string{wantPrint1},
			restoreErrMsg: "no restore",
			errMsg:        fmt.Sprintf(errCouldNotRestoreFmt, shortFile1, "no restore"),
		},
	}

	oldFilepathGlobFn := filepathGlobFn
	for _, c := range cases {
		filepathGlobFn = func(_ string) ([]string, error) {
			if c.globErr {
				return nil, fmt.Errorf(errGlob)
			}
			return c.onDiskFiles, nil
		}

		handler := setupDefaultTestIO(c.input)
		tuner := newTunerWithDefaultFlags(handler, nil)

		err := tuner.restore(&testRestorer{c.restoreErrMsg}, c.filePath)
		if c.errMsg == "" && err != nil {
			t.Errorf("%s: unexpected error: got %v", c.desc, err)
		} else if c.errMsg != "" {
			if err == nil {
				t.Errorf("%s: unexpected lack of error", c.desc)
			} else if got := err.Error(); got != c.errMsg {
				t.Errorf("%s: incorrect error: got\n%s\nwant\n%s", c.desc, got, c.errMsg)
			}
		}

		tp := tuner.handler.p.(*testPrinter)
		if got := tp.statementCalls; got != c.statements {
			t.Errorf("%s: incorrect number of statements: got %d want %d", c.desc, got, c.statements)
		}

		if c.errMsg == "" {
			out := handler.out.(*testWriter)
			// subtract one for the ending newline
			if got := len(out.lines) - 1; got != len(c.wantPrints) {
				t.Errorf("%s: incorrect number of prints: got %d want %d", c.desc, got, len(c.wantPrints))
			}

			if got := tp.promptCalls; got != c.prompts {
				t.Errorf("%s: incorrect number of prompts: got %d want %d", c.desc, got, c.prompts)
			}

			if got := tp.successCalls; got != c.successes {
				t.Errorf("%s: incorrect number of successes: got %d want %d", c.desc, got, c.successes)
			}
		}
	}
	filepathGlobFn = oldFilepathGlobFn
}

type limitChecker struct {
	limit     uint64
	calls     uint64
	shouldErr bool
	checks    []string
}

func (c *limitChecker) Check(r string) (bool, error) {
	c.calls++
	c.checks = append(c.checks, r)
	if c.calls >= c.limit {
		if c.shouldErr {
			return false, fmt.Errorf("errored")
		}
		return true, nil
	}
	return false, nil
}

func newTunerWithDefaultFlagsForInputs(t *testing.T, input string, lines []string) *Tuner {
	handler := setupDefaultTestIO(input)
	cfs := newConfigFileStateFromSlice(t, lines)
	return newTunerWithDefaultFlags(handler, cfs)
}

func TestPromptUntilValidInput(t *testing.T) {
	cases := []struct {
		desc      string
		limit     uint64
		shouldErr bool
	}{
		{
			desc:      "always returns true",
			limit:     1,
			shouldErr: false,
		},
		{
			desc:      "always errors",
			limit:     1,
			shouldErr: true,
		},
		{
			desc:      "skip once, then success",
			limit:     2,
			shouldErr: false,
		},
		{
			desc:      "skip once, then error",
			limit:     2,
			shouldErr: true,
		},
		{
			desc:      "skip twice",
			limit:     3,
			shouldErr: false,
		},
		{
			desc:      "check all are lower",
			limit:     5,
			shouldErr: false,
		},
	}

	testInput := "foo\nFoo\nFOO\nfOo\nfOO\n\n"
	for _, c := range cases {
		handler := setupDefaultTestIO(testInput)
		tuner := newTunerWithDefaultFlags(handler, nil)
		checker := &limitChecker{limit: c.limit, shouldErr: c.shouldErr}
		err := tuner.promptUntilValidInput("test prompt", checker)
		if err != nil && !c.shouldErr {
			t.Errorf("%s: unexpected error: %v", c.desc, err)
		} else if err == nil && c.shouldErr {
			t.Errorf("%s: unexpected lack of error", c.desc)
		}

		if got := handler.p.(*testPrinter).promptCalls; got != c.limit {
			t.Errorf("%s: incorrect number of prompts: got %d want %d", c.desc, got, c.limit)
		}

		if got := len(checker.checks); got != int(c.limit) {
			t.Errorf("%s: incorrect number of checks: got %d want %d", c.desc, got, c.limit)
		}

		for i, check := range checker.checks {
			if check != strings.ToLower(check) {
				t.Errorf("%s: check was not lowercase: %s (idx %d)", c.desc, check, i)
			}
		}
	}

	// check --yes case works
	for _, c := range cases {
		handler := setupDefaultTestIO(testInput)
		tuner := &Tuner{handler: handler, flags: &TunerFlags{YesAlways: true}}
		checker := &limitChecker{limit: c.limit, shouldErr: c.shouldErr}
		err := tuner.promptUntilValidInput("test prompt", checker)
		if err != nil {
			t.Errorf("%s: unexpected error in yesAlways case: %v", c.desc, err)
		}
	}
}

func TestProcessConfFileCheck(t *testing.T) {
	cases := []struct {
		desc        string
		input       string
		promptCalls uint64
		filePath    string
		flagPath    string
		errMsg      string
	}{
		{
			desc:        "success - provided path",
			input:       "",
			promptCalls: 0,
			filePath:    "/path/to/postgresql.conf",
			flagPath:    "/path/to/postgresql.conf",
		},
		{
			desc:        "success - append default filename to directory",
			input:       "",
			promptCalls: 0,
			filePath:    "postgresql.conf",
			flagPath:    ".",
		},
		{
			desc:        "success - input yes",
			input:       "yeS\n",
			promptCalls: 1,
			filePath:    "/path/to/postgresql.conf",
		},
		{
			desc:        "success - eventually yes",
			input:       "si\nyes\n",
			promptCalls: 2,
			filePath:    "/path/to/postgresql.conf",
		},
		{
			desc:        "error - said no",
			input:       "maybe\nno\n",
			promptCalls: 2,
			filePath:    "/path/to/postgresql.conf",
			errMsg:      errConfFileCheckNo,
		},
		{
			desc:     "error - mismatch",
			input:    "",
			filePath: "/path/to/postgresql.conf",
			flagPath: "postgresql.conf",
			errMsg:   fmt.Sprintf(errConfFileMismatchFmt, "postgresql.conf", "/path/to/postgresql.conf"),
		},
	}

	for _, c := range cases {
		tuner := newTunerWithDefaultFlagsForInputs(t, c.input, []string{})
		tuner.flags.ConfPath = dirPathToFile(c.flagPath, "postgresql.conf")

		err := tuner.processConfFileCheck(c.filePath)
		tp := tuner.handler.p.(*testPrinter)
		if got := tp.statementCalls; got != 1 {
			t.Errorf("%s: incorrect number of statements: got %d want %d", c.desc, got, 1)
		} else if got := tp.statements[0]; got != statementConfFileCheck {
			t.Errorf("%s: incorrect statement: got\n%s\nwant\n%s", c.desc, got, statementConfFileCheck)
		}

		if got := tp.promptCalls; got != c.promptCalls {
			t.Errorf("%s: incorrect number of prompt calls: got %d want %d", c.desc, got, c.promptCalls)
		}

		out := tuner.handler.out.(*testWriter)
		if got := len(out.lines); got != 1 {
			t.Errorf("%s: incorrect number of prints: got %d want %d", c.desc, got, 1)
		} else if got := out.lines[0]; got != c.filePath+"\n\n" {
			t.Errorf("%s: incorrect print: got\n%s\nwant\n%s", c.desc, got, c.filePath+"\n\n")
		}

		if len(c.errMsg) == 0 {
			if err != nil {
				t.Errorf("%s: unexpected error: got %v", c.desc, err)
			}
		} else {
			if err == nil {
				t.Errorf("%s: unexpected lack of error", c.desc)
			}
		}
	}
}

func TestProcessNoSharedLibLine(t *testing.T) {
	cases := []struct {
		desc      string
		input     string
		shouldErr bool
		prompts   uint64
	}{
		{
			desc:      "success on first prompt (y)",
			input:     "y\n",
			shouldErr: false,
			prompts:   1,
		},
		{
			desc:      "success on first prompt (yes)",
			input:     "yes\n",
			shouldErr: false,
			prompts:   1,
		},
		{
			desc:      "success on later try",
			input:     " \nYES\n",
			shouldErr: false,
			prompts:   2,
		},
		{
			desc:      "error on first prompt (n)",
			input:     "n\n\n",
			shouldErr: true,
			prompts:   1,
		},
		{
			desc:      "error on first prompt (no)",
			input:     "no\n",
			shouldErr: true,
			prompts:   1,
		},
		{
			desc:      "error on later prompt (n)",
			input:     "x\nx\nNO\n",
			shouldErr: true,
			prompts:   3,
		},
		{
			desc:      "error closed stream",
			input:     "",
			shouldErr: true,
			prompts:   1,
		},
	}
	for _, c := range cases {
		tuner := newTunerWithDefaultFlagsForInputs(t, c.input, []string{})
		err := tuner.processNoSharedLibLine()

		tp := tuner.handler.p.(*testPrinter)
		if got := tp.statementCalls; got != 1 {
			t.Errorf("%s: incorrect number of statements: got %d want %d", c.desc, got, 1)
		}

		if err != nil && !c.shouldErr {
			t.Errorf("%s: unexpected error: %v", c.desc, err)
		} else if err == nil && c.shouldErr {
			t.Errorf("%s: unexpected lack of error", c.desc)
		}

		if got := tp.promptCalls; got != c.prompts {
			t.Errorf("%s: incorrect number of prompts: got %d want %d", c.desc, got, c.prompts)
		}
		if err == nil {
			if got := tp.successCalls; got != 1 {
				t.Errorf("%s: incorrect number of successes: got %d want %d", c.desc, got, 1)
			}
		}
	}
}

func TestProcessSharedLibLine(t *testing.T) {
	okLine := "shared_preload_libraries = 'timescaledb' # (need restart)"
	okLinePrint := "shared_preload_libraries = 'timescaledb'"
	cases := []struct {
		desc       string
		lines      []string
		input      string
		shouldErr  bool
		prompts    uint64
		prints     []string
		statements uint64
		successMsg string
	}{
		{
			desc:       "no change",
			lines:      []string{okLine},
			input:      "\n",
			shouldErr:  false,
			prompts:    0,
			statements: 0,
			successMsg: successSharedLibCorrect,
		},
		{
			desc:       "success on prompt",
			lines:      []string{"#" + okLine},
			input:      "y\n",
			shouldErr:  false,
			prompts:    1,
			statements: 3,
			prints:     []string{"#" + okLinePrint + "\n", okLinePrint + "\n"},
			successMsg: successSharedLibUpdated,
		},
		{
			desc:       "success on 2nd prompt",
			lines:      []string{"  ##  " + okLine},
			input:      " \ny\n",
			shouldErr:  false,
			prompts:    2,
			statements: 3,
			prints:     []string{"##  " + okLinePrint + "\n", okLinePrint + "\n"},
			successMsg: successSharedLibUpdated,
		},
		{
			desc:       "fail",
			lines:      []string{"#" + okLine},
			input:      " \nn\n",
			shouldErr:  true,
			prompts:    2,
			statements: 3,
			prints:     []string{"#" + okLinePrint + "\n", okLinePrint + "\n"},
			successMsg: "",
		},
		{
			desc:       "no shared lib, success",
			lines:      []string{""},
			input:      "y\n",
			shouldErr:  false,
			prompts:    1,
			statements: 1,
			successMsg: "",
		},
		{
			desc:       "no shared lib, fail",
			lines:      []string{""},
			input:      "n\n",
			shouldErr:  true,
			prompts:    1,
			statements: 1,
			successMsg: "",
		},
	}

	for _, c := range cases {
		tuner := newTunerWithDefaultFlagsForInputs(t, c.input, c.lines)

		err := tuner.processSharedLibLine()
		if err != nil && !c.shouldErr {
			t.Errorf("%s: unexpected error: %v", c.desc, err)
		} else if err == nil && c.shouldErr {
			t.Errorf("%s: unexpected lack of err", c.desc)
		}

		tp := tuner.handler.p.(*testPrinter)
		if got := tp.promptCalls; got != c.prompts {
			t.Errorf("%s: incorrect number of prompts: got %d want %d", c.desc, got, c.prompts)
		}
		if got := tp.statementCalls; got != c.statements {
			t.Errorf("%s: incorrect number of statements: got %d want %d", c.desc, got, c.statements)
		}

		if len(c.prints) > 0 {
			out := tuner.handler.out.(*testWriter)
			for i, want := range c.prints {
				if got := out.lines[i]; got != want {
					t.Errorf("%s: incorrect print at %d: got\n%s\nwant\n%s", c.desc, i, got, want)
				}
			}
		}

		if len(c.successMsg) > 0 {
			if got := tp.successes[0]; got != c.successMsg {
				t.Errorf("%s: incorrect success msg: got\n%s\nwant\n%s", c.desc, got, c.successMsg)
			}
		}
	}
}

type badRecommender struct{}

func (r *badRecommender) IsAvailable() bool       { return true }
func (r *badRecommender) Recommend(string) string { return "not a number" }

func TestCheckIfShouldShowSetting(t *testing.T) {
	valSharedBuffers := "2GB"
	valEffective := "6GB"
	valWorkMem := "52428kB"
	valMaintenance := "1GB"
	okSharedBuffers := &tunableParseResult{
		idx:       0,
		commented: false,
		key:       pgtune.SharedBuffersKey,
		value:     valSharedBuffers,
	}
	okEffective := &tunableParseResult{
		idx:       1,
		commented: false,
		key:       pgtune.EffectiveCacheKey,
		value:     valEffective,
	}
	okWorkMem := &tunableParseResult{
		idx:       2,
		commented: false,
		key:       pgtune.WorkMemKey,
		value:     valWorkMem,
	}
	okMaintenance := &tunableParseResult{
		idx:       3,
		commented: false,
		key:       pgtune.MaintenanceWorkMemKey,
		value:     valMaintenance,
	}
	badWorkMem := &tunableParseResult{
		idx:       2,
		commented: false,
		key:       pgtune.WorkMemKey,
		value:     "0B",
	}
	cases := []struct {
		desc         string
		parseResults map[string]*tunableParseResult
		okFudge      []string
		highFudge    []string
		lowFudge     []string
		commented    []string
		want         []string
		errMsg       string
	}{
		{
			desc: "show nothing",
			parseResults: map[string]*tunableParseResult{
				pgtune.SharedBuffersKey:      okSharedBuffers,
				pgtune.EffectiveCacheKey:     okEffective,
				pgtune.WorkMemKey:            okWorkMem,
				pgtune.MaintenanceWorkMemKey: okMaintenance,
			},
			want: []string{},
		},
		{
			desc: "show 1, missing",
			parseResults: map[string]*tunableParseResult{
				pgtune.EffectiveCacheKey:     okEffective,
				pgtune.WorkMemKey:            okWorkMem,
				pgtune.MaintenanceWorkMemKey: okMaintenance,
			},
			want: []string{pgtune.SharedBuffersKey},
		},
		{
			desc: "show 1, unparseable",
			parseResults: map[string]*tunableParseResult{
				pgtune.SharedBuffersKey:      okSharedBuffers,
				pgtune.EffectiveCacheKey:     okEffective,
				pgtune.WorkMemKey:            badWorkMem,
				pgtune.MaintenanceWorkMemKey: okMaintenance,
			},
			want: []string{pgtune.WorkMemKey},
		},
		{
			desc: "show 2, 1 unparseable + 1 missing",
			parseResults: map[string]*tunableParseResult{
				pgtune.EffectiveCacheKey:     okEffective,
				pgtune.WorkMemKey:            badWorkMem,
				pgtune.MaintenanceWorkMemKey: okMaintenance,
			},
			want: []string{pgtune.SharedBuffersKey, pgtune.WorkMemKey},
		},
		{
			desc: "show all, all commented",
			parseResults: map[string]*tunableParseResult{
				pgtune.SharedBuffersKey:      okSharedBuffers,
				pgtune.EffectiveCacheKey:     okEffective,
				pgtune.WorkMemKey:            okWorkMem,
				pgtune.MaintenanceWorkMemKey: okMaintenance,
			},
			commented: []string{pgtune.SharedBuffersKey, pgtune.EffectiveCacheKey, pgtune.WorkMemKey, pgtune.MaintenanceWorkMemKey},
			want:      []string{pgtune.SharedBuffersKey, pgtune.EffectiveCacheKey, pgtune.WorkMemKey, pgtune.MaintenanceWorkMemKey},
		},
		{
			desc: "show one, 1 commented",
			parseResults: map[string]*tunableParseResult{
				pgtune.SharedBuffersKey:      okSharedBuffers,
				pgtune.EffectiveCacheKey:     okEffective,
				pgtune.WorkMemKey:            okWorkMem,
				pgtune.MaintenanceWorkMemKey: okMaintenance,
			},
			commented: []string{pgtune.EffectiveCacheKey},
			want:      []string{pgtune.EffectiveCacheKey},
		},
		{
			desc: "show none, 1 ok fudge",

			parseResults: map[string]*tunableParseResult{
				pgtune.SharedBuffersKey:      okSharedBuffers,
				pgtune.EffectiveCacheKey:     okEffective,
				pgtune.WorkMemKey:            okWorkMem,
				pgtune.MaintenanceWorkMemKey: okMaintenance,
			},
			okFudge:   []string{},
			commented: []string{},
			want:      []string{},
		},
		{
			desc: "show 2, 1 high fudge, 1 low fudge",

			parseResults: map[string]*tunableParseResult{
				pgtune.SharedBuffersKey:      okSharedBuffers,
				pgtune.EffectiveCacheKey:     okEffective,
				pgtune.WorkMemKey:            okWorkMem,
				pgtune.MaintenanceWorkMemKey: okMaintenance,
			},
			highFudge: []string{pgtune.SharedBuffersKey},
			lowFudge:  []string{pgtune.WorkMemKey},
			commented: []string{},
			want:      []string{pgtune.SharedBuffersKey, pgtune.WorkMemKey},
		},
		{
			desc: "show 2, 1 high fudge commented too, 1 low fudge",

			parseResults: map[string]*tunableParseResult{
				pgtune.SharedBuffersKey:      okSharedBuffers,
				pgtune.EffectiveCacheKey:     okEffective,
				pgtune.WorkMemKey:            okWorkMem,
				pgtune.MaintenanceWorkMemKey: okMaintenance,
			},
			highFudge: []string{pgtune.SharedBuffersKey},
			lowFudge:  []string{pgtune.WorkMemKey},
			commented: []string{pgtune.SharedBuffersKey},
			want:      []string{pgtune.SharedBuffersKey, pgtune.WorkMemKey},
		},
	}

	reset := func() {
		okSharedBuffers.commented = false
		okSharedBuffers.value = valSharedBuffers
		okEffective.commented = false
		okEffective.value = valEffective
		okWorkMem.commented = false
		okWorkMem.value = valWorkMem
		okMaintenance.commented = false
		okMaintenance.value = valMaintenance
	}

	for _, c := range cases {
		reset()
		// change those keys who are supposed to be commented out
		for _, k := range c.commented {
			c.parseResults[k].commented = true
		}
		// change values, but still within fudge factor so it shouldn't be shown
		for _, k := range c.okFudge {
			temp, err := parse.PGFormatToBytes(c.parseResults[k].value)
			if err != nil {
				t.Errorf("%s: unexpected err in parsing: %v", c.desc, err)
			}
			temp = temp + uint64(float64(temp)*(fudgeFactor-.01))
			c.parseResults[k].value = parse.BytesToPGFormat(temp)
		}
		// change values to higher fudge factor, so it should be shown
		for _, k := range c.highFudge {
			temp, err := parse.PGFormatToBytes(c.parseResults[k].value)
			if err != nil {
				t.Errorf("%s: unexpected err in parsing: %v", c.desc, err)
			}
			temp = temp + uint64(float64(temp)*(fudgeFactor+.01))
			c.parseResults[k].value = parse.BytesToPGFormat(temp)
		}
		// change values to lower fudge factor, so it should be shown
		for _, k := range c.lowFudge {
			temp, err := parse.PGFormatToBytes(c.parseResults[k].value)
			if err != nil {
				t.Errorf("%s: unexpected err in parsing: %v", c.desc, err)
			}
			temp = temp - uint64(float64(temp)*(fudgeFactor+.01))
			c.parseResults[k].value = parse.BytesToPGFormat(temp)
		}
		mr := pgtune.NewMemoryRecommender(8*parse.Gigabyte, 1, 20)
		show, err := checkIfShouldShowSetting(pgtune.MemoryKeys, c.parseResults, mr)
		if len(c.errMsg) > 0 {

		} else if err != nil {
			t.Errorf("%s: unexpected err: %v", c.desc, err)
		} else {
			if got := len(show); got != len(c.want) {
				t.Errorf("%s: incorrect show length: got %d want %d", c.desc, got, len(c.want))
			}
			for _, k := range c.want {
				if _, ok := show[k]; !ok {
					t.Errorf("%s: key not found: %s", c.desc, k)
				}
			}
		}
	}
}

func TestCheckIfShouldShowSettingErr(t *testing.T) {
	keys := []string{"foo"}
	parseResults := map[string]*tunableParseResult{
		"foo": {value: "5.0"},
	}
	show, err := checkIfShouldShowSetting(keys, parseResults, &badRecommender{})
	if show != nil {
		t.Errorf("show map is not nil: %v", show)
	}
	if err == nil {
		t.Errorf("err is nil")
	}
}

var (
	memSettingsCorrect = []string{
		"shared_buffers = 2GB",
		"work_mem = 26214kB",
		"effective_cache_size = 6GB",
		"maintenance_work_mem = 1GB",
	}
	memSettingsCommented = []string{
		"#shared_buffers = 2GB  # should be uncommented",
		"work_mem = 26214kB",
		"effective_cache_size = 6GB",
		"maintenance_work_mem = 1GB",
	}
	memSettingsWrongVal = []string{
		"shared_buffers = 2GB",
		"work_mem = 0kB				# 0kb is wrong",
		"effective_cache_size = 6GB",
		"maintenance_work_mem = 1GB",
	}
	memSettingsMissing = []string{
		"shared_buffers = 2GB",
		"work_mem = 26214kB",
		// missing effective cache size
		"maintenance_work_mem = 1GB",
	}
	memSettingsCommentWrong = []string{
		"#shared_buffers = 0GB  # should be uncommented, and 2GB",
		"work_mem = 26214kB",
		"effective_cache_size = 6GB",
		"maintenance_work_mem = 0GB  # should be non-0",
	}
	memSettingsCommentWrongMissing = []string{
		"shared_buffers = 2GB",
		// missing work_mem
		"effective_cache_size = 0GB  # should be non-0",
		"#maintenance_work_mem = 1GB  # should be uncommented",
	}
	memSettingsAllWrong = []string{
		"shared_buffers = 0GB",
		"work_mem = 0kB",
		"effective_cache_size = 0GB",
		"maintenance_work_mem = 0GB",
	}
)

const (
	testMaxConns        = 20
	testMem      uint64 = 8 * parse.Gigabyte
	testCPUs            = 4
	testWorkers         = 8
	testWALDisk  uint64 = 0
)

type testSettingsGroup struct {
	keys []string
}

func (sg *testSettingsGroup) Label() string                      { return "foo" }
func (sg *testSettingsGroup) Keys() []string                     { return sg.keys }
func (sg *testSettingsGroup) GetRecommender() pgtune.Recommender { return &badRecommender{} }

func getDefaultSystemConfig(t *testing.T) *pgtune.SystemConfig {
	config, err := pgtune.NewSystemConfig(testMem, testCPUs, pgutils.MajorVersion10, testWALDisk, testMaxConns, testWorkers)
	if err != nil {
		t.Fatalf("unexpected error in config creation: got %v", err)
	}
	return config
}

func TestTunerProcessSettingsGroup(t *testing.T) {
	config := getDefaultSystemConfig(t)
	cases := []struct {
		desc           string
		ts             pgtune.SettingsGroup
		lines          []string
		input          string
		wantStatements uint64
		wantPrompts    uint64
		wantPrints     int
		wantErrors     uint64
		successMsg     string
		shouldErr      bool
	}{
		{
			desc:           "bad recommender",
			ts:             &testSettingsGroup{pgtune.ParallelKeys},
			lines:          []string{fmt.Sprintf("%s = 1.0", pgtune.ParallelKeys[0])},
			wantStatements: 1, // only intro remark
			wantPrints:     1, // one for initial newline
			shouldErr:      true,
		},
		{
			desc:           "no keys, no need to prompt",
			ts:             &testSettingsGroup{},
			lines:          memSettingsCorrect,
			wantStatements: 1, // only intro remark
			wantPrompts:    0,
			wantPrints:     1, // one for initial newline
			successMsg:     "foo settings are already tuned",
			shouldErr:      false,
		},
		{
			desc:           "memory - commented",
			ts:             pgtune.GetSettingsGroup(pgtune.MemoryLabel, config),
			lines:          memSettingsCommented,
			input:          "y\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    1,
			wantPrints:     3, // one for initial newline + one setting, displayed twice
			successMsg:     "memory settings will be updated",
			shouldErr:      false,
		},
		{
			desc:           "memory - wrong",
			ts:             pgtune.GetSettingsGroup(pgtune.MemoryLabel, config),
			lines:          memSettingsWrongVal,
			input:          "y\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    1,
			wantPrints:     3, // one for initial newline + one setting, displayed twice
			successMsg:     "memory settings will be updated",
			shouldErr:      false,
		},
		{
			desc:           "memory - missing",
			ts:             pgtune.GetSettingsGroup(pgtune.MemoryLabel, config),
			lines:          memSettingsMissing,
			input:          "y\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    1,
			wantPrints:     2, // one for initial newline + one setting, displayed once (missing is now in printer.Error)
			wantErrors:     1, // for missing
			successMsg:     "memory settings will be updated",
			shouldErr:      false,
		},
		{
			desc:           "memory - comment+wrong",
			ts:             pgtune.GetSettingsGroup(pgtune.MemoryLabel, config),
			lines:          memSettingsCommentWrong,
			input:          " \ny\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    2, // first input is blank
			wantPrints:     5, // one for initial newline + two settings, displayed twice
			successMsg:     "memory settings will be updated",
			shouldErr:      false,
		},
		{
			desc:           "memory - comment+wrong+missing",
			ts:             pgtune.GetSettingsGroup(pgtune.MemoryLabel, config),
			lines:          memSettingsCommentWrongMissing,
			input:          " \n \ny\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    3, // first input is blank
			wantPrints:     6, // one for initial newline + two settings, displayed twice, 1 setting once
			wantErrors:     1, // for missing
			successMsg:     "memory settings will be updated",
			shouldErr:      false,
		},
		{
			desc:           "memory - all wrong, but skip",
			ts:             pgtune.GetSettingsGroup(pgtune.MemoryLabel, config),
			lines:          memSettingsAllWrong,
			input:          "s\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    1,
			wantPrints:     9, // one for initial newline + four settings, displayed twice
			wantErrors:     1,
			successMsg:     "",
			shouldErr:      false,
		},
		{
			desc:           "memory - all wrong, but quit",
			ts:             pgtune.GetSettingsGroup(pgtune.MemoryLabel, config),
			lines:          memSettingsAllWrong,
			input:          " \nqUIt\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    2,
			wantPrints:     9, // one for initial newline + four settings, displayed twice
			successMsg:     "",
			shouldErr:      true,
		},
		{
			desc:           "memory - all wrong",
			ts:             pgtune.GetSettingsGroup(pgtune.MemoryLabel, config),
			lines:          memSettingsAllWrong,
			input:          "y\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    1,
			wantPrints:     9, // one for initial newline + four settings, displayed twice
			successMsg:     "memory settings will be updated",
			shouldErr:      false,
		},
		{
			desc:           "label capitalized",
			ts:             pgtune.GetSettingsGroup(pgtune.WALLabel, config),
			input:          "y\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    1,
			wantPrints:     4, // one for initial newline + 3 for recommendations
			wantErrors:     3, // everything is missing
			successMsg:     "WAL settings will be updated",
			shouldErr:      false,
		},
	}

	for _, c := range cases {
		tuner := newTunerWithDefaultFlagsForInputs(t, c.input, c.lines)

		err := tuner.processSettingsGroup(c.ts)
		if err != nil && !c.shouldErr {
			t.Errorf("%s: unexpected error: %v", c.desc, err)
		} else if err == nil && c.shouldErr {
			t.Errorf("%s: unexpected lack of error", c.desc)
		}

		tp := tuner.handler.p.(*testPrinter)
		if got := strings.ToUpper(strings.TrimSpace(tp.statements[0])[:1]); got != strings.ToUpper(c.ts.Label()[:1]) {
			t.Errorf("%s: label not capitalized in first statement: got %s want %s", c.desc, got, strings.ToUpper(c.ts.Label()[:1]))
		}

		if got := tp.statementCalls; got != c.wantStatements {
			t.Errorf("%s: incorrect number of statements: got %d want %d", c.desc, got, c.wantStatements)
		}

		if got := tp.promptCalls; got != c.wantPrompts {
			t.Errorf("%s: incorrect number of prompts: got %d want %d", c.desc, got, c.wantPrompts)
		}

		out := tuner.handler.out.(*testWriter)
		if got := len(out.lines); got != c.wantPrints {
			t.Errorf("%s: incorrect number of prints: got %d want %d", c.desc, got, c.wantPrints)
		}

		if got := tp.errorCalls; got != c.wantErrors {
			t.Errorf("%s: incorrect number of errors: got %d want %d", c.desc, got, c.wantErrors)
		} else if len(c.successMsg) > 0 {
			if got := tp.successCalls; got != 1 {
				t.Errorf("%s: incorrect number of successes: got %d want %d", c.desc, got, 1)
			}
			if got := tp.successes[0]; got != c.successMsg {
				t.Errorf("%s: incorrect success message: got\n%s\nwant\n%s\n", c.desc, got, c.successMsg)
			}
		} else if tp.successCalls > 0 {
			t.Errorf("%s: got success without expecting it: %s", c.desc, tp.successes[0])
		}
	}
}

func TestTunerProcessTunables(t *testing.T) {
	check := func(handler *ioHandler, config *pgtune.SystemConfig, wantGroups uint64) {
		// Total number of statements is intro statement and then 3 per group of settings;
		// each group has a heading and then the current/recommended labels.
		wantStatements := uint64(1 + 3*wantGroups)

		tp := handler.p.(*testPrinter)
		if got := tp.statementCalls; got != wantStatements {
			t.Errorf("incorrect number of statements: got %d, want %d", got, wantStatements)
		}

		wantStatement := fmt.Sprintf(statementTunableIntro, parse.BytesToDecimalFormat(config.Memory), config.CPUs, config.PGMajorVersion)
		if got := tp.statements[0]; got != wantStatement {
			t.Errorf("incorrect first statement: got\n%s\nwant\n%s\n", got, wantStatement)
		}

		for i := 2; i < len(tp.statements); i += 3 {
			if got := tp.statements[i]; got != currentLabel {
				t.Errorf("did not get current label as expected: got %s", got)
			}
			if got := tp.statements[i+1]; got != recommendLabel {
				t.Errorf("did not get recommend label as expected: got %s", got)
			}
		}

		idx := 1
		checkStmt := func(want string) {
			if got := tp.statements[idx]; got != want {
				t.Errorf("incorrect statement at %d: got\n%s\nwant\n%s", idx, got, want)
			}
			idx += 3
		}
		checkStmt("Memory settings recommendations")
		if wantGroups > 3 {
			checkStmt("Parallelism settings recommendations")
		}
		checkStmt("WAL settings recommendations")
		checkStmt("Miscellaneous settings recommendations")
	}
	input := "y\ny\ny\ny\n"

	config := getDefaultSystemConfig(t)
	handler := setupDefaultTestIO(input)
	cfs := &configFileState{tuneParseResults: make(map[string]*tunableParseResult)}
	tuner := newTunerWithDefaultFlags(handler, cfs)
	tuner.processTunables(config)
	check(tuner.handler, config, 4)

	config.CPUs = 1
	handler = setupDefaultTestIO(input)
	cfs = &configFileState{tuneParseResults: make(map[string]*tunableParseResult)}
	tuner = newTunerWithDefaultFlags(handler, cfs)
	tuner.processTunables(config)
	check(tuner.handler, config, 3)
}

var (
	wantedQuietCorrectShared   = plainSharedLibLine
	wantedQuietCommentedShared = "#" + plainSharedLibLine
	wantedQuietMissingShared   = "shared_preload_libraries = ''"
	wantedQuietLines           = []string{
		wantedQuietCorrectShared,
		"shared_buffers = 2GB",
		"effective_cache_size = 6GB",
		"maintenance_work_mem = 1GB",
		"work_mem = 26214kB",
		"timescaledb.max_background_workers = 8",
		"max_worker_processes = 15",
		"max_parallel_workers_per_gather = 2",
		"max_parallel_workers = 4",
		"wal_buffers = 16MB",
		"min_wal_size = 512MB",
		"max_wal_size = 1GB",
		"default_statistics_target = 500",
		"random_page_cost = 1.1",
		"checkpoint_completion_target = 0.9",
		fmt.Sprintf("max_connections = %d", testMaxConns),
		"autovacuum_max_workers = 10",
		"autovacuum_naptime = 10",
		"max_locks_per_transaction = 64",
		"effective_io_concurrency = 200",
		"max_locks_per_transaction = 128",
	}
)

func TestTunerProcessOurParams(t *testing.T) {
	defaultWantLines := []string{
		ourParamString(lastTunedParam),
		ourParamString(lastTunedVersionParam),
	}
	cases := []struct {
		desc      string
		lines     []string
		wantLines []string
	}{
		{
			desc:      "no params found",
			lines:     []string{},
			wantLines: defaultWantLines,
		},
		{
			desc: "one param found",
			lines: []string{
				ourParamString(lastTunedParam),
			},
			wantLines: defaultWantLines,
		},
		{
			desc: "all param found",
			lines: []string{
				ourParamString(lastTunedParam),
				ourParamString(lastTunedVersionParam),
			},
			wantLines: defaultWantLines,
		},
		{
			desc: "all param found, early stop",
			lines: []string{
				"not a useful line",
				ourParamString(lastTunedParam),
				ourParamString(lastTunedParam), //repeat
				ourParamString(lastTunedVersionParam),
			},
			wantLines: []string{
				"not a useful line",
				ourParamString(lastTunedParam),
				ourParamString(lastTunedParam), //repeat
				ourParamString(lastTunedVersionParam),
			},
		},
	}

	for _, c := range cases {
		tuner := newTunerWithDefaultFlagsForInputs(t, "", c.lines)
		tuner.processOurParams()

		if got := len(tuner.cfs.lines); got != len(c.wantLines) {
			t.Errorf("%s: incorrect number of lines: got %d want %d", c.desc, got, len(c.wantLines))
		} else {
			for i, want := range c.wantLines {
				if got := tuner.cfs.lines[i].content; got != want {
					t.Errorf("%s: incorrect line at %d: got %s want %s", c.desc, i, got, want)
				}
			}
		}
	}
}

func TestTunerProcessQuiet(t *testing.T) {
	lastTuned := removeSecsFromLastTuned(ourParamString(lastTunedParam)) + "\n"
	lastTunedVersion := ourParamString(lastTunedVersionParam) + "\n"
	cases := []struct {
		desc          string
		lines         []string
		wantedPrints  []string
		wantPrompts   uint64
		wantSuccesses uint64
		shouldErr     bool
	}{
		{
			desc:         "missing shared",
			lines:        wantedQuietLines[1:],
			wantedPrints: []string{wantedQuietCorrectShared},
			wantPrompts:  1,
		},
		{
			desc:          "all correct",
			lines:         wantedQuietLines,
			wantedPrints:  []string{},
			wantPrompts:   0,
			wantSuccesses: 1,
		},
		{
			desc:         "commented shared",
			lines:        append([]string{wantedQuietCommentedShared}, wantedQuietLines[1:]...),
			wantedPrints: []string{wantedQuietCorrectShared},
			wantPrompts:  1,
		},
		{
			desc:         "wrong shared",
			lines:        append([]string{wantedQuietMissingShared}, wantedQuietLines[1:]...),
			wantedPrints: []string{wantedQuietCorrectShared},
			wantPrompts:  1,
		},
		{
			desc:         "missing tunables",
			lines:        append([]string{wantedQuietCorrectShared}, wantedQuietLines[6:]...),
			wantedPrints: wantedQuietLines[1:6],
			wantPrompts:  1,
		},
		{
			desc:         "no = error",
			lines:        wantedQuietLines[1:],
			wantedPrints: []string{wantedQuietCorrectShared},
			wantPrompts:  1,
			shouldErr:    true,
		},
	}

	for _, c := range cases {
		config := getDefaultSystemConfig(t)
		input := "y\n"
		if c.shouldErr {
			input = "n\n"
		}
		tuner := newTunerWithDefaultFlagsForInputs(t, input, c.lines)
		tuner.flags.Quiet = true

		err := tuner.processQuiet(config)
		if err != nil && !c.shouldErr {
			t.Errorf("%s: unexpected error: %v", c.desc, err)
		} else if err == nil && c.shouldErr {
			t.Errorf("%s: unexpected lack of an error", c.desc)
		}

		// If there are no prints, then our "extra" prints for last_tuned GUCs
		// are not printed either, so the default is 0. However, if any other
		// setting is printed, then we add our GUCs too, therefore upping the
		// wanted prints len by 2.
		wantPrintsLen := 0
		if len(c.wantedPrints) > 0 {
			wantPrintsLen = len(c.wantedPrints) + 2
		}

		out := tuner.handler.out.(*testWriter)
		if got := len(out.lines); got != wantPrintsLen {
			t.Errorf("%s: incorrect prints len: got %d want %d", c.desc, got, wantPrintsLen)
		} else if len(c.wantedPrints) > 0 {
			for i, want := range c.wantedPrints {
				if got := out.lines[i]; got != want+"\n" {
					t.Errorf("%s: incorrect print at idx %d: got\n%s\nwant\n%s", c.desc, i, got, want+"\n")
				}
			}
			lastTuneIdx := len(c.wantedPrints)
			lastTuneVersionIdx := len(c.wantedPrints) + 1
			if got := removeSecsFromLastTuned(out.lines[lastTuneIdx]); got != lastTuned {
				t.Errorf("%s: lastTuned print is missing/incorrect: got\n%s\nwant\n%s", c.desc, got, lastTuned)
			}
			if got := out.lines[lastTuneVersionIdx]; got != lastTunedVersion {
				t.Errorf("%s: lastTunedVersion print is missing/incorrect: got\n%s\nwant\n%s", c.desc, got, lastTunedVersion)
			}
		}

		tp := tuner.handler.p.(*testPrinter)
		if got := tp.statementCalls; got != 1 {
			t.Errorf("%s: incorrect number of statements: got %d want %d", c.desc, got, 1)
		} else {
			want := fmt.Sprintf(statementTunableIntro, parse.BytesToDecimalFormat(config.Memory), config.CPUs, config.PGMajorVersion)
			if got := tp.statements[0]; got != want {
				t.Errorf("%s: incorrect statement: got\n%s\nwant\n%s", c.desc, got, want)
			}
		}

		if got := tp.promptCalls; got != c.wantPrompts {
			t.Errorf("%s: incorrect number of prompts: got %d want %d", c.desc, got, c.wantPrompts)
		}

		if got := tp.successCalls; got != c.wantSuccesses {
			t.Errorf("%s: incorrect number of successes: got %d want %d", c.desc, got, c.wantSuccesses)
		}

		if c.wantSuccesses == 1 {
			if got := tp.successes[0]; got != successQuiet {
				t.Errorf("%s: incorrect success: got\n%s\nwant\n%s", c.desc, got, successQuiet)
			}
		}
	}
}

func TestTunerWriteConfFile(t *testing.T) {
	wantPath := "postgresql.conf"
	errCreateFmt := "path does not exist: %s"
	errAbsPath := "could not get absolute path"
	confFileLines := []string{"shared_preload_libraries = 'timescaledb'", "foo"}

	cases := []struct {
		desc             string
		destPath         string
		confPath         string
		statements       uint64
		shouldErrOnWrite bool
		errMsg           string
	}{
		{
			desc:       "success",
			destPath:   wantPath,
			statements: 1,
		},
		{
			desc:       "success with derived path",
			confPath:   wantPath,
			statements: 1,
		},
		{
			desc:     "error on absolute path",
			confPath: "foo.out",
			errMsg:   fmt.Sprintf(errCouldNotWriteFmt, "foo.out", errAbsPath),
		},
		{
			desc:     "error on create",
			destPath: "foo.out",
			errMsg:   fmt.Sprintf(errCouldNotWriteFmt, "foo.out", fmt.Sprintf(errCreateFmt, "foo.out")),
		},
		{
			desc:             "error on writeTo",
			destPath:         wantPath,
			shouldErrOnWrite: true,
			errMsg:           fmt.Sprintf(errCouldNotWriteFmt, wantPath, errTestWriter),
		},
	}

	oldOSCreateFn := osCreateFn
	oldFilepathAbsFn := filepathAbsFn
	filepathAbsFn = func(p string) (string, error) {
		if p == wantPath {
			return p, nil
		}
		return "", fmt.Errorf(errAbsPath)
	}

	for _, c := range cases {
		var buf testBufferCloser
		buf.shouldErr = c.shouldErrOnWrite
		osCreateFn = func(p string) (io.WriteCloser, error) {
			if !fileExists(p) && p != wantPath {
				return nil, fmt.Errorf(errCreateFmt, p)
			}
			return &buf, nil
		}

		tuner := newTunerWithDefaultFlagsForInputs(t, "", confFileLines)
		tuner.flags.DestPath = c.destPath

		err := tuner.writeConfFile(c.confPath)
		if c.errMsg == "" && err != nil {
			t.Errorf("%s: unexpected error: got %v", c.desc, err)
		} else if c.errMsg != "" {
			if err == nil {
				t.Errorf("%s: unexpected lack of error", c.desc)
			} else if got := err.Error(); got != c.errMsg {
				t.Errorf("%s: incorrect error:\ngot\n%s\nwant\n%s", c.desc, got, c.errMsg)
			}
		} else {
			tp := tuner.handler.p.(*testPrinter)
			if got := tp.statementCalls; got != c.statements {
				t.Errorf("%s: incorrect number of statements: got %d want %d", c.desc, got, c.statements)
			}

			scanner := bufio.NewScanner(bytes.NewReader(buf.b.Bytes()))
			i := 0
			for scanner.Scan() {
				if scanner.Err() != nil {
					t.Errorf("%s: unexpected error while scanning: %v", c.desc, scanner.Err())
				}
				got := scanner.Text()
				if want := confFileLines[i]; got != want {
					t.Errorf("%s: incorrect line at %d:\ngot\n%s\nwant\n%s", c.desc, i, got, want)
				}
				i++
			}
		}
	}

	filepathAbsFn = oldFilepathAbsFn
	osCreateFn = oldOSCreateFn
}
