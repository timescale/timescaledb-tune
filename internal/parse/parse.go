// Package parse provides internal constants and functions for parsing byte
// totals as presented in string of uint64 forms.
package parse

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
)

// Byte equivalents (using 1024) of common byte measurements
const (
	Terabyte = 1 << 40
	Gigabyte = 1 << 30
	Megabyte = 1 << 20
	Kilobyte = 1 << 10
)

// Suffixes for byte measurements that are valid to PostgreSQL
const (
	TB = "TB" // terabyte
	GB = "GB" // gigabyte
	MB = "MB" // megabyte
	KB = "kB" // kilobyte
	B  = ""   // no unit, therefore: bytes
)

const (
	errIncorrectFormatFmt    = "incorrect PostgreSQL bytes format: '%s'"
	errCouldNotParseBytesFmt = "could not parse bytes number: %v"
)

var (
	pgBytesRegex = regexp.MustCompile("^([0-9]+)((?:k|M|G|T)B)?$")
)

func parseIntToFloatUnits(bytes uint64) (float64, string) {
	if bytes <= 0 {
		panic(fmt.Sprintf("bytes must be at least 1 byte (got %d)", bytes))
	}
	divisor := 1.0
	units := B
	if bytes >= Terabyte {
		divisor = float64(Terabyte)
		units = TB
	} else if bytes >= Gigabyte {
		divisor = float64(Gigabyte)
		units = GB
	} else if bytes >= Megabyte {
		divisor = float64(Megabyte)
		units = MB
	} else if bytes >= Kilobyte {
		divisor = float64(Kilobyte)
		units = KB
	}
	return float64(bytes) / divisor, units
}

// BytesToDecimalFormat converts a given amount of bytes into string with a two decimal
// precision float value.
func BytesToDecimalFormat(bytes uint64) string {
	val, units := parseIntToFloatUnits(bytes)
	return fmt.Sprintf("%0.2f %s", val, units)
}

// BytesToPGFormat converts a given amount of bytes into an acceptable PostgreSQL
// string, such as 1024 -> 1kB.
func BytesToPGFormat(bytes uint64) string {
	val, units := parseIntToFloatUnits(bytes)
	if units == B { // nothing less than 1kB allowed
		val = 1.0
		units = KB
	} else if units == KB {
		val = math.Round(val)
	} else {
		if val-float64(uint64(val)) > 0.001 { // (anything less than .001 is not going to meaningfully change at 1024x)
			val = val * 1024
			if units == TB {
				units = GB
			} else if units == GB {
				units = MB
			} else if units == MB {
				units = KB
			} else {
				panic(fmt.Sprintf("unknown units: %s", units))
			}
		}
	}
	return fmt.Sprintf("%d%s", uint64(val), units)
}

// PGFormatToBytes parses a string to match it to the PostgreSQL byte string format,
// which is <int value><string suffix>, e.g., 10GB, 1520MB, 20kB, etc.
func PGFormatToBytes(val string) (uint64, error) {
	res := pgBytesRegex.FindStringSubmatch(val)
	if len(res) != 3 {
		return 0.0, fmt.Errorf(errIncorrectFormatFmt, val)
	}
	num, err := strconv.ParseInt(res[1], 10, 64)
	if err != nil {
		return 0.0, fmt.Errorf(errCouldNotParseBytesFmt, err)
	}
	units := res[2]
	var ret uint64
	if units == KB {
		ret = uint64(num) * Kilobyte
	} else if units == MB {
		ret = uint64(num) * Megabyte
	} else if units == GB {
		ret = uint64(num) * Gigabyte
	} else if units == TB {
		ret = uint64(num) * Terabyte
	} else if units == B {
		ret = uint64(num)
	} else {
		return 0, fmt.Errorf("unknown units: %s", units)
	}
	return ret, nil
}
