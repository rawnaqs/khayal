package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/rawnaqs/khayal/internal/queue"
)

// graphStressShape describes a synthetic vault for the graph endpoint.
type graphStressShape struct {
	Notes       int
	ConnJobs    int
	ConnsPerJob int
	People      int
}

// seedGraphStress fills the queue with synthetic notes, connection results
// and people, mimicking what the enrichment pipeline would have produced.
func seedGraphStress(t testing.TB, ts *testServer, shape graphStressShape) {
	t.Helper()
	ctx := context.Background()

	for i := 0; i < shape.Notes; i++ {
		path := fmt.Sprintf("khayal/note-%05d.md", i)
		if err := ts.Queue.IndexNote(ctx, path, fmt.Sprintf("Note %d", i), "body", "stress"); err != nil {
			t.Fatalf("index note: %v", err)
		}
	}

	for i := 0; i < shape.ConnJobs; i++ {
		src := fmt.Sprintf("khayal/note-%05d.md", i%shape.Notes)
		job := &queue.Job{
			ID:        fmt.Sprintf("conn-%05d", i),
			Type:      "connections",
			Status:    "done",
			NotePath:  src,
			CreatedAt: time.Now().UTC(),
		}
		if err := ts.Queue.CreateJob(ctx, job); err != nil {
			t.Fatalf("create conn job: %v", err)
		}
		conns := make([]map[string]any, 0, shape.ConnsPerJob)
		for k := 0; k < shape.ConnsPerJob; k++ {
			target := fmt.Sprintf("khayal/note-%05d.md", (i*shape.ConnsPerJob+k)%shape.Notes)
			conns = append(conns, map[string]any{
				"type":      "similar",
				"note_path": target,
				"excerpt":   "x",
				"score":     0.9,
				"label":     "similar note",
			})
		}
		payload, _ := json.Marshal(map[string]any{"connections": conns})
		if err := ts.Queue.UpdateJobResult(ctx, job.ID, payload); err != nil {
			t.Fatalf("update conn result: %v", err)
		}
	}

	for p := 0; p < shape.People; p++ {
		path := fmt.Sprintf("khayal/note-%05d.md", p%shape.Notes)
		if err := ts.Queue.SaveEntities(ctx, path, queue.NoteEntities{
			People: []string{fmt.Sprintf("Person %04d", p)},
		}); err != nil {
			t.Fatalf("save entities: %v", err)
		}
	}
}

type graphStressResult struct {
	Latency time.Duration
	Nodes   int
	Edges   int
	Bytes   int
}

