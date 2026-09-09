package api

import (
	"context"

	"encoding/json"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestMediaHandler(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	// seed one media file inside the inbox media dir
	mediaDir := filepath.Join(ts.Config.Vault.Path, ts.Config.Vault.InboxDir, "media")
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mediaDir, "pic.jpg"), []byte("JPEGDATA"), 0644); err != nil {
		t.Fatal(err)
	}

	get := func(url string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("X-Khayal-Token", "test-token")
		rec := httptest.NewRecorder()
		ts.Server.mediaHandler(rec, req)
		return rec
	}

	t.Run("serves file with content type", func(t *testing.T) {
		rec := get("/v1/media?path=media/pic.jpg")
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); ct != "image/jpeg" {
			t.Errorf("content type: %s", ct)
		}
		if rec.Body.String() != "JPEGDATA" {
			t.Errorf("body: %q", rec.Body.String())
		}
	})

	t.Run("vault-relative path accepted", func(t *testing.T) {
		rec := get("/v1/media?path=" + ts.Config.Vault.InboxDir + "/media/pic.jpg")
		if rec.Code != http.StatusOK || rec.Body.String() != "JPEGDATA" {
			t.Errorf("status %d body %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("traversal rejected", func(t *testing.T) {
		rec := get("/v1/media?path=../../etc/passwd")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("outside media dir rejected", func(t *testing.T) {
		rec := get("/v1/media?path=khayal/some-note.md")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for non-media path, got %d", rec.Code)
		}
	})

	t.Run("missing param rejected", func(t *testing.T) {
		rec := get("/v1/media")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("not found is 404", func(t *testing.T) {
		rec := get("/v1/media?path=media/ghost.png")
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d body %s", rec.Code, rec.Body.String())
		}
	})
}

// Health must advertise STT capability so the PWA can hide the voice tab.
// A timed-out transcription must fire the async preload and tell the
// user to retry — the load often completes server-side right after.

// Note response carries backlinks: sources whose connections block
// references this note, resolved to real paths with titles.
func TestNoteBacklinks(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	// seed three notes on disk + index rows (jobs give titles)
	note := func(relPath, connections string) {
		abs := filepath.Join(ts.Config.Vault.Path, relPath)
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			t.Fatal(err)
		}
		fm := ""
		if connections != "" {
			fm = "---\ntype: text\n" + connections + "---\n"
		}
		os.WriteFile(abs, []byte(fm+"\n# T\n"), 0644)
	}
	note(filepath.Join("inbox", "target.md"), "")
	note(filepath.Join("inbox", "fan1.md"), "connections:\n  - \"[[target]]\"\n")
	note(filepath.Join("inbox", "fan2.md"), "connections:\n  - \"[[target]]\"\n")

	ctx := context.Background()
	if err := ts.Queue.IndexNote(ctx, filepath.ToSlash(filepath.Join("inbox", "target.md")), "Target", "body", ""); err != nil {
		t.Fatal(err)
	}
	if err := ts.Queue.IndexNote(ctx, filepath.ToSlash(filepath.Join("inbox", "fan1.md")), "Fan One", "body", ""); err != nil {
		t.Fatal(err)
	}
	if err := ts.Queue.IndexNote(ctx, filepath.ToSlash(filepath.Join("inbox", "fan2.md")), "Fan Two", "body", ""); err != nil {
		t.Fatal(err)
	}
	// IndexNote has no explicit title column? it stores title — names here are paths; titles resolve from fts
	// (BatchGetNoteTitles reads notes_fts title col — indexed above with path as title)

	relPath := filepath.ToSlash(filepath.Join("inbox", "target.md"))
	r := chi.NewRouter()
	r.Get("/v1/notes/{path}", ts.Server.noteHandler)
	req := httptest.NewRequest(http.MethodGet, "/v1/notes/"+url.PathEscape(relPath), nil)
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Backlinks []struct {
			NotePath string `json:"note_path"`
			Title    string `json:"title"`
		} `json:"backlinks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Backlinks) != 2 {
		t.Fatalf("expected 2 backlinks, got %+v", resp.Backlinks)
	}
	paths := map[string]bool{}
	for _, bl := range resp.Backlinks {
		paths[bl.NotePath] = true
	}
	if !paths[filepath.ToSlash(filepath.Join("inbox", "fan1.md"))] || !paths[filepath.ToSlash(filepath.Join("inbox", "fan2.md"))] {
		t.Errorf("missing expected backlinks: %+v", resp.Backlinks)
	}
}
