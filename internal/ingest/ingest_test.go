package ingest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rawnaqs/khayal/internal/chunk"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
)

type mockLLMForIngest struct {
	embedCalls atomic.Int32
	batchCalls atomic.Int32

	batchSizesMu   sync.Mutex
	batchSizesList []int
}

func newMockLLMForIngest() *mockLLMForIngest {
	return &mockLLMForIngest{}
}

func (m *mockLLMForIngest) recordBatch(n int) {
	m.batchSizesMu.Lock()
	defer m.batchSizesMu.Unlock()
	m.batchSizesList = append(m.batchSizesList, n)
}

func (m *mockLLMForIngest) batchSizesSnapshot() []int {
	m.batchSizesMu.Lock()
	defer m.batchSizesMu.Unlock()
	out := make([]int, len(m.batchSizesList))
	copy(out, m.batchSizesList)
	return out
}

func (m *mockLLMForIngest) Embed(text string) ([]float32, error) {
	m.embedCalls.Add(1)
	return make([]float32, 384), nil
}

func (m *mockLLMForIngest) EmbedBatch(texts []string) ([][]float32, error) {
	m.batchCalls.Add(1)
	m.recordBatch(len(texts))
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, 384)
	}
	return result, nil
}

func (m *mockLLMForIngest) Generate(prompt string) (string, error) {
	return "mock response", nil
}

