// Package dates resolves relative date references ("tomorrow", "in 3 days")
// into absolute times. Deterministic and pure: the reference clock is always
// supplied by the caller, never read.
package dates

import (
	"strconv"
	"strings"
	"time"
)

var weekdays = map[string]time.Weekday{
	"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
	"wednesday": time.Wednesday, "thursday": time.Thursday,
	"friday": time.Friday, "saturday": time.Saturday,
}

// ResolveRelative resolves a relative date reference against now.
// Absolute date strings are deliberately NOT parsed here — callers decide
// whether an unresolved ref is acceptable. Returns ok=false for anything
// outside the supported grammar (including "in 0 days" and past-relative
// forms beyond "yesterday").
func ResolveRelative(ref string, now time.Time) (time.Time, bool) {
	ref = strings.ToLower(strings.TrimSpace(ref))
	if ref == "" {
		return time.Time{}, false
	}

	day := func(offset int) (time.Time, bool) {
		return now.AddDate(0, 0, offset), true
	}

	switch ref {
	case "today", "tonight":
		return day(0)
	case "tomorrow":
		return day(1)
	case "yesterday":
		return day(-1)
	case "next week":
		return day(7)
	case "next month":
		return now.AddDate(0, 1, 0), true
	}

	if rest, found := strings.CutPrefix(ref, "in "); found {
		fields := strings.Fields(rest)
		if len(fields) == 2 {
			n, err := strconv.Atoi(fields[0])
			if err != nil || n <= 0 {
				return time.Time{}, false
			}
			switch fields[1] {
			case "day", "days":
				return day(n)
			case "week", "weeks":
				return now.AddDate(0, 0, 7*n), true
			case "month", "months":
				return now.AddDate(0, n, 0), true
			}
		}
		return time.Time{}, false
	}

	name := ref
	next := false
	if rest, found := strings.CutPrefix(ref, "next "); found {
		next = true
		name = strings.TrimSpace(rest)
	}
	if wd, ok := weekdays[name]; ok {
		ahead := (int(wd) - int(now.Weekday()) + 7) % 7
		if next && ahead == 0 {
			ahead = 7
		}
		return day(ahead)
	}

	return time.Time{}, false
}
