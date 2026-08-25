package memory

import (
	"strings"
	"testing"
)

func TestBuildContextBlock(t *testing.T) {
	t.Run("empty sources yield empty block", func(t *testing.T) {
		if got := BuildContextBlock(Sources{}); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("user context leads, matched blurbs and recall follow", func(t *testing.T) {
		got := BuildContextBlock(Sources{
			UserContext:   "Engineer in Mangalore.",
			People:        map[string]string{"alice": "colleague at Acme"},
			MatchedPeople: []string{"Alice"},
			Glossary:      []string{"Alice", "Acme"},
			Recall:        []string{"older note about alice budgets"},
		})
		for _, want := range []string{
			"About the user:", "Engineer in Mangalore.",
			"Known people:", "- alice: colleague at Acme",
			"Known names: Alice, Acme",
			"- older note about alice budgets",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q in:\n%s", want, got)
			}
		}
		if strings.Index(got, "About the user:") > strings.Index(got, "Known names") {
			t.Error("user context must lead")
		}
	})

	t.Run("unmatched config entries are not injected", func(t *testing.T) {
		got := BuildContextBlock(Sources{
			People:        map[string]string{"alice": "x"},
			MatchedPeople: []string{},
			Orgs:          map[string]string{"acme": "y"},
			MatchedOrgs:   []string{},
		})
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("recall capped at three entries", func(t *testing.T) {
		got := BuildContextBlock(Sources{Recall: []string{"a", "b", "c", "d", "e"}})
		if strings.Contains(got, "\n- d\n") || strings.Contains(got, "- e") {
			t.Errorf("more than 3 recall entries:\n%s", got)
		}
	})

	t.Run("user context truncated at cap", func(t *testing.T) {
		got := BuildContextBlock(Sources{UserContext: strings.Repeat("x", 5000)})
		if len(got) > userContextCap+100 {
			t.Errorf("user context not truncated: %d chars", len(got))
		}
	})
}
