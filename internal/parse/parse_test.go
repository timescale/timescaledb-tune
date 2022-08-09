package parse

import (
	"fmt"
	"strconv"
	"testing"
)

func TestParseIntToFloatUnits(t *testing.T) {
	cases := []struct {
		desc      string
		input     uint64
		wantNum   float64
		wantUnits string
	}{
		{
			desc:      "no limit to TB",
			input:     2000 * Terabyte,
			wantNum:   2000,
			wantUnits: TB,
		},
		{
			desc:      "1 TB",
			input:     Terabyte,
			wantNum:   1.0,
			wantUnits: TB,
		},
		{
			desc:      "1.5 TB",
			input:     uint64(1.5 * float64(Terabyte)),
			wantNum:   1.5,
			wantUnits: TB,
		},
		{
			desc:      "1TB - 1GB",
			input:     Terabyte - Gigabyte,
			wantNum:   1023,
			wantUnits: GB,
		},
		{
			desc:      "1 GB",
			input:     Gigabyte,
			wantNum:   1.0,
			wantUnits: GB,
		},
		{
			desc:      "1.5 GB",
			input:     uint64(1.5 * float64(Gigabyte)),
			wantNum:   1.5,
			wantUnits: GB,
		},
		{
			desc:      "2.0 GB",
			input:     2 * Gigabyte,
			wantNum:   2.0,
			wantUnits: GB,
		},
		{
			desc:      "1 GB - 1 MB",
			input:     Gigabyte - Megabyte,
			wantNum:   1023.0,
			wantUnits: MB,
		},
		{
			desc:      "1 MB",
			input:     Megabyte,
			wantNum:   1.0,
			wantUnits: MB,
		},
		{
			desc:      "1.5 MB",
			input:     uint64(1.5 * float64(Megabyte)),
			wantNum:   1.5,
			wantUnits: MB,
		},
		{
			desc:      "1020 kB",
			input:     Megabyte - 4*Kilobyte,
			wantNum:   1020.0,
			wantUnits: KB,
		},
		{
			desc:      "1 kB",
			input:     Kilobyte,
			wantNum:   1.0,
			wantUnits: KB,
		},
		{
			desc:      "1.5 kB",
			input:     uint64(1.5 * float64(Kilobyte)),
			wantNum:   1.5,
			wantUnits: KB,
		},
		{
			desc:      "1000 bytes",
			input:     Kilobyte - 24,
			wantNum:   1000,
			wantUnits: B,
		},
	}

	for _, c := range cases {
		val, units := parseIntToFloatUnits(c.input)
		if got := val; got != c.wantNum {
			t.Errorf("%s: incorrect val: got %f want %f", c.desc, got, c.wantNum)
		}
		if got := units; got != c.wantUnits {
			t.Errorf("%s: incorrect units: got %s want %s", c.desc, got, c.wantUnits)
		}
	}
}

func TestParseIntToFloatUnitsPanic(t *testing.T) {
	func() {
		defer func() {
			if re := recover(); re == nil {
				t.Errorf("did not panic when should")
			}
		}()
		parseIntToFloatUnits(0)
	}()
}

func TestBytesToDecimalFormat(t *testing.T) {
	cases := []struct {
		desc  string
		input uint64
		want  string
	}{
		{
			desc:  "no limit to TB",
			input: 2000 * Terabyte,
			want:  "2000.00 " + TB,
		},
		{
			desc:  "1 TB",
			input: Terabyte,
			want:  "1.00 " + TB,
		},
		{
			desc:  "1.5 TB",
			input: uint64(1.5 * float64(Terabyte)),
			want:  "1.50 " + TB,
		},
		{
			desc:  "1.25 TB",
			input: uint64(1.25 * float64(Terabyte)),
			want:  "1.25 " + TB,
		},
		{
			desc:  ".50 TB",
			input: uint64(.50 * float64(Terabyte)),
			want:  "512.00 " + GB,
		},
	}

	for _, c := range cases {
		if got := BytesToDecimalFormat(c.input); got != c.want {
			t.Errorf("%s: incorrect return: got %s want %s", c.desc, got, c.want)
		}
	}
}

