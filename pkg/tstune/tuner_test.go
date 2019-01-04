package tstune

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pbnjay/memory"
	"github.com/timescale/timescaledb-tune/internal/parse"
	"github.com/timescale/timescaledb-tune/pkg/pgtune"
)

func TestGetPGMajorVersion(t *testing.T) {
	okPath96 := "pg_config_9.6"
	okPath10 := "pg_config_10"
	okPath11 := "pg_config_11"
	okPath95 := "pg_config_9.5"
	okPath60 := "pg_config_6.0"
	cases := []struct {
		desc    string
		binPath string
		want    string
		errMsg  string
	}{
		{
			desc:    "failed execute",
			binPath: "pg_config_bad",
			errMsg:  fmt.Sprintf(errCouldNotExecuteFmt, "pg_config_bad", exec.ErrNotFound),
		},
		{
			desc:    "failed major parse",
			binPath: okPath60,
			errMsg:  fmt.Sprintf("unknown major PG version: PostgreSQL 6.0.5"),
		},
		{
			desc:    "failed unsupported",
			binPath: okPath95,
			errMsg:  fmt.Sprintf(errUnsupportedMajorFmt, "9.5"),
		},
		{
			desc:    "success 9.6",
			binPath: okPath96,
			want:    pgMajor96,
		},
		{
			desc:    "success 10",
			binPath: okPath10,
			want:    pgMajor10,
		},
		{
			desc:    "success 11",
			binPath: okPath11,
			want:    pgMajor11,
		},
	}

	oldVersionFn := getPGConfigVersionFn
	getPGConfigVersionFn = func(binPath string) ([]byte, error) {
		switch binPath {
		case okPath60:
			return []byte("PostgreSQL 6.0.5"), nil
		case okPath95:
			return []byte("PostgreSQL 9.5.10"), nil
		case okPath96:
			return []byte("PostgreSQL 9.6.6"), nil
		case okPath10:
			return []byte("PostgreSQL 10.5 (Debian7)"), nil
		case okPath11:
			return []byte("PostgreSQL 11.1"), nil
		default:
			return nil, exec.ErrNotFound
		}
	}

	for _, c := range cases {
		got, err := getPGMajorVersion(c.binPath)
		if len(c.errMsg) == 0 {
			if err != nil {
				t.Errorf("%s: unexpected error: got %v", c.desc, err)
			}
			if got != c.want {
				t.Errorf("%s: incorrect major version: got %s want %s", c.desc, got, c.want)
			}
		} else {
			if err == nil {
				t.Errorf("%s: unexpected lack of error", c.desc)
			}
			if got := err.Error(); got != c.errMsg {
				t.Errorf("%s: incorrect error:\ngot\n%s\nwant\n%s", c.desc, got, c.errMsg)
			}
		}
	}

	getPGConfigVersionFn = oldVersionFn
}

