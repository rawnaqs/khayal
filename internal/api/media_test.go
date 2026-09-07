package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rawnaqs/khayal/internal/config"
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

func TestAudioCapture(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	// fake STT service
	sttSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"text":"remember to water the plants"}`))
	}))
	defer sttSrv.Close()
	on := true
	ts.Config.STT = config.STTConfig{Enabled: &on, Endpoint: sttSrv.URL, API: "openai"}

	post := func(url string) *httptest.ResponseRecorder {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "note.webm")
		part.Write([]byte("FAKEAUDIO"))
		writer.WriteField("note", "from mic")
		writer.Close()
		req := httptest.NewRequest(http.MethodPost, url, body)
		req.Header.Set("X-Khayal-Token", "test-token")
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		ts.Server.handleAudioCapture(rec, req)
		return rec
	}

	t.Run("transcribes and enqueues voice job", func(t *testing.T) {
		rec := post("/v1/capture/audio")
		if rec.Code != http.StatusCreated {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Type != "voice" {
			t.Errorf("type: %s", resp.Type)
		}
		job, err := ts.Queue.GetJob(context.Background(), resp.ID)
		if err != nil {
			t.Fatal(err)
		}
		if job.Content != "remember to water the plants" {
			t.Errorf("content: %q", job.Content)
		}
	})

	t.Run("stt not configured is 503", func(t *testing.T) {
		off := false
		ts.Config.STT = config.STTConfig{Enabled: &off}
		rec := post("/v1/capture/audio")
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", rec.Code)
		}
	})

	t.Run("stt failure is 502 and nothing captured", func(t *testing.T) {
		failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer failSrv.Close()
		ts.Config.STT = config.STTConfig{Enabled: &on, Endpoint: failSrv.URL, API: "openai"}
		before := countJobs(ts)
		rec := post("/v1/capture/audio")
		if rec.Code != http.StatusBadGateway {
			t.Errorf("expected 502, got %d", rec.Code)
		}
		if countJobs(ts) != before {
			t.Error("failed transcription must not enqueue a job")
		}
	})
}

func countJobs(ts *testServer) int {
	jobs, total, _ := ts.Queue.ListJobs(context.Background(), "all", 1000, 0)
	_ = jobs
	return total
}

// Health must advertise STT capability so the PWA can hide the voice tab.
func TestHealthReportsSTT(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()
	ts.Server.healthHandler(rec, req)

	var resp struct {
		STT *struct {
			Enabled bool `json:"enabled"`
		} `json:"stt"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.STT == nil {
		t.Fatal("expected stt capability object, got nil")
	}
	if resp.STT.Enabled {
		t.Error("stt must be disabled in the default test config")
	}
}
