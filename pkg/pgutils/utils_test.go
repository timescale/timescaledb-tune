package pgutils

import (
	"fmt"
	"testing"
)

func TestToPGMajorVersion(t *testing.T) {
	okPrefix := "PostgreSQL "
	cases := []struct {
		desc    string
		version string
		want    string
		errMsg  string
	}{
		{
			desc:    "pg11",
			version: okPrefix + "11.1",
			want:    "11",
		},
		{
			desc:    "pg11 w/ extra",
			version: okPrefix + "11.1 (Debian)",
			want:    "11",
		},
		{
			desc:    "pg10",
			version: okPrefix + "10.5",
			want:    "10",
		},
		{
			desc:    "pg10 w/ extra",
			version: okPrefix + "10.2 (Debian)",
			want:    "10",
		},
		{
			desc:    "9.6",
			version: okPrefix + "9.6.3",
			want:    "9.6",
		},

		{
			desc:    "9.5",
			version: okPrefix + "9.5.9",
			want:    "9.5",
		},
		{
			desc:    "8.1",
			version: okPrefix + "8.1.9",
			want:    "8.1",
		},
		{
			desc:    "7.3",
			version: okPrefix + "7.3.9",
			want:    "7.3",
		},
		{
			desc:    "bad parse",
			version: "10.0",
			want:    "",
			errMsg:  fmt.Sprintf(errCouldNotParseVersionFmt, "10.0"),
		},
		{
			desc:    "old version",
			version: "PostgreSQL 6.3.2",
			want:    "",
			errMsg:  fmt.Sprintf(errUnknownMajorVersionFmt, "PostgreSQL 6.3.2"),
		},
	}

	for _, c := range cases {
		got, err := ToPGMajorVersion(c.version)
		if got != c.want {
			t.Errorf("%s: incorrect version: got %s want %s", c.desc, got, c.want)
		}
		if len(c.errMsg) > 0 {
			if err == nil {
				t.Errorf("%s: unexpected lack of error", c.desc)
			}
			if got := err.Error(); got != c.errMsg {
				t.Errorf("%s: incorrect error msg: got\n%s\nwant\n%s", c.desc, got, c.errMsg)
			}
		} else {
			if err != nil {
				t.Errorf("%s: unexpected error: %v", c.desc, err)
			}
		}
	}
}
