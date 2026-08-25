package dates

import (
	"testing"
	"time"
)

var now = time.Date(2026, 8, 25, 14, 30, 0, 0, time.UTC) // a Tuesday

func at(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, now.Hour(), now.Minute(), now.Second(), 0, time.UTC)
}

func TestResolveRelative(t *testing.T) {
	tests := []struct {
		name  string
		ref   string
		want  time.Time
		exact bool // compare full timestamp; else only "resolves" flag matters
	}{
		{"empty", "", time.Time{}, false},
		{"whitespace", "   ", time.Time{}, false},
		{"today", "today", at(2026, time.August, 25), true},
		{"tonight case-insensitive", "Tonight", at(2026, time.August, 25), true},
		{"tomorrow", "tomorrow", at(2026, time.August, 26), true},
		{"yesterday", "yesterday", at(2026, time.August, 24), true},
		{"in 3 days", "in 3 days", at(2026, time.August, 28), true},
		{"in 2 weeks", "in 2 weeks", at(2026, time.September, 8), true},
		{"in 1 month", "in 1 month", at(2026, time.September, 25), true},
		{"in 0 days is meaningless", "in 0 days", time.Time{}, false},
		{"next week", "next week", at(2026, time.September, 1), true},
		{"next month", "next month", at(2026, time.September, 25), true},
		{"bare weekday later this week", "friday", at(2026, time.August, 28), true},
		{"weekday today resolves to today", "tuesday", at(2026, time.August, 25), true},
		{"next weekday strictly future", "next tuesday", at(2026, time.September, 1), true},
		{"monday wraps forward", "monday", at(2026, time.August, 31), true},
		{"unparseable phrase", "sometime soonish", time.Time{}, false},
		{"absolute dates pass through unresolved", "March 2024", time.Time{}, false},
		{"past-relative unsupported", "last week", time.Time{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ResolveRelative(tt.ref, now)
			if !ok {
				if tt.exact {
					t.Errorf("ResolveRelative(%q) did not resolve, want %v", tt.ref, tt.want)
				}
				return
			}
			if !tt.exact {
				t.Errorf("ResolveRelative(%q) unexpectedly resolved to %v", tt.ref, got)
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("ResolveRelative(%q) = %v, want %v", tt.ref, got, tt.want)
			}
		})
	}
}
