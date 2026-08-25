package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
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

func TestDeleteChunksByNote(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()

	// Deleting chunks for an unknown note is a no-op, not an error.
	if err := q.DeleteChunksByNote(ctx, "inbox/missing.md"); err != nil {
		t.Fatalf("DeleteChunksByNote(missing) error = %v", err)
	}

	for i := range 3 {
		if err := q.SaveChunk(ctx, "inbox/note.md", i,
			fmt.Sprintf("chunk %d content", i), make([]float32, 384)); err != nil {
			t.Fatalf("SaveChunk() error = %v", err)
		}
	}
	if err := q.SaveChunk(ctx, "inbox/other.md", 0,
		"other note chunk", make([]float32, 384)); err != nil {
		t.Fatalf("SaveChunk() error = %v", err)
	}

	if n, err := q.CountChunks(ctx, "inbox/note.md"); err != nil || n != 3 {
		t.Fatalf("expected 3 chunks before delete, got %d (err=%v)", n, err)
	}

	if err := q.DeleteChunksByNote(ctx, "inbox/note.md"); err != nil {
		t.Fatalf("DeleteChunksByNote() error = %v", err)
	}

	if n, err := q.CountChunks(ctx, "inbox/note.md"); err != nil || n != 0 {
		t.Errorf("expected 0 chunks after delete, got %d (err=%v)", n, err)
	}
	if n, err := q.CountChunks(ctx, "inbox/other.md"); err != nil || n != 1 {
		t.Errorf("expected other note untouched (1 chunk), got %d (err=%v)", n, err)
	}

	// Idempotent re-save: delete then insert again replaces cleanly.
	for i := range 2 {
		if err := q.SaveChunk(ctx, "inbox/note.md", i,
			fmt.Sprintf("new chunk %d", i), make([]float32, 384)); err != nil {
			t.Fatalf("SaveChunk() re-save error = %v", err)
		}
	}
	if n, err := q.CountChunks(ctx, "inbox/note.md"); err != nil || n != 2 {
		t.Errorf("expected 2 chunks after re-save, got %d (err=%v)", n, err)
	}
}

func TestSaveEntities(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()

	// Deleting entities for an unknown note is a no-op.
	if err := q.DeleteEntities(ctx, "inbox/missing.md"); err != nil {
		t.Fatalf("DeleteEntities(missing) error = %v", err)
	}

	ents := NoteEntities{
		People:  []string{"John Doe"},
		Amounts: []string{"2000", "3000"},
		Dates:   []string{"March 2024"},
	}
	if err := q.SaveEntities(ctx, "inbox/note.md", ents); err != nil {
		t.Fatalf("SaveEntities() error = %v", err)
	}

	var n int
	if err := q.db.QueryRow(
		`SELECT COUNT(*) FROM entities WHERE note_path='inbox/note.md' AND entity_type='person'`,
	).Scan(&n); err != nil || n != 1 {
		t.Errorf("person rows = %d (err=%v), want 1", n, err)
	}
	if err := q.db.QueryRow(
		`SELECT COUNT(*) FROM entities WHERE note_path='inbox/note.md' AND entity_type='amount'`,
	).Scan(&n); err != nil || n != 2 {
		t.Errorf("amount rows = %d (err=%v), want 2", n, err)
	}

	// Idempotent: saving again replaces the enrichment rows.
	if err := q.SaveEntities(ctx, "inbox/note.md", NoteEntities{
		People: []string{"Jane Smith"},
	}); err != nil {
		t.Fatalf("SaveEntities(retry) error = %v", err)
	}
	if err := q.db.QueryRow(
		`SELECT COUNT(*) FROM entities WHERE note_path='inbox/note.md'`,
	).Scan(&n); err != nil || n != 1 {
		t.Errorf("rows after re-save = %d (err=%v), want 1", n, err)
	}
}