func TestBytesToPGFormat(t *testing.T) {
	cases := []struct {
		desc  string
		input uint64
		want  string
	}{
		{
			desc:  "no limit to TB",
			input: 2000 * Terabyte,
			want:  "2000" + TB,
		},
		{
			desc:  "1 TB",
			input: Terabyte,
			want:  "1" + TB,
		},
		{
			desc:  "1.5 TB",
			input: uint64(1.5 * float64(Terabyte)),
			want:  "1536" + GB,
		},
		{
			desc:  "1TB - 1GB",
			input: Terabyte - Gigabyte,
			want:  "1023" + GB,
		},
		{
			desc:  "1TB - 1MB",
			input: Terabyte - Megabyte,
			want:  "1048575" + MB,
		},
		{
			desc:  "1 GB",
			input: Gigabyte,
			want:  "1" + GB,
		},
		{
			desc:  "1.5 GB",
			input: uint64(1.5 * float64(Gigabyte)),
			want:  "1536" + MB,
		},
		{
			desc:  "2.0 GB",
			input: 2 * Gigabyte,
			want:  "2" + GB,
		},
		{
			desc:  "1 GB - 1MB",
			input: Gigabyte - Megabyte,
			want:  "1023" + MB,
		},
		{
			desc:  "1 MB",
			input: Megabyte,
			want:  "1" + MB,
		},
		{
			desc:  "1.5 MB",
			input: uint64(1.5 * float64(Megabyte)),
			want:  "1536" + KB,
		},
		{
			desc:  "1020 kB",
			input: Megabyte - 4*Kilobyte,
			want:  "1020" + KB,
		},
		{
			desc:  "1 kB",
			input: Kilobyte,
			want:  "1" + KB,
		},
		{
			desc:  "1.5 kB, round up",
			input: uint64(1.5 * float64(Kilobyte)),
			want:  "2" + KB,
		},
		{
			desc:  "1.4 kB, round down",
			input: 1400,
			want:  "1" + KB,
		},
		{
			desc:  "1000 bytes",
			input: Kilobyte - 24,
			want:  "1" + KB,
		},
	}

	for _, c := range cases {
		if got := BytesToPGFormat(c.input); got != c.want {
			t.Errorf("%s: incorrect return: got %s want %s", c.desc, got, c.want)
		}
	}
}

