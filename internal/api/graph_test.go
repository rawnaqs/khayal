package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rawnaqs/khayal/internal/queue"
)

func TestGraphHandler(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	ctx := context.Background()
	j1 := &queue.Job{ID: "n1", Type: "text", Status: "done", NotePath: "khayal/a.md", CreatedAt: time.Now()}
	j2 := &queue.Job{ID: "c1", Type: "connections", Status: "done", NotePath: "khayal/a.md", CreatedAt: time.Now()}
	j3 := &queue.Job{ID: "b1", Type: "text", Status: "done", NotePath: "khayal/b.md", CreatedAt: time.Now()}
	for _, j := range []*queue.Job{j1, j2, j3} {
		if err := ts.Queue.CreateJob(ctx, j); err != nil {
			t.Fatal(err)
		}
	}
	if err := ts.Queue.SaveEntities(ctx, "khayal/a.md", queue.NoteEntities{People: []string{"Bob"}}); err != nil {
		t.Fatal(err)
	}
	payload := `{"connections":[{"type":"person","note_path":"khayal/b.md","excerpt":"x","score":1,"label":"Bob also appears"}]}`
	if err := ts.Queue.UpdateJobResult(ctx, j2.ID, json.RawMessage(payload)); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/graph", nil)
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()
	ts.Server.graphHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}

	var g struct {
		Nodes []struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
			Name string `json:"name"`
		} `json:"nodes"`
		Edges []struct {
			Source string   `json:"source"`
			Target string   `json:"target"`
			Types  []string `json:"types,omitempty"`
		} `json:"edges"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&g); err != nil {
		t.Fatal(err)
	}

	nodes := map[string]string{}
	for _, n := range g.Nodes {
		nodes[n.ID] = n.Kind
	}
	if nodes["khayal/a.md"] != "note" || nodes["khayal/b.md"] != "note" {
		t.Errorf("note nodes missing: %v", nodes)
	}
	if nodes["person:bob"] != "person" {
		t.Errorf("person node missing: %v", nodes)
	}

	var noteEdge bool
	for _, e := range g.Edges {
		if e.Source == "khayal/a.md" && e.Target == "khayal/b.md" {
			noteEdge = true
		}
	}
	if !noteEdge {
		t.Errorf("note-note edge missing: %+v", g.Edges)
	}
	var personEdge bool
	for _, e := range g.Edges {
		if e.Source == "person:bob" && e.Target == "khayal/a.md" {
			personEdge = true
		}
	}
	if !personEdge {
		t.Errorf("person edge missing: %+v", g.Edges)
	}
}
