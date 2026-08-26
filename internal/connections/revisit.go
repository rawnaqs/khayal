package connections

import (
	"fmt"
	"time"
)

// revisitMinMatches is how many prior notes on the same topic constitute a
// pattern rather than a coincidence.
const revisitMinMatches = 3

// revisitSpan is the minimum time between the oldest and newest match
// before repeated mentions count as a recurring idea.
const revisitSpan = 6 * 30 * 24 * time.Hour

// findRevisit detects a recurring-idea pattern from the semantic-similar
// matches already gathered for Type 1. When enough matches span a long
// enough window, it emits one connection anchored to the oldest note with
// the newest as its excerpt. Nil when no pattern exists.
func findRevisit(similar []Connection, now time.Time) *Connection {
	dated := make([]Connection, 0, len(similar))
	for _, c := range similar {
		if c.CreatedAt != nil {
			dated = append(dated, c)
		}
	}
	if len(dated) < revisitMinMatches {
		return nil
	}

	oldest, newest := dated[0], dated[0]
	for _, c := range dated[1:] {
		if c.CreatedAt.Before(*oldest.CreatedAt) {
			oldest = c
		}
		if c.CreatedAt.After(*newest.CreatedAt) {
			newest = c
		}
	}

	span := newest.CreatedAt.Sub(*oldest.CreatedAt)
	if span < revisitSpan {
		return nil
	}

	label := fmt.Sprintf("you've returned to this idea %d times since %s",
		len(dated), oldest.CreatedAt.Format("January 2006"))
	return &Connection{
		Type:     "revisit",
		NotePath: oldest.NotePath,
		Excerpt:  newest.Excerpt,
		Score:    newest.Score,
		Label:    label,
	}
}
