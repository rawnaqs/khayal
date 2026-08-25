// Package memory assembles the per-capture context block injected into the
// enrichment prompts: who the user is, what khayal already knows, and what
// past notes are relevant. Pure assembly — retrieval happens elsewhere.
package memory

import (
	"fmt"
	"strings"
)

const (
	userContextCap = 2000
	derivedCap     = 1200
	MaxRecall      = 3
)

// Sources carries every ingredient for one context block. Empty fields are
// skipped; an entirely empty Sources yields "".
type Sources struct {
	UserContext string
	// People/Orgs are the config maps; MatchedPeople/MatchedOrgs are the
	// entity values found in this capture. Only entries whose key matches a
	// matched value (or appears in it) are injected.
	People        map[string]string
	Orgs          map[string]string
	MatchedPeople []string
	MatchedOrgs   []string
	// MemoryFile is the current content of the LLM-maintained memory file.
	MemoryFile string
	Glossary   []string // distinct known person/org values
	Recall     []string // top similar past note summaries, best first
}

// BuildContextBlock assembles the injection block. The user's own context is
// capped separately (most authoritative); everything derived shares the
// derived cap.
func BuildContextBlock(s Sources) string {
	var b strings.Builder

	if uc := strings.TrimSpace(s.UserContext); uc != "" {
		b.WriteString("About the user:\n")
		b.WriteString(truncate(uc, userContextCap))
		b.WriteString("\n")
	}

	var derived strings.Builder

	for _, kv := range [][2]string{
		{"people", matchedBlurbs(s.People, s.MatchedPeople)},
		{"orgs", matchedBlurbs(s.Orgs, s.MatchedOrgs)},
	} {
		if kv[1] != "" {
			fmt.Fprintf(&derived, "Known %s:\n%s\n", kv[0], kv[1])
		}
	}

	if mf := strings.TrimSpace(s.MemoryFile); mf != "" {
		fmt.Fprintf(&derived, "Long-term memory notes:\n%s\n", truncate(mf, 800))
	}

	if len(s.Glossary) > 0 {
		fmt.Fprintf(&derived, "Known names: %s\n",
			truncate(strings.Join(s.Glossary, ", "), 300))
	}

	for i, r := range s.Recall {
		if i >= MaxRecall {
			break
		}
		r = truncate(strings.TrimSpace(r), 250)
		if r != "" {
			fmt.Fprintf(&derived, "- %s\n", r)
		}
	}

	out := derived.String()
	if out == "" {
		return strings.TrimRight(b.String(), "\n")
	}
	result := b.String() + "\nContext from your knowledge base:\n" + truncate(out, derivedCap)
	return strings.TrimRight(result, "\n")
}

func matchedBlurbs(m map[string]string, matched []string) string {
	var lines []string
	for key, blurb := range m {
		for _, val := range matched {
			if strings.EqualFold(key, val) ||
				strings.Contains(strings.ToLower(val), strings.ToLower(key)) {
				lines = append(lines, fmt.Sprintf("- %s: %s", key, truncate(blurb, 200)))
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// inputLabelMarkers are the user-prompt section labels the consolidation
// model sometimes echoes into its output; everything from the first marker
// onward is input leakage, not memory content.
var inputLabelMarkers = []string{
	"RECENT CAPTURED FACTS:",
	"CURRENT MEMORY FILE:",
	"NEW PEOPLE SINCE LAST RUN:",
}

// requiredHeadings must all be present for output to be accepted.
var requiredHeadings = []string{
	"# Memory", "## About the author", "## People",
	"## Ongoing threads", "## Preferences",
}

// SanitizeConsolidatedOutput validates and cleans a consolidation rewrite:
// truncates any echoed input-section labels, requires the canonical start
// and all five headings, and normalizes the trailing newline. Broken output
// returns an error so the job retries instead of persisting garbage.
func SanitizeConsolidatedOutput(out string) (string, error) {
	cut := len(out)
	for _, marker := range inputLabelMarkers {
		if i := strings.Index(out, marker); i >= 0 && i < cut {
			cut = i
		}
	}
	out = strings.TrimRight(out[:cut], " \n-")

	trimmed := strings.TrimSpace(out)
	if !strings.HasPrefix(trimmed, "# Memory") {
		return "", fmt.Errorf("consolidation output does not start with '# Memory'")
	}
	for _, h := range requiredHeadings {
		if !strings.Contains(trimmed, h) {
			return "", fmt.Errorf("consolidation output missing heading %q", h)
		}
	}
	return trimmed + "\n", nil
}