func TestSaveEntities_PreservesTitleAndTagRows(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()

	// IndexNote writes 'title' and 'tag' rows into the same table.
	if err := q.IndexNote(ctx, "inbox/note.md", "Test Title", "content", "golang,test"); err != nil {
		t.Fatalf("IndexNote() error = %v", err)
	}

	if err := q.SaveEntities(ctx, "inbox/note.md", NoteEntities{
		People: []string{"John Doe"},
	}); err != nil {
		t.Fatalf("SaveEntities() error = %v", err)
	}

	var n int
	if err := q.db.QueryRow(
		`SELECT COUNT(*) FROM entities WHERE note_path='inbox/note.md' AND entity_type IN ('title','tag')`,
	).Scan(&n); err != nil || n != 3 {
		t.Errorf("title/tag rows = %d (err=%v), want 3", n, err)
	}
	if err := q.db.QueryRow(
		`SELECT COUNT(*) FROM entities WHERE note_path='inbox/note.md' AND entity_type='person'`,
	).Scan(&n); err != nil || n != 1 {
		t.Errorf("person rows = %d (err=%v), want 1", n, err)
	}

	// And re-indexing must not wipe enrichment rows either.
	if err := q.UpdateNoteIndex(ctx, "inbox/note.md", "Test Title", "content", "golang,test"); err != nil {
		t.Fatalf("UpdateNoteIndex() error = %v", err)
	}
	if err := q.db.QueryRow(
		`SELECT COUNT(*) FROM entities WHERE note_path='inbox/note.md' AND entity_type='person'`,
	).Scan(&n); err != nil || n != 1 {
		t.Errorf("person rows after UpdateNoteIndex = %d (err=%v), want 1", n, err)
	}
}

func TestJobStoreInterface(t *testing.T) {
	var store JobStore = &Queue{}
	_ = store
}

func TestConnectionsHelpers(t *testing.T) {
	tmpDir := t.TempDir()
	q, err := NewQueue(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	mkJob := func(id, path string, age time.Duration) {
		j := &Job{ID: id, Type: "text", Status: "done",
			NotePath: path, Content: "Alice and Bob met. " + id,
			CreatedAt: now.Add(-age)}
		if err := q.CreateJob(ctx, j); err != nil {
			t.Fatal(err)
		}
	}
	mkJob("old-1", "khayal/old-1.md", 10*24*time.Hour)
	mkJob("old-2", "khayal/old-2.md", 30*24*time.Hour)
	mkJob("new-1", "khayal/new-1.md", 1*time.Hour)

	for _, p := range []string{"khayal/old-1.md", "khayal/old-2.md", "khayal/new-1.md"} {
		if err := q.SaveEntities(ctx, p, NoteEntities{People: []string{"Alice"}}); err != nil {
			t.Fatal(err)
		}
	}

	cutoff := now.Add(-7 * 24 * time.Hour)

	t.Run("GetEntitiesByNote returns values for type", func(t *testing.T) {
		vals, err := q.GetEntitiesByNote(ctx, "khayal/old-1.md", "person")
		if err != nil || len(vals) != 1 || vals[0] != "Alice" {
			t.Fatalf("got %v (err=%v)", vals, err)
		}
	})

	t.Run("GetNotesByEntity respects cutoff, most recent first", func(t *testing.T) {
		notes, err := q.GetNotesByEntity(ctx, "Alice", "person", cutoff)
		if err != nil {
			t.Fatal(err)
		}
		if len(notes) != 2 {
			t.Fatalf("expected only in-window notes, got %d", len(notes))
		}
		if notes[0].NotePath != "khayal/old-1.md" {
			t.Errorf("expected most-recent-first, got %s", notes[0].NotePath)
		}
	})

	t.Run("CountNotesByEntity excludes given note", func(t *testing.T) {
		n, err := q.CountNotesByEntity(ctx, "Alice", "person", cutoff, "khayal/old-1.md")
		if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("expected 1 other note, got %d", n)
		}
	})

	t.Run("GetChunkEmbeddingForNote round-trips", func(t *testing.T) {
		vec := []float32{0.25, 0.5, 0.75}
		if err := q.SaveChunk(ctx, "khayal/old-1.md", 0, "some content", vec); err != nil {
			t.Fatal(err)
		}
		got, ok, err := q.GetChunkEmbeddingForNote(ctx, "khayal/old-1.md")
		if err != nil || !ok || len(got) != 3 || got[2] != 0.75 {
			t.Fatalf("got %v ok=%v (err=%v)", got, ok, err)
		}
		if _, ok, _ := q.GetChunkEmbeddingForNote(ctx, "khayal/none.md"); ok {
			t.Error("expected no embedding for unknown note")
		}
	})
}

