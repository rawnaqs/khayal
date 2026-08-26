package connections

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rawnaqs/khayal/internal/constants"
)

// maxContradictionCandidates bounds the LLM verdict calls per capture.
const maxContradictionCandidates = 5

// ContradictionChecker is the slice of llm.LLMExt the contradiction
// detector needs. Nil checker = type skipped entirely.
type ContradictionChecker interface {
	GenerateWithSystemTemp(system, user string, temperature float64) (string, error)
}

// findContradictions runs an LLM verdict per semantic candidate above the
// contradiction threshold and keeps only confirmed conflicts. Fail-open:
// any parse or call error skips that candidate.
func findContradictions(ctx context.Context, checker ContradictionChecker,
	system string, similar []Connection, now time.Time) []Connection {
	if checker == nil || len(similar) == 0 {
		return nil
	}

	candidates := similar
	if len(candidates) > maxContradictionCandidates {
		candidates = candidates[:maxContradictionCandidates]
	}

	var conns []Connection
	for _, c := range candidates {
		if c.CreatedAt == nil {
			continue
		}
		user := fmt.Sprintf("NOTE A (new):\n%s\n\nNOTE B (%s):\n%s",
			now.Format("January 2, 2006"), c.CreatedAt.Format("January 2, 2006"),
			strings.TrimSpace(c.Excerpt))
		resp, err := checker.GenerateWithSystemTemp(system, user, 0.2)
		if err != nil {
			continue
		}
		var verdict struct {
			Contradicts bool   `json:"contradicts"`
			Because     string `json:"because"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(resp)), &verdict); err != nil || !verdict.Contradicts {
			continue
		}
		label := "contradicts something you wrote " + c.CreatedAt.Format("January 2, 2006")
		if verdict.Because != "" {
			label += " — " + verdict.Because
		}
		conns = append(conns, Connection{
			Type:     "contradiction",
			NotePath: c.NotePath,
			Excerpt:  c.Excerpt,
			Score:    c.Score,
			Label:    label,
		})
	}
	return conns
}

// ContradictionSystemPrompt exposes the default for the worker adapter.
func ContradictionSystemPrompt() string {
	return constants.DefaultSystemPrompts.CheckContradiction
}
