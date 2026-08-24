// Package connections finds related past notes after a capture and ranks
// them under strict quality gates (see SPEC.md "Proactive Connections").
package connections

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/queue"
)

// Connection is one surfaced relation between the new note and an older one.
type Connection struct {
	Type     string  `json:"type"` // "similar" | "person" | "amount"
	NotePath string  `json:"note_path"`
	Excerpt  string  `json:"excerpt"`
	Score    float64 `json:"score"`
	Label    string  `json:"label"`
}

// priority per SPEC: person > amount > similar.
var priority = map[string]int{
	"person":  3,
	"amount":  2,
	"similar": 1,
}

// Store is the slice of queue.Queue the engine needs. *queue.Queue
// satisfies it; narrow for tests.
type Store interface {
	GetChunkEmbeddingForNote(ctx context.Context, notePath string) ([]float32, bool, error)
	SearchSemantic(ctx context.Context, queryEmbedding []float32, limit int, minScore float64, from, to *time.Time) ([]queue.SearchResult, error)
	GetEntitiesByNote(ctx context.Context, notePath, entityType string) ([]string, error)
	GetNotesByEntity(ctx context.Context, entityValue, entityType string, cutoff time.Time) ([]queue.EntityMatch, error)
	CountNotesByEntity(ctx context.Context, entityValue, entityType string, cutoff time.Time, excludePath string) (int, error)
}

// Find runs every enabled detector against older notes and returns at most
// cfg.MaxPerCapture ranked connections. Detector errors degrade to skipping
// that type — a connections job must not fail because one source hiccups.
func Find(ctx context.Context, q Store, notePath string, cfg config.ConnectionsConfig) ([]Connection, error) {
	if !config.IsOn(cfg.Enabled) {
		return nil, nil
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -cfg.MinAgeDays)

	var conns []Connection

	sim, err := findSimilar(ctx, q, notePath, cutoff, cfg.SimilarityThreshold)
	if err == nil {
		conns = append(conns, sim...)
	} else {
		fmt.Println("connections: similar skipped:", err)
	}

	if config.IsOn(cfg.Types.Person) {
		p, err := findByEntity(ctx, q, notePath, "person", cutoff)
		if err == nil {
			conns = append(conns, p...)
		}
	}
	if config.IsOn(cfg.Types.Amount) {
		a, err := findByEntity(ctx, q, notePath, "amount", cutoff)
		if err == nil {
			conns = append(conns, a...)
		}
	}

	return rankAndLimit(conns, cfg.MaxPerCapture), nil
}

// findSimilar reuses the note's own stored chunk embedding — no extra LLM
// call. SearchSemantic's date filter (to=cutoff) enforces the age window.
func findSimilar(ctx context.Context, q Store, notePath string, cutoff time.Time, threshold float64) ([]Connection, error) {
	emb, ok, err := q.GetChunkEmbeddingForNote(ctx, notePath)
	if err != nil || !ok {
		return nil, err
	}
	results, err := q.SearchSemantic(ctx, emb, 5, threshold, nil, &cutoff)
	if err != nil {
		return nil, err
	}

	var conns []Connection
	for _, r := range results {
		if r.NotePath == notePath || r.Score < threshold {
			continue
		}
		created, _ := time.Parse(time.RFC3339, r.CreatedAt)
		conns = append(conns, Connection{
			Type:     "similar",
			NotePath: r.NotePath,
			Excerpt:  r.Excerpt,
			Score:    r.Score,
			Label:    fmt.Sprintf("you thought about this %s", formatAge(created)),
		})
	}
	return conns, nil
}

func findByEntity(ctx context.Context, q Store, notePath, entityType string, cutoff time.Time) ([]Connection, error) {
	values, err := q.GetEntitiesByNote(ctx, notePath, entityType)
	if err != nil {
		return nil, err
	}

	var conns []Connection
	for _, val := range values {
		matches, err := q.GetNotesByEntity(ctx, val, entityType, cutoff)
		if err != nil {
			continue
		}
		for _, m := range matches {
			if m.NotePath == notePath {
				continue
			}
			label := fmt.Sprintf("%s also appears in %d other notes", val, otherCount(q, ctx, val, entityType, cutoff, notePath))
			if entityType == "amount" {
				label = "you've mentioned this amount before"
			}
			conns = append(conns, Connection{
				Type:     entityType,
				NotePath: m.NotePath,
				Excerpt:  m.Excerpt,
				Score:    1.0,
				Label:    label,
			})
		}
	}
	return conns, nil
}

func otherCount(q Store, ctx context.Context, val, typ string, cutoff time.Time, exclude string) int {
	n, err := q.CountNotesByEntity(ctx, val, typ, cutoff, exclude)
	if err != nil {
		return 0
	}
	return n
}

// rankAndLimit dedupes by note (keeping the highest-priority occurrence),
// sorts by priority then score, and caps the output.
func rankAndLimit(conns []Connection, max int) []Connection {
	best := make(map[string]Connection, len(conns))
	for _, c := range conns {
		cur, seen := best[c.NotePath]
		if !seen || priority[c.Type] > priority[cur.Type] ||
			(priority[c.Type] == priority[cur.Type] && c.Score > cur.Score) {
			best[c.NotePath] = c
		}
	}

	deduped := make([]Connection, 0, len(best))
	for _, c := range best {
		deduped = append(deduped, c)
	}
	sort.Slice(deduped, func(i, j int) bool {
		pi, pj := priority[deduped[i].Type], priority[deduped[j].Type]
		if pi != pj {
			return pi > pj
		}
		return deduped[i].Score > deduped[j].Score
	})

	if len(deduped) > max {
		deduped = deduped[:max]
	}
	return deduped
}

func formatAge(t time.Time) string {
	days := int(time.Since(t).Hours() / 24)
	switch {
	case days < 30:
		return fmt.Sprintf("%d days ago", days)
	case days < 365:
		return fmt.Sprintf("%d months ago", days/30)
	default:
		return fmt.Sprintf("%d years ago", days/365)
	}
}
