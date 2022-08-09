// Package parse provides internal constants and functions for parsing byte
// totals as presented in string of uint64 forms.
package parse

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
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

// TimeUnit represents valid suffixes for time measurements used by PostgreSQL settings
// https://www.postgresql.org/docs/current/config-setting.html#20.1.1.%20Parameter%20Names%20and%20Values
type TimeUnit int64

const (
	Microseconds TimeUnit = iota
	Milliseconds
	Seconds
	Minutes
	Hours
	Days
)

func (tu TimeUnit) String() string {
	switch tu {
	case Microseconds:
		return "us"
	case Milliseconds:
		return "ms"
	case Seconds:
		return "s"
	case Minutes:
		return "min"
	case Hours:
		return "h"
	case Days:
		return "d"
	default:
		return "unrecognized"
	}
}

func ParseTimeUnit(val string) (TimeUnit, error) {
	switch strings.ToLower(val) {
	case "us":
		return Microseconds, nil
	case "ms":
		return Milliseconds, nil
	case "s":
		return Seconds, nil
	case "min":
		return Minutes, nil
	case "h":
		return Hours, nil
	case "d":
		return Days, nil
	default:
		return TimeUnit(-1), fmt.Errorf(errUnrecognizedTimeUnitsFmt, val)
	}
}

// VarType represents the values from the vartype column of the pg_settings table
type VarType int64

const (
	VarTypeReal VarType = iota
	VarTypeInteger
)

func (v VarType) String() string {
	switch v {
	case VarTypeReal:
		return "real"
	case VarTypeInteger:
		return "integer"
	default:
		return "unrecognized"
	}
}

const (
	errIncorrectBytesFormatFmt  = "incorrect PostgreSQL bytes format: '%s'"
	errIncorrectTimeFormatFmt   = "incorrect PostgreSQL time format: '%s'"
	errCouldNotParseBytesFmt    = "could not parse bytes number: %v"
	errUnrecognizedTimeUnitsFmt = "unrecognized time units: %s"
)

var (
	pgBytesRegex = regexp.MustCompile("^(?:')?([0-9]+)((?:k|M|G|T)B)?(?:')?$")
	pgTimeRegex  = regexp.MustCompile(`^(?:')?([0-9]+(\.[0-9]+)?)(?:\s*)(us|ms|s|min|h|d)?(?:')?$`)
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
		return 0.0, fmt.Errorf(errIncorrectBytesFormatFmt, val)
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

func NextSmallerTimeUnits(units TimeUnit) TimeUnit {
	switch units {
	case Microseconds:
		return Microseconds
	case Milliseconds:
		return Microseconds
	case Seconds:
		return Milliseconds
	case Minutes:
		return Seconds
	case Hours:
		return Minutes
	default: // Days
		return Hours
	}
}

func UnitsToDuration(units TimeUnit) time.Duration {
	switch units {
	case Microseconds:
		return time.Microsecond
	case Milliseconds:
		return time.Millisecond
	case Seconds:
		return time.Second
	case Minutes:
		return time.Minute
	case Hours:
		return time.Hour
	case Days:
		return 24 * time.Hour
	default:
		return time.Nanosecond
	}
}

func TimeConversion(fromUnits, toUnits TimeUnit) (float64, error) {
	return float64(UnitsToDuration(fromUnits)) / float64(UnitsToDuration(toUnits)), nil
}

func PGFormatToTime(val string, defaultUnits TimeUnit, vt VarType) (float64, TimeUnit, error) {
	// the default units, whether units were specified, and the variable type ALL impact how the value is interpreted
	// https://www.postgresql.org/docs/current/config-setting.html#20.1.1.%20Parameter%20Names%20and%20Values

	// select unit, vartype, array_agg(name) from pg_settings where unit in ('us', 'ms', 's', 'm', 'h', 'd') group by 1, 2;

	// parse it
	res := pgTimeRegex.FindStringSubmatch(val)
	if res == nil || len(res) < 2 {
		return -1.0, TimeUnit(-1), fmt.Errorf(errIncorrectTimeFormatFmt, val)
	}

	// extract the numeric portion
	v, err := strconv.ParseFloat(res[1], 64)
	if err != nil {
		return -1.0, TimeUnit(-1), fmt.Errorf(errIncorrectTimeFormatFmt, val)
	}

	// extract the units or use the default
	unitsWereUnspecified := true
	units := defaultUnits
	if len(res) >= 4 && res[3] != "" {
		unitsWereUnspecified = false
		units, err = ParseTimeUnit(res[3])
		if err != nil {
			return -1.0, TimeUnit(-1), err
		}
	}

	convert := func(v float64, units TimeUnit, vt VarType) (float64, TimeUnit, error) {
		if _, fract := math.Modf(v); fract < math.Nextafter(0.0, 1.0) {
			// not distinguishable as a fractional value
			switch vt {
			case VarTypeInteger:
				return math.Trunc(v), units, nil
			case VarTypeReal:
				return v, units, nil
			}
		} else {
			// IS a fractional value. it had a decimal component
			toUnits := NextSmallerTimeUnits(units)
			if err != nil {
				return -1.0, TimeUnit(-1), err
			}
			conv, err := TimeConversion(units, toUnits)
			if err != nil {
				return -1.0, TimeUnit(-1), err
			}
			return math.Round(v * conv), toUnits, nil
		}
		return -1.0, TimeUnit(-1), fmt.Errorf(errIncorrectTimeFormatFmt, val)
	}

	if unitsWereUnspecified {
		switch vt {
		case VarTypeInteger:
			return math.Round(v), units, nil
		case VarTypeReal:
			return convert(v, units, vt)
		}
	} else /* units WERE specified */ {
		if units == defaultUnits {
			switch vt {
			case VarTypeInteger:
				return math.Round(v), units, nil
			case VarTypeReal:
				return convert(v, units, vt)
			}
		} else /* specified units are different from the default units */ {
			return convert(v, units, vt)
		}
	}
	// should never get here!
	return -1.0, TimeUnit(-1), fmt.Errorf(errIncorrectTimeFormatFmt, val)
}
