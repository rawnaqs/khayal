package connections

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/queue"
)

func testCfg() config.ConnectionsConfig {
	return config.ConnectionsConfig{
		Enabled:             nil, // on
		MinAgeDays:          7,
		MaxPerCapture:       3,
		SimilarityThreshold: 0.85,
	}
}

func setup(t *testing.T) (*queue.Queue, func()) {
	t.Helper()
	q, err := queue.NewQueue(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	return q, func() { _ = q.Close() }
}

func mkOldNote(t *testing.T, ctx context.Context, q *queue.Queue, id, path, content string,
	vec []float32, people []string) {
	t.Helper()
	now := time.Now().UTC()
	j := &queue.Job{ID: id, Type: "text", Status: "done", NotePath: path,
		Content: content, CreatedAt: now.Add(-10 * 24 * time.Hour)}
	if err := q.CreateJob(ctx, j); err != nil {
		t.Fatal(err)
	}
	if vec != nil {
		if err := q.SaveChunk(ctx, path, 0, content, vec); err != nil {
			t.Fatal(err)
		}
	}
	if len(people) > 0 {
		if err := q.SaveEntities(ctx, path, queue.NoteEntities{People: people}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFind_DisabledYieldsNothing(t *testing.T) {
	q, closeQ := setup(t)
	defer closeQ()
	cfg := testCfg()
	off := false
	cfg.Enabled = &off

	got, err := Find(context.Background(), q, "khayal/x.md", cfg)
	if err != nil || got != nil {
		t.Fatalf("disabled must return nil,nil, got %v (err=%v)", got, err)
	}
}

func TestFind_SemimilarDetectedWithAgeAndSelfFilters(t *testing.T) {
	q, closeQ := setup(t)
	defer closeQ()
	ctx := context.Background()

	mkOldNote(t, ctx, q, "cur", "khayal/current.md", "current note about raft consensus",
		[]float32{1, 0, 0}, []string{"Alice"})
	mkOldNote(t, ctx, q, "old-sim", "khayal/old-sim.md", "older note about raft consensus",
		[]float32{0.95, 0.05, 0}, []string{"Bob"})
	mkOldNote(t, ctx, q, "old-diff", "khayal/old-diff.md", "cooking pasta today",
		[]float32{0, 0, 1}, nil)

	got, err := Find(ctx, q, "khayal/current.md", testCfg())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly the one similar note, got %+v", got)
	}
	c := got[0]
	if c.Type != "similar" || c.NotePath != "khayal/old-sim.md" {
		t.Errorf("unexpected connection %+v", c)
	}
	if c.Score < 0.85 {
		t.Errorf("score %v below threshold leaked through", c.Score)
	}
	if c.Label == "" {
		t.Error("expected an age label")
	}
}

func TestFind_PersonAndAmountLabels(t *testing.T) {
	q, closeQ := setup(t)
	defer closeQ()
	ctx := context.Background()

	// Current note: no chunks (similarity skips), shares Alice + 2000.
	now := time.Now().UTC()
	cur := &queue.Job{ID: "cur", Type: "text", Status: "done",
		NotePath: "khayal/current.md", Content: "Paid Alice 2000 again",
		CreatedAt: now.Add(-30 * 24 * time.Hour)}
	if err := q.CreateJob(ctx, cur); err != nil {
		t.Fatal(err)
	}
	if err := q.SaveEntities(ctx, "khayal/current.md",
		queue.NoteEntities{People: []string{"Alice"}, Amounts: []string{"2000"}}); err != nil {
		t.Fatal(err)
	}

	mkOldNote(t, ctx, q, "p1", "khayal/p1.md", "Lunch with Alice about the 2000 budget", nil, nil)
	if err := q.SaveEntities(ctx, "khayal/p1.md",
		queue.NoteEntities{People: []string{"Alice"}, Amounts: []string{"2000"}}); err != nil {
		t.Fatal(err)
	}
	// A second older note with Alice, to exercise the count label.
	mkOldNote(t, ctx, q, "p2", "khayal/p2.md", "Coffee with Alice", nil, []string{"Alice"})

	got, err := Find(ctx, q, "khayal/current.md", testCfg())
	if err != nil {
		t.Fatal(err)
	}

	var personConn *Connection
	for i := range got {
		if got[i].Type == "person" && got[i].NotePath == "khayal/p1.md" {
			personConn = &got[i]
		}
	}
	if personConn == nil {
		t.Fatalf("expected a person connection for p1, got %+v", got)
	}
	// SPEC label format: "[Name] also appears in N other notes" —
	// N = every older note besides the current capture (p1 and p2).
	want := "Alice also appears in 2 other notes"
	if personConn.Label != want {
		t.Errorf("label = %q, want %q", personConn.Label, want)
	}
	if personConn.Excerpt == "" {
		t.Error("expected non-empty excerpt")
	}
}

func TestRankAndLimit(t *testing.T) {
	conns := []Connection{
		{Type: "similar", NotePath: "a", Score: 0.9},
		{Type: "person", NotePath: "b", Score: 1.0},
		{Type: "amount", NotePath: "c", Score: 1.0},
		{Type: "similar", NotePath: "d", Score: 0.95},
		{Type: "person", NotePath: "e", Score: 1.0},
		// duplicate note across types — keep highest priority occurrence
		{Type: "similar", NotePath: "b", Score: 0.99},
	}

	got := rankAndLimit(conns, 3)

	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].Type != "person" {
		t.Errorf("first = %+v, want a person connection", got[0])
	}
	foundB := false
	for _, c := range got {
		if c.NotePath == "b" && c.Type == "person" {
			foundB = true
		}
	}
	if !foundB {
		t.Errorf("b must survive as its higher-priority person occurrence: %+v", got)
	}
	for _, c := range got {
		if c.NotePath == "" {
			t.Error("nil path leaked")
		}
	}
	paths := map[string]bool{}
	for _, c := range got {
		if paths[c.NotePath] {
			t.Errorf("duplicate note_path in output: %s", c.NotePath)
		}
		paths[c.NotePath] = true
	}
}
