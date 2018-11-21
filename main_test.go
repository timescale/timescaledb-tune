package main

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/timescale/timescaledb-tune/internal/parse"
	"github.com/timescale/timescaledb-tune/pkg/pgtune"
)

func TestGetConfigFilePath(t *testing.T) {
	cases := []struct {
		desc      string
		os        string
		files     []string
		wantFile  string
		shouldErr bool
	}{
		{
			desc:      "mac - yes",
			os:        osMac,
			files:     []string{fileNameMac},
			wantFile:  fileNameMac,
			shouldErr: false,
		},
		{
			desc:      "mac - no",
			os:        osMac,
			files:     []string{"/etc"},
			wantFile:  "",
			shouldErr: true,
		},
		{
			desc:      "linux - pg10+debian",
			os:        osLinux,
			files:     []string{fmt.Sprintf(fileNameDebianFmt, "10")},
			wantFile:  fmt.Sprintf(fileNameDebianFmt, "10"),
			shouldErr: false,
		},
		{
			desc:      "linux - pg9.6+debian",
			os:        osLinux,
			files:     []string{fmt.Sprintf(fileNameDebianFmt, "9.6")},
			wantFile:  fmt.Sprintf(fileNameDebianFmt, "9.6"),
			shouldErr: false,
		},
		{
			desc:      "linux - pg10+rpm",
			os:        osLinux,
			files:     []string{fmt.Sprintf(fileNameRPMFmt, "10")},
			wantFile:  fmt.Sprintf(fileNameRPMFmt, "10"),
			shouldErr: false,
		},
		{
			desc:      "linux - pg9.6+rpm",
			os:        osLinux,
			files:     []string{fmt.Sprintf(fileNameDebianFmt, "9.6")},
			wantFile:  fmt.Sprintf(fileNameDebianFmt, "9.6"),
			shouldErr: false,
		},
		{
			desc:      "linux - no",
			os:        osLinux,
			files:     []string{fmt.Sprintf(fileNameDebianFmt, "9.0")},
			wantFile:  "",
			shouldErr: true,
		},
	}

	oldFileExistsFn := fileExistsFn
	for _, c := range cases {
		fileExistsFn = func(fn string) bool {
			for _, s := range c.files {
				if fn == s {
					return true
				}
			}
			return false
		}
		filename, err := getConfigFilePath(c.os)
		if err != nil && !c.shouldErr {
			t.Errorf("%s: unexpected error: %v", c.desc, err)
		} else if err == nil && c.shouldErr {
			t.Errorf("%s: unexpected lack of error", c.desc)
		}

		if c.shouldErr && filename != "" {
			t.Errorf("%s: unexpected filename in error case: got %s", c.desc, filename)
		}

		if got := filename; got != c.wantFile {
			t.Errorf("%s: incorrect filename: got %s want %s", c.desc, got, c.wantFile)
		}
	}
	fileExistsFn = oldFileExistsFn
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
		checker := &limitChecker{limit: c.limit, shouldErr: c.shouldErr}
		err := promptUntilValidInput(handler, "test prompt", checker)
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
}

