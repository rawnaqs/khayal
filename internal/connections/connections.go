// Package connections finds related past notes after a capture and ranks
// them under strict quality gates (see SPEC.md "Proactive Connections").
package connections

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/constants"
	"github.com/rawnaqs/khayal/internal/queue"
)

// Connection is one surfaced relation between the new note and an older one.
type Connection struct {
	Type     string  `json:"type"` // "similar" | "person" | "amount" | "revisit" | ...
	NotePath string  `json:"note_path"`
	Excerpt  string  `json:"excerpt"`
	Score    float64 `json:"score"` // true cosine for "similar"; 1.0 for entity matches
	Label    string  `json:"label"`
	// CreatedAt carries the target note's creation time between detectors
	// and ranking; hidden from API output.
	CreatedAt *time.Time `json:"-"`
}

// priority per SPEC: person > contradiction > follow_up ~ amount >
// similar > revisit.
var priority = map[string]int{
	"person":        5,
	"contradiction": 4,
	"follow_up":     3,
	"amount":        3,
	"similar":       2,
	"revisit":       1,
}

// Store is the slice of queue.Queue the engine needs. *queue.Queue
// satisfies it; narrow for tests.
type Store interface {
	GetChunkEmbeddingForNote(ctx context.Context, notePath string) ([]float32, bool, error)
	TopSimilarChunks(ctx context.Context, embedding []float32, limit int, minScore float64, cutoff time.Time, excludePath string) ([]queue.RawChunkMatch, error)
	GetEntitiesByNote(ctx context.Context, notePath, entityType string) ([]string, error)
	GetNotesByEntity(ctx context.Context, entityValue, entityType string, cutoff time.Time) ([]queue.EntityMatch, error)
	CountNotesByEntity(ctx context.Context, entityValue, entityType string, cutoff time.Time, excludePath string) (int, error)
	FindFollowupCandidates(ctx context.Context, person string, keywords []string, before time.Time, excludePath string) ([]queue.FollowupCandidate, error)
	PersonMentionedSince(ctx context.Context, person string, since time.Time, excludePaths ...string) (bool, error)
}

// Find runs every enabled detector against older notes and returns at most
// cfg.MaxPerCapture ranked connections. Detector errors degrade to skipping
// that type — a connections job must not fail because one source hiccups.
func Find(ctx context.Context, q Store, notePath string, cfg config.ConnectionsConfig,
	checker ContradictionChecker) ([]Connection, error) {
	if !config.IsOn(cfg.Enabled) {
		return nil, nil
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -cfg.AgeDays())

	// The current note's own embedding powers both the similar detector and
	// amount corroboration. Missing chunks (pre-v1.1 notes) degrade both.
	selfEmb, hasEmb, err := q.GetChunkEmbeddingForNote(ctx, notePath)
	if err != nil {
		return nil, err
	}

	var conns []Connection

	var similarConns []Connection

	if hasEmb {
		sim, err := findSimilar(ctx, q, selfEmb, notePath, cutoff, cfg.SimilarityThreshold)
		if err == nil {
			conns = append(conns, sim...)
			similarConns = sim
		} else {
			fmt.Println("connections: similar skipped:", err)
		}

		// Revisit rides on the same semantic matches (v1.2 Type 6).
		if config.IsOn(cfg.Types.Revisit) {
			if r := findRevisit(similarConns, time.Now().UTC()); r != nil {
				conns = append(conns, *r)
			}
		}

		// Contradiction runs LLM verdicts over high-similarity candidates
		// (v1.2 Type 4). Nil checker skips it entirely.
		if config.IsOn(cfg.Types.Contradiction) && checker != nil {
			floor := cfg.SimilarityThreshold - 0.10
			if cfg.ContradictionThreshold > 0 {
				floor = cfg.ContradictionThreshold
			}
			hot := make([]Connection, 0, len(similarConns))
			for _, c := range similarConns {
				if c.Score >= floor {
					hot = append(hot, c)
				}
			}
			if cds := findContradictions(ctx, checker,
				constants.DefaultSystemPrompts.CheckContradiction, hot, time.Now().UTC()); len(cds) > 0 {
				conns = append(conns, cds...)
			}
		}
	}

	personByPath := map[string]bool{}
	if config.IsOn(cfg.Types.Person) {
		p, err := findByEntity(ctx, q, notePath, "person", cutoff)
		if err == nil {
			conns = append(conns, p...)
			for _, c := range p {
				personByPath[c.NotePath] = true
			}
		}
	}
	if config.IsOn(cfg.Types.FollowUp) {
		if f := findFollowups(ctx, q, notePath, time.Now().UTC()); len(f) > 0 {
			conns = append(conns, f...)
		}
	}

	if config.IsOn(cfg.Types.Amount) {
		a, err := findAmounts(ctx, q, selfEmb, hasEmb, personByPath, notePath, cutoff, cfg.SimilarityThreshold)
		if err == nil {
			conns = append(conns, a...)
		}
	}

	return rankAndLimit(conns, cfg.MaxPerCapture), nil
}

// findSimilar reports raw-cosine matches from older notes — scores are true
// confidence values, never rescaled.
func findSimilar(ctx context.Context, q Store, selfEmb []float32, notePath string, cutoff time.Time, threshold float64) ([]Connection, error) {
	matches, err := q.TopSimilarChunks(ctx, selfEmb, 5, threshold, cutoff, notePath)
	if err != nil {
		return nil, err
	}

	var conns []Connection
	for _, m := range matches {
		created, _ := time.Parse(time.RFC3339, m.CreatedAt)
		createdCopy := created
		conns = append(conns, Connection{
			Type:      "similar",
			NotePath:  m.NotePath,
			Excerpt:   m.Content,
			Score:     m.Score,
			Label:     fmt.Sprintf("you thought about this %s", formatAge(created)),
			CreatedAt: &createdCopy,
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

// findAmounts surfaces amount matches only when corroborated: the pair must
// ALSO share a person entity or be semantically close (within 0.10 of the
// similarity threshold). Bare numeric equality across unrelated notes is
// almost always coincidence.
func findAmounts(ctx context.Context, q Store, selfEmb []float32, hasEmb bool,
	sharedPerson map[string]bool, notePath string, cutoff time.Time, threshold float64) ([]Connection, error) {

	values, err := q.GetEntitiesByNote(ctx, notePath, "amount")
	if err != nil {
		return nil, err
	}

	var conns []Connection
	for _, val := range values {
		matches, err := q.GetNotesByEntity(ctx, val, "amount", cutoff)
		if err != nil {
			continue
		}
		for _, m := range matches {
			if m.NotePath == notePath {
				continue
			}
			if !sharedPerson[m.NotePath] {
				if !hasEmb {
					continue // no way to corroborate
				}
				otherEmb, ok, err := q.GetChunkEmbeddingForNote(ctx, m.NotePath)
				if err != nil || !ok {
					continue
				}
				if cosineSim(selfEmb, otherEmb) < threshold-0.10 {
					continue
				}
			}
			label := "you've mentioned this amount before"
			conns = append(conns, Connection{
				Type:     "amount",
				NotePath: m.NotePath,
				Excerpt:  m.Excerpt,
				Score:    1.0,
				Label:    label,
			})
		}
	}
	return conns, nil
}

func cosineSim(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
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
