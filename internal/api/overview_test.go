package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rawnaqs/khayal/internal/queue"
)

type countingLLM struct {
	mockLLM
	generateCalls atomic.Int32
	fail          bool
}

func (m *countingLLM) GenerateWithSystem(system, user string) (string, error) {
	m.generateCalls.Add(1)
	if m.fail {
		return "", context.DeadlineExceeded
	}
	return "Bob owes money per [0] and also [7] and again [0]." +
		"", nil
}

func TestExtractCitations(t *testing.T) {
	tests := []struct {
		name string
		text string
		max  int
		want []int
	}{
		{
			name: "valid citations kept in order",
			text: "Bob owes [0] money and Alice called [2] today.",
			max:  3,
			want: []int{0, 2},
		},
		{
			name: "out-of-range dropped",
			text: "See [7] and [1].",
			max:  3,
			want: []int{1},
		},
		{
			name: "duplicates deduped",
			text: "[0] said [0] again per [1]",
			max:  3,
			want: []int{0, 1},
		},
		{
			name: "negative refs are not citations",
			text: "array[-1] is code, [0] is a cite",
			max:  3,
			want: []int{0},
		},
		{
			text: "no citations here",
			max:  3,
			want: nil,
		},
		{
			text: "[9][8][7]",
			max:  0,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCitations(tt.text, tt.max)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractCitations(%q, %d) = %v, want %v", tt.text, tt.max, got, tt.want)
			}
		})
	}
}

func searchOverviewReq(ts *testServer, query string, overview bool) (*httptest.ResponseRecorder, *countingLLM) {
	llm := &countingLLM{}
	ts.Server.llm = llm
	url := "/v1/search?q=" + query + "&mode=keyword"
	if overview {
		url += "&overview=true"
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()
	ts.Server.searchHandler(rec, req)
	return rec, llm
}

func seedDoneNote(t *testing.T, ts *testServer) {
	t.Helper()
	ctx := context.Background()
	job := &queue.Job{Type: "text", Status: "done", NotePath: "bob-note.md", CreatedAt: time.Now()}
	if err := ts.Queue.CreateJob(ctx, job); err != nil {
		t.Fatal(err)
	}
	if err := ts.Queue.IndexNote(ctx, "bob-note.md", "Bob note", "bob owes me money", "bob"); err != nil {
		t.Fatal(err)
	}
}

func TestSearchOverview(t *testing.T) {
	t.Run("no param: zero LLM calls, no overview field", func(t *testing.T) {
		ts := setupTestServer(t)
		defer ts.close()
		seedDoneNote(t, ts)

		rec, llm := searchOverviewReq(ts, "bob", false)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d", rec.Code)
		}
		var resp SearchResponse
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Overview != nil {
			t.Errorf("expected nil overview, got %+v", resp.Overview)
		}
		if llm.generateCalls.Load() != 0 {
			t.Errorf("expected 0 LLM calls, got %d", llm.generateCalls.Load())
		}
	})

	t.Run("param + results: answer with clamped citations", func(t *testing.T) {
		ts := setupTestServer(t)
		defer ts.close()
		seedDoneNote(t, ts)

		rec, _ := searchOverviewReq(ts, "bob", true)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
		var resp SearchResponse
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Overview == nil {
			t.Fatal("expected non-nil overview")
		}
		if len(resp.Results) == 0 {
			t.Fatal("expected results intact")
		}
		want := []int{0}
		if !reflect.DeepEqual(resp.Overview.Citations, want) {
			t.Errorf("citations = %v, want %v (out-of-range and dupes must drop)", resp.Overview.Citations, want)
		}
	})

	t.Run("param + LLM failure: fail-open with null overview", func(t *testing.T) {
		ts := setupTestServer(t)
		defer ts.close()
		seedDoneNote(t, ts)

		failing := &countingLLM{fail: true}
		ts.Server.llm = failing
		rec2 := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/search?q=bob&mode=keyword&overview=true", nil)
		req.Header.Set("X-Khayal-Token", "test-token")
		ts.Server.searchHandler(rec2, req)

		if rec2.Code != http.StatusOK {
			t.Fatalf("search must not fail: status %d", rec2.Code)
		}
		var resp SearchResponse
		json.NewDecoder(rec2.Body).Decode(&resp)
		if resp.Overview != nil {
			t.Errorf("expected nil overview on LLM failure")
		}
		if len(resp.Results) == 0 {
			t.Errorf("results must stay intact")
		}
	})

	t.Run("param + zero results: null overview, zero LLM calls", func(t *testing.T) {
		ts := setupTestServer(t)
		defer ts.close()

		rec, llm := searchOverviewReq(ts, "qqqq", true)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d", rec.Code)
		}
		var resp SearchResponse
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Overview != nil {
			t.Errorf("expected nil overview for empty results")
		}
		if llm.generateCalls.Load() != 0 {
			t.Errorf("expected 0 LLM calls, got %d", llm.generateCalls.Load())
		}
	})
}

