package queue

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
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
	SearchSemantic(ctx context.Context, queryEmbedding []float32, limit int, minScore float64) ([]SearchResult, error)
	SaveChunk(ctx context.Context, notePath string, chunkIdx int, content string, embedding []float32) error
	IndexNote(ctx context.Context, notePath, title, content, tags string) error
	UpdateNoteIndex(ctx context.Context, notePath, title, content, tags string) error
	DeleteFromIndex(ctx context.Context, notePath string) error
}

func NewQueue(dbPath string) (*Queue, error) {
	return NewQueueWithLogger(dbPath, slog.Default())
}

func NewQueueWithLogger(dbPath string, logger *slog.Logger) (*Queue, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy_timeout: %w", err)
	}

	// No connection limit - let SQLite handle contention via busy_timeout
	// WAL mode supports concurrent readers + 1 writer
	db.SetMaxOpenConns(0)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	q := &Queue{db: db, logger: logger}
	if err := q.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return q, nil
}

type Queue struct {
	db     *sql.DB
	logger *slog.Logger
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
	JobID     string   `json:"id"`
	NotePath  string   `json:"note_path"`
	Title     string   `json:"title"`
	Excerpt   string   `json:"excerpt"`
	Score     float64  `json:"score"`
	Type      string   `json:"type"`
	CreatedAt string   `json:"created_at"`
	Tags      []string `json:"tags,omitempty"`
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
	// Drop existing table if it exists (migration for schema change)
	q.db.Exec(`DROP TABLE IF EXISTS notes_fts`)

	_, err := q.db.Exec(`CREATE VIRTUAL TABLE notes_fts USING fts5(
		note_path UNINDEXED,
		content,
		title,
		tags,
		tokenize = 'porter unicode61'
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

	const maxRetries = 3
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		_, err = q.db.ExecContext(ctx, `
			INSERT INTO jobs (id, type, status, note_path, source_url, source_file, content, user_context, created_at, processed_at, error, retries)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			job.ID, job.Type, job.Status, job.NotePath, job.SourceURL, job.SourceFile,
			job.Content, job.UserContext, job.CreatedAt.Format(time.RFC3339),
			nullableTime(job.ProcessedAt), job.Error, job.Retries)

		if err == nil {
			q.logger.Debug("job created",
				"job_id", job.ID,
				"type", job.Type,
			)
			return nil
		}

		if isLockError(err) {
			q.logger.Warn("sqlite locked, retrying create job",
				"job_id", job.ID,
				"attempt", attempt+1,
				"error", err,
			)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}

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
	const maxRetries = 3
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		_, err = q.db.ExecContext(ctx, `
			UPDATE jobs SET status = ?, note_path = ?, processed_at = ?, error = ?, retries = ?
			WHERE id = ?`,
			job.Status, job.NotePath, nullableTime(job.ProcessedAt), job.Error, job.Retries, job.ID)
		if err == nil {
			return nil
		}
		if isLockError(err) {
			q.logger.Warn("sqlite locked, retrying update job",
				"job_id", job.ID,
				"attempt", attempt+1,
				"error", err,
			)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}
	return err
}

func (q *Queue) UpdateJobStatus(ctx context.Context, id, status string) error {
	const maxRetries = 3
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		_, err = q.db.ExecContext(ctx, "UPDATE jobs SET status = ? WHERE id = ?", status, id)
		if err == nil {
			q.logger.Debug("job status changed",
				"job_id", id,
				"new_status", status,
			)
			return nil
		}
		if isLockError(err) {
			q.logger.Warn("sqlite locked, retrying update job status",
				"job_id", id,
				"attempt", attempt+1,
				"error", err,
			)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}
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

func (q *Queue) FetchAndLockPendingJobs(ctx context.Context, limit int) ([]Job, error) {
	rows, err := q.db.QueryContext(ctx, `
		UPDATE jobs 
		SET status = 'queued'
		WHERE id IN (
			SELECT id FROM jobs 
			WHERE status = 'pending' 
			ORDER BY created_at ASC 
			LIMIT ?
		)
		RETURNING id, type, status, note_path, source_url, source_file, content, user_context, created_at, processed_at, error, retries`,
		limit)
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
	_, err := q.db.ExecContext(ctx, `UPDATE jobs SET status = 'pending' WHERE status = 'processing' OR status = 'queued'`)
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
		ORDER BY bm25(notes_fts, 0, 3.0, 1.0, 1.0)
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

		tags, err := q.GetNoteTags(ctx, r.NotePath)
		if err == nil {
			r.Tags = tags
		}

		title, err := q.GetNoteTitle(ctx, r.NotePath)
		if err == nil {
			r.Title = title
		}

		results = append(results, r)
	}

	return results, rows.Err()
}

