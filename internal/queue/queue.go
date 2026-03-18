package queue

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

const (
	batchSize       = 1000
	maxSearchChunks = 100000
)

type JobStore interface {
	CreateJob(ctx context.Context, job *Job) error
	GetJob(ctx context.Context, id string) (*Job, error)
	UpdateJob(ctx context.Context, job *Job) error
	UpdateJobStatus(ctx context.Context, id, status string) error
	ListJobs(ctx context.Context, status string, limit, offset int) ([]Job, int, error)
	GetPendingJobs(ctx context.Context, limit int) ([]Job, error)
	ResetStuckJobs(ctx context.Context) error
	CountByStatus(ctx context.Context, status string) (int, error)
	DeleteJob(ctx context.Context, id string) error
	SaveEmbedding(ctx context.Context, jobID, model string, embedding []float32) error
	SearchKeyword(ctx context.Context, query string, limit int) ([]SearchResult, error)
	SearchSemantic(ctx context.Context, queryEmbedding []float32, limit int) ([]SearchResult, error)
	SaveChunk(ctx context.Context, notePath string, chunkIdx int, content string, embedding []float32) error
	IndexNote(ctx context.Context, notePath, title, content, tags string) error
	UpdateNoteIndex(ctx context.Context, notePath, title, content, tags string) error
	DeleteFromIndex(ctx context.Context, notePath string) error
}

func NewQueue(dbPath string) (*Queue, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	q := &Queue{db: db}
	if err := q.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return q, nil
}

type Queue struct {
	db *sql.DB
}

type Job struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	NotePath    string     `json:"note_path,omitempty"`
	SourceURL   string     `json:"source_url,omitempty"`
	SourceFile  string     `json:"source_file,omitempty"`
	Content     string     `json:"content,omitempty"`
	UserContext string     `json:"user_context,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
	Retries     int        `json:"retries"`
}

type SearchResult struct {
	JobID     string  `json:"id"`
	NotePath  string  `json:"note_path"`
	Title     string  `json:"title"`
	Excerpt   string  `json:"excerpt"`
	Score     float64 `json:"score"`
	Type      string  `json:"type"`
	CreatedAt string  `json:"created_at"`
}

func (q *Queue) initSchema() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			note_path TEXT,
			source_url TEXT,
			source_file TEXT,
			content TEXT,
			user_context TEXT,
			created_at TEXT NOT NULL,
			processed_at TEXT,
			error TEXT,
			retries INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_created ON jobs(created_at)`,
		`CREATE TABLE IF NOT EXISTS embeddings (
			id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL,
			vector BLOB NOT NULL,
			model TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (job_id) REFERENCES jobs(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_embeddings_job ON embeddings(job_id)`,
		`CREATE TABLE IF NOT EXISTS entities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			note_path TEXT NOT NULL,
			chunk_idx INTEGER,
			entity_type TEXT NOT NULL,
			entity_value TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_entities_note ON entities(note_path)`,
		`CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(entity_type)`,
		`CREATE TABLE IF NOT EXISTS chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			note_path TEXT NOT NULL,
			chunk_idx INTEGER NOT NULL,
			content TEXT NOT NULL,
			embedding BLOB NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_note ON chunks(note_path)`,
	}

	for _, stmt := range statements {
		if _, err := q.db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute: %w", err)
		}
	}

	if err := q.initFTS(); err != nil {
		return fmt.Errorf("FTS5 not available: %w", err)
	}

	return nil
}

func (q *Queue) initFTS() error {
	_, err := q.db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
		note_path,
		content,
		title,
		tags
	)`)
	return err
}

func (q *Queue) Close() error {
	return q.db.Close()
}

func (q *Queue) CreateJob(ctx context.Context, job *Job) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.Status == "" {
		job.Status = "pending"
	}

	_, err := q.db.ExecContext(ctx, `
		INSERT INTO jobs (id, type, status, note_path, source_url, source_file, content, user_context, created_at, processed_at, error, retries)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.Type, job.Status, job.NotePath, job.SourceURL, job.SourceFile,
		job.Content, job.UserContext, job.CreatedAt.Format(time.RFC3339),
		nullableTime(job.ProcessedAt), job.Error, job.Retries)
	return err
}

func (q *Queue) GetJob(ctx context.Context, id string) (*Job, error) {
	var createdAtStr string
	var processedAtStr sql.NullString
	job := &Job{}
	err := q.db.QueryRowContext(ctx, `
		SELECT id, type, status, note_path, source_url, source_file, content, user_context, created_at, processed_at, error, retries
		FROM jobs WHERE id = ?`, id).Scan(
		&job.ID, &job.Type, &job.Status, &job.NotePath, &job.SourceURL, &job.SourceFile,
		&job.Content, &job.UserContext, &createdAtStr, &processedAtStr, &job.Error, &job.Retries)
	if err != nil {
		return nil, err
	}

	var parseErr error
	job.CreatedAt, parseErr = time.Parse(time.RFC3339, createdAtStr)
	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", parseErr)
	}

	if processedAtStr.Valid {
		t, err := time.Parse(time.RFC3339, processedAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse processed_at: %w", err)
		}
		job.ProcessedAt = &t
	}

	return job, nil
}

