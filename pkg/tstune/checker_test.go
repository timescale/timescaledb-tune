package tstune

import (
	"fmt"
	"testing"
)

type isCases struct {
	s    string
	want bool
}

func testIsFunc(t *testing.T, cases []isCases, fn func(string) bool) {
	for _, c := range cases {
		if got := fn(c.s); got != c.want {
			t.Errorf("'%s' returned wrong answer: got %v want %v", c.s, got, c.want)
		}
	}
}

func TestIsYes(t *testing.T) {
	cases := []isCases{
		{
			s:    "y",
			want: true,
		},
		{
			s:    "yes",
			want: true,
		},
		{
			s:    "ye",
			want: false,
		},
		{
			s:    "n",
			want: false,
		},
		{
			s:    "no",
			want: false,
		},
		{
			s:    "",
			want: false,
		},
	}

	testIsFunc(t, cases, isYes)
}

func TestIsNo(t *testing.T) {
	cases := []isCases{
		{
			s:    "y",
			want: false,
		},
		{
			s:    "yes",
			want: false,
		},
		{
			s:    "ye",
			want: false,
		},
		{
			s:    "n",
			want: true,
		},
		{
			s:    "no",
			want: true,
		},
		{
			s:    "",
			want: false,
		},
	}

	testIsFunc(t, cases, isNo)
}

func TestIsSkip(t *testing.T) {
	cases := []isCases{
		{
			s:    "s",
			want: true,
		},
		{
			s:    "skip",
			want: true,
		},
		{
			s:    "sk",
			want: false,
		},
		{
			s:    "y",
			want: false,
		},
		{
			s:    "yes",
			want: false,
		},
		{
			s:    "ye",
			want: false,
		},
		{
			s:    "n",
			want: false,
		},
		{
			s:    "no",
			want: false,
		},
		{
			s:    "",
			want: false,
		},
	}

	testIsFunc(t, cases, isSkip)
}

func TestIsQuit(t *testing.T) {
	cases := []isCases{
		{
			s:    "q",
			want: true,
		},
		{
			s:    "quit",
			want: true,
		},
		{
			s:    "qu",
			want: false,
		},
		{
			s:    "y",
			want: false,
		},
		{
			s:    "yes",
			want: false,
		},
		{
			s:    "ye",
			want: false,
		},
		{
			s:    "n",
			want: false,
		},
		{
			s:    "no",
			want: false,
		},
		{
			s:    "",
			want: false,
		},
	}

	testIsFunc(t, cases, isQuit)
}

func TestNewNumberedListCheckerCheck(t *testing.T) {
	defaultErrMsg := "default error"
	defaultLimit := 3
	cases := []struct {
		s      string
		want   bool
		errMsg string
	}{
		{
			s:      "q",
			want:   false,
			errMsg: defaultErrMsg,
		},
		{
			s:      "not a number",
			want:   false,
			errMsg: "strconv.ParseInt: parsing \"not a number\": invalid syntax",
		},
		{
			s:    "0",
			want: false,
		},
		{
			s:    fmt.Sprintf("%d", defaultLimit+1),
			want: false,
		},
		{
			s:    "1",
			want: true,
		},
		{
			s:    "2",
			want: true,
		},
		{
			s:    fmt.Sprintf("%d", defaultLimit),
			want: true,
		},
	}

	for _, c := range cases {
		checker := newNumberedListChecker(defaultLimit, defaultErrMsg)
		got, err := checker.Check(c.s)
		if c.errMsg == "" && err != nil {
			t.Errorf("%s: unexpected err: got %v", c.s, err)
		} else if c.errMsg != "" {
			if err == nil {
				t.Errorf("%s: unexpected lack of error", c.s)
			} else if got := err.Error(); got != c.errMsg {
				t.Errorf("%s: incorrect error: got %s want %s", c.s, got, c.errMsg)
			}
		} else if got != c.want {
			t.Errorf("%s: incorrect value: got %v want %v", c.s, got, c.want)
		}
	}
}