func TestParseLineForSharedLibResult(t *testing.T) {
	cases := []struct {
		desc  string
		input string
		want  *sharedLibResult
	}{
		{
			desc: "initial config value",
			input: "#shared_preload_libraries = ''		# (change requires restart)",
			want: &sharedLibResult{
				commented:    true,
				hasTimescale: false,
				libs:         "",
			},
		},
		{
			desc: "extra commented out",
			input: "###shared_preload_libraries = ''		# (change requires restart)",
			want: &sharedLibResult{
				commented:    true,
				hasTimescale: false,
				libs:         "",
			},
		},
		{
			desc: "commented with space after",
			input: "# shared_preload_libraries = ''		# (change requires restart)",
			want: &sharedLibResult{
				commented:    true,
				hasTimescale: false,
				libs:         "",
			},
		},
		{
			desc: "extra commented with space after",
			input: "## shared_preload_libraries = ''		# (change requires restart)",
			want: &sharedLibResult{
				commented:    true,
				hasTimescale: false,
				libs:         "",
			},
		},
		{
			desc: "initial config value, uncommented",
			input: "shared_preload_libraries = ''		# (change requires restart)",
			want: &sharedLibResult{
				commented:    false,
				hasTimescale: false,
				libs:         "",
			},
		},
		{
			desc: "initial config value, uncommented with leading space",
			input: "  shared_preload_libraries = ''		# (change requires restart)",
			want: &sharedLibResult{
				commented:    false,
				hasTimescale: false,
				libs:         "",
			},
		},
		{
			desc: "timescaledb already there but commented",
			input: "#shared_preload_libraries = 'timescaledb'		# (change requires restart)",
			want: &sharedLibResult{
				commented:    true,
				hasTimescale: true,
				libs:         "timescaledb",
			},
		},
		{
			desc:  "other libraries besides timescaledb, commented",
			input: "#shared_preload_libraries = 'pg_stats' # (change requires restart)   ",
			want: &sharedLibResult{
				commented:    true,
				hasTimescale: false,
				libs:         "pg_stats",
			},
		},
		{
			desc:  "no string after the quotes",
			input: "shared_preload_libraries = 'pg_stats,timescaledb'",
			want: &sharedLibResult{
				commented:    false,
				hasTimescale: true,
				libs:         "pg_stats,timescaledb",
			},
		},
		{
			desc: "don't be greedy with things between single quotes",
			input: "#shared_preload_libraries = 'timescaledb'		# comment with single quote ' test",
			want: &sharedLibResult{
				commented:    true,
				hasTimescale: true,
				libs:         "timescaledb",
			},
		},
		{
			desc:  "not shared preload line",
			input: "data_dir = '/path/to/data'",
			want:  nil,
		},
	}
	for _, c := range cases {
		res := parseLineForSharedLibResult(c.input)
		if res == nil && c.want != nil {
			t.Errorf("%s: result was unexpectedly nil: want %v", c.desc, c.want)
		} else if res != nil && c.want == nil {
			t.Errorf("%s: result was unexpectedly non-nil: got %v", c.desc, res)
		} else if c.want != nil {
			if got := res.commented; got != c.want.commented {
				t.Errorf("%s: incorrect commented: got %v want %v", c.desc, got, c.want.commented)
			}
			if got := res.hasTimescale; got != c.want.hasTimescale {
				t.Errorf("%s: incorrect hasTimescale: got %v want %v", c.desc, got, c.want.hasTimescale)
			}
			if got := res.libs; got != c.want.libs {
				t.Errorf("%s: incorrect libs: got %s want %s", c.desc, got, c.want.libs)
			}
		}
	}
}