func runGraphStress(t testing.TB, shape graphStressShape) graphStressResult {
	t.Helper()
	ts := setupTestServer(t)
	defer ts.close()

	seedGraphStress(t, ts, shape)

	// phase breakdown: how much is DB vs in-memory edge assembly
	ctx := context.Background()
	qStart := time.Now()
	results, _ := ts.Queue.AllConnectionResults(ctx)
	tConns := time.Since(qStart)
	qStart = time.Now()
	ents, _ := ts.Queue.AllPersonEntities(ctx)
	tEnts := time.Since(qStart)
	qStart = time.Now()
	meta, _ := ts.Queue.GraphNoteMeta(ctx)
	tMeta := time.Since(qStart)
	t.Logf("  db phases: connResults=%d in %s | personEntities=%d in %s | noteMeta=%d in %s",
		len(results), tConns.Round(time.Millisecond),
		len(ents), tEnts.Round(time.Millisecond),
		len(meta), tMeta.Round(time.Millisecond))

	req := httptest.NewRequest(http.MethodGet, "/v1/graph", nil)
	req.Header.Set("X-Khayal-Token", "test-token")

	// warm-up (prepared statements, page cache) then measure
	rec := httptest.NewRecorder()
	ts.Server.graphHandler(rec, req)

	start := time.Now()
	rec = httptest.NewRecorder()
	ts.Server.graphHandler(rec, req)
	latency := time.Since(start)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var g struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
		Edges []struct {
			Source string `json:"source"`
			Target string `json:"target"`
		} `json:"edges"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &g); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// every edge must reference a node that survived the cap
	ids := make(map[string]bool, len(g.Nodes))
	for _, n := range g.Nodes {
		ids[n.ID] = true
	}
	for _, e := range g.Edges {
		if !ids[e.Source] || !ids[e.Target] {
			t.Fatalf("dangling edge %s -> %s", e.Source, e.Target)
		}
	}

	return graphStressResult{
		Latency: latency,
		Nodes:   len(g.Nodes),
		Edges:   len(g.Edges),
		Bytes:   rec.Body.Len(),
	}
}

// TestGraphStress measures /v1/graph latency as the vault grows. The heavy
// scale is opt-in (KHAYAL_STRESS=1) so the normal suite stays fast.
func TestGraphStress(t *testing.T) {
	shapes := []struct {
		name  string
		shape graphStressShape
	}{
		{"small", graphStressShape{Notes: 200, ConnJobs: 100, ConnsPerJob: 5, People: 100}},
		{"medium", graphStressShape{Notes: 1000, ConnJobs: 500, ConnsPerJob: 10, People: 500}},
	}
	if os.Getenv("KHAYAL_STRESS") == "1" {
		shapes = append(shapes,
			struct {
				name  string
				shape graphStressShape
			}{"large", graphStressShape{Notes: 2000, ConnJobs: 1000, ConnsPerJob: 30, People: 2000}},
		)
	}

	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			r := runGraphStress(t, s.shape)
			t.Logf("notes=%d connjobs=%d conns/job=%d people=%d -> latency=%s nodes=%d edges=%d payload=%dB",
				s.shape.Notes, s.shape.ConnJobs, s.shape.ConnsPerJob, s.shape.People,
				r.Latency.Round(time.Millisecond), r.Nodes, r.Edges, r.Bytes)
			if r.Nodes > graphMaxNodes {
				t.Fatalf("node cap violated: %d > %d", r.Nodes, graphMaxNodes)
			}
		})
	}
}

// TestGraphHandlerCapsConnectionsPerSource verifies the per-note edge cap
// keeps the strongest links rather than an arbitrary slice.
func TestGraphHandlerCapsConnectionsPerSource(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()
	ctx := context.Background()

	const total = 30
	src := "khayal/src.md"
	if err := ts.Queue.IndexNote(ctx, src, "Src", "body", "stress"); err != nil {
		t.Fatal(err)
	}
	job := &queue.Job{ID: "cap-job", Type: "connections", Status: "done", NotePath: src, CreatedAt: time.Now().UTC()}
	if err := ts.Queue.CreateJob(ctx, job); err != nil {
		t.Fatal(err)
	}
	conns := make([]map[string]any, 0, total)
	for k := 0; k < total; k++ {
		target := fmt.Sprintf("khayal/t-%02d.md", k)
		if err := ts.Queue.IndexNote(ctx, target, target, "body", "stress"); err != nil {
			t.Fatal(err)
		}
		// score == k, so the strongest 20 are k = 10..29
		conns = append(conns, map[string]any{"type": "similar", "note_path": target, "score": k, "excerpt": "x"})
	}
	payload, _ := json.Marshal(map[string]any{"connections": conns})
	if err := ts.Queue.UpdateJobResult(ctx, job.ID, payload); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/graph", nil)
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()
	ts.Server.graphHandler(rec, req)

	var g struct {
		Edges []struct {
			Source string `json:"source"`
			Target string `json:"target"`
		} `json:"edges"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &g); err != nil {
		t.Fatal(err)
	}
	fromSrc := map[string]bool{}
	for _, e := range g.Edges {
		if e.Source == src {
			fromSrc[e.Target] = true
		}
	}
	if len(fromSrc) != graphMaxConnsPerSource {
		t.Fatalf("expected %d edges from source, got %d", graphMaxConnsPerSource, len(fromSrc))
	}
	for k := 0; k < total; k++ {
		target := fmt.Sprintf("khayal/t-%02d.md", k)
		want := k >= total-graphMaxConnsPerSource // k = 10..29 are strongest
		if fromSrc[target] != want {
			t.Errorf("target %s kept=%v, want %v", target, fromSrc[target], want)
		}
	}
}

func BenchmarkGraphHandler(b *testing.B) {
	ts := setupTestServer(b)
	defer ts.close()
	seedGraphStress(b, ts, graphStressShape{Notes: 1000, ConnJobs: 500, ConnsPerJob: 10, People: 500})

	req := httptest.NewRequest(http.MethodGet, "/v1/graph", nil)
	req.Header.Set("X-Khayal-Token", "test-token")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		ts.Server.graphHandler(rec, req)
	}
}