func (q *Queue) UpdateJob(ctx context.Context, job *Job) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE jobs SET status = ?, note_path = ?, processed_at = ?, error = ?, retries = ?
		WHERE id = ?`,
		job.Status, job.NotePath, nullableTime(job.ProcessedAt), job.Error, job.Retries, job.ID)
	return err
}

func (q *Queue) UpdateJobStatus(ctx context.Context, id, status string) error {
	_, err := q.db.ExecContext(ctx, "UPDATE jobs SET status = ? WHERE id = ?", status, id)
	return err
}

func (q *Queue) ListJobs(ctx context.Context, status string, limit, offset int) ([]Job, int, error) {
	var total int

	query := "SELECT COUNT(*) FROM jobs"
	args := []interface{}{}
	if status != "" && status != "all" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	if err := q.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query = `SELECT id, type, status, note_path, source_url, source_file, content, user_context, created_at, processed_at, error, retries FROM jobs`
	if status != "" && status != "all" {
		query += " WHERE status = ?"
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var job Job
		var createdAtStr string
		var processedAtStr sql.NullString
		if err := rows.Scan(&job.ID, &job.Type, &job.Status, &job.NotePath, &job.SourceURL, &job.SourceFile,
			&job.Content, &job.UserContext, &createdAtStr, &processedAtStr, &job.Error, &job.Retries); err != nil {
			return nil, 0, err
		}

		var parseErr error
		job.CreatedAt, parseErr = time.Parse(time.RFC3339, createdAtStr)
		if parseErr != nil {
			return nil, 0, fmt.Errorf("failed to parse created_at: %w", parseErr)
		}
		if processedAtStr.Valid {
			t, err := time.Parse(time.RFC3339, processedAtStr.String)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to parse processed_at: %w", err)
			}
			job.ProcessedAt = &t
		}
		jobs = append(jobs, job)
	}

	return jobs, total, rows.Err()
}

func (q *Queue) GetPendingJobs(ctx context.Context, limit int) ([]Job, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT id, type, status, note_path, source_url, source_file, content, user_context, created_at, processed_at, error, retries
		FROM jobs WHERE status = 'pending' ORDER BY created_at ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var job Job
		var createdAtStr string
		var processedAtStr sql.NullString
		if err := rows.Scan(&job.ID, &job.Type, &job.Status, &job.NotePath, &job.SourceURL, &job.SourceFile,
			&job.Content, &job.UserContext, &createdAtStr, &processedAtStr, &job.Error, &job.Retries); err != nil {
			return nil, err
		}

		var parseErr error
		job.CreatedAt, parseErr = time.Parse(time.RFC3339, createdAtStr)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", parseErr)
		}
		if processedAtStr.Valid {
			t, err := time.Parse(time.RFC3339, processedAtStr.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse processed_at: %w", err)
			}
			job.ProcessedAt = &t
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

func (q *Queue) ResetStuckJobs(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, `UPDATE jobs SET status = 'pending' WHERE status = 'processing'`)
	return err
}

func (q *Queue) CountByStatus(ctx context.Context, status string) (int, error) {
	var count int
	err := q.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs WHERE status = ?", status).Scan(&count)
	return count, err
}

func (q *Queue) DeleteJob(ctx context.Context, id string) error {
	_, err := q.db.ExecContext(ctx, "DELETE FROM jobs WHERE id = ?", id)
	return err
}

func (q *Queue) SaveEmbedding(ctx context.Context, jobID, model string, embedding []float32) error {
	id := uuid.New().String()
	blob := encodeEmbedding(embedding)

	_, err := q.db.ExecContext(ctx, `
		INSERT INTO embeddings (id, job_id, vector, model, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		id, jobID, blob, model, time.Now().Format(time.RFC3339))
	return err
}

func (q *Queue) SearchKeyword(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT j.id, j.note_path, j.type, j.created_at,
			   snippet(notes_fts, 1, '...', '...', '', 50) as excerpt
		FROM notes_fts fts
		JOIN jobs j ON fts.note_path = j.note_path
		WHERE notes_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, query, limit)
	if err != nil {
		if isFTSErr(err) {
			return []SearchResult{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.JobID, &r.NotePath, &r.Type, &r.CreatedAt, &r.Excerpt); err != nil {
			return nil, err
		}
		r.Score = 1.0
		results = append(results, r)
	}

	return results, rows.Err()
}

