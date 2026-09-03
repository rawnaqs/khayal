package connections

import (
	"strings"
	"testing"
	"time"
)

func simConn(path string, created time.Time) Connection {
	return Connection{
		Type:      "similar",
		NotePath:  path,
		Score:     0.9,
		Label:     "you thought about this",
		CreatedAt: &created,
	}
}

func TestFindRevisit(t *testing.T) {
	now := time.Now().UTC()

	t.Run("three matches spanning over six months emit one revisit", func(t *testing.T) {
		similar := []Connection{
			simConn("khayal/newest.md", now.AddDate(0, 0, -30)),
			simConn("khayal/mid.md", now.AddDate(0, -4, -5)),
			simConn("khayal/oldest.md", now.AddDate(-1, 0, 0)), // >1 year ago
		}
		got := findRevisit(similar, now)
		if got == nil {
			t.Fatal("expected a revisit connection")
		}
		if got.Type != "revisit" || got.NotePath != "khayal/oldest.md" {
			t.Errorf("type/path: %+v", got)
		}
		if !strings.Contains(got.Label, "returned to this idea") ||
			!strings.Contains(got.Label, "3 times") {
			t.Errorf("label missing phrase/count: %q", got.Label)
		}
	})

	t.Run("short span does not trigger", func(t *testing.T) {
		similar := []Connection{
			simConn("a", now.AddDate(0, 0, -10)),
			simConn("b", now.AddDate(0, 0, -20)),
			simConn("c", now.AddDate(0, -1, 0)),
		}
		if got := findRevisit(similar, now); got != nil {
			t.Errorf("expected nil for <6 month span, got %+v", got)
		}
	})

	t.Run("fewer than three matches does not trigger", func(t *testing.T) {
		similar := []Connection{
			simConn("a", now.AddDate(-1, 0, 0)),
			simConn("b", now.AddDate(0, 0, -30)),
		}
		if got := findRevisit(similar, now); got != nil {
			t.Errorf("expected nil with 2 matches, got %+v", got)
		}
	})

	t.Run("matches without timestamps are ignored", func(t *testing.T) {
		similar := []Connection{
			simConn("a", now),
			{Type: "similar", NotePath: "b"}, // no CreatedAt
			simConn("c", now.AddDate(-1, 0, 0)),
		}
		if got := findRevisit(similar, now); got != nil {
			t.Errorf("expected nil when timestamps missing, got %+v", got)
		}
	})
}