func TestPGFormatToBytes(t *testing.T) {
	tooBigInt := "9223372036854775808"
	_, tooBigErr := strconv.ParseInt(tooBigInt, 10, 64)
	cases := []struct {
		desc   string
		input  string
		want   uint64
		errMsg string
	}{
		{
			desc:   "incorrect format #1",
			input:  " 64MB", // no leading spaces
			errMsg: fmt.Sprintf(errIncorrectBytesFormatFmt, " 64MB"),
		},
		{
			desc:   "incorrect format #2",
			input:  "64b", // bytes not allowed
			errMsg: fmt.Sprintf(errIncorrectBytesFormatFmt, "64b"),
		},
		{
			desc:   "incorrect format #3",
			input:  "64 GB", // no space between num and units,
			errMsg: fmt.Sprintf(errIncorrectBytesFormatFmt, "64 GB"),
		},
		{
			desc:   "incorrect format #4",
			input:  "-64MB", // negative memory is a no-no
			errMsg: fmt.Sprintf(errIncorrectBytesFormatFmt, "-64MB"),
		},
		{
			desc:   "incorrect format #5",
			input:  tooBigInt + MB,
			errMsg: fmt.Sprintf(errCouldNotParseBytesFmt, tooBigErr),
		},
		{
			desc:   "incorrect format #6",
			input:  "5.5" + MB, // decimal memory is a no-no
			errMsg: fmt.Sprintf(errIncorrectBytesFormatFmt, "5.5"+MB),
		},
		{
			desc:  "valid bytes",
			input: "65536" + B,
			want:  64 * Kilobyte,
		},
		{
			desc:  "valid kilobytes",
			input: "64" + KB,
			want:  64 * Kilobyte,
		},
		{
			desc:  "valid kilobytes, oversized",
			input: "2048" + KB,
			want:  2048 * Kilobyte,
		},
		{
			desc:  "valid megabytes",
			input: "64" + MB,
			want:  64 * Megabyte,
		},
		{
			desc:  "valid megabytes, oversized",
			input: "2048" + MB,
			want:  2048 * Megabyte,
		},
		{
			desc:  "valid gigabytes",
			input: "64" + GB,
			want:  64 * Gigabyte,
		},
		{
			desc:  "valid gigabytes, oversized",
			input: "2048" + GB,
			want:  2048 * Gigabyte,
		},
		{
			desc:  "valid terabytes",
			input: "64" + TB,
			want:  64 * Terabyte,
		},
		{
			desc:  "valid terabytes, oversized",
			input: "2048" + TB,
			want:  2048 * Terabyte,
		},
		{
			desc:  "valid megabytes, wrapped in single-quotes",
			input: "'64MB'",
			want:  64 * Megabyte,
		},
	}

	for _, c := range cases {
		bytes, err := PGFormatToBytes(c.input)
		if len(c.errMsg) > 0 { // failure cases
			if err == nil {
				t.Errorf("%s: unexpectedly err is nil: want %s", c.desc, c.errMsg)
			} else if got := err.Error(); got != c.errMsg {
				t.Errorf("%s: unexpected err msg: got\n%s\nwant\n%s", c.desc, got, c.errMsg)
			}
		} else {
			if err != nil {
				t.Errorf("%s: unexpected err: got %v", c.desc, err)
			}
			if got := bytes; got != c.want {
				t.Errorf("%s: incorrect bytes: got %d want %d", c.desc, got, c.want)
			}
		}

	}
}