func (q *Queue) SearchSemantic(ctx context.Context, queryEmbedding []float32, limit int) ([]SearchResult, error) {
	type scoredChunk struct {
		notePath  string
		chunkIdx  int
		content   string
		score     float64
		jobType   string
		createdAt string
		jobID     string
	}

	queryNorm := normalize(queryEmbedding)
	if queryNorm == 0 {
		return []SearchResult{}, nil
	}

	noteBest := make(map[string]scoredChunk)
	offset := 0

	for {
		rows, err := q.db.QueryContext(ctx, `
			SELECT c.note_path, c.chunk_idx, c.content, c.embedding, j.type, j.created_at, j.id
			FROM chunks c
			JOIN jobs j ON c.note_path = j.note_path
			WHERE c.embedding IS NOT NULL
			LIMIT ? OFFSET ?`, batchSize, offset)
		if err != nil {
			return nil, err
		}

		var hasRows bool
		for rows.Next() {
			hasRows = true
			var notePath string
			var chunkIdx int
			var content string
			var blob []byte
			var jobType, createdAt, jobID string

			if err := rows.Scan(&notePath, &chunkIdx, &content, &blob, &jobType, &createdAt, &jobID); err != nil {
				rows.Close()
				return nil, err
			}

			embedding := decodeEmbedding(blob)
			score := cosine(queryEmbedding, embedding)

			best, exists := noteBest[notePath]
			if !exists || score > best.score {
				noteBest[notePath] = scoredChunk{
					notePath:  notePath,
					chunkIdx:  chunkIdx,
					content:   content,
					score:     score,
					jobType:   jobType,
					createdAt: createdAt,
					jobID:     jobID,
				}
			}
		}
		rows.Close()

		if err := rows.Err(); err != nil {
			return nil, err
		}

		if !hasRows || offset >= maxSearchChunks {
			break
		}
		offset += batchSize
	}

	results := make([]scoredChunk, 0, len(noteBest))
	for _, chunk := range noteBest {
		results = append(results, chunk)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if len(results) > limit {
		results = results[:limit]
	}

	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			JobID:     r.jobID,
			NotePath:  r.notePath,
			Excerpt:   r.content,
			Score:     r.score,
			Type:      r.jobType,
			CreatedAt: r.createdAt,
		}
	}

	return searchResults, nil
}

func (q *Queue) SaveChunk(ctx context.Context, notePath string, chunkIdx int, content string, embedding []float32) error {
	blob := encodeEmbedding(embedding)

	_, err := q.db.ExecContext(ctx, `
		INSERT INTO chunks (note_path, chunk_idx, content, embedding, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		notePath, chunkIdx, content, blob, time.Now().Format(time.RFC3339))
	return err
}

func (q *Queue) IndexNote(ctx context.Context, notePath, title, content, tags string) error {
	_, err := q.db.ExecContext(ctx, `INSERT INTO notes_fts (note_path, content, title, tags) VALUES (?, ?, ?, ?)`, notePath, content, title, tags)
	if err != nil && isFTSErr(err) {
		return nil
	}
	return err
}

func (q *Queue) UpdateNoteIndex(ctx context.Context, notePath, title, content, tags string) error {
	if _, err := q.db.ExecContext(ctx, `DELETE FROM notes_fts WHERE note_path = ?`, notePath); err != nil && !isFTSErr(err) {
		return err
	}
	_, err := q.db.ExecContext(ctx, `INSERT INTO notes_fts (note_path, content, title, tags) VALUES (?, ?, ?, ?)`, notePath, content, title, tags)
	if err != nil && isFTSErr(err) {
		return nil
	}
	return err
}

func (q *Queue) DeleteFromIndex(ctx context.Context, notePath string) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM notes_fts WHERE note_path = ?`, notePath)
	if err != nil && isFTSErr(err) {
		return nil
	}
	return err
}

func isFTSErr(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "no such table: notes_fts") || contains(errStr, "no such module")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func nullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

func encodeEmbedding(embedding []float32) []byte {
	blob := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		binary.LittleEndian.PutUint32(blob[i*4:], math.Float32bits(v))
	}
	return blob
}

func decodeEmbedding(blob []byte) []float32 {
	n := len(blob) / 4
	v := make([]float32, n)
	for i := 0; i < n; i++ {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(blob[i*4:]))
	}
	return v
}

func normalize(v []float32) float64 {
	sum := float64(0)
	for _, f := range v {
		sum += float64(f) * float64(f)
	}
	return math.Sqrt(sum)
}

func cosine(a, b []float32) float64 {
	dot := float64(0)
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return dot
}