func (q *Queue) SearchSemantic(ctx context.Context, queryEmbedding []float32, limit int, minScore float64) ([]SearchResult, error) {
	type scoredChunk struct {
		notePath  string
		chunkIdx  int
		content   string
		score     float64
		jobType   string
		createdAt string
		jobID     string
	}

	if len(queryEmbedding) == 0 {
		return []SearchResult{}, nil
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

			if score < minScore {
				continue
			}

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
		sr := SearchResult{
			JobID:     r.jobID,
			NotePath:  r.notePath,
			Excerpt:   r.content,
			Score:     r.score,
			Type:      r.jobType,
			CreatedAt: r.createdAt,
		}

		tags, err := q.GetNoteTags(ctx, r.notePath)
		if err == nil {
			sr.Tags = tags
		}

		title, err := q.GetNoteTitle(ctx, r.notePath)
		if err == nil && title != "" {
			sr.Title = title
		}

		searchResults[i] = sr
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
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		now := time.Now().Format(time.RFC3339)

		_, err := q.db.ExecContext(ctx, `DELETE FROM entities WHERE note_path = ?`, notePath)
		if err != nil {
			lastErr = err
			if isLockError(err) {
				q.logger.Warn("sqlite locked, retrying index note",
					"note_path", notePath,
					"attempt", attempt+1,
				)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return err
		}

		if title != "" {
			_, err = q.db.ExecContext(ctx, `
				INSERT OR REPLACE INTO entities (note_path, chunk_idx, entity_type, entity_value, created_at) 
				VALUES (?, NULL, 'title', ?, ?)`,
				notePath, title, now)
			if err != nil {
				lastErr = err
				if isLockError(err) {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				return err
			}
		}

		if tags != "" {
			tagList := strings.Split(tags, ",")
			for _, tag := range tagList {
				tag = strings.TrimSpace(tag)
				if tag == "" {
					continue
				}
				_, err = q.db.ExecContext(ctx, `
					INSERT INTO entities (note_path, chunk_idx, entity_type, entity_value, created_at) 
					VALUES (?, NULL, 'tag', ?, ?)`,
					notePath, tag, now)
				if err != nil {
					lastErr = err
					if isLockError(err) {
						time.Sleep(100 * time.Millisecond)
						break
					}
					return err
				}
			}
			if isLockError(lastErr) {
				continue
			}
		}

		_, err = q.db.ExecContext(ctx, `INSERT INTO notes_fts (note_path, content, title, tags) VALUES (?, ?, ?, ?)`, notePath, content, title, tags)
		if err != nil && isFTSErr(err) {
			return nil
		}
		if err != nil {
			lastErr = err
			if isLockError(err) {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return err
		}
		return nil
	}
	return lastErr
}

func (q *Queue) UpdateNoteIndex(ctx context.Context, notePath, title, content, tags string) error {
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		now := time.Now().Format(time.RFC3339)

		_, err := q.db.ExecContext(ctx, `DELETE FROM entities WHERE note_path = ?`, notePath)
		if err != nil {
			lastErr = err
			if isLockError(err) {
				q.logger.Warn("sqlite locked, retrying update note index",
					"note_path", notePath,
					"attempt", attempt+1,
				)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return err
		}

		if title != "" {
			_, err = q.db.ExecContext(ctx, `
				INSERT OR REPLACE INTO entities (note_path, chunk_idx, entity_type, entity_value, created_at) 
				VALUES (?, NULL, 'title', ?, ?)`,
				notePath, title, now)
			if err != nil {
				lastErr = err
				if isLockError(err) {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				return err
			}
		}

		if tags != "" {
			tagList := strings.Split(tags, ",")
			for _, tag := range tagList {
				tag = strings.TrimSpace(tag)
				if tag == "" {
					continue
				}
				_, err = q.db.ExecContext(ctx, `
					INSERT INTO entities (note_path, chunk_idx, entity_type, entity_value, created_at) 
					VALUES (?, NULL, 'tag', ?, ?)`,
					notePath, tag, now)
				if err != nil {
					lastErr = err
					if isLockError(err) {
						time.Sleep(100 * time.Millisecond)
						break
					}
					return err
				}
			}
			if isLockError(lastErr) {
				continue
			}
		}

		if _, err := q.db.ExecContext(ctx, `DELETE FROM notes_fts WHERE note_path = ?`, notePath); err != nil && !isFTSErr(err) {
			lastErr = err
			if isLockError(err) {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return err
		}
		_, err = q.db.ExecContext(ctx, `INSERT INTO notes_fts (note_path, content, title, tags) VALUES (?, ?, ?, ?)`, notePath, content, title, tags)
		if err != nil && isFTSErr(err) {
			return nil
		}
		if err != nil {
			lastErr = err
			if isLockError(err) {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return err
		}
		return nil
	}
	return lastErr
}

func (q *Queue) DeleteFromIndex(ctx context.Context, notePath string) error {
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		_, err := q.db.ExecContext(ctx, `DELETE FROM notes_fts WHERE note_path = ?`, notePath)
		if err != nil && isFTSErr(err) {
			return nil
		}
		if err != nil {
			lastErr = err
			if isLockError(err) {
				q.logger.Warn("sqlite locked, retrying delete from index",
					"note_path", notePath,
					"attempt", attempt+1,
				)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return err
		}
		return nil
	}
	return lastErr
}

func isFTSErr(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "no such table: notes_fts") || contains(errStr, "no such module")
}

func isLockError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "database is locked") ||
		contains(errStr, "database table is locked") ||
		contains(errStr, "SQLITE_BUSY") ||
		contains(errStr, "SQLITE_LOCKED")
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
	if len(v) == 0 {
		return 0
	}
	sum := float64(0)
	for _, f := range v {
		sum += float64(f) * float64(f)
	}
	return math.Sqrt(sum)
}

func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}

	dot := float64(0)
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}

	normA := normalize(a)
	normB := normalize(b)

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (normA * normB)
}

