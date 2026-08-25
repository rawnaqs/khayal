package worker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
)

func intPtr(v int) *int { return &v }

func boolPtr(v bool) *bool { return &v }

type chainFixture struct {
	w  *Worker
	q  *queue.Queue
	v  *vault.Writer
	cl func()
}

func newChainFixture(t *testing.T, mutate func(*config.Config)) *chainFixture {
	t.Helper()
	tmp := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.Vault.Path = tmp
	if mutate != nil {
		mutate(cfg)
	}
	cfg.ApplyDefaults() // fills memory file name and other derived defaults

	q, err := queue.NewQueue(filepath.Join(tmp, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	v, err := vault.NewWriter(cfg, filepath.Join(tmp, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	mockLLM := &chainMockLLM{}
	w := NewWorker(cfg.Worker, cfg.Search.ChunkOptions(), cfg.Connections, cfg.Memory,
		q, v, mockLLM, nil)
	return &chainFixture{w: w, q: q, v: v, cl: func() { _ = q.Close() }}
}

// chainMockLLM satisfies llm.LLMExt minimally; chaining never calls it.
type chainMockLLM struct{}

func (m *chainMockLLM) Embed(string) ([]float32, error) { return make([]float32, 4), nil }
func (m *chainMockLLM) EmbedBatch(t []string) ([][]float32, error) {
	out := make([][]float32, len(t))
	return out, nil
}
func (m *chainMockLLM) Generate(string) (string, error) { return "", nil }
func (m *chainMockLLM) GenerateWithSystem(s, u string) (string, error) {
	return "# Memory\n\n## About the author\n\n## People\n\n## Ongoing threads\n\n## Preferences\n", nil
}
func (m *chainMockLLM) DescribeImage(string) (string, error) { return "", nil }
func (m *chainMockLLM) Ping() error                          { return nil }
func (m *chainMockLLM) Type() string                         { return "mock" }
func (m *chainMockLLM) ExtractTags(string, string) ([]string, error) {
	return []string{"t"}, nil
}
func (m *chainMockLLM) Summarize(string, string) (string, error) { return "s", nil }
func (m *chainMockLLM) ExtractKeyIdeas(string, string) ([]string, error) {
	return []string{"k"}, nil
}
func (m *chainMockLLM) ExtractEntities(string, string) (llm.EntityResult, error) {
	return llm.EntityResult{}, nil
}

func (f *chainFixture) ingestJob(t *testing.T, ctx context.Context, path string) string {
	t.Helper()
	j := &queue.Job{Type: "text", Status: "done", NotePath: path, CreatedAt: time.Now().UTC()}
	if err := f.q.CreateJob(ctx, j); err != nil {
		t.Fatal(err)
	}
	return j.ID
}

func TestChainConnections_CreatesJobAndSetsPointer(t *testing.T) {
	f := newChainFixture(t, nil)
	defer f.cl()
	ctx := context.Background()

	id := f.ingestJob(t, ctx, "khayal/n.md")
	f.w.chainConnections(ctx, id, "khayal/n.md")

	got, err := f.q.GetJob(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.ConnectionsJobID == "" {
		t.Fatal("connections_job_id not set")
	}
	conn, err := f.q.GetJob(ctx, got.ConnectionsJobID)
	if err != nil {
		t.Fatal(err)
	}
	if conn.Type != "connections" || conn.NotePath != "khayal/n.md" {
		t.Errorf("unexpected connections job %+v", conn)
	}
}

func TestChainConnections_DisabledOrEmptyPathCreatesNothing(t *testing.T) {
	ctx := context.Background()

	f := newChainFixture(t, func(c *config.Config) { c.Connections.Enabled = boolPtr(false) })
	defer f.cl()
	id := f.ingestJob(t, ctx, "khayal/n.md")
	f.w.chainConnections(ctx, id, "khayal/n.md")
	got, _ := f.q.GetJob(ctx, id)
	if got.ConnectionsJobID != "" {
		t.Error("disabled connections must not chain a job")
	}

	f2 := newChainFixture(t, nil)
	defer f2.cl()
	id2 := f2.ingestJob(t, ctx, "khayal/n.md")
	f2.w.chainConnections(ctx, id2, "")
	got2, _ := f2.q.GetJob(ctx, id2)
	if got2.ConnectionsJobID != "" {
		t.Error("empty note_path must not chain a job")
	}
}

func TestChainMemory_DoesNotClobberConnectionsPointer(t *testing.T) {
	// THE regression: memory chaining must leave connections_job_id alone.
	f := newChainFixture(t, nil)
	defer f.cl()
	ctx := context.Background()

	id := f.ingestJob(t, ctx, "khayal/n.md")
	f.w.chainConnections(ctx, id, "khayal/n.md")

	before, _ := f.q.GetJob(ctx, id)
	f.w.chainMemoryConsolidation(id)

	after, _ := f.q.GetJob(ctx, id)
	if after.ConnectionsJobID != before.ConnectionsJobID {
		t.Fatalf("connections pointer clobbered: %s -> %s",
			before.ConnectionsJobID, after.ConnectionsJobID)
	}

	// Memory job must still have been created (first run is always due).
	mem, _, err := f.q.ListJobs(ctx, "pending", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, j := range mem {
		if j.Type == "memory" {
			found = true
		}
	}
	if !found {
		t.Error("expected a pending memory job")
	}
}

func TestChainMemory_GatesAndEdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("disabled memory never chains", func(t *testing.T) {
		f := newChainFixture(t, func(c *config.Config) { c.Memory.Enabled = boolPtr(false) })
		defer f.cl()
		id := f.ingestJob(t, ctx, "khayal/n.md")
		f.w.chainConnections(ctx, id, "khayal/n.md")
		f.w.chainMemoryConsolidation(id)
		jobs, _, _ := f.q.ListJobs(ctx, "pending", 20, 0)
		for _, j := range jobs {
			if j.Type == "memory" {
				t.Error("memory job created while disabled")
			}
		}
	})

	t.Run("fresh throttle marker suppresses chaining", func(t *testing.T) {
		f := newChainFixture(t, nil)
		defer f.cl()
		id := f.ingestJob(t, ctx, "khayal/n.md")
		f.w.chainConnections(ctx, id, "khayal/n.md")
		if err := f.q.SetStat(ctx, "memory_last_consolidation",
			time.Now().UTC().Add(-1*time.Hour).Format(time.RFC3339)); err != nil {
			t.Fatal(err)
		}
		f.w.chainMemoryConsolidation(id)
		jobs, _, _ := f.q.ListJobs(ctx, "pending", 20, 0)
		for _, j := range jobs {
			if j.Type == "memory" {
				t.Error("memory job chained despite fresh marker and no new persons")
			}
		}
	})

	t.Run("new persons past threshold trigger despite fresh marker", func(t *testing.T) {
		f := newChainFixture(t, func(c *config.Config) {
			c.Memory.NewPersonsThreshold = intPtr(1)
		})
		defer f.cl()
		id := f.ingestJob(t, ctx, "khayal/n.md")
		f.w.chainConnections(ctx, id, "khayal/n.md")
		lastRun := time.Now().UTC().Add(-1 * time.Hour)
		if err := f.q.SetStat(ctx, "memory_last_consolidation",
			lastRun.Format(time.RFC3339)); err != nil {
			t.Fatal(err)
		}
		// New person created AFTER the marker.
		if err := f.q.SaveEntities(ctx, "khayal/other.md",
			queue.NoteEntities{People: []string{"Brandnew"}}); err != nil {
			t.Fatal(err)
		}
		f.w.chainMemoryConsolidation(id)
		jobs, _, _ := f.q.ListJobs(ctx, "pending", 20, 0)
		found := false
		for _, j := range jobs {
			if j.Type == "memory" {
				found = true
			}
		}
		if !found {
			t.Error("expected memory job due to new-persons threshold")
		}
	})
}

func TestProcessConnections_SkipsManagedFileTargets(t *testing.T) {
	f := newChainFixture(t, nil)
	defer f.cl()
	ctx := context.Background()
	now := time.Now().UTC()

	// Managed file exists in the inbox as an old note.
	if err := f.v.WriteManagedFile(f.w.memCfg.File, "# Memory\n\n## People\n- Alice\n"); err != nil {
		t.Fatal(err)
	}

	// The current note must exist on disk with frontmatter (ingest writes it).
	if err := os.WriteFile(filepath.Join(f.v.InboxPath(), "current.md"),
		[]byte("---\ncreated: 2026-08-01T00:00:00Z\ntype: text\nstatus: done\n---\n\nwith Alice\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cur := &queue.Job{ID: "cur", Type: "text", Status: "done",
		NotePath: "khayal/current.md", Content: "with Alice",
		CreatedAt: now.Add(-30 * 24 * time.Hour)}
	old := &queue.Job{ID: "oldmem", Type: "text", Status: "done",
		NotePath: "khayal/" + f.w.memCfg.File, Content: "managed file note",
		CreatedAt: now.Add(-40 * 24 * time.Hour)}
	for _, j := range []*queue.Job{cur, old} {
		if err := f.q.CreateJob(ctx, j); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{"khayal/current.md", "khayal/" + f.w.memCfg.File} {
		if err := f.q.SaveEntities(ctx, p, queue.NoteEntities{People: []string{"Alice"}}); err != nil {
			t.Fatal(err)
		}
	}

	connJob := &queue.Job{ID: "conn-1", Type: "connections", Status: "processing",
		NotePath: "khayal/current.md", CreatedAt: now}
	if err := f.q.CreateJob(ctx, connJob); err != nil {
		t.Fatal(err)
	}

	if err := f.w.processConnections(ctx, connJob); err != nil {
		t.Fatalf("processConnections: %v", err)
	}

	// Result stored.
	got, _ := f.q.GetJob(ctx, "conn-1")
	if got.Result == nil || !strings.Contains(string(got.Result), "person") {
		t.Fatalf("result missing: %s", string(got.Result))
	}

	// Managed target excluded from wikilinks even though detected.
	full := readVaultFile(t, f.v, "khayal/current.md")
	if strings.Contains(full, "[[memory]]") {
		t.Errorf("managed file linked:\n%s", full)
	}
	if !strings.Contains(string(got.Result), "memory") {
		t.Logf("note: result should still contain the detection: %s", string(got.Result))
	}
	if strings.Contains(full, "connections:") && strings.Contains(full, "[[") {
		t.Errorf("unexpected non-managed links present:\n%s", full)
	}
	if !strings.Contains(full, "with Alice") {
		t.Error("body mutated")
	}
}

func readVaultFile(t *testing.T, v *vault.Writer, p string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(v.BasePath(), p))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
