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
			errMsg: fmt.Sprintf(errIncorrectFormatFmt, " 64MB"),
		},
		{
			desc:   "incorrect format #2",
			input:  "64b", // bytes not allowed
			errMsg: fmt.Sprintf(errIncorrectFormatFmt, "64b"),
		},
		{
			desc:   "incorrect format #3",
			input:  "64 GB", // no space between num and units,
			errMsg: fmt.Sprintf(errIncorrectFormatFmt, "64 GB"),
		},
		{
			desc:   "incorrect format #4",
			input:  "-64MB", // negative memory is a no-no
			errMsg: fmt.Sprintf(errIncorrectFormatFmt, "-64MB"),
		},
		{
			desc:   "incorrect format #5",
			input:  tooBigInt + MB,
			errMsg: fmt.Sprintf(errCouldNotParseBytesFmt, tooBigErr),
		},
		{
			desc:   "incorrect format #6",
			input:  "5.5" + MB, // decimal memory is a no-no
			errMsg: fmt.Sprintf(errIncorrectFormatFmt, "5.5"+MB),
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