func TestUpdateJobResultAndLink(t *testing.T) {
	tmpDir := t.TempDir()
	q, err := NewQueue(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	ctx := context.Background()
	ingest := &Job{ID: "ingest-1", Type: "text", Status: "done",
		NotePath: "khayal/n.md", CreatedAt: time.Now()}
	if err := q.CreateJob(ctx, ingest); err != nil {
		t.Fatal(err)
	}
	job := &Job{ID: "j1", Type: "connections", Status: "processing",
		NotePath: "khayal/n.md", CreatedAt: time.Now()}
	if err := q.CreateJob(ctx, job); err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{"connections": []map[string]any{
		{"type": "person", "note_path": "khayal/old.md"},
	}}
	blob, _ := json.Marshal(payload)
	if err := q.UpdateJobResult(ctx, "j1", blob); err != nil {
		t.Fatalf("UpdateJobResult: %v", err)
	}

	if err := q.LinkConnectionsJob(ctx, "ingest-1", "j1"); err != nil {
		t.Fatalf("LinkConnectionsJob: %v", err)
	}

	got, err := q.GetJob(ctx, "j1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Result == nil || !strings.Contains(string(got.Result), "khayal/old.md") {
		t.Errorf("result not persisted: %s", string(got.Result))
	}

	src, err := q.GetJob(ctx, "ingest-1")
	if err != nil {
		t.Fatal(err)
	}
	if src.ConnectionsJobID != "j1" {
		t.Errorf("link not persisted: %q", src.ConnectionsJobID)
	}
}

func TestInitSchemaIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	q, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}

	// Second open must tolerate already-applied migrations.
	q2, err := NewQueue(dbPath)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	_ = q2.Close()
}

