package api

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/rawnaqs/khayal/internal/constants"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/queue"
)

const overviewMaxExcerpts = 5

// Overview is the on-demand AI answer above search results.
type Overview struct {
	Text      string `json:"text"`
	Citations []int  `json:"citations"`
}

var citationRe = regexp.MustCompile(`\[(\d+)\]`)

// extractCitations returns unique, in-range citation indices in order of
// first appearance. Out-of-range refs are dropped, never clamped silently
// into wrong results.
func extractCitations(text string, maxIdx int) []int {
	if maxIdx <= 0 {
		return nil
	}
	matches := citationRe.FindAllStringSubmatch(text, -1)
	var out []int
	seen := make(map[int]bool, len(matches))
	for _, m := range matches {
		n, err := strconv.Atoi(m[1])
		if err != nil || n < 0 || n >= maxIdx || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

// buildExcerptBlock renders numbered excerpt lines for the RAG prompt:
// [1] (title, date) excerpt-text
func buildExcerptBlock(results []queue.SearchResult) string {
	block := ""
	for i, r := range results[:min(len(results), overviewMaxExcerpts)] {
		block += fmt.Sprintf("[%d] (%s, %s) %s\n", i+1, r.Title, r.CreatedAt, r.Excerpt)
	}
	return block
}

// generateOverview synthesizes the AI answer from search results. Any
// failure returns nil — search must never fail because of the answer.
func generateOverview(ctx context.Context, l llm.LLMExt, query string, results []queue.SearchResult) *Overview {
	if len(results) == 0 || l == nil {
		return nil
	}
	prompt := fmt.Sprintf("Question: %s\n\nExcerpts:\n%s", query, buildExcerptBlock(results))
	system := constants.DefaultSystemPrompts.SearchOverview

	generator := tempGenerator(l)
	text, err := generator(system, prompt)
	if err != nil || text == "" {
		return nil
	}
	cites := extractCitations(text, min(len(results), overviewMaxExcerpts))
	if cites == nil {
		cites = []int{}
	}
	return &Overview{Text: text, Citations: cites}
}

// tempGenerator prefers a low temperature when the client supports it — an
// answer synthesis is closer to summarization than creative writing.
func tempGenerator(l llm.LLMExt) func(system, user string) (string, error) {
	if tg, ok := l.(interface {
		GenerateWithSystemTemp(system, user string, temperature float64) (string, error)
	}); ok {
		return func(system, user string) (string, error) {
			return tg.GenerateWithSystemTemp(system, user, 0.3)
		}
	}
	return l.GenerateWithSystem
}