func TestUpdateSharedLibLine(t *testing.T) {
	confKey := "shared_preload_libraries = "
	simpleOkayCase := confKey + "'" + extName + "'"
	simpleOkayCaseExtra := simpleOkayCase + " # (change requires restart)"
	cases := []struct {
		desc     string
		original string
		want     string
	}{
		{
			desc:     "original = ok",
			original: simpleOkayCase,
			want:     simpleOkayCase,
		},
		{
			desc:     "original = ok w/ ending comments",
			original: simpleOkayCaseExtra,
			want:     simpleOkayCaseExtra,
		},
		{
			desc:     "original = ok w/ prepended spaces",
			original: "   " + simpleOkayCase,
			want:     "   " + simpleOkayCase,
		},
		{
			desc:     "just need to uncomment",
			original: "#" + simpleOkayCase,
			want:     simpleOkayCase,
		},
		{
			desc:     "just need to uncomment w/ ending comments",
			original: "#" + simpleOkayCaseExtra,
			want:     simpleOkayCaseExtra,
		},
		{
			desc:     "just need to uncomment multiple times",
			original: "###" + simpleOkayCase,
			want:     simpleOkayCase,
		},
		{
			desc:     "uncomment + spaces",
			original: "###  " + simpleOkayCase,
			want:     simpleOkayCase,
		},
		{
			desc:     "needs to be added, empty list",
			original: confKey + "''",
			want:     simpleOkayCase,
		},
		{
			desc:     "needs to be added, empty list, commented out",
			original: "#" + confKey + "''",
			want:     simpleOkayCase,
		},
		{
			desc:     "needs to be added, empty list, trailing comment",
			original: confKey + "'' # (change requires restart)",
			want:     simpleOkayCaseExtra,
		},
		{
			desc:     "needs to be added, one item",
			original: confKey + "'pg_stats'",
			want:     confKey + "'pg_stats," + extName + "'",
		},
		{
			desc:     "needs to be added, t item, commented out",
			original: "#" + confKey + "'pg_stats,ext2'",
			want:     confKey + "'pg_stats,ext2," + extName + "'",
		},
		{
			desc:     "needs to be added, two items",
			original: confKey + "'pg_stats'",
			want:     confKey + "'pg_stats," + extName + "'",
		},
		{
			desc:     "needs to be added, two items, commented out",
			original: "#" + confKey + "'pg_stats,ext2'",
			want:     confKey + "'pg_stats,ext2," + extName + "'",
		},
		{
			desc:     "in list with others",
			original: confKey + "'timescaledb,pg_stats'",
			want:     confKey + "'timescaledb,pg_stats'",
		},
		{
			desc:     "in list with others, commented out",
			original: "#" + confKey + "'timescaledb,pg_stats'",
			want:     confKey + "'timescaledb,pg_stats'",
		},
	}

	for _, c := range cases {
		res := parseLineForSharedLibResult(c.original)
		if res == nil {
			t.Errorf("%s: parsing gave unexpected nil", c.desc)
		}
		got := updateSharedLibLine(c.original, res)
		if got != c.want {
			t.Errorf("%s: incorrect result: got\n%s\nwant\n%s", c.desc, got, c.want)
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
		buf := bytes.NewBuffer([]byte(c.input))
		br := bufio.NewReader(buf)
		handler := &ioHandler{
			p:  &testPrinter{},
			br: br,
		}
		cfs := &configFileState{lines: []string{}}
		err := processNoSharedLibLine(handler, cfs)
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
			prints:     []string{"#" + okLine + "\n", okLine + "\n"},
			successMsg: successSharedLibUpdated,
		},
		{
			desc:       "success on 2nd prompt",
			lines:      []string{"#" + okLine},
			input:      " \ny\n",
			shouldErr:  false,
			prompts:    2,
			statements: 3,
			prints:     []string{"#" + okLine + "\n", okLine + "\n"},
			successMsg: successSharedLibUpdated,
		},
		{
			desc:       "fail",
			lines:      []string{"#" + okLine},
			input:      " \nn\n",
			shouldErr:  true,
			prompts:    2,
			statements: 3,
			prints:     []string{"#" + okLine + "\n", okLine + "\n"},
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

		prints := []string{}
		printFn = func(format string, args ...interface{}) (int, error) {
			prints = append(prints, fmt.Sprintf(format, args...))
			return 0, nil
		}

		err := processSharedLibLine(handler, cfs)
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
			temp = temp + float64(temp)*(fudgeFactor-.01)
			c.parseResults[k].value = parse.BytesToPGFormat(uint64(temp))
		}
		// change values to higher fudge factor, so it should be shown
		for _, k := range c.highFudge {
			temp, err := parse.PGFormatToBytes(c.parseResults[k].value)
			if err != nil {
				t.Errorf("%s: unexpected err in parsing: %v", c.desc, err)
			}
			temp = temp + float64(temp)*(fudgeFactor+.01)
			c.parseResults[k].value = parse.BytesToPGFormat(uint64(temp))
		}
		// change values to lower fudge factor, so it should be shown
		for _, k := range c.lowFudge {
			temp, err := parse.PGFormatToBytes(c.parseResults[k].value)
			if err != nil {
				t.Errorf("%s: unexpected err in parsing: %v", c.desc, err)
			}
			temp = temp - float64(temp)*(fudgeFactor+.01)
			c.parseResults[k].value = parse.BytesToPGFormat(uint64(temp))
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

func TestProcessSettingsGroup(t *testing.T) {
	cases := []struct {
		desc           string
		ts             *settingsGroup
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
			desc: "no keys, no need to prompt",
			ts: &settingsGroup{
				label: "foo",
				rec:   nil,
				keys:  nil,
			},
			lines:          memSettingsCorrect,
			wantStatements: 1, // only intro remark
			wantPrompts:    0,
			wantPrints:     1, // one for initial newline
			successMsg:     "foo settings are already tuned",
			shouldErr:      false,
		},
		{
			desc: "memory - commented",
			ts: &settingsGroup{
				label: "memory",
				rec:   pgtune.NewMemoryRecommender(8*parse.Gigabyte, 4),
				keys:  pgtune.MemoryKeys,
			},
			lines:          memSettingsCommented,
			stdin:          "y\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    1,
			wantPrints:     3, // one for initial newline + one setting, displayed twice
			successMsg:     "memory settings will be updated",
			shouldErr:      false,
		},
		{
			desc: "memory - wrong",
			ts: &settingsGroup{
				label: "memory",
				rec:   pgtune.NewMemoryRecommender(8*parse.Gigabyte, 4),
				keys:  pgtune.MemoryKeys,
			},
			lines:          memSettingsWrongVal,
			stdin:          "y\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    1,
			wantPrints:     3, // one for initial newline + one setting, displayed twice
			successMsg:     "memory settings will be updated",
			shouldErr:      false,
		},
		{
			desc: "memory - missing",
			ts: &settingsGroup{
				label: "memory",
				rec:   pgtune.NewMemoryRecommender(8*parse.Gigabyte, 4),
				keys:  pgtune.MemoryKeys,
			},
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
			desc: "memory - comment+wrong",
			ts: &settingsGroup{
				label: "memory",
				rec:   pgtune.NewMemoryRecommender(8*parse.Gigabyte, 4),
				keys:  pgtune.MemoryKeys,
			},
			lines:          memSettingsCommentWrong,
			stdin:          " \ny\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    2, // first input is blank
			wantPrints:     5, // one for initial newline + two settings, displayed twice
			successMsg:     "memory settings will be updated",
			shouldErr:      false,
		},
		{
			desc: "memory - comment+wrong+missing",
			ts: &settingsGroup{
				label: "memory",
				rec:   pgtune.NewMemoryRecommender(8*parse.Gigabyte, 4),
				keys:  pgtune.MemoryKeys,
			},
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
			desc: "memory - all wrong, but skip",
			ts: &settingsGroup{
				label: "memory",
				rec:   pgtune.NewMemoryRecommender(8*parse.Gigabyte, 4),
				keys:  pgtune.MemoryKeys,
			},
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
			desc: "memory - all wrong, but quit",
			ts: &settingsGroup{
				label: "memory",
				rec:   pgtune.NewMemoryRecommender(8*parse.Gigabyte, 4),
				keys:  pgtune.MemoryKeys,
			},
			lines:          memSettingsAllWrong,
			stdin:          " \nqUIt\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    2,
			wantPrints:     9, // one for initial newline + four settings, displayed twice
			successMsg:     "",
			shouldErr:      true,
		},
		{
			desc: "memory - all wrong",
			ts: &settingsGroup{
				label: "memory",
				rec:   pgtune.NewMemoryRecommender(8*parse.Gigabyte, 4),
				keys:  pgtune.MemoryKeys,
			},
			lines:          memSettingsAllWrong,
			stdin:          "y\n",
			wantStatements: 3, // intro remark + current label + recommend label
			wantPrompts:    1,
			wantPrints:     9, // one for initial newline + four settings, displayed twice
			successMsg:     "memory settings will be updated",
			shouldErr:      false,
		},
		{
			desc: "label capitalized",
			ts: &settingsGroup{
				label: "WAL",
				rec:   nil,
				keys:  []string{},
			},
			lines:          []string{},
			stdin:          "y\n",
			wantStatements: 1,
			wantPrints:     1, // one for initial newline
			successMsg:     "WAL settings are already tuned",
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
			for _, k := range c.ts.keys {
				p := parseWithRegex(l, regexes[k])
				if p != nil {
					p.idx = i
					cfs.tuneParseResults[k] = p
				}
			}
		}
		numPrints := uint64(0)
		printFn = func(format string, args ...interface{}) (int, error) {
			numPrints++
			return 0, nil
		}

		err := c.ts.process(handler, cfs, false)
		if err != nil && !c.shouldErr {
			t.Errorf("%s: unexpected error: %v", c.desc, err)
		} else if err == nil && c.shouldErr {
			t.Errorf("%s: unexpected lack of error", c.desc)
		}

		tp := handler.p.(*testPrinter)
		if got := strings.ToUpper(strings.TrimSpace(tp.statements[0])[:1]); got != strings.ToUpper(c.ts.label[:1]) {
			t.Errorf("%s: label not capitalized in first statement: got %s want %s", c.desc, got, strings.ToUpper(c.ts.label[:1]))
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
	oldPrintFn := printFn
	printFn = func(_ string, _ ...interface{}) (int, error) {
		return 0, nil
	}

	buf := bytes.NewBuffer([]byte("y\ny\ny\ny\n"))
	br := bufio.NewReader(buf)
	handler := &ioHandler{
		p:  &testPrinter{},
		br: br,
	}

	cfs := &configFileState{lines: []string{}, tuneParseResults: make(map[string]*tunableParseResult)}
	processTunables(handler, cfs, mem, cpus, false)

	tp := handler.p.(*testPrinter)
	// Total number of statements is intro statement and then 3 per group of settings;
	// each group has a heading and then the current/recommended labels.
	if got := tp.statementCalls; got != 1+3*4 {
		t.Errorf("incorrect number of statements: got %d, want %d", got, 1+3*4)
	}

	wantStatement := fmt.Sprintf(statementTunableIntro, parse.BytesToDecimalFormat(mem), cpus)
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
	oldPrintFn := printFn
	printFn = func(_ string, _ ...interface{}) (int, error) {
		return 0, nil
	}

	buf := bytes.NewBuffer([]byte("y\ny\ny\ny\n"))
	br := bufio.NewReader(buf)
	handler := &ioHandler{
		p:  &testPrinter{},
		br: br,
	}

	cfs := &configFileState{lines: []string{}, tuneParseResults: make(map[string]*tunableParseResult)}
	processTunables(handler, cfs, mem, cpus, false)

	tp := handler.p.(*testPrinter)
	// Total number of statements is intro statement and then 3 per group of settings;
	// each group has a heading and then the current/recommended labels.
	// On a single CPU, only 3 groups since parallelism does not apply
	if got := tp.statementCalls; got != 1+3*3 {
		t.Errorf("incorrect number of statements: got %d, want %d", got, 1+3*3)
	}

	wantStatement := fmt.Sprintf(statementTunableIntro, parse.BytesToDecimalFormat(mem), cpus)
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
