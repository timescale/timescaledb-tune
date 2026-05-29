package parse

import (
	"fmt"
	"time"
)

const lessThanMin = "less than a minute"

// PrettyDuration returns a human-readable duration that should fit the
// phrase "X ago", e.g., "less than a minute ago", "2 minutes ago", etc.
func PrettyDuration(d time.Duration) string {
	if d < time.Minute {
		return lessThanMin
	} else if d < time.Hour {
		mins := int64(d.Minutes())
		ending := ""
		if mins > 1 {
			ending = "s"
		}
		return fmt.Sprintf("%d minute%s", mins, ending)
	} else if d < 48*time.Hour {
		hrs := int64(d.Hours())
		ending := ""
		if hrs > 1 {
			ending = "s"
		}
		return fmt.Sprintf("%d hour%s", hrs, ending)
	}

	days := int64(d.Hours()) / 24
	return fmt.Sprintf("%d days", days)
}
