package ingest

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
)

type mockLLMForIngest struct {
	embedCalls atomic.Int32
}

func (m *mockLLMForIngest) Embed(text string) ([]float32, error) {
	m.embedCalls.Add(1)
	return make([]float32, 384), nil
}

func (m *mockLLMForIngest) EmbedBatch(texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, 384)
	}
	return result, nil
}

func (m *mockLLMForIngest) Generate(prompt string) (string, error) {
	return "mock response", nil
}

func (m *mockLLMForIngest) DescribeImage(imagePath string) (string, error) {
	return "mock image description", nil
}

func (m *mockLLMForIngest) Ping() error {
	return nil
}

func (m *mockLLMForIngest) Type() string {
	return "mock"
}

func (m *mockLLMForIngest) ExtractTags(content string) ([]string, error) {
	return []string{"test", "mock"}, nil
}

func (m *mockLLMForIngest) Summarize(content string) (string, error) {
	return "mock summary", nil
}

func (m *mockLLMForIngest) ExtractKeyIdeas(content string) ([]string, error) {
	return []string{"key idea 1", "key idea 2"}, nil
}

type mockLLMWithDelay struct {
	delay time.Duration
}

func (m *mockLLMWithDelay) Embed(text string) ([]float32, error) {
	time.Sleep(m.delay)
	return make([]float32, 384), nil
}

func (m *mockLLMWithDelay) EmbedBatch(texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, 384)
	}
	return result, nil
}

func (m *mockLLMWithDelay) Generate(prompt string) (string, error) {
	time.Sleep(m.delay)
	return "mock response", nil
}

func (m *mockLLMWithDelay) DescribeImage(imagePath string) (string, error) {
	time.Sleep(m.delay)
	return "mock image description", nil
}

func (m *mockLLMWithDelay) Ping() error {
	return nil
}

func (m *mockLLMWithDelay) Type() string {
	return "mock"
}

func (m *mockLLMWithDelay) ExtractTags(content string) ([]string, error) {
	time.Sleep(m.delay)
	return []string{"test"}, nil
}

func (m *mockLLMWithDelay) Summarize(content string) (string, error) {
	time.Sleep(m.delay)
	return "mock summary", nil
}

func (m *mockLLMWithDelay) ExtractKeyIdeas(content string) ([]string, error) {
	time.Sleep(m.delay)
	return []string{"key idea 1"}, nil
}

type mockLLMFail struct {
	failExtractTags bool
	failSummarize   bool
	failKeyIdeas    bool
}

func (m *mockLLMFail) Embed(text string) ([]float32, error) {
	return make([]float32, 384), nil
}

func (m *mockLLMFail) EmbedBatch(texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, 384)
	}
	return result, nil
}

func (m *mockLLMFail) Generate(prompt string) (string, error) {
	return "mock response", nil
}

func (m *mockLLMFail) DescribeImage(imagePath string) (string, error) {
	return "mock image description", nil
}

func (m *mockLLMFail) Ping() error {
	return nil
}

func (m *mockLLMFail) Type() string {
	return "mock"
}

func (m *mockLLMFail) ExtractTags(content string) ([]string, error) {
	if m.failExtractTags {
		return nil, errors.New("extract tags failed")
	}
	return []string{"test"}, nil
}

func (m *mockLLMFail) Summarize(content string) (string, error) {
	if m.failSummarize {
		return "", errors.New("summarize failed")
	}
	return "mock summary", nil
}

func (m *mockLLMFail) ExtractKeyIdeas(content string) ([]string, error) {
	if m.failKeyIdeas {
		return nil, errors.New("extract key ideas failed")
	}
	return []string{"key idea 1"}, nil
}

func setupTestIngest(t *testing.T) (*queue.Queue, *vault.Writer, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := queue.NewQueue(dbPath)
	if err != nil {
		t.Fatalf("failed to create queue: %v", err)
	}

	cfg := &config.Config{
		Vault: config.VaultConfig{
			Path:     tmpDir,
			InboxDir: "inbox",
		},
		Search: config.SearchConfig{
			MaxResults: 50,
			MaxExcerpt: 500,
		},
	}

	v, err := vault.NewWriter(cfg)
	if err != nil {
		t.Fatalf("failed to create vault: %v", err)
	}

	return q, v, func() {
		q.Close()
	}
}

func TestIngestText_BasicSuccess(t *testing.T) {
	q, v, cleanup := setupTestIngest(t)
	defer cleanup()

	llm := &mockLLMForIngest{}
	ctx := context.Background()

	job := &queue.Job{
		ID:        "test-job-1",
		Type:      "text",
		Content:   "This is a test note about programming",
		CreatedAt: time.Now(),
	}

	notePath, err := IngestText(ctx, job, v, q, llm)
	if err != nil {
		t.Fatalf("IngestText failed: %v", err)
	}

	if notePath == "" {
		t.Error("expected note path to be set")
	}

	if llm.embedCalls.Load() != 1 {
		t.Errorf("expected 1 embed call, got %d", llm.embedCalls.Load())
	}
}

func TestIngestText_ConcurrentExecution(t *testing.T) {
	q, v, cleanup := setupTestIngest(t)
	defer cleanup()

	delay := 50 * time.Millisecond
	llm := &mockLLMWithDelay{delay: delay}
	ctx := context.Background()

	job := &queue.Job{
		ID:        "test-job-2",
		Type:      "text",
		Content:   "Test content for concurrency",
		CreatedAt: time.Now(),
	}

	start := time.Now()
	_, err := IngestText(ctx, job, v, q, llm)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("IngestText failed: %v", err)
	}

	sequentialTime := delay * 3
	if elapsed > sequentialTime {
		t.Errorf("expected concurrent execution, took %v (sequential would be %v)", elapsed, sequentialTime)
	}
}

func TestIngestText_FailFastOnError(t *testing.T) {
	_, v, cleanup := setupTestIngest(t)
	defer cleanup()

	ctx := context.Background()

	failLLM := &mockLLMFail{failExtractTags: true}
	job := &queue.Job{
		ID:        "test-job-fail",
		Type:      "text",
		Content:   "Test content",
		CreatedAt: time.Now(),
	}

	_, err := IngestText(ctx, job, v, nil, failLLM)
	if err == nil {
		t.Error("expected error when ExtractTags fails")
	}
}