func TestPGFormatToTime(t *testing.T) {
	cases := []struct {
		desc      string
		input     string
		defUnits  TimeUnit
		varType   VarType
		wantNum   float64
		wantUnits TimeUnit
		errMsg    string
	}{
		{
			desc:      "set statement_timeout to '13ms';",
			input:     "13ms",
			defUnits:  Milliseconds,
			varType:   VarTypeInteger,
			wantNum:   13.0,
			wantUnits: Milliseconds,
		},
		{
			desc:      "set statement_timeout to '13ms'; #2",
			input:     "'13ms'",
			defUnits:  Milliseconds,
			varType:   VarTypeInteger,
			wantNum:   13.0,
			wantUnits: Milliseconds,
		},
		{
			desc:      "set statement_timeout to 7;",
			input:     "7",
			defUnits:  Milliseconds,
			varType:   VarTypeInteger,
			wantNum:   7.0,
			wantUnits: Milliseconds,
		},
		{
			desc:      "set statement_timeout to 13;",
			input:     "13",
			defUnits:  Milliseconds,
			varType:   VarTypeInteger,
			wantNum:   13.0,
			wantUnits: Milliseconds,
		},
		{
			desc:      "set statement_timeout to 13.5;",
			input:     "13.5",
			defUnits:  Milliseconds,
			varType:   VarTypeInteger,
			wantNum:   14.0,
			wantUnits: Milliseconds,
		},
		{
			desc:      "set statement_timeout to 13.4;",
			input:     "13.4",
			defUnits:  Milliseconds,
			varType:   VarTypeInteger,
			wantNum:   13.0,
			wantUnits: Milliseconds,
		},
		{
			desc:      "set statement_timeout to '13.4ms';",
			input:     "13.4ms",
			defUnits:  Milliseconds,
			varType:   VarTypeInteger,
			wantNum:   13.0,
			wantUnits: Milliseconds,
		},
		{
			desc:      "set statement_timeout to '13min';",
			input:     "13min",
			defUnits:  Milliseconds,
			varType:   VarTypeInteger,
			wantNum:   13.0,
			wantUnits: Minutes,
		},
		{
			desc:      "set statement_timeout to '13.0min';",
			input:     "13.0min",
			defUnits:  Milliseconds,
			varType:   VarTypeInteger,
			wantNum:   13.0,
			wantUnits: Minutes,
		},
		{
			desc:      "set statement_timeout to '1.5s';",
			input:     "1.5s",
			defUnits:  Milliseconds,
			varType:   VarTypeInteger,
			wantNum:   1500.0,
			wantUnits: Milliseconds,
		},
		{
			desc:      "set statement_timeout to '1.5min';",
			input:     "1.5min",
			defUnits:  Milliseconds,
			varType:   VarTypeInteger,
			wantNum:   90.0,
			wantUnits: Seconds,
		},
		{
			desc:      "set statement_timeout to '1.3h';",
			input:     "1.3h",
			defUnits:  Milliseconds,
			varType:   VarTypeInteger,
			wantNum:   78.0,
			wantUnits: Minutes,
		},
		{
			desc:      "set statement_timeout to '42.0 min';",
			input:     "42.0 min",
			defUnits:  Milliseconds,
			varType:   VarTypeInteger,
			wantNum:   42.0,
			wantUnits: Minutes,
		},
		{
			desc:      "set statement_timeout to '42.1 min';",
			input:     "42.1 min",
			defUnits:  Milliseconds,
			varType:   VarTypeInteger,
			wantNum:   2526.0,
			wantUnits: Seconds,
		},
		{
			desc:     "set statement_timeout to 'bob';",
			input:    "bob",
			defUnits: Milliseconds,
			varType:  VarTypeInteger,
			errMsg:   fmt.Sprintf(errIncorrectTimeFormatFmt, "bob"),
		},
		{
			desc:     "set statement_timeout to '42 bob';",
			input:    "42 bob",
			defUnits: Milliseconds,
			varType:  VarTypeInteger,
			errMsg:   fmt.Sprintf(errIncorrectTimeFormatFmt, "42 bob"),
		},
		{
			desc:      "set vacuum_cost_delay to 250;",
			input:     "250",
			defUnits:  Milliseconds,
			varType:   VarTypeReal,
			wantNum:   250.0,
			wantUnits: Milliseconds,
		},
		{
			desc:      "set vacuum_cost_delay to 250.0;",
			input:     "250.0",
			defUnits:  Milliseconds,
			varType:   VarTypeReal,
			wantNum:   250.0,
			wantUnits: Milliseconds,
		},
		{
			desc:      "set vacuum_cost_delay to 1.3;",
			input:     "1.3",
			defUnits:  Milliseconds,
			varType:   VarTypeReal,
			wantNum:   1300.0,
			wantUnits: Microseconds,
		},
		{
			desc:      "set vacuum_cost_delay to '1.3ms';",
			input:     "1.3ms",
			defUnits:  Milliseconds,
			varType:   VarTypeReal,
			wantNum:   1300.0,
			wantUnits: Microseconds,
		},
		{
			desc:      "set vacuum_cost_delay to '1300us';",
			input:     "1300us",
			defUnits:  Milliseconds,
			varType:   VarTypeReal,
			wantNum:   1300.0,
			wantUnits: Microseconds,
		},
		{
			desc:     "37.1 goats",
			input:    "37.1 goats",
			defUnits: Milliseconds,
			varType:  VarTypeReal,
			errMsg:   fmt.Sprintf(errIncorrectTimeFormatFmt, "37.1 goats"),
		},
		{
			desc:     "37.42.1min",
			input:    "37.42.1min",
			defUnits: Milliseconds,
			varType:  VarTypeReal,
			errMsg:   fmt.Sprintf(errIncorrectTimeFormatFmt, "37.42.1min"),
		},
	}

	for _, c := range cases {
		v, u, err := PGFormatToTime(c.input, c.defUnits, c.varType)
		if len(c.errMsg) > 0 { // failure cases
			if err == nil {
				t.Errorf("%s: unexpectedly err is nil: want %s", c.desc, c.errMsg)
			} else if got := err.Error(); got != c.errMsg {
				t.Errorf("%s: unexpected err msg: got\n%s\nwant\n%s", c.desc, got, c.errMsg)
			}
		} else {
			if err != nil {
				t.Errorf("%s: unexpected err: got %v", c.desc, err)
			}
			if got := v; got != c.wantNum {
				t.Errorf("%s: incorrect num: got %f want %f", c.desc, got, c.wantNum)
			}
			if got := u; got != c.wantUnits {
				t.Errorf("%s: incorrect units: got %s want %s", c.desc, got, c.wantUnits)
			}
		}
	}
}

