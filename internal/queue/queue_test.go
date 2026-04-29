package queue

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestNewQueue(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	if q.db == nil {
		t.Error("expected non-nil database")
	}
}

func TestCreateJob(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	job := &Job{
		Type:      "text",
		Status:    "pending",
		Content:   "test content",
		CreatedAt: time.Now(),
	}

	if err := q.CreateJob(ctx, job); err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}

	if job.ID == "" {
		t.Error("expected job ID to be set")
	}

	retrieved, err := q.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}

	if retrieved.Content != job.Content {
		t.Errorf("expected content %s, got %s", job.Content, retrieved.Content)
	}
	if retrieved.Type != job.Type {
		t.Errorf("expected type %s, got %s", job.Type, retrieved.Type)
	}
}

func TestUpdateJobStatus(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	job := &Job{
		Type:      "text",
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	q.CreateJob(ctx, job)

	if err := q.UpdateJobStatus(ctx, job.ID, "processing"); err != nil {
		t.Fatalf("UpdateJobStatus() error = %v", err)
	}

	retrieved, err := q.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if retrieved.Status != "processing" {
		t.Errorf("expected status 'processing', got %s", retrieved.Status)
	}
}

func TestGetPendingJobs(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		q.CreateJob(ctx, &Job{
			Type:      "text",
			Status:    "pending",
			Content:   "test",
			CreatedAt: time.Now(),
		})
	}

	q.CreateJob(ctx, &Job{
		Type:      "text",
		Status:    "done",
		Content:   "test",
		CreatedAt: time.Now(),
	})

	pending, err := q.GetPendingJobs(ctx, 10)
	if err != nil {
		t.Fatalf("GetPendingJobs() error = %v", err)
	}

	if len(pending) != 5 {
		t.Errorf("expected 5 pending jobs, got %d", len(pending))
	}
}

func TestCountByStatus(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	q.CreateJob(ctx, &Job{Type: "text", Status: "pending", CreatedAt: time.Now()})
	q.CreateJob(ctx, &Job{Type: "text", Status: "pending", CreatedAt: time.Now()})
	q.CreateJob(ctx, &Job{Type: "text", Status: "done", CreatedAt: time.Now()})

	pending, err := q.CountByStatus(ctx, "pending")
	if err != nil {
		t.Fatalf("CountByStatus() error = %v", err)
	}
	if pending != 2 {
		t.Errorf("expected 2 pending, got %d", pending)
	}

	done, err := q.CountByStatus(ctx, "done")
	if err != nil {
		t.Fatalf("CountByStatus() error = %v", err)
	}
	if done != 1 {
		t.Errorf("expected 1 done, got %d", done)
	}
}

func TestDeleteJob(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	job := &Job{
		Type:      "text",
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	q.CreateJob(ctx, job)

	if err := q.DeleteJob(ctx, job.ID); err != nil {
		t.Fatalf("DeleteJob() error = %v", err)
	}

	_, err = q.GetJob(ctx, job.ID)
	if err == nil {
		t.Error("expected error when getting deleted job")
	}
}

func TestResetStuckJobs(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	job := &Job{
		Type:      "text",
		Status:    "processing",
		CreatedAt: time.Now(),
	}
	q.CreateJob(ctx, job)

	if err := q.ResetStuckJobs(ctx); err != nil {
		t.Fatalf("ResetStuckJobs() error = %v", err)
	}

	retrieved, err := q.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if retrieved.Status != "pending" {
		t.Errorf("expected status 'pending' after reset, got %s", retrieved.Status)
	}
}

func TestListJobs(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	for i := 0; i < 15; i++ {
		q.CreateJob(ctx, &Job{
			Type:      "text",
			Status:    "done",
			Content:   "test",
			CreatedAt: time.Now(),
		})
	}

	jobs, total, err := q.ListJobs(ctx, "done", 10, 0)
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}

	if total != 15 {
		t.Errorf("expected total 15, got %d", total)
	}
	if len(jobs) != 10 {
		t.Errorf("expected 10 jobs, got %d", len(jobs))
	}
}

