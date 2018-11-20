package main

import (
	"fmt"
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
			input:     2000 * terabyte,
			wantNum:   2000,
			wantUnits: tb,
		},
		{
			desc:      "1 TB",
			input:     terabyte,
			wantNum:   1.0,
			wantUnits: tb,
		},
		{
			desc:      "1.5 TB",
			input:     uint64(1.5 * float64(terabyte)),
			wantNum:   1.5,
			wantUnits: tb,
		},
		{
			desc:      "1TB - 1GB",
			input:     terabyte - gigabyte,
			wantNum:   1023,
			wantUnits: gb,
		},
		{
			desc:      "1 GB",
			input:     gigabyte,
			wantNum:   1.0,
			wantUnits: gb,
		},
		{
			desc:      "1.5 GB",
			input:     uint64(1.5 * float64(gigabyte)),
			wantNum:   1.5,
			wantUnits: gb,
		},
		{
			desc:      "2.0 GB",
			input:     2 * gigabyte,
			wantNum:   2.0,
			wantUnits: gb,
		},
		{
			desc:      "1 GB - 1 MB",
			input:     gigabyte - megabyte,
			wantNum:   1023.0,
			wantUnits: mb,
		},
		{
			desc:      "1 MB",
			input:     megabyte,
			wantNum:   1.0,
			wantUnits: mb,
		},
		{
			desc:      "1.5 MB",
			input:     uint64(1.5 * float64(megabyte)),
			wantNum:   1.5,
			wantUnits: mb,
		},
		{
			desc:      "1020 kB",
			input:     megabyte - 4*kilobyte,
			wantNum:   1020.0,
			wantUnits: kb,
		},
		{
			desc:      "1 kB",
			input:     kilobyte,
			wantNum:   1.0,
			wantUnits: kb,
		},
		{
			desc:      "1.5 kB",
			input:     uint64(1.5 * float64(kilobyte)),
			wantNum:   1.5,
			wantUnits: kb,
		},
		{
			desc:      "1000 bytes",
			input:     kilobyte - 24,
			wantNum:   1000,
			wantUnits: b,
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

func TestBytesPGFormat(t *testing.T) {
	cases := []struct {
		desc  string
		input uint64
		want  string
	}{
		{
			desc:  "no limit to TB",
			input: 2000 * terabyte,
			want:  "2000" + tb,
		},
		{
			desc:  "1 TB",
			input: terabyte,
			want:  "1" + tb,
		},
		{
			desc:  "1.5 TB",
			input: uint64(1.5 * float64(terabyte)),
			want:  "1536" + gb,
		},
		{
			desc:  "1TB - 1GB",
			input: terabyte - gigabyte,
			want:  "1023" + gb,
		},
		{
			desc:  "1TB - 1MB",
			input: terabyte - megabyte,
			want:  "1048575" + mb,
		},
		{
			desc:  "1 GB",
			input: gigabyte,
			want:  "1" + gb,
		},
		{
			desc:  "1.5 GB",
			input: uint64(1.5 * float64(gigabyte)),
			want:  "1536" + mb,
		},
		{
			desc:  "2.0 GB",
			input: 2 * gigabyte,
			want:  "2" + gb,
		},
		{
			desc:  "1 GB - 1MB",
			input: gigabyte - megabyte,
			want:  "1023" + mb,
		},
		{
			desc:  "1 MB",
			input: megabyte,
			want:  "1" + mb,
		},
		{
			desc:  "1.5 MB",
			input: uint64(1.5 * float64(megabyte)),
			want:  "1536" + kb,
		},
		{
			desc:  "1020 kB",
			input: megabyte - 4*kilobyte,
			want:  "1020" + kb,
		},
		{
			desc:  "1 kB",
			input: kilobyte,
			want:  "1" + kb,
		},
		{
			desc:  "1.5 kB, round up",
			input: uint64(1.5 * float64(kilobyte)),
			want:  "2" + kb,
		},
		{
			desc:  "1.4 kB, round down",
			input: 1400,
			want:  "1" + kb,
		},
		{
			desc:  "1000 bytes",
			input: kilobyte - 24,
			want:  "1" + kb,
		},
	}

	for _, c := range cases {
		if got := bytesPGFormat(c.input); got != c.want {
			t.Errorf("%s: incorrect return: got %s want %s", c.desc, got, c.want)
		}
	}
}

func TestParsePGStringToBytes(t *testing.T) {
	cases := []struct {
		desc   string
		input  string
		want   float64
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
			desc:  "valid kilobytes",
			input: "64" + kb,
			want:  64 * kilobyte,
		},
		{
			desc:  "valid kilobytes, oversized",
			input: "2048" + kb,
			want:  2048 * kilobyte,
		},
		{
			desc:  "valid megabytes",
			input: "64" + mb,
			want:  64 * megabyte,
		},
		{
			desc:  "valid megabytes, oversized",
			input: "2048" + mb,
			want:  2048 * megabyte,
		},
		{
			desc:  "valid gigabytes",
			input: "64" + gb,
			want:  64 * gigabyte,
		},
		{
			desc:  "valid gigabytes, oversized",
			input: "2048" + gb,
			want:  2048 * gigabyte,
		},
		{
			desc:  "valid terabytes",
			input: "64" + tb,
			want:  64 * terabyte,
		},
		{
			desc:  "valid terabytes, oversized",
			input: "2048" + tb,
			want:  2048 * terabyte,
		},
	}

	for _, c := range cases {
		bytes, err := parsePGStringToBytes(c.input)
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
				t.Errorf("%s: incorrect bytes: got %f want %f", c.desc, got, c.want)
			}
		}

	}
}