func TestTimeConversion(t *testing.T) {
	// test cases generated with the following query
	/*
		with x(unit, const, val) as
		(
		    values
		        ('us',  'Microseconds' , interval '1 microsecond'),
		        ('ms',  'Milliseconds' , interval '1 millisecond'),
		        ('s',   'Seconds'      , interval '1 second'),
		        ('min', 'Minutes'      , interval '1 minute'),
		        ('h',   'Hours'        , interval '1 hour'),
		        ('d',   'Days'         , interval '24 hours')
		)
		select string_agg(format
		(
		$${
		    desc: "%s -> %s",
		    from: %s,
		    to:   %s,
		    want: %s / %s,
		}$$,
		    f.unit,
		    t.unit,
		    f.const,
		    t.const,
		    extract(epoch from f.val),
		    extract(epoch from t.val)
		), E',\n' order by f.const, t.const)
		from x f
		cross join x t
		;
	*/

	cases := []struct {
		desc   string
		from   TimeUnit
		to     TimeUnit
		want   float64
		errMsg string
	}{
		{
			desc: "d -> d",
			from: Days,
			to:   Days,
			want: 86400.000000 / 86400.000000,
		},
		{
			desc: "d -> h",
			from: Days,
			to:   Hours,
			want: 86400.000000 / 3600.000000,
		},
		{
			desc: "d -> us",
			from: Days,
			to:   Microseconds,
			want: 86400.000000 / 0.000001,
		},
		{
			desc: "d -> ms",
			from: Days,
			to:   Milliseconds,
			want: 86400.000000 / 0.001000,
		},
		{
			desc: "d -> min",
			from: Days,
			to:   Minutes,
			want: 86400.000000 / 60.000000,
		},
		{
			desc: "d -> s",
			from: Days,
			to:   Seconds,
			want: 86400.000000 / 1.000000,
		},
		{
			desc: "h -> d",
			from: Hours,
			to:   Days,
			want: 3600.000000 / 86400.000000,
		},
		{
			desc: "h -> h",
			from: Hours,
			to:   Hours,
			want: 3600.000000 / 3600.000000,
		},
		{
			desc: "h -> us",
			from: Hours,
			to:   Microseconds,
			want: 3600.000000 / 0.000001,
		},
		{
			desc: "h -> ms",
			from: Hours,
			to:   Milliseconds,
			want: 3600.000000 / 0.001000,
		},
		{
			desc: "h -> min",
			from: Hours,
			to:   Minutes,
			want: 3600.000000 / 60.000000,
		},
		{
			desc: "h -> s",
			from: Hours,
			to:   Seconds,
			want: 3600.000000 / 1.000000,
		},
		{
			desc: "us -> d",
			from: Microseconds,
			to:   Days,
			want: 0.000001 / 86400.000000,
		},
		{
			desc: "us -> h",
			from: Microseconds,
			to:   Hours,
			want: 0.000001 / 3600.000000,
		},
		{
			desc: "us -> us",
			from: Microseconds,
			to:   Microseconds,
			want: 0.000001 / 0.000001,
		},
		{
			desc: "us -> ms",
			from: Microseconds,
			to:   Milliseconds,
			want: 0.000001 / 0.001000,
		},
		{
			desc: "us -> min",
			from: Microseconds,
			to:   Minutes,
			want: 0.000001 / 60.000000,
		},
		{
			desc: "us -> s",
			from: Microseconds,
			to:   Seconds,
			want: 0.000001 / 1.000000,
		},
		{
			desc: "ms -> d",
			from: Milliseconds,
			to:   Days,
			want: 0.001000 / 86400.000000,
		},
		{
			desc: "ms -> h",
			from: Milliseconds,
			to:   Hours,
			want: 0.001000 / 3600.000000,
		},
		{
			desc: "ms -> us",
			from: Milliseconds,
			to:   Microseconds,
			want: 0.001000 / 0.000001,
		},
		{
			desc: "ms -> ms",
			from: Milliseconds,
			to:   Milliseconds,
			want: 0.001000 / 0.001000,
		},
		{
			desc: "ms -> min",
			from: Milliseconds,
			to:   Minutes,
			want: 0.001000 / 60.000000,
		},
		{
			desc: "ms -> s",
			from: Milliseconds,
			to:   Seconds,
			want: 0.001000 / 1.000000,
		},
		{
			desc: "min -> d",
			from: Minutes,
			to:   Days,
			want: 60.000000 / 86400.000000,
		},
		{
			desc: "min -> h",
			from: Minutes,
			to:   Hours,
			want: 60.000000 / 3600.000000,
		},
		{
			desc: "min -> us",
			from: Minutes,
			to:   Microseconds,
			want: 60.000000 / 0.000001,
		},
		{
			desc: "min -> ms",
			from: Minutes,
			to:   Milliseconds,
			want: 60.000000 / 0.001000,
		},
		{
			desc: "min -> min",
			from: Minutes,
			to:   Minutes,
			want: 60.000000 / 60.000000,
		},
		{
			desc: "min -> s",
			from: Minutes,
			to:   Seconds,
			want: 60.000000 / 1.000000,
		},
		{
			desc: "s -> d",
			from: Seconds,
			to:   Days,
			want: 1.000000 / 86400.000000,
		},
		{
			desc: "s -> h",
			from: Seconds,
			to:   Hours,
			want: 1.000000 / 3600.000000,
		},
		{
			desc: "s -> us",
			from: Seconds,
			to:   Microseconds,
			want: 1.000000 / 0.000001,
		},
		{
			desc: "s -> ms",
			from: Seconds,
			to:   Milliseconds,
			want: 1.000000 / 0.001000,
		},
		{
			desc: "s -> min",
			from: Seconds,
			to:   Minutes,
			want: 1.000000 / 60.000000,
		},
		{
			desc: "s -> s",
			from: Seconds,
			to:   Seconds,
			want: 1.000000 / 1.000000,
		},
	}

	for _, c := range cases {
		conv, err := TimeConversion(c.from, c.to)
		if c.errMsg != "" {
			if err != nil {
				t.Errorf("%s: unexpectedly err is nil: want %s", c.desc, c.errMsg)
			} else if got := err.Error(); got != c.errMsg {
				t.Errorf("%s: unexpected err msg: got\n%s\nwant\n%s", c.desc, got, c.errMsg)
			}
		} else {
			if err != nil {
				t.Errorf("%s: unexpected err: got %v", c.desc, err)
			}
			if got := conv; got != c.want {
				t.Errorf("%s: incorrect conv: got %f want %f", c.desc, got, c.want)
			}
		}
	}
}