func TestIndexNote(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	err = q.IndexNote(ctx, "inbox/test.md", "Test Title", "test content here", "golang,test")
	if err != nil {
		t.Fatalf("IndexNote() error = %v", err)
	}
}

func TestSaveEmbedding(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	job := &Job{
		Type:      "text",
		Status:    "done",
		CreatedAt: time.Now(),
	}
	q.CreateJob(ctx, job)

	vector := make([]float32, 384)
	for i := range vector {
		vector[i] = float32(i) * 0.01
	}

	if err := q.SaveEmbedding(ctx, job.ID, "nomic-embed-text", vector); err != nil {
		t.Fatalf("SaveEmbedding() error = %v", err)
	}
}

func TestUpdateJob(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	job := &Job{
		Type:      "text",
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	q.CreateJob(ctx, job)

	now := time.Now()
	job.Status = "done"
	job.NotePath = "inbox/test.md"
	job.ProcessedAt = &now

	if err := q.UpdateJob(ctx, job); err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}

	retrieved, err := q.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if retrieved.Status != "done" {
		t.Errorf("expected status 'done', got %s", retrieved.Status)
	}
	if retrieved.NotePath != "inbox/test.md" {
		t.Errorf("expected note_path 'inbox/test.md', got %s", retrieved.NotePath)
	}
}

func TestSaveChunk(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	embedding := make([]float32, 4)
	for i := range embedding {
		embedding[i] = float32(i) * 0.25
	}

	err = q.SaveChunk(ctx, "inbox/test.md", 0, "This is a test chunk", embedding)
	if err != nil {
		t.Fatalf("SaveChunk() error = %v", err)
	}

	err = q.SaveChunk(ctx, "inbox/test.md", 1, "Another test chunk", embedding)
	if err != nil {
		t.Fatalf("SaveChunk() second chunk error = %v", err)
	}
}

