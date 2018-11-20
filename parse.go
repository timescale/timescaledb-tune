package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
)

const (
	terabyte = 1 << 40
	gigabyte = 1 << 30
	megabyte = 1 << 20
	kilobyte = 1 << 10
	tb       = "TB"
	gb       = "GB"
	mb       = "MB"
	kb       = "kB"
	b        = "bytes"

	errIncorrectFormatFmt = "incorrect format for '%s'"
)

var pgBytesRegex = regexp.MustCompile("^([0-9]+)((?:k|M|G|T)B)$")

func parseIntToFloatUnits(bytes uint64) (float64, string) {
	if bytes <= 0 {
		panic("bytes must be at least 1 byte")
	}
	divisor := 1.0
	units := b
	if bytes >= terabyte {
		divisor = float64(terabyte)
		units = tb
	} else if bytes >= gigabyte {
		divisor = float64(gigabyte)
		units = gb
	} else if bytes >= megabyte {
		divisor = float64(megabyte)
		units = mb
	} else if bytes >= kilobyte {
		divisor = float64(kilobyte)
		units = kb
	}
	return float64(bytes) / divisor, units
}

func bytesFormat(bytes uint64) string {
	val, units := parseIntToFloatUnits(bytes)
	return fmt.Sprintf("%0.2f %s", val, units)
}

func bytesPGFormat(bytes uint64) string {
	val, units := parseIntToFloatUnits(bytes)
	if units == b { // nothing less than 1kB allowed
		val = 1.0
		units = kb
	} else if units == kb {
		val = math.Round(val)
	} else {
		if val-float64(uint64(val)) > 0.001 { // (anything less than .001 is not going to meaningfully change at 1024x)
			val = val * 1024
			if units == tb {
				units = gb
			} else if units == gb {
				units = mb
			} else if units == mb {
				units = kb
			} else {
				panic(fmt.Sprintf("unknown units: %s", units))
			}
		}
	}
	return fmt.Sprintf("%d%s", uint64(val), units)
}

func parsePGStringToBytes(val string) (float64, error) {
	res := pgBytesRegex.FindStringSubmatch(val)
	if len(res) != 3 {
		return 0.0, fmt.Errorf(errIncorrectFormatFmt, val)
	}
	num, err := strconv.ParseInt(res[1], 10, 64)
	if err != nil {
		return 0.0, fmt.Errorf("could not parse bytes number: %v", err)
	}
	units := res[2]
	var ret uint64
	if units == kb {
		ret = uint64(num) * kilobyte
	} else if units == mb {
		ret = uint64(num) * megabyte
	} else if units == gb {
		ret = uint64(num) * gigabyte
	} else if units == tb {
		ret = uint64(num) * terabyte
	} else {
		return 0, fmt.Errorf("unknown units: %s", units)
	}
	return float64(ret), nil
}
