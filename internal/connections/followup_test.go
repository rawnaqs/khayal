package connections

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rawnaqs/khayal/internal/queue"
)

// seedFollowup creates a done note with content, optional person entity,
// created at the given offset from now.
func seedFollowup(t *testing.T, ctx context.Context, q *queue.Queue, id, path, content string, people []string, ageDays int) {
	t.Helper()
	now := time.Now().UTC()
	j := &queue.Job{ID: id, Type: "text", Status: "done", NotePath: path,
		Content: content, CreatedAt: now.Add(-time.Duration(ageDays) * 24 * time.Hour)}
	if err := q.CreateJob(ctx, j); err != nil {
		t.Fatal(err)
	}
	if err := q.IndexNote(ctx, path, path, content, ""); err != nil {
		t.Fatal(err)
	}
	if len(people) > 0 {
		if err := q.SaveEntities(ctx, path, queue.NoteEntities{People: people}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFindFollowups(t *testing.T) {
	ctx := context.Background()

	t.Run("old intent with no completion surfaces", func(t *testing.T) {
		q, closeQ := setup(t)
		defer closeQ()

		seedFollowup(t, ctx, q, "intent", "khayal/intent.md",
			"need to follow up with bob about the invoice", []string{"Bob"}, 30)

		seedFollowup(t, ctx, q, "new", "khayal/new.md",
			"meeting bob later today", []string{"Bob"}, 0)

		got := findFollowups(ctx, q, "khayal/new.md", time.Now().UTC())
		found := false
		for _, c := range got {
			if c.Type == "follow_up" && strings.Contains(c.Label, "Bob") &&
				c.NotePath == "khayal/intent.md" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected follow_up connection for Bob, got %+v", got)
		}
	})

	t.Run("recent intent is below the 14-day floor", func(t *testing.T) {
		q, closeQ := setup(t)
		defer closeQ()

		seedFollowup(t, ctx, q, "recent", "khayal/recent.md",
			"need to follow up with bob", []string{"Bob"}, 5)

		if got := findFollowups(ctx, q, "khayal/new.md", time.Now().UTC()); len(got) != 0 {
			t.Errorf("expected none for fresh intent, got %+v", got)
		}
	})

	t.Run("completion mention suppresses the follow-up", func(t *testing.T) {
		q, closeQ := setup(t)
		defer closeQ()

		seedFollowup(t, ctx, q, "old-intent", "khayal/old.md",
			"must follow up with bob on the contract", []string{"Bob"}, 40)
		// a later note mentioning Bob = record of contact
		seedFollowup(t, ctx, q, "later", "khayal/later.md",
			"called bob today, all settled", []string{"Bob"}, 3)

		if got := findFollowups(ctx, q, "khayal/new.md", time.Now().UTC()); len(got) != 0 {
			t.Errorf("expected suppression after completion, got %+v", got)
		}
	})

	t.Run("no intent keywords means no candidates", func(t *testing.T) {
		q, closeQ := setup(t)
		defer closeQ()

		seedFollowup(t, ctx, q, "plain", "khayal/plain.md",
			"had a call with bob about pricing", []string{"Bob"}, 30)

		if got := findFollowups(ctx, q, "khayal/new.md", time.Now().UTC()); len(got) != 0 {
			t.Errorf("expected none without intent keyword, got %+v", got)
		}
	})
}
