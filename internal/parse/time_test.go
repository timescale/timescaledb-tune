package parse

import (
	"testing"
	"time"
)

func TestPrettyDuration(t *testing.T) {
	now := time.Now()
	cases := []struct {
		given time.Time
		want  string
	}{
		{
			given: now,
			want:  lessThanMin,
		},
		{
			given: now.Add(59 * time.Second),
			want:  lessThanMin,
		},
		{
			given: now.Add(60 * time.Second),
			want:  "1 minute",
		},
		{
			given: now.Add(61 * time.Second),
			want:  "1 minute",
		},
		{
			given: now.Add(2 * time.Minute),
			want:  "2 minutes",
		},
		{
			given: now.Add(59 * time.Minute),
			want:  "59 minutes",
		},
		{
			given: now.Add(60 * time.Minute),
			want:  "1 hour",
		},
		{
			given: now.Add(61 * time.Minute),
			want:  "1 hour",
		},
		{
			given: now.Add(2 * time.Hour),
			want:  "2 hours",
		},
		{
			given: now.Add(47 * time.Hour),
			want:  "47 hours",
		},
		{
			given: now.Add(48 * time.Hour),
			want:  "2 days",
		},
	}

	for _, c := range cases {
		d := c.given.Sub(now)
		if got := PrettyDuration(d); got != c.want {
			t.Errorf("incorrect value for %v: got\n%s\nwant\n%s", c.given, got, c.want)
		}
	}
}
