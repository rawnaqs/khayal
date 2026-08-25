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

func TestSanitizeConsolidatedOutput(t *testing.T) {
	valid := "# Memory\n\n## About the author\n\n## People\n- Alice: x\n\n## Ongoing threads\n- y\n\n## Preferences\n- z"

	t.Run("clean output passes through", func(t *testing.T) {
		got, err := SanitizeConsolidatedOutput(valid)
		if err != nil {
			t.Fatal(err)
		}
		if got != valid+"\n" {
			t.Errorf("got:\n%q", got)
		}
	})

	t.Run("input-label leak truncated at first marker", func(t *testing.T) {
		leaked := valid + "\n\nRECENT CAPTURED FACTS:\n- stale fact one\nNEW PEOPLE SINCE LAST RUN: 3"
		got, err := SanitizeConsolidatedOutput(leaked)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(got, "RECENT CAPTURED FACTS") ||
			strings.Contains(got, "stale fact") ||
			strings.Contains(got, "NEW PEOPLE") {
			t.Errorf("prompt fragments survived:\n%s", got)
		}
		if !strings.HasSuffix(got, "- z\n") {
			t.Errorf("valid content damaged:\n%s", got)
		}
	})

	t.Run("dangling bullet dash after truncation is cleaned", func(t *testing.T) {
		leaked := valid + "\n-"
		got, err := SanitizeConsolidatedOutput(leaked)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(got, "-") || strings.Contains(got, "--\n") && false {
			t.Errorf("dangling dash survived: %q", got)
		}
		if !strings.HasSuffix(got, "- z\n") {
			t.Errorf("content damaged: %q", got)
		}
	})

	t.Run("missing required heading rejected", func(t *testing.T) {
		broken := "# Memory\n\n## People\n- a"
		if _, err := SanitizeConsolidatedOutput(broken); err == nil {
			t.Error("expected rejection for missing headings")
		}
	})

	t.Run("wrong start rejected", func(t *testing.T) {
		if _, err := SanitizeConsolidatedOutput("Sure! Here is the file:\n# Memory\n## People"); err == nil {
			t.Error("expected rejection for preamble before # Memory")
		}
	})

	t.Run("empty output rejected", func(t *testing.T) {
		if _, err := SanitizeConsolidatedOutput(""); err == nil {
			t.Error("expected rejection for empty output")
		}
	})
}