func TestSearchSemantic(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	job1 := &Job{
		Type:      "text",
		Status:    "done",
		NotePath:  "inbox/doc1.md",
		Content:   "First document about AI",
		CreatedAt: time.Now(),
	}
	q.CreateJob(ctx, job1)

	job2 := &Job{
		Type:      "text",
		Status:    "done",
		NotePath:  "inbox/doc2.md",
		Content:   "Second document about cooking",
		CreatedAt: time.Now(),
	}
	q.CreateJob(ctx, job2)

	embedding1 := []float32{1.0, 0.0, 0.0, 0.0}
	err = q.SaveChunk(ctx, "inbox/doc1.md", 0, "First document about AI", embedding1)
	if err != nil {
		t.Fatalf("SaveChunk() error = %v", err)
	}

	embedding2 := []float32{0.0, 1.0, 0.0, 0.0}
	err = q.SaveChunk(ctx, "inbox/doc2.md", 0, "Second document about cooking", embedding2)
	if err != nil {
		t.Fatalf("SaveChunk() error = %v", err)
	}

	query := []float32{0.9, 0.1, 0.0, 0.0}
	results, err := q.SearchSemantic(ctx, query, 10, 0.1, nil, nil)
	if err != nil {
		t.Fatalf("SearchSemantic() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].NotePath != "inbox/doc1.md" {
		t.Errorf("expected first result to be doc1.md, got %s", results[0].NotePath)
	}

	if results[0].Score < 0.8 {
		t.Errorf("expected high score for similar vector, got %f", results[0].Score)
	}

	if results[1].NotePath != "inbox/doc2.md" {
		t.Errorf("expected second result to be doc2.md, got %s", results[1].NotePath)
	}
}

func TestSearchKeyword_Normalization(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()

	// Create test notes with known content
	job1 := &Job{
		Type:      "text",
		Status:    "done",
		NotePath:  "inbox/doc1.md",
		Content:   "artificial intelligence machine learning",
		CreatedAt: time.Now(),
	}
	q.CreateJob(ctx, job1)
	q.IndexNote(ctx, "inbox/doc1.md", "AI", "artificial intelligence machine learning", "ai,ml")

	job2 := &Job{
		Type:      "text",
		Status:    "done",
		NotePath:  "inbox/doc2.md",
		Content:   "cooking recipes food",
		CreatedAt: time.Now(),
	}
	q.CreateJob(ctx, job2)
	q.IndexNote(ctx, "inbox/doc2.md", "Cooking", "cooking recipes food", "cooking,food")

	// Search for "intelligence" - doc1 should be first with score ≈ 1.0
	results, err := q.SearchKeyword(ctx, "intelligence", 10, nil, nil)
	if err != nil {
		t.Fatalf("SearchKeyword() error = %v", err)
	}

	if len(results) < 1 {
		t.Fatal("expected at least 1 result")
	}

	// Best result should have score ≈ 1.0
	if results[0].NotePath != "inbox/doc1.md" {
		t.Errorf("expected first result to be doc1.md, got %s", results[0].NotePath)
	}

	// Score should be close to 1.0 (normalized)
	if results[0].Score < 0.9 || results[0].Score > 1.1 {
		t.Errorf("expected score close to 1.0, got %f", results[0].Score)
	}

	// All scores should be in (0, 1]
	for i, r := range results {
		if r.Score < 0 || r.Score > 1 {
			t.Errorf("result %d score %f not in (0,1]", i, r.Score)
		}
	}
}

func TestBatchGetNoteTags(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()

	// Create test notes with tags
	q.CreateJob(ctx, &Job{
		Type:      "text",
		Status:    "done",
		NotePath:  "inbox/note1.md",
		CreatedAt: time.Now(),
	})
	q.IndexNote(ctx, "inbox/note1.md", "Note 1", "content 1", "tag1,tag2")

	q.CreateJob(ctx, &Job{
		Type:      "text",
		Status:    "done",
		NotePath:  "inbox/note2.md",
		CreatedAt: time.Now(),
	})
	q.IndexNote(ctx, "inbox/note2.md", "Note 2", "content 2", "tag2,tag3")

	// Test batch fetch
	tagsMap, err := q.BatchGetNoteTags(ctx, []string{"inbox/note1.md", "inbox/note2.md"})
	if err != nil {
		t.Fatalf("BatchGetNoteTags() error = %v", err)
	}

	if len(tagsMap) != 2 {
		t.Errorf("expected 2 entries, got %d", len(tagsMap))
	}

	if len(tagsMap["inbox/note1.md"]) != 2 {
		t.Errorf("expected 2 tags for note1, got %d", len(tagsMap["inbox/note1.md"]))
	}

	if len(tagsMap["inbox/note2.md"]) != 2 {
		t.Errorf("expected 2 tags for note2, got %d", len(tagsMap["inbox/note2.md"]))
	}
}

func TestBatchGetNoteTitles(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()

	// Create test notes with titles
	q.CreateJob(ctx, &Job{
		Type:      "text",
		Status:    "done",
		NotePath:  "inbox/note1.md",
		CreatedAt: time.Now(),
	})
	q.IndexNote(ctx, "inbox/note1.md", "Note 1", "content 1", "tag1")

	q.CreateJob(ctx, &Job{
		Type:      "text",
		Status:    "done",
		NotePath:  "inbox/note2.md",
		CreatedAt: time.Now(),
	})
	q.IndexNote(ctx, "inbox/note2.md", "Note 2", "content 2", "tag2")

	// Test batch fetch
	titlesMap, err := q.BatchGetNoteTitles(ctx, []string{"inbox/note1.md", "inbox/note2.md"})
	if err != nil {
		t.Fatalf("BatchGetNoteTitles() error = %v", err)
	}

	if len(titlesMap) != 2 {
		t.Errorf("expected 2 entries, got %d", len(titlesMap))
	}

	if titlesMap["inbox/note1.md"] != "Note 1" {
		t.Errorf("expected 'Note 1', got '%s'", titlesMap["inbox/note1.md"])
	}

	if titlesMap["inbox/note2.md"] != "Note 2" {
		t.Errorf("expected 'Note 2', got '%s'", titlesMap["inbox/note2.md"])
	}
}

func TestContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	job := &Job{
		Type:      "text",
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	err = q.CreateJob(ctx, job)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestJobStoreInterface(t *testing.T) {
	var store JobStore = &Queue{}
	_ = store
}
