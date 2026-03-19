package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
)

type mockLLM struct{}

func (m *mockLLM) Embed(text string) ([]float32, error) {
	return make([]float32, 384), nil
}

func (m *mockLLM) Generate(prompt string) (string, error) {
	return "mock response", nil
}

func (m *mockLLM) DescribeImage(imagePath string) (string, error) {
	return "mock image description", nil
}

func (m *mockLLM) Ping() error {
	return nil
}

func (m *mockLLM) Type() string {
	return "mock"
}

func (m *mockLLM) ExtractTags(content string) ([]string, error) {
	return []string{"test", "mock"}, nil
}

func (m *mockLLM) Summarize(content string) (string, error) {
	return "mock summary", nil
}

func (m *mockLLM) ExtractKeyIdeas(content string) ([]string, error) {
	return []string{"key idea 1", "key idea 2"}, nil
}

type testServer struct {
	Server *Server
	Queue  *queue.Queue
	Vault  *vault.Writer
	Config *config.Config
}

func setupTestServer(t *testing.T) *testServer {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := queue.NewQueue(dbPath)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:             "127.0.0.1",
			Port:             7766,
			Token:            "test-token",
			MaxTextBodyMB:    1,
			MaxImageBodyMB:   10,
			ShutdownTimeoutS: 30,
		},
		Vault: config.VaultConfig{
			Path:     tmpDir,
			InboxDir: "inbox",
		},
		Search: config.SearchConfig{
			MaxResults: 50,
			MaxExcerpt: 500,
			RRFK:       60,
		},
		LLM: config.LLMConfig{
			Provider:   "mock",
			OllamaHost: "http://localhost:11434",
		},
	}

	v, err := vault.NewWriter(cfg)
	if err != nil {
		t.Fatalf("failed to create vault: %v", err)
	}

	llm := &mockLLM{}
	srv := NewServer(cfg, q, v, llm)

	return &testServer{
		Server: srv,
		Queue:  q,
		Vault:  v,
		Config: cfg,
	}
}

func (ts *testServer) close() {
	ts.Queue.Close()
}

func TestHealthHandler(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()

	ts.Server.healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp.Status)
	}
}

func TestCaptureText(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	body := `{"type": "text", "content": "test content"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/capture", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()

	ts.Server.captureHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	var resp CaptureResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID == "" {
		t.Error("expected job ID to be set")
	}
	if resp.Type != "text" {
		t.Errorf("expected type 'text', got '%s'", resp.Type)
	}
	if resp.Status != "pending" {
		t.Errorf("expected status 'pending', got '%s'", resp.Status)
	}
	if resp.NotePath != "" {
		t.Errorf("expected note_path to be empty for async capture, got '%s'", resp.NotePath)
	}
}

func TestCaptureText_MissingContent(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	body := `{"type": "text"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/capture", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()

	ts.Server.captureHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestCaptureText_InvalidJSON(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/v1/capture", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()

	ts.Server.captureHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestQueueList(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		ts.Queue.CreateJob(ctx, &queue.Job{
			Type:      "text",
			Status:    "pending",
			CreatedAt: time.Now(),
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/queue", nil)
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()

	ts.Server.queueListHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp QueueListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Total != 5 {
		t.Errorf("expected total 5, got %d", resp.Total)
	}
	if len(resp.Jobs) != 5 {
		t.Errorf("expected 5 jobs, got %d", len(resp.Jobs))
	}
}

func TestQueueGet_WithRouter(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	ctx := context.Background()
	job := &queue.Job{
		Type:      "text",
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	ts.Queue.CreateJob(ctx, job)

	r := chi.NewRouter()
	r.Get("/v1/queue/{id}", ts.Server.queueGetHandler)

	req := httptest.NewRequest(http.MethodGet, "/v1/queue/"+job.ID, nil)
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestQueueGet_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	r := chi.NewRouter()
	r.Get("/v1/queue/{id}", ts.Server.queueGetHandler)

	req := httptest.NewRequest(http.MethodGet, "/v1/queue/nonexistent", nil)
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestQueueRetry_WithRouter(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	ctx := context.Background()
	job := &queue.Job{
		Type:      "text",
		Status:    "failed",
		CreatedAt: time.Now(),
	}
	ts.Queue.CreateJob(ctx, job)

	r := chi.NewRouter()
	r.Post("/v1/queue/{id}/retry", ts.Server.queueRetryHandler)

	req := httptest.NewRequest(http.MethodPost, "/v1/queue/"+job.ID+"/retry", nil)
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	updatedJob, _ := ts.Queue.GetJob(ctx, job.ID)
	if updatedJob.Status != "pending" {
		t.Errorf("expected status 'pending', got '%s'", updatedJob.Status)
	}
}

func TestQueueDiscard_WithRouter(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	ctx := context.Background()
	job := &queue.Job{
		Type:      "text",
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	ts.Queue.CreateJob(ctx, job)

	r := chi.NewRouter()
	r.Post("/v1/queue/{id}/discard", ts.Server.queueDiscardHandler)

	req := httptest.NewRequest(http.MethodPost, "/v1/queue/"+job.ID+"/discard", nil)
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	_, err := ts.Queue.GetJob(ctx, job.ID)
	if err == nil {
		t.Error("expected job to be deleted")
	}
}

func TestSearchKeyword(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	ctx := context.Background()
	job := &queue.Job{
		Type:      "text",
		Status:    "done",
		CreatedAt: time.Now(),
	}
	ts.Queue.CreateJob(ctx, job)

	ts.Queue.IndexNote(ctx, "test-note.md", "Test", "golang programming", "golang")

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=golang", nil)
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()

	ts.Server.searchHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp SearchResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Query != "golang" {
		t.Errorf("expected query 'golang', got '%s'", resp.Query)
	}
}

func TestSearchMissingQuery(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.close()

	req := httptest.NewRequest(http.MethodGet, "/v1/search", nil)
	req.Header.Set("X-Khayal-Token", "test-token")
	rec := httptest.NewRecorder()

	ts.Server.searchHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
