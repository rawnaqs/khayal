package ingest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/queue"
)

func TestNormalizeAmount(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"$2,000", "2000"},
		{"2k", "2000"},
		{"2.5k", "2500"},
		{"£1m", "1000000"},
		{"500", "500"},
		{"10B", "10000000000"},
		{"€3.2m", "3200000"},
		{"weird", ""},
		{"", ""},
		{"$", ""},
		{"1,234.56", "1235"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeAmount(tt.input); got != tt.want {
				t.Errorf("normalizeAmount(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeNames(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "short form dropped when subset of long form",
			input: []string{"John", "John Doe"},
			want:  []string{"John Doe"},
		},
		{
			name:  "short form dropped regardless of order",
			input: []string{"Sarah Connor", "Sarah"},
			want:  []string{"Sarah Connor"},
		},
		{
			name:  "unrelated names both kept",
			input: []string{"Alice", "Bob"},
			want:  []string{"Alice", "Bob"},
		},
		{
			name:  "partial prefix without word boundary kept",
			input: []string{"Jo", "John"},
			want:  []string{"Jo", "John"},
		},
		{
			name:  "identical names deduplicated",
			input: []string{"John Doe", "John Doe"},
			want:  []string{"John Doe"},
		},
		{
			name:  "empty input",
			input: []string{},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeNames(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("name[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNormalizeEntities(t *testing.T) {
	raw := llm.EntityResult{
		People:  []string{"Sarah", "Sarah Connor", "James"},
		Amounts: []string{"$2,000", "weird", "3k"},
		Dates:   []string{"March 2024"},
		Places:  []string{"London"},
		Orgs:    []string{},
		URLs:    []string{"https://example.com"},
	}
	got := NormalizeEntities(raw)

	if len(got.People) != 2 || got.People[0] != "Sarah Connor" || got.People[1] != "James" {
		t.Errorf("People = %v, want [Sarah Connor James]", got.People)
	}
	if len(got.Amounts) != 2 || got.Amounts[0] != "2000" || got.Amounts[1] != "3000" {
		t.Errorf("Amounts = %v, want [2000 3000]", got.Amounts)
	}
	if len(got.Dates) != 1 || got.Dates[0] != "March 2024" {
		t.Errorf("Dates = %v, want [March 2024]", got.Dates)
	}
	if len(got.Places) != 1 || got.Places[0] != "London" {
		t.Errorf("Places = %v, want [London]", got.Places)
	}
	if len(got.Orgs) != 0 {
		t.Errorf("Orgs = %v, want empty", got.Orgs)
	}
	if len(got.URLs) != 1 || got.URLs[0] != "https://example.com" {
		t.Errorf("URLs = %v, want [https://example.com]", got.URLs)
	}
}

func TestNormalizeEntities_PeopleJunkFiltered(t *testing.T) {
	raw := llm.EntityResult{
		People: []string{"I", "Alice", "X", "We", "bob"},
	}
	got := NormalizeEntities(raw)
	want := []string{"Alice", "bob"}
	if len(got.People) != len(want) {
		t.Fatalf("People = %v, want %v", got.People, want)
	}
	for i := range want {
		if got.People[i] != want[i] {
			t.Errorf("People[%d] = %q, want %q", i, got.People[i], want[i])
		}
	}
}

func TestResolveRelativeDates(t *testing.T) {
	e := Entities{Dates: []string{"tomorrow", "March 2024", "in 3 days"}}
	e.ResolveRelativeDates(time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC))

	if e.ResolvedDates[0] != "2026-08-26" {
		t.Errorf("tomorrow = %q, want 2026-08-26", e.ResolvedDates[0])
	}
	if e.ResolvedDates[1] != "" {
		t.Errorf("absolute date should not resolve, got %q", e.ResolvedDates[1])
	}
	if e.ResolvedDates[2] != "2026-08-28" {
		t.Errorf("in 3 days = %q, want 2026-08-28", e.ResolvedDates[2])
	}

	q := e.toQueue()
	if len(q.ResolvedDates) != len(q.Dates) || q.ResolvedDates[0] != "2026-08-26" {
		t.Errorf("toQueue lost resolutions: %+v", q)
	}
	b := e.toVaultBlock()
	if b.DateResolutions[0] != "2026-08-26" {
		t.Errorf("toVaultBlock lost resolutions: %+v", b)
	}
}

func TestRescuePeopleFromGlossary(t *testing.T) {
	q, err := queue.NewQueue(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()
	ctx := context.Background()

	// Glossary source rows (any note path).
	if err := q.SaveEntities(ctx, "khayal/g.md", queue.NoteEntities{
		People: []string{"Bob", "John Doe"}, Orgs: []string{"Acme Corp"},
	}); err != nil {
		t.Fatal(err)
	}

	t.Run("empty extraction rescues known names from text", func(t *testing.T) {
		e := Entities{}
		rescuePeople(ctx, q, &e, "what is Bob's issue with Acme Corp")
		want := []string{"Bob", "Acme Corp"}
		if len(e.People) != len(want) {
			t.Fatalf("People = %v, want %v", e.People, want)
		}
		for i := range want {
			if e.People[i] != want[i] {
				t.Errorf("People[%d] = %q, want %q", i, e.People[i], want[i])
			}
		}
	})

	t.Run("non-empty extraction left alone", func(t *testing.T) {
		e := Entities{People: []string{"Someone Else"}}
		rescuePeople(ctx, q, &e, "Bob was here")
		if len(e.People) != 1 || e.People[0] != "Someone Else" {
			t.Errorf("model judgment overridden: %v", e.People)
		}
	})

	t.Run("unknown names not invented", func(t *testing.T) {
		e := Entities{}
		rescuePeople(ctx, q, &e, "totally unrelated Zephyr content")
		if len(e.People) != 0 {
			t.Errorf("invented people: %v", e.People)
		}
	})
}