func TestNoteDeleteHandler(t *testing.T) {
	seed := func(t *testing.T, ts *testServer) string {
		t.Helper()
		ctx := context.Background()
		job := &queue.Job{Type: "text", Status: "done", NotePath: "inbox/victim.md", CreatedAt: time.Now()}
		if err := ts.Queue.CreateJob(ctx, job); err != nil {
			t.Fatal(err)
		}
		if err := ts.Queue.IndexNote(ctx, "inbox/victim.md", "Victim", "deletable body words", "x"); err != nil {
			t.Fatal(err)
		}
		if err := ts.Queue.SaveChunk(ctx, "inbox/victim.md", 0, "c", make([]float32, 4)); err != nil {
			t.Fatal(err)
		}
		if err := ts.Queue.SaveEntities(ctx, "inbox/victim.md", queue.NoteEntities{People: []string{"Bob"}}); err != nil {
			t.Fatal(err)
		}
		noteRel := filepath.Join("inbox", "victim.md")
		abs := filepath.Join(ts.Config.Vault.Path, noteRel)
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("---\ntitle: Victim\n---\nbody"), 0644); err != nil {
			t.Fatal(err)
		}
		return noteRel
	}

	del := func(ts *testServer, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodDelete, "/v1/note?path="+url.QueryEscape(path), nil)
		req.Header.Set("X-Khayal-Token", "test-token")
		rec := httptest.NewRecorder()
		ts.Server.noteDeleteHandler(rec, req)
		return rec
	}

	t.Run("happy path: trashed + index purged", func(t *testing.T) {
		ts := setupTestServer(t)
		defer ts.close()
		noteRel := seed(t, ts)

		rec := del(ts, noteRel)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Deleted   bool   `json:"deleted"`
			TrashPath string `json:"trash_path"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)
		if !resp.Deleted || resp.TrashPath == "" {
			t.Fatalf("bad response: %+v", resp)
		}

		// file gone from inbox, present in trash
		if _, err := os.Stat(filepath.Join(ts.Config.Vault.Path, noteRel)); !os.IsNotExist(err) {
			t.Error("note still in inbox")
		}
		matches, _ := filepath.Glob(filepath.Join(ts.Config.Vault.Path, "inbox", ".khayal-trash", "victim.md.*"))
		if len(matches) != 1 {
			t.Errorf("expected exactly one trash file, got %v", matches)
		}

		// index purged: keyword search no longer finds it
		results, err := ts.Queue.SearchKeyword(context.Background(), "deletable", 10, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range results {
			if r.NotePath == noteRel {
				t.Error("deleted note still searchable")
			}
		}
		if c, _ := ts.Queue.CountChunks(context.Background(), noteRel); c != 0 {
			t.Error("chunks survived delete")
		}
		if e, _ := ts.Queue.CountEntities(context.Background(), noteRel, "person"); e != 0 {
			t.Error("entities survived delete")
		}
	})

	t.Run("traversal rejected", func(t *testing.T) {
		ts := setupTestServer(t)
		defer ts.close()

		for _, p := range []string{"../../etc/passwd", "../outside.md", "inbox/../secret.md"} {
			rec := del(ts, p)
			if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
				t.Errorf("path %q: expected 4xx, got %d", p, rec.Code)
			}
		}
	})

	t.Run("missing path param rejected", func(t *testing.T) {
		ts := setupTestServer(t)
		defer ts.close()
		req := httptest.NewRequest(http.MethodDelete, "/v1/note", nil)
		rec := httptest.NewRecorder()
		ts.Server.noteDeleteHandler(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("nonexistent note is 404", func(t *testing.T) {
		ts := setupTestServer(t)
		defer ts.close()
		rec := del(ts, filepath.Join("inbox", "ghost.md"))
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
