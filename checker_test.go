package main

import "testing"

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