func newTunerWithDefaultFlags(handler *ioHandler, cfs *configFileState) *Tuner {
	return &Tuner{handler, cfs, &TunerFlags{}}
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
	cases := []struct {
		desc         string
		flagPGConfig string
		flagMemory   string
		flagNumCPUs  uint
		wantMemory   uint64
		wantCPUs     int
		errMsg       string
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
			desc:         "use mem flag only",
			flagPGConfig: okPGConfig,
			flagMemory:   "1" + parse.GB,
			wantMemory:   1 * parse.Gigabyte,
			wantCPUs:     runtime.NumCPU(),
		},
		{
			desc:         "use cpu flag only",
			flagPGConfig: okPGConfig,
			flagNumCPUs:  2,
			wantMemory:   totalMemory,
			wantCPUs:     2,
		},
		{
			desc:         "both flags",
			flagPGConfig: okPGConfig,
			flagMemory:   "128" + parse.GB,
			flagNumCPUs:  1,
			wantMemory:   128 * parse.Gigabyte,
			wantCPUs:     1,
		},
		{
			desc:         "neither flags",
			flagPGConfig: okPGConfig,
			wantMemory:   totalMemory,
			wantCPUs:     runtime.NumCPU(),
		},
	}

	oldVersionFn := getPGConfigVersionFn
	getPGConfigVersionFn = func(binPath string) ([]byte, error) {
		if binPath == okPGConfig {
			return []byte("PostgreSQL 10.5"), nil
		}
		return nil, exec.ErrNotFound
	}

	for _, c := range cases {
		tuner := &Tuner{nil, nil, &TunerFlags{
			PGConfig: c.flagPGConfig,
			Memory:   c.flagMemory,
			NumCPUs:  c.flagNumCPUs,
		}}
		config, err := tuner.initializeSystemConfig()
		if len(c.errMsg) == 0 {
			if err != nil {
				t.Errorf("%s: unexpected error: got %v", c.desc, err)
			}

			if got := config.Memory; got != c.wantMemory {
				t.Errorf("%s: incorrect amount of memory: got %d want %d", c.desc, got, c.wantMemory)
			}

			if got := config.CPUs; got != c.wantCPUs {
				t.Errorf("%s: incorrect number of CPUs: got %d want %d", c.desc, got, c.wantCPUs)
			}
		} else {
			if err == nil {
				t.Errorf("%s: unexpected lack of error", c.desc)
			}

			if got := err.Error(); got != c.errMsg {
				t.Errorf("%s: incorrect error: got\n%s\nwant\n%s", c.desc, got, c.errMsg)
			}
		}
	}

	getPGConfigVersionFn = oldVersionFn
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

	testString := "foo\nFoo\nFOO\nfOo\nfOO\n\n"
	for _, c := range cases {
		buf := bytes.NewBuffer([]byte(testString))
		br := bufio.NewReader(buf)
		handler := &ioHandler{
			p:  &testPrinter{},
			br: br,
		}
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
		buf := bytes.NewBuffer([]byte(testString))
		br := bufio.NewReader(buf)
		handler := &ioHandler{
			p:  &testPrinter{},
			br: br,
		}
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

	oldPrintFn := printFn

	for _, c := range cases {
		prints := []string{}
		printFn = func(_ io.Writer, format string, args ...interface{}) (int, error) {
			prints = append(prints, fmt.Sprintf(format, args...))
			return 0, nil
		}

		buf := bytes.NewBufferString(c.input)
		br := bufio.NewReader(buf)
		handler := &ioHandler{
			p:  &testPrinter{},
			br: br,
		}
		cfs := &configFileState{lines: []string{}}
		tuner := newTunerWithDefaultFlags(handler, cfs)
		tuner.flags.ConfPath = c.flagPath

		err := tuner.processConfFileCheck(c.filePath)
		tp := handler.p.(*testPrinter)
		if got := tp.statementCalls; got != 1 {
			t.Errorf("%s: incorrect number of statements: got %d want %d", c.desc, got, 1)
		} else if got := tp.statements[0]; got != statementConfFileCheck {
			t.Errorf("%s: incorrect statement: got\n%s\nwant\n%s", c.desc, got, statementConfFileCheck)
		}

		if got := len(prints); got != 1 {
			t.Errorf("%s: incorrect number of prints: got %d want %d", c.desc, got, 1)
		} else if got := prints[0]; got != c.filePath+"\n\n" {
			t.Errorf("%s: incorrect print: got\n%s\nwant\n%s", c.desc, got, c.filePath+"\n\n")
		}

		if got := tp.promptCalls; got != c.promptCalls {
			t.Errorf("%s: incorrect number of prompt calls: got %d want %d", c.desc, got, c.promptCalls)
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

	printFn = oldPrintFn
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
		buf := bytes.NewBuffer([]byte(c.input))
		br := bufio.NewReader(buf)
		handler := &ioHandler{
			p:  &testPrinter{},
			br: br,
		}
		cfs := &configFileState{lines: []string{}}
		tuner := newTunerWithDefaultFlags(handler, cfs)
		err := tuner.processNoSharedLibLine()
		if got := handler.p.(*testPrinter).statementCalls; got != 1 {
			t.Errorf("%s: incorrect number of statements: got %d want %d", c.desc, got, 1)
		}

		if err != nil && !c.shouldErr {
			t.Errorf("%s: unexpected error: %v", c.desc, err)
		} else if err == nil && c.shouldErr {
			t.Errorf("%s: unexpected lack of error", c.desc)
		}

		if got := handler.p.(*testPrinter).promptCalls; got != c.prompts {
			t.Errorf("%s: incorrect number of prompts: got %d want %d", c.desc, got, c.prompts)
		}
		if err == nil {
			if got := handler.p.(*testPrinter).successCalls; got != 1 {
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

	oldPrintFn := printFn

	for _, c := range cases {
		buf := bytes.NewBufferString(c.input)
		br := bufio.NewReader(buf)
		handler := &ioHandler{
			p:  &testPrinter{},
			br: br,
		}
		cfs := &configFileState{lines: c.lines}
		cfs.sharedLibResult = parseLineForSharedLibResult(c.lines[0])
		tuner := newTunerWithDefaultFlags(handler, cfs)

		prints := []string{}
		printFn = func(_ io.Writer, format string, args ...interface{}) (int, error) {
			prints = append(prints, fmt.Sprintf(format, args...))
			return 0, nil
		}

		err := tuner.processSharedLibLine()
		if err != nil && !c.shouldErr {
			t.Errorf("%s: unexpected error: %v", c.desc, err)
		} else if err == nil && c.shouldErr {
			t.Errorf("%s: unexpected lack of err", c.desc)
		}

		tp := handler.p.(*testPrinter)
		if got := tp.promptCalls; got != c.prompts {
			t.Errorf("%s: incorrect number of prompts: got %d want %d", c.desc, got, c.prompts)
		}
		if got := tp.statementCalls; got != c.statements {
			t.Errorf("%s: incorrect number of statements: got %d want %d", c.desc, got, c.statements)
		}

		if len(c.prints) > 0 {
			for i, want := range c.prints {
				if got := prints[i]; got != want {
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

	printFn = oldPrintFn
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
		mr := pgtune.NewMemoryRecommender(8*parse.Gigabyte, 1)
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
		"foo": &tunableParseResult{
			value: "5.0",
		},
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

type testSettingsGroup struct {
	keys []string
}

func (sg *testSettingsGroup) Label() string                      { return "foo" }
func (sg *testSettingsGroup) Keys() []string                     { return sg.keys }
func (sg *testSettingsGroup) GetRecommender() pgtune.Recommender { return &badRecommender{} }

func TestProcessSettingsGroup(t *testing.T) {
	mem := uint64(8 * parse.Gigabyte)
	cpus := 4
	config := pgtune.NewSystemConfig(mem, cpus, pgMajor10)
	cases := []struct {
		desc           string
		ts             pgtune.SettingsGroup
		lines          []string
		stdin          string
		wantStatements uint64
		wantPrompts    uint64
		wantPrints     uint64
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
			stdin:          "y\n",
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
			stdin:          "y\n",
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
			stdin:          "y\n",
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
			stdin:          " \ny\n",
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
			stdin:          " \n \ny\n",
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
			stdin:          "s\n",
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
			stdin:          " \nqUIt\n",
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
			stdin:          "y\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    1,
			wantPrints:     9, // one for initial newline + four settings, displayed twice
			successMsg:     "memory settings will be updated",
			shouldErr:      false,
		},
		{
			desc:           "label capitalized",
			ts:             pgtune.GetSettingsGroup(pgtune.WALLabel, config),
			stdin:          "y\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    1,
			wantPrints:     4, // one for initial newline + 3 for recommendations
			wantErrors:     3, // everything is missing
			successMsg:     "WAL settings will be updated",
			shouldErr:      false,
		},
	}

	oldPrintFn := printFn

	for _, c := range cases {
		buf := bytes.NewBuffer([]byte(c.stdin))
		br := bufio.NewReader(buf)
		handler := &ioHandler{
			p:  &testPrinter{},
			br: br,
		}
		cfs := &configFileState{tuneParseResults: make(map[string]*tunableParseResult)}
		cfs.lines = append(cfs.lines, c.lines...)
		for i, l := range cfs.lines {
			for _, k := range c.ts.Keys() {
				p := parseWithRegex(l, regexes[k])
				if p != nil {
					p.idx = i
					cfs.tuneParseResults[k] = p
				}
			}
		}
		tuner := newTunerWithDefaultFlags(handler, cfs)

		numPrints := uint64(0)
		printFn = func(_ io.Writer, _ string, _ ...interface{}) (int, error) {
			numPrints++
			return 0, nil
		}

		err := tuner.processSettingsGroup(c.ts)
		if err != nil && !c.shouldErr {
			t.Errorf("%s: unexpected error: %v", c.desc, err)
		} else if err == nil && c.shouldErr {
			t.Errorf("%s: unexpected lack of error", c.desc)
		}

		tp := handler.p.(*testPrinter)
		if got := strings.ToUpper(strings.TrimSpace(tp.statements[0])[:1]); got != strings.ToUpper(c.ts.Label()[:1]) {
			t.Errorf("%s: label not capitalized in first statement: got %s want %s", c.desc, got, strings.ToUpper(c.ts.Label()[:1]))
		}

		if got := tp.statementCalls; got != c.wantStatements {
			t.Errorf("%s: incorrect number of statements: got %d want %d", c.desc, got, c.wantStatements)
		}

		if got := tp.promptCalls; got != c.wantPrompts {
			t.Errorf("%s: incorrect number of prompts: got %d want %d", c.desc, got, c.wantPrompts)
		}

		if got := numPrints; got != c.wantPrints {
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

	printFn = oldPrintFn
}

func TestProcessTunables(t *testing.T) {
	mem := uint64(10 * parse.Gigabyte)
	cpus := 6
	config := pgtune.NewSystemConfig(mem, cpus, pgMajor10)

	oldPrintFn := printFn
	printFn = func(_ io.Writer, _ string, _ ...interface{}) (int, error) {
		return 0, nil
	}

	buf := bytes.NewBuffer([]byte("y\ny\ny\ny\n"))
	br := bufio.NewReader(buf)
	handler := &ioHandler{
		p:  &testPrinter{},
		br: br,
	}

	cfs := &configFileState{lines: []string{}, tuneParseResults: make(map[string]*tunableParseResult)}
	tuner := newTunerWithDefaultFlags(handler, cfs)
	tuner.processTunables(config)

	tp := handler.p.(*testPrinter)
	// Total number of statements is intro statement and then 3 per group of settings;
	// each group has a heading and then the current/recommended labels.
	if got := tp.statementCalls; got != 1+3*4 {
		t.Errorf("incorrect number of statements: got %d, want %d", got, 1+3*4)
	}

	wantStatement := fmt.Sprintf(statementTunableIntro, parse.BytesToDecimalFormat(mem), cpus, pgMajor10)
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

	wantStatement = "Memory settings recommendations"
	if got := tp.statements[1]; got != wantStatement {
		t.Errorf("incorrect statement at 1: got\n%s\nwant\n%s", got, wantStatement)
	}
	wantStatement = "Parallelism settings recommendations"
	if got := tp.statements[4]; got != wantStatement {
		t.Errorf("incorrect statement at 4: got\n%s\nwant\n%s", got, wantStatement)
	}
	wantStatement = "WAL settings recommendations"
	if got := tp.statements[7]; got != wantStatement {
		t.Errorf("incorrect statement at 7: got\n%s\nwant\n%s", got, wantStatement)
	}
	wantStatement = "Miscellaneous settings recommendations"
	if got := tp.statements[10]; got != wantStatement {
		t.Errorf("incorrect statement at 10: got\n%s\nwant\n%s", got, wantStatement)
	}

	printFn = oldPrintFn
}

func TestProcessTunablesSingleCPU(t *testing.T) {
	mem := uint64(10 * parse.Gigabyte)
	cpus := 1
	config := pgtune.NewSystemConfig(mem, cpus, pgMajor10)

	oldPrintFn := printFn
	printFn = func(_ io.Writer, _ string, _ ...interface{}) (int, error) {
		return 0, nil
	}

	buf := bytes.NewBuffer([]byte("y\ny\ny\ny\n"))
	br := bufio.NewReader(buf)
	handler := &ioHandler{
		p:  &testPrinter{},
		br: br,
	}

	cfs := &configFileState{lines: []string{}, tuneParseResults: make(map[string]*tunableParseResult)}
	tuner := newTunerWithDefaultFlags(handler, cfs)
	tuner.processTunables(config)

	tp := handler.p.(*testPrinter)
	// Total number of statements is intro statement and then 3 per group of settings;
	// each group has a heading and then the current/recommended labels.
	// On a single CPU, only 3 groups since parallelism does not apply
	if got := tp.statementCalls; got != 1+3*3 {
		t.Errorf("incorrect number of statements: got %d, want %d", got, 1+3*3)
	}

	wantStatement := fmt.Sprintf(statementTunableIntro, parse.BytesToDecimalFormat(mem), cpus, pgMajor10)
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

	wantStatement = "Memory settings recommendations"
	if got := tp.statements[1]; got != wantStatement {
		t.Errorf("incorrect statement at 1: got\n%s\nwant\n%s", got, wantStatement)
	}
	// no parallelism on single CPU
	wantStatement = "WAL settings recommendations"
	if got := tp.statements[4]; got != wantStatement {
		t.Errorf("incorrect statement at 7: got\n%s\nwant\n%s", got, wantStatement)
	}
	wantStatement = "Miscellaneous settings recommendations"
	if got := tp.statements[7]; got != wantStatement {
		t.Errorf("incorrect statement at 10: got\n%s\nwant\n%s", got, wantStatement)
	}

	printFn = oldPrintFn
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
		"max_worker_processes = 4",
		"max_parallel_workers_per_gather = 2",
		"max_parallel_workers = 4",
		"wal_buffers = 16MB",
		"min_wal_size = 4GB",
		"max_wal_size = 8GB",
		"default_statistics_target = 500",
		"random_page_cost = 1.1",
		"checkpoint_completion_target = 0.9",
		"max_connections = 20",
		"max_locks_per_transaction = 64",
		"effective_io_concurrency = 200",
	}
)

func TestTunerProcessQuiet(t *testing.T) {
	// We should only expect the first part of the timestamps to be the same,
	// since the number of seconds + milliseconds will change on each invocation
	// inside the function (which we cannot control). Therefore, only compare a defined prefix.
	timeMatchIdx := len("timescaledb.last_tuned = '") + 16
	lastTuned := fmt.Sprintf(fmtLastTuned, time.Now().Format(time.RFC3339))[:timeMatchIdx]
	lastTunedVersion := fmt.Sprintf(fmtLastTunedVersion+"\n", Version)
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
	oldPrintFn := printFn

	for _, c := range cases {
		prints := []string{}
		printFn = func(_ io.Writer, format string, args ...interface{}) (int, error) {
			prints = append(prints, fmt.Sprintf(format, args...))
			return 0, nil
		}

		mem := uint64(8 * parse.Gigabyte)
		cpus := 4
		config := pgtune.NewSystemConfig(mem, cpus, pgMajor10)
		input := "y\n"
		if c.shouldErr {
			input = "n\n"
		}
		buf := bytes.NewBufferString(input)
		br := bufio.NewReader(buf)
		handler := &ioHandler{
			p:  &testPrinter{},
			br: br,
		}
		confFile := bytes.NewBufferString(strings.Join(c.lines, "\n"))
		cfs, err := getConfigFileState(confFile)
		if err != nil {
			t.Fatalf("could not parse config lines")
		}

		tuner := newTunerWithDefaultFlags(handler, cfs)
		tuner.flags.Quiet = true
		err = tuner.processQuiet(config)

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

		if got := len(prints); got != wantPrintsLen {
			t.Errorf("%s: incorrect prints len: got %d want %d", c.desc, got, wantPrintsLen)
		} else if len(c.wantedPrints) > 0 {
			for i, want := range c.wantedPrints {
				if got := prints[i]; got != want+"\n" {
					t.Errorf("%s: incorrect print at idx %d: got\n%s\nwant\n%s", c.desc, i, got, want+"\n")
				}
			}
			lastTuneIdx := len(c.wantedPrints)
			lastTuneVersionIdx := len(c.wantedPrints) + 1
			if got := prints[lastTuneIdx][:timeMatchIdx]; got != lastTuned {
				t.Errorf("%s: lastTuned print is missing/incorrect: got\n%s\nwant\n%s", c.desc, got, lastTuned)
			}
			if got := prints[lastTuneVersionIdx]; got != lastTunedVersion {
				t.Errorf("%s: lastTunedVersion print is missing/incorrect: got\n%s\nwant\n%s", c.desc, got, lastTunedVersion)
			}
		}

		tp := handler.p.(*testPrinter)
		if got := tp.statementCalls; got != 1 {
			t.Errorf("%s: incorrect number of statements: got %d want %d", c.desc, got, 1)
		} else {
			want := fmt.Sprintf(statementTunableIntro, parse.BytesToDecimalFormat(mem), cpus, pgMajor10)
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
	printFn = oldPrintFn
}
