package connections

import (
	"context"
	"strings"
	"time"
)

// followupKeywords are the intent markers FTS hunts for in past notes.
var followupKeywords = []string{
	"follow up", "follow-up", "todo", "need to",
	"will send", "promise", "must remember",
}

// followupMinAgeDays: intents younger than this still have time.
const followupMinAgeDays = 14

// findFollowups surfaces old intent notes ("need to follow up with X")
// whose person never appears again afterwards — the planned contact has
// no record of happening. At most one connection per person.
func findFollowups(ctx context.Context, q Store, notePath string, now time.Time) []Connection {
	persons, err := q.GetEntitiesByNote(ctx, notePath, "person")
	if err != nil || len(persons) == 0 {
		return nil
	}

	before := now.AddDate(0, 0, -followupMinAgeDays)
	var conns []Connection
	seen := map[string]bool{}

	for _, person := range persons {
		if seen[strings.ToLower(person)] {
			continue
		}
		candidates, err := q.FindFollowupCandidates(ctx, person, followupKeywords, before, notePath)
		if err != nil {
			continue // fail-open per person
		}
		for _, cand := range candidates {
			completed, err := q.PersonMentionedSince(ctx, person, cand.CreatedAt, notePath, cand.NotePath)
			if err != nil || completed {
				continue
			}
			seen[strings.ToLower(person)] = true
			conns = append(conns, Connection{
				Type:     "follow_up",
				NotePath: cand.NotePath,
				Excerpt:  cand.Content,
				Score:    1.0,
				Label:    "you planned to follow up with " + person + " — no record of this happening",
			})
			break // one intent per person is enough
		}
	}
	return conns
}