func (m *mockLLMForIngest) GenerateWithSystem(system, user string) (string, error) {
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

func (m *mockLLMForIngest) ExtractTags(content string, bucket string) ([]string, error) {
	return []string{"test", "mock"}, nil
}

func (m *mockLLMForIngest) Summarize(content string, bucket string) (string, error) {
	return "mock summary", nil
}

func (m *mockLLMForIngest) ExtractKeyIdeas(content string, bucket string) ([]string, error) {
	return []string{"key idea 1", "key idea 2"}, nil
}

func (m *mockLLMForIngest) ExtractEntities(content string, bucket string) (llm.EntityResult, error) {
	return llm.EntityResult{
		People:  []string{"John", "John Doe"},
		Amounts: []string{"$2,000"},
	}, nil
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

func (m *mockLLMWithDelay) GenerateWithSystem(system, user string) (string, error) {
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

func (m *mockLLMWithDelay) ExtractTags(content string, bucket string) ([]string, error) {
	time.Sleep(m.delay)
	return []string{"test"}, nil
}

func (m *mockLLMWithDelay) Summarize(content string, bucket string) (string, error) {
	time.Sleep(m.delay)
	return "mock summary", nil
}

func (m *mockLLMWithDelay) ExtractKeyIdeas(content string, bucket string) ([]string, error) {
	time.Sleep(m.delay)
	return []string{"key idea 1"}, nil
}

func (m *mockLLMWithDelay) ExtractEntities(content string, bucket string) (llm.EntityResult, error) {
	time.Sleep(m.delay)
	return llm.EntityResult{}, nil
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

func (m *mockLLMFail) GenerateWithSystem(system, user string) (string, error) {
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

func (m *mockLLMFail) ExtractTags(content string, bucket string) ([]string, error) {
	if m.failExtractTags {
		return nil, errors.New("extract tags failed")
	}
	return []string{"test"}, nil
}

func (m *mockLLMFail) Summarize(content string, bucket string) (string, error) {
	if m.failSummarize {
		return "", errors.New("summarize failed")
	}
	return "mock summary", nil
}

func (m *mockLLMFail) ExtractKeyIdeas(content string, bucket string) ([]string, error) {
	if m.failKeyIdeas {
		return nil, errors.New("extract key ideas failed")
	}
	return []string{"key idea 1"}, nil
}

func (m *mockLLMFail) ExtractEntities(content string, bucket string) (llm.EntityResult, error) {
	return llm.EntityResult{}, nil
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

	v, err := vault.NewWriter(cfg, filepath.Join(tmpDir, "config.yaml"))
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

	llm := newMockLLMForIngest()
	ctx := context.Background()

	job := &queue.Job{
		ID:        "test-job-1",
		Type:      "text",
		Content:   "This is a test note about programming",
		CreatedAt: time.Now(),
	}

	notePath, err := IngestText(ctx, job, v, q, llm, chunk.DefaultOptions())
	if err != nil {
		t.Fatalf("IngestText failed: %v", err)
	}

	if notePath == "" {
		t.Error("expected note path to be set")
	}

	if got := llm.batchCalls.Load(); got != 1 {
		t.Errorf("expected 1 EmbedBatch call, got %d", got)
	}
	if sizes := llm.batchSizesSnapshot(); len(sizes) != 1 || sizes[0] != 1 {
		t.Errorf("expected single-chunk batch, got sizes %v", sizes)
	}
	if n, err := q.CountChunks(ctx, notePath); err != nil || n != 1 {
		t.Errorf("expected 1 stored chunk, got %d (err=%v)", n, err)
	}
}

func TestIngestText_SavesEntities(t *testing.T) {
	q, v, cleanup := setupTestIngest(t)
	defer cleanup()

	llm := newMockLLMForIngest()
	ctx := context.Background()

	job := &queue.Job{
		ID:        "test-job-entities",
		Type:      "text",
		Content:   "Met John Doe about the $2,000 invoice",
		CreatedAt: time.Now(),
	}

	notePath, err := IngestText(ctx, job, v, q, llm, chunk.DefaultOptions())
	if err != nil {
		t.Fatalf("IngestText failed: %v", err)
	}

	// Normalized enrichment rows in the store.
	if n, err := q.CountEntities(ctx, notePath, "person"); err != nil || n != 1 {
		t.Errorf("person rows = %d (err=%v), want 1 (John Doe only)", n, err)
	}
	if n, err := q.CountEntities(ctx, notePath, "amount"); err != nil || n != 1 {
		t.Errorf("amount rows = %d (err=%v), want 1", n, err)
	}

	// Entities block in the written frontmatter.
	data, err := os.ReadFile(filepath.Join(v.BasePath(), notePath))
	if err != nil {
		t.Fatalf("failed to read note: %v", err)
	}
	src := string(data)
	if !strings.Contains(src, "entities:\n") ||
		!strings.Contains(src, "- John Doe") ||
		!strings.Contains(src, "- 2000\n") {
		t.Errorf("frontmatter missing normalized entities:\n%s", src)
	}
}

func TestIngestText_ChunksLongNote(t *testing.T) {
	q, v, cleanup := setupTestIngest(t)
	defer cleanup()

	llm := newMockLLMForIngest()
	ctx := context.Background()

	para := strings.Repeat("alpha beta gamma delta epsilon ", 20) // 100 words
	job := &queue.Job{
		ID:   "test-job-long",
		Type: "text",
		Content: para + "\n\n" +
			strings.Repeat(para+"\n\n", 5) + para, // 6 paragraphs
		CreatedAt: time.Now(),
	}

	notePath, err := IngestText(ctx, job, v, q, llm, chunk.DefaultOptions())
	if err != nil {
		t.Fatalf("IngestText failed: %v", err)
	}

	sizes := llm.batchSizesSnapshot()
	if len(sizes) != 1 {
		t.Fatalf("expected exactly one EmbedBatch call, got %d", len(sizes))
	}
	if sizes[0] < 3 {
		t.Errorf("expected long note to produce >=3 chunks, got batch size %d", sizes[0])
	}

	stored, err := q.CountChunks(ctx, notePath)
	if err != nil {
		t.Fatalf("CountChunks failed: %v", err)
	}
	if stored != sizes[0] {
		t.Errorf("stored chunks = %d, but EmbedBatch received %d texts", stored, sizes[0])
	}
}

func TestIngestText_ShortNoteSingleChunk(t *testing.T) {
	q, v, cleanup := setupTestIngest(t)
	defer cleanup()

	llm := newMockLLMForIngest()
	ctx := context.Background()

	job := &queue.Job{
		ID:        "test-job-short",
		Type:      "text",
		Content:   "tiny thought",
		CreatedAt: time.Now(),
	}

	notePath, err := IngestText(ctx, job, v, q, llm, chunk.DefaultOptions())
	if err != nil {
		t.Fatalf("IngestText failed: %v", err)
	}

	if n, err := q.CountChunks(ctx, notePath); err != nil || n != 1 {
		t.Errorf("expected short note stored as 1 chunk, got %d (err=%v)", n, err)
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
	_, err := IngestText(ctx, job, v, q, llm, chunk.DefaultOptions())
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

	_, err := IngestText(ctx, job, v, nil, failLLM, chunk.DefaultOptions())
	if err == nil {
		t.Error("expected error when ExtractTags fails")
	}
}