func TestListJobsHandlesNullColumns(t *testing.T) {
	tmpDir := t.TempDir()
	q, err := NewQueue(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	ctx := context.Background()
	// Raw insert leaves every optional column NULL — exactly what external
	// fixtures or future writers could produce.
	if _, err := q.db.ExecContext(ctx, `
		INSERT INTO jobs (id, type, status, created_at)
		VALUES ('raw-1', 'text', 'done', ?)`,
		time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}

	jobs, total, err := q.ListJobs(ctx, "all", 10, 0)
	if err != nil {
		t.Fatalf("ListJobs with NULL columns: %v", err)
	}
	if total != 1 || len(jobs) != 1 || jobs[0].ID != "raw-1" {
		t.Fatalf("unexpected result: total=%d jobs=%+v", total, jobs)
	}

	pending, err := q.GetPendingJobs(ctx, 10)
	if err != nil {
		t.Fatalf("GetPendingJobs with NULL columns: %v", err)
	}
	_ = pending
}

func TestTopSimilarChunks(t *testing.T) {
	tmpDir := t.TempDir()
	q, err := NewQueue(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	mk := func(id, path string, age time.Duration, vec []float32) {
		j := &Job{ID: id, Type: "text", Status: "done", NotePath: path,
			Content: "content " + id, CreatedAt: now.Add(-age)}
		if err := q.CreateJob(ctx, j); err != nil {
			t.Fatal(err)
		}
		if err := q.SaveChunk(ctx, path, 0, "chunk text "+id, vec); err != nil {
			t.Fatal(err)
		}
	}

	mk("old-a", "khayal/a.md", 30*24*time.Hour, []float32{1, 0, 0})
	mk("old-b", "khayal/b.md", 20*24*time.Hour, []float32{0.9, 0.1, 0}) // slightly less similar
	mk("recent", "khayal/recent.md", 1*time.Hour, []float32{1, 0, 0})   // age-excluded
	mk("old-c", "khayal/c.md", 15*24*time.Hour, []float32{0, 1, 0})     // low similarity

	query := []float32{1, 0, 0}
	cutoff := now.Add(-7 * 24 * time.Hour)

	got, err := q.TopSimilarChunks(ctx, query, 5, 0.5, cutoff, "khayal/self.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected a and b only, got %+v", got)
	}
	if got[0].NotePath != "khayal/a.md" || got[0].Score < 0.999 {
		t.Errorf("top = %+v, want self-similar ~1.0", got[0])
	}
	if got[1].NotePath != "khayal/b.md" {
		t.Errorf("second = %+v", got[1])
	}
	// Raw scores, not rescaled: b must keep its true cosine (~0.994).
	if got[1].Score > 0.999 {
		t.Errorf("scores must not be rescaled; b=%v", got[1].Score)
	}

	// Self exclusion.
	self, err := q.TopSimilarChunks(ctx, query, 5, 0.5, cutoff, "khayal/a.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range self {
		if m.NotePath == "khayal/a.md" {
			t.Error("self not excluded")
		}
	}
}

func TestGetNotesByEntityCaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	q, err := NewQueue(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	for _, j := range []*Job{
		{ID: "b1", Type: "text", Status: "done", NotePath: "khayal/b1.md",
			Content: "met Bob", CreatedAt: now.Add(-30 * 24 * time.Hour)},
		{ID: "b2", Type: "text", Status: "done", NotePath: "khayal/b2.md",
			Content: "met bob again", CreatedAt: now.Add(-20 * 24 * time.Hour)},
	} {
		if err := q.CreateJob(ctx, j); err != nil {
			t.Fatal(err)
		}
	}
	casing := map[string]string{"khayal/b1.md": "Bob", "khayal/b2.md": "bob"}
	for path, val := range casing {
		if err := q.SaveEntities(ctx, path, NoteEntities{People: []string{val}}); err != nil {
			t.Fatal(err)
		}
	}

	// Query with either casing must find BOTH notes.
	for _, query := range []string{"Bob", "bob", "BOB"} {
		matches, err := q.GetNotesByEntity(ctx, query, "person", now.Add(-10*24*time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 2 {
			t.Errorf("query %q matched %d notes, want 2", query, len(matches))
		}
	}
}

func TestSaveEntitiesResolvedDates(t *testing.T) {
	tmpDir := t.TempDir()
	q, err := NewQueue(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()

	ctx := context.Background()
	if err := q.SaveEntities(ctx, "khayal/n.md", NoteEntities{
		Dates:         []string{"tomorrow", "March 2024"},
		ResolvedDates: []string{"2026-08-26", ""},
	}); err != nil {
		t.Fatal(err)
	}

	var resolved sql.NullString
	rows, err := q.db.Query(`SELECT resolved_date FROM entities WHERE note_path='khayal/n.md' ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&resolved); err != nil {
			t.Fatal(err)
		}
	}
	first := struct{ R sql.NullString }{}
	_ = first
	// Re-query individually for deterministic assertions.
	var a, b sql.NullString
	_ = q.db.QueryRow(`SELECT resolved_date FROM entities WHERE note_path='khayal/n.md' AND entity_value='tomorrow'`).Scan(&a)
	_ = q.db.QueryRow(`SELECT resolved_date FROM entities WHERE note_path='khayal/n.md' AND entity_value='March 2024'`).Scan(&b)
	if !a.Valid || !strings.HasPrefix(a.String, "2026-08-26") {
		t.Errorf("tomorrow resolved_date = %v, want starting 2026-08-26", a)
	}
	if b.Valid {
		t.Errorf("March 2024 should have NULL resolution, got %q", b.String)
	}
}

func TestRemoveNote(t *testing.T) {
	tmpDir := t.TempDir()
	q, err := NewQueue(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("NewQueue() error = %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	notePath := "inbox/doomed.md"

	if err := q.IndexNote(ctx, notePath, "Doomed", "searchable body text", "tag1"); err != nil {
		t.Fatal(err)
	}
	if err := q.SaveChunk(ctx, notePath, 0, "chunk text", make([]float32, 4)); err != nil {
		t.Fatal(err)
	}
	if err := q.SaveEntities(ctx, notePath, NoteEntities{People: []string{"Bob"}}); err != nil {
		t.Fatal(err)
	}

	// sanity: all three stores populated
	var ftsN int
	if err := q.db.QueryRow(`SELECT COUNT(*) FROM notes_fts WHERE note_path = ?`, notePath).Scan(&ftsN); err != nil || ftsN == 0 {
		t.Fatalf("fts not seeded: n=%d err=%v", ftsN, err)
	}
	if n, _ := q.CountChunks(ctx, notePath); n == 0 {
		t.Fatal("chunks not seeded")
	}
	if n, _ := q.CountEntities(ctx, notePath, "person"); n == 0 {
		t.Fatal("entities not seeded")
	}

	if err := q.RemoveNote(ctx, notePath); err != nil {
		t.Fatalf("RemoveNote() error = %v", err)
	}

	if err := q.db.QueryRow(`SELECT COUNT(*) FROM notes_fts WHERE note_path = ?`, notePath).Scan(&ftsN); err != nil || ftsN != 0 {
		t.Errorf("fts rows survived: n=%d err=%v", ftsN, err)
	}
	if n, _ := q.CountChunks(ctx, notePath); n != 0 {
		t.Errorf("chunks survived: %d", n)
	}
	if n, _ := q.CountEntities(ctx, notePath, "person"); n != 0 {
		t.Errorf("entities survived: %d", n)
	}

	// idempotent
	if err := q.RemoveNote(ctx, notePath); err != nil {
		t.Errorf("RemoveNote must be idempotent, got %v", err)
	}

	// keyword search no longer finds it
	results, err := q.SearchKeyword(ctx, "searchable", 10, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.NotePath == notePath {
			t.Error("deleted note still searchable")
		}
	}
}