type TagCount struct {
	Name  string
	Count int
}

type PersonCount struct {
	Name  string
	Count int
}

func (q *Queue) CountNotes(ctx context.Context) (int, error) {
	var count int
	err := q.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT note_path) FROM chunks
		WHERE note_path LIKE 'khayal/%'
	`).Scan(&count)
	return count, err
}

func (q *Queue) CountNotesSince(ctx context.Context, since time.Time) (int, error) {
	var count int
	err := q.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT note_path) FROM chunks
		WHERE note_path LIKE 'khayal/%'
		AND created_at >= ?
	`, since.Format(time.RFC3339)).Scan(&count)
	return count, err
}

func (q *Queue) CountByType(ctx context.Context) (map[string]int, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT type, COUNT(*) FROM jobs
		WHERE status = 'done' AND type IN ('text', 'article', 'image')
		GROUP BY type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var t string
		var count int
		if err := rows.Scan(&t, &count); err != nil {
			return nil, err
		}
		result[t] = count
	}
	return result, rows.Err()
}

func (q *Queue) GetTopTags(ctx context.Context, limit int) ([]TagCount, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT entity_value, COUNT(*) as cnt
		FROM entities
		WHERE entity_type = 'tag'
		GROUP BY entity_value
		ORDER BY cnt DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TagCount
	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Name, &tc.Count); err != nil {
			return nil, err
		}
		result = append(result, tc)
	}
	return result, rows.Err()
}

func (q *Queue) GetTopPeople(ctx context.Context, limit int) ([]PersonCount, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT entity_value, COUNT(*) as cnt
		FROM entities
		WHERE entity_type = 'person'
		GROUP BY entity_value
		ORDER BY cnt DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PersonCount
	for rows.Next() {
		var pc PersonCount
		if err := rows.Scan(&pc.Name, &pc.Count); err != nil {
			return nil, err
		}
		result = append(result, pc)
	}
	return result, rows.Err()
}

func (q *Queue) GetStreaks(ctx context.Context) (int, int, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT DISTINCT DATE(created_at) as capture_date
		FROM jobs
		WHERE status = 'done'
		ORDER BY capture_date DESC
	`)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err != nil {
			return 0, 0, err
		}
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		dates = append(dates, t)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	if len(dates) == 0 {
		return 0, 0, nil
	}

	today := time.Now().Truncate(24 * time.Hour)
	currentStreak := 0
	longestStreak := 0
	streak := 1

	for i := 0; i < len(dates)-1; i++ {
		diff := dates[i].Sub(dates[i+1]).Hours() / 24
		if diff == 1 {
			streak++
		} else {
			if streak > longestStreak {
				longestStreak = streak
			}
			streak = 1
		}
	}
	if streak > longestStreak {
		longestStreak = streak
	}

	if len(dates) > 0 && dates[0].Equal(today) || dates[0].Equal(today.AddDate(0, 0, -1)) {
		streak = 1
		for i := 0; i < len(dates)-1; i++ {
			diff := dates[i].Sub(dates[i+1]).Hours() / 24
			if diff == 1 {
				streak++
			} else {
				break
			}
		}
		currentStreak = streak
	}

	return currentStreak, longestStreak, nil
}

func (q *Queue) GetNoteTags(ctx context.Context, notePath string) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT DISTINCT entity_value
		FROM entities
		WHERE note_path = ? AND entity_type = 'tag'
		ORDER BY entity_value
	`, notePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (q *Queue) GetNoteTitle(ctx context.Context, notePath string) (string, error) {
	var title string
	err := q.db.QueryRowContext(ctx, `
		SELECT entity_value FROM entities WHERE note_path = ? AND entity_type = 'title'
	`, notePath).Scan(&title)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return title, err
}
