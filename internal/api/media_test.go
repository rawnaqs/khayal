package api

import (
	"net/http"
	"net/http/httptest"
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
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})
}
