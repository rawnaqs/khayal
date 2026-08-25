package queue

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rawnaqs/khayal/internal/constants"
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
	SearchKeyword(ctx context.Context, query string, limit int, from, to *time.Time) ([]SearchResult, error)
	SearchSemantic(ctx context.Context, queryEmbedding []float32, limit int, minScore float64, from, to *time.Time) ([]SearchResult, error)
	SaveChunk(ctx context.Context, notePath string, chunkIdx int, content string, embedding []float32) error
	DeleteChunksByNote(ctx context.Context, notePath string) error
	CountChunks(ctx context.Context, notePath string) (int, error)
	SaveEntities(ctx context.Context, notePath string, ents NoteEntities) error
	DeleteEntities(ctx context.Context, notePath string) error
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
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to set busy_timeout: %w", err)
	}

	// No connection limit - let SQLite handle contention via busy_timeout
	// WAL mode supports concurrent readers + 1 writer
	db.SetMaxOpenConns(0)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	q := &Queue{db: db, logger: logger}
	if err := q.initSchema(); err != nil {
		_ = db.Close()
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
	// Connections chaining (phase 2): set on ingest jobs after their
	// connections job is created; the connections job itself stores its
	// payload in Result.
	Result           json.RawMessage `json:"result,omitempty"`
	ConnectionsJobID string          `json:"connections_job_id,omitempty"`
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
		// v1.1: drop the legacy per-job embeddings table; chunks is the
		// canonical vector store.
		`DROP TABLE IF EXISTS embeddings`,
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
		`CREATE TABLE IF NOT EXISTS stats_cache (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	}

	for _, stmt := range statements {
		if _, err := q.db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute: %w", err)
		}
	}

	// v1.1 phase 2 migrations: idempotent column adds. SQLite has no
	// "ADD COLUMN IF NOT EXISTS", so a duplicate-column error means an
	// already-migrated database and is treated as success.
	for _, alter := range []string{
		`ALTER TABLE jobs ADD COLUMN result TEXT`,
		`ALTER TABLE jobs ADD COLUMN connections_job_id TEXT`,
		`ALTER TABLE entities ADD COLUMN resolved_date DATETIME`,
	} {
		if _, err := q.db.Exec(alter); err != nil &&
			!strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("failed to migrate: %w", err)
		}
	}

	if err := q.initFTS(); err != nil {
		return fmt.Errorf("FTS5 not available: %w", err)
	}

	return nil
}

func (q *Queue) initFTS() error {
	// Check if FTS table exists
	var count int
	_ = q.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='notes_fts'").Scan(&count)

	if count == 0 {
		_, err := q.db.Exec(`CREATE VIRTUAL TABLE notes_fts USING fts5(
			note_path UNINDEXED,
			content,
			title,
			tags,
			tokenize = 'porter unicode61'
		)`)
		return err
	}

	// Table exists - use 'khayal reindex --force' to rebuild if schema changed
	return nil
}

func (q *Queue) Close() error {
	return q.db.Close()
}

func (q *Queue) CreateJob(ctx context.Context, job *Job) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	if job.Status == "" {
		job.Status = "pending"
	}

	const maxRetries = constants.SQLiteMaxRetries
	var err error
	for attempt := range maxRetries {
		_, err = q.db.ExecContext(ctx, `
			INSERT INTO jobs (id, type, status, note_path, source_url, source_file, content, user_context, created_at, processed_at, error, retries)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			job.ID, job.Type, job.Status, job.NotePath, job.SourceURL, job.SourceFile,
			job.Content, job.UserContext, job.CreatedAt.UTC().Format(time.RFC3339),
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
			time.Sleep(constants.SQLiteRetrySleep)
			continue
		}
		break
	}

	return err
}

func (q *Queue) GetJob(ctx context.Context, id string) (*Job, error) {
	var createdAtStr string
	var processedAtStr sql.NullString
	var resultStr sql.NullString
	var connJobID sql.NullString
	job := &Job{}
	err := q.db.QueryRowContext(ctx, `
		SELECT id, type, status, note_path, source_url, source_file, content, user_context,
		       created_at, processed_at, error, retries, result, connections_job_id
		FROM jobs WHERE id = ?`, id).Scan(
		&job.ID, &job.Type, &job.Status, &job.NotePath, &job.SourceURL, &job.SourceFile,
		&job.Content, &job.UserContext, &createdAtStr, &processedAtStr, &job.Error, &job.Retries,
		&resultStr, &connJobID)
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

	if resultStr.Valid && resultStr.String != "" {
		job.Result = json.RawMessage(resultStr.String)
	}
	job.ConnectionsJobID = connJobID.String

	return job, nil
}

func (q *Queue) UpdateJob(ctx context.Context, job *Job) error {
	const maxRetries = constants.SQLiteMaxRetries
	var err error
	for attempt := range maxRetries {
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
			time.Sleep(constants.SQLiteRetrySleep)
			continue
		}
		break
	}
	return err
}

func (q *Queue) UpdateJobStatus(ctx context.Context, id, status string) error {
	const maxRetries = constants.SQLiteMaxRetries
	var err error
	for attempt := range maxRetries {
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
			time.Sleep(constants.SQLiteRetrySleep)
			continue
		}
		break
	}
	return err
}

func (q *Queue) ListJobs(ctx context.Context, status string, limit, offset int) ([]Job, int, error) {
	var total int

	query := "SELECT COUNT(*) FROM jobs"
	args := []any{}
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

	jobs := make([]Job, 0, limit)
	for rows.Next() {
		var job Job
		var createdAtStr string
		var notePath, sourceURL, sourceFile, userContext, content, errorStr sql.NullString
		var processedAtStr sql.NullString
		if err := rows.Scan(&job.ID, &job.Type, &job.Status, &notePath, &sourceURL, &sourceFile,
			&content, &userContext, &createdAtStr, &processedAtStr, &errorStr, &job.Retries); err != nil {
			return nil, 0, err
		}

		job.NotePath = notePath.String
		job.SourceURL = sourceURL.String
		job.SourceFile = sourceFile.String
		job.UserContext = userContext.String
		job.Content = content.String
		job.Error = errorStr.String

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

	jobs := make([]Job, 0, limit)
	for rows.Next() {
		var job Job
		var createdAtStr string
		var notePath, sourceURL, sourceFile, userContext, content, errorStr sql.NullString
		var processedAtStr sql.NullString
		if err := rows.Scan(&job.ID, &job.Type, &job.Status, &notePath, &sourceURL, &sourceFile,
			&content, &userContext, &createdAtStr, &processedAtStr, &errorStr, &job.Retries); err != nil {
			return nil, err
		}
		job.NotePath = notePath.String
		job.SourceURL = sourceURL.String
		job.SourceFile = sourceFile.String
		job.UserContext = userContext.String
		job.Content = content.String
		job.Error = errorStr.String

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

	jobs := make([]Job, 0, limit)
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

func (q *Queue) SearchKeyword(ctx context.Context, query string, limit int, from, to *time.Time) ([]SearchResult, error) {
	baseSQL := `
		SELECT j.id, j.note_path, j.type, j.created_at,
			   snippet(notes_fts, 1, '...', '...', '', 50) as excerpt,
			   bm25(notes_fts, 0, 3.0, 1.0, 1.0) as bm25_score
		FROM notes_fts fts
		JOIN jobs j ON fts.note_path = j.note_path
		WHERE notes_fts MATCH ?`

	args := []any{query}

	if from != nil {
		baseSQL += ` AND j.created_at >= ?`
		args = append(args, from.UTC().Format(time.RFC3339))
	}
	if to != nil {
		baseSQL += ` AND j.created_at <= ?`
		args = append(args, to.UTC().Format(time.RFC3339))
	}

	baseSQL += `
		ORDER BY bm25(notes_fts, 0, 3.0, 1.0, 1.0)
		LIMIT ?`
	args = append(args, limit)

	rows, err := q.db.QueryContext(ctx, baseSQL, args...)
	if err != nil {
		if isFTSErr(err) {
			return []SearchResult{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	results := make([]SearchResult, 0, limit)
	notePaths := make([]string, 0, limit)

	for rows.Next() {
		var r SearchResult
		var bm25Score float64
		if err := rows.Scan(&r.JobID, &r.NotePath, &r.Type, &r.CreatedAt, &r.Excerpt, &bm25Score); err != nil {
			return nil, err
		}
		relevance := -bm25Score
		r.Score = relevance / (relevance + 1)
		results = append(results, r)
		notePaths = append(notePaths, r.NotePath)
	}

	// Normalize keyword scores to (0,1] - best result gets 1.0
	if len(results) > 0 {
		maxScore := results[0].Score // Already sorted by BM25, first is best
		if maxScore > 0 {
			for i := range results {
				results[i].Score = results[i].Score / maxScore
			}
		}
	}

	// Batch fetch tags and titles to avoid N+1 queries
	tagsMap, _ := q.BatchGetNoteTags(ctx, notePaths)
	titlesMap, _ := q.BatchGetNoteTitles(ctx, notePaths)

	for i := range results {
		results[i].Tags = tagsMap[results[i].NotePath]
		if title, ok := titlesMap[results[i].NotePath]; ok {
			results[i].Title = title
		}
	}

	return results, rows.Err()
}

func (q *Queue) SearchSemantic(ctx context.Context, queryEmbedding []float32, limit int, minScore float64, from, to *time.Time) ([]SearchResult, error) {
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

	noteBest := make(map[string]scoredChunk)
	offset := 0

	dateFilter := ""
	var args []any
	if from != nil {
		dateFilter += ` AND j.created_at >= ?`
		args = append(args, from.UTC().Format(time.RFC3339))
	}
	if to != nil {
		dateFilter += ` AND j.created_at <= ?`
		args = append(args, to.UTC().Format(time.RFC3339))
	}

	for {
		query := fmt.Sprintf(`
			SELECT c.note_path, c.chunk_idx, c.content, c.embedding, j.type, j.created_at, j.id
			FROM chunks c
			JOIN jobs j ON c.note_path = j.note_path
			WHERE c.embedding IS NOT NULL%s
			LIMIT ? OFFSET ?`, dateFilter)

		qArgs := append([]any{}, args...)
		qArgs = append(qArgs, batchSize, offset)

		rows, err := q.db.QueryContext(ctx, query, qArgs...)
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

	// Normalize semantic scores to (0,1] - best result gets 1.0
	if len(results) > 0 && results[0].score > 0 {
		maxScore := results[0].score
		for i := range results {
			results[i].score = results[i].score / maxScore
		}
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
		notePath, chunkIdx, content, blob, time.Now().UTC().Format(time.RFC3339))
	return err
}

// CountChunks returns the number of stored chunks for a note.
func (q *Queue) CountChunks(ctx context.Context, notePath string) (int, error) {
	var n int
	err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chunks WHERE note_path = ?`, notePath).Scan(&n)
	return n, err
}

// DeleteChunksByNote removes all stored chunks for a note.
func (q *Queue) DeleteChunksByNote(ctx context.Context, notePath string) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM chunks WHERE note_path = ?`, notePath)
	return err
}

// RemoveNote purges every index trace of a note — FTS row, chunk
// embeddings, and entity rows — in one transaction. Idempotent: removing
// a note that was never indexed is a no-op.
func (q *Queue) RemoveNote(ctx context.Context, notePath string) error {
	const maxRetries = constants.SQLiteMaxRetries
	var lastErr error

	for range maxRetries {
		tx, err := q.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM notes_fts WHERE note_path = ?`, notePath); err != nil && !isFTSErr(err) {
			_ = tx.Rollback()
			lastErr = err
			if isLockError(err) {
				time.Sleep(constants.SQLiteRetrySleep)
				continue
			}
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM chunks WHERE note_path = ?`, notePath); err != nil {
			_ = tx.Rollback()
			lastErr = err
			if isLockError(err) {
				time.Sleep(constants.SQLiteRetrySleep)
				continue
			}
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM entities WHERE note_path = ?`, notePath); err != nil {
			_ = tx.Rollback()
			lastErr = err
			if isLockError(err) {
				time.Sleep(constants.SQLiteRetrySleep)
				continue
			}
			return err
		}
		if err := tx.Commit(); err != nil {
			lastErr = err
			if isLockError(err) {
				time.Sleep(constants.SQLiteRetrySleep)
				continue
			}
			return err
		}
		return nil
	}
	return lastErr
}

// NoteEntities holds enrichment entities extracted from a note. Mirrored
// from the ingest package's Entities to avoid an import cycle.
type NoteEntities struct {
	People  []string
	Amounts []string
	Dates   []string
	// ResolvedDates is index-aligned with Dates: the absolute date for
	// relative references ("tomorrow" -> 2026-08-27), empty when a date
	// could not be resolved.
	ResolvedDates []string
	Places        []string
	Orgs          []string
	URLs          []string
}

// enrichmentEntityTypes are the entity types owned by SaveEntities.
// IndexNote separately manages 'title' and 'tag' rows in the same table;
// scoping deletes to this list keeps the two writers independent.
var enrichmentEntityTypes = []string{"person", "amount", "date", "place", "org", "url"}

func (e *NoteEntities) rows() []entityRow {
	rows := make([]entityRow, 0, 16)
	add := func(typ string, vals, resolved []string) {
		for i, v := range vals {
			if v = strings.TrimSpace(v); v != "" {
				var res string
				if typ == "date" && i < len(resolved) {
					res = strings.TrimSpace(resolved[i])
				}
				rows = append(rows, entityRow{typ: typ, value: v, resolved: res})
			}
		}
	}
	add("person", e.People, nil)
	add("amount", e.Amounts, nil)
	add("date", e.Dates, e.ResolvedDates)
	add("place", e.Places, nil)
	add("org", e.Orgs, nil)
	add("url", e.URLs, nil)
	return rows
}

type entityRow struct {
	typ      string
	value    string
	resolved string
}

// deleteEnrichmentEntities removes only the enrichment rows for a note,
// leaving the title and tag rows managed by IndexNote untouched.
func (q *Queue) deleteEnrichmentEntities(ctx context.Context, notePath string) error {
	placeholders := strings.TrimSuffix(
		strings.Repeat("?,", len(enrichmentEntityTypes)), ",")
	args := make([]any, 0, len(enrichmentEntityTypes)+1)
	args = append(args, notePath)
	for _, typ := range enrichmentEntityTypes {
		args = append(args, typ)
	}
	_, err := q.db.ExecContext(ctx,
		`DELETE FROM entities WHERE note_path = ? AND entity_type IN (`+placeholders+`)`,
		args...)
	return err
}

// deleteIndexedEntities removes only the title and tag rows a note-index
// rewrite owns, leaving enrichment entities untouched.
func (q *Queue) deleteIndexedEntities(ctx context.Context, notePath string) error {
	_, err := q.db.ExecContext(ctx,
		`DELETE FROM entities WHERE note_path = ? AND entity_type IN ('title', 'tag')`,
		notePath)
	return err
}

// SaveEntities stores a note's enrichment entities, replacing any previous
// ones for that note. Title and tag rows written by IndexNote are preserved.
func (q *Queue) SaveEntities(ctx context.Context, notePath string, ents NoteEntities) error {
	const maxRetries = constants.SQLiteMaxRetries
	var lastErr error

	rows := ents.rows()
	now := time.Now().UTC().Format(time.RFC3339)

	for attempt := range maxRetries {
		if err := q.deleteEnrichmentEntities(ctx, notePath); err != nil {
			lastErr = err
			if isLockError(err) {
				q.logger.Warn("sqlite locked, retrying save entities",
					"note_path", notePath, "attempt", attempt+1)
				time.Sleep(constants.SQLiteRetrySleep)
				continue
			}
			return err
		}

		ok := true
		for _, r := range rows {
			if _, err := q.db.ExecContext(ctx, `
				INSERT INTO entities (note_path, chunk_idx, entity_type, entity_value, resolved_date, created_at)
				VALUES (?, NULL, ?, ?, ?, ?)`,
				notePath, r.typ, r.value, nullableString(r.resolved), now); err != nil {
				if isLockError(err) {
					lastErr = err
					ok = false
					break
				}
				return err
			}
		}
		if ok {
			return nil
		}
		time.Sleep(constants.SQLiteRetrySleep)
	}
	return lastErr
}

// DeleteEntities removes all enrichment entities for a note.
func (q *Queue) DeleteEntities(ctx context.Context, notePath string) error {
	return q.deleteEnrichmentEntities(ctx, notePath)
}

// CountEntities returns how many rows exist for a note and entity type.
// Useful for tests and vault health reporting.
func (q *Queue) CountEntities(ctx context.Context, notePath, entityType string) (int, error) {
	var n int
	err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM entities WHERE note_path = ? AND entity_type = ?`,
		notePath, entityType).Scan(&n)
	return n, err
}

// ChunkRow is one chunk of a note: its index, text, and embedding.
type ChunkRow struct {
	Idx       int
	Content   string
	Embedding []float32
}

// ReplaceChunks atomically replaces all stored chunks for a note. Either the
// note ends up with exactly the given rows, or its previous chunks survive —
// a partial set is never persisted.
func (q *Queue) ReplaceChunks(ctx context.Context, notePath string, rows []ChunkRow) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM chunks WHERE note_path = ?`, notePath); err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO chunks (note_path, chunk_idx, content, embedding, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			notePath, r.Idx, r.Content, encodeEmbedding(r.Embedding),
			time.Now().UTC().Format(time.RFC3339)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (q *Queue) IndexNote(ctx context.Context, notePath, title, content, tags string) error {
	const maxRetries = constants.SQLiteMaxRetries
	var lastErr error

	// Pre-split tags once outside retry loop
	var tagList []string
	if tags != "" {
		for tag := range strings.SplitSeq(tags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagList = append(tagList, tag)
			}
		}
	}

	for attempt := range maxRetries {
		now := time.Now().UTC().Format(time.RFC3339)

		if err := q.deleteIndexedEntities(ctx, notePath); err != nil {
			lastErr = err
			if isLockError(err) {
				q.logger.Warn("sqlite locked, retrying index note",
					"note_path", notePath,
					"attempt", attempt+1,
				)
				time.Sleep(constants.SQLiteRetrySleep)
				continue
			}
			return err
		}

		if title != "" {
			_, err := q.db.ExecContext(ctx, `
				INSERT OR REPLACE INTO entities (note_path, chunk_idx, entity_type, entity_value, created_at) 
				VALUES (?, NULL, 'title', ?, ?)`,
				notePath, title, now)
			if err != nil {
				lastErr = err
				if isLockError(err) {
					time.Sleep(constants.SQLiteRetrySleep)
					continue
				}
				return err
			}
		}

		if len(tagList) > 0 {
			for _, tag := range tagList {
				_, err := q.db.ExecContext(ctx, `
					INSERT INTO entities (note_path, chunk_idx, entity_type, entity_value, created_at) 
					VALUES (?, NULL, 'tag', ?, ?)`,
					notePath, tag, now)
				if err != nil {
					lastErr = err
					if isLockError(err) {
						time.Sleep(constants.SQLiteRetrySleep)
						break
					}
					return err
				}
			}
			if isLockError(lastErr) {
				continue
			}
		}

		_, err := q.db.ExecContext(ctx, `INSERT INTO notes_fts (note_path, content, title, tags) VALUES (?, ?, ?, ?)`, notePath, content, title, tags)
		if err != nil && isFTSErr(err) {
			return nil
		}
		if err != nil {
			lastErr = err
			if isLockError(err) {
				time.Sleep(constants.SQLiteRetrySleep)
				continue
			}
			return err
		}
		return nil
	}
	return lastErr
}

func (q *Queue) UpdateNoteIndex(ctx context.Context, notePath, title, content, tags string) error {
	const maxRetries = constants.SQLiteMaxRetries
	var lastErr error

	// Pre-split tags once outside retry loop
	var tagList []string
	if tags != "" {
		for tag := range strings.SplitSeq(tags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagList = append(tagList, tag)
			}
		}
	}

	for attempt := range maxRetries {
		now := time.Now().UTC().Format(time.RFC3339)

		if err := q.deleteIndexedEntities(ctx, notePath); err != nil {
			lastErr = err
			if isLockError(err) {
				q.logger.Warn("sqlite locked, retrying update note index",
					"note_path", notePath,
					"attempt", attempt+1,
				)
				time.Sleep(constants.SQLiteRetrySleep)
				continue
			}
			return err
		}

		if title != "" {
			_, err := q.db.ExecContext(ctx, `
				INSERT OR REPLACE INTO entities (note_path, chunk_idx, entity_type, entity_value, created_at) 
				VALUES (?, NULL, 'title', ?, ?)`,
				notePath, title, now)
			if err != nil {
				lastErr = err
				if isLockError(err) {
					time.Sleep(constants.SQLiteRetrySleep)
					continue
				}
				return err
			}
		}

		if len(tagList) > 0 {
			for _, tag := range tagList {
				_, err := q.db.ExecContext(ctx, `
					INSERT INTO entities (note_path, chunk_idx, entity_type, entity_value, created_at) 
					VALUES (?, NULL, 'tag', ?, ?)`,
					notePath, tag, now)
				if err != nil {
					lastErr = err
					if isLockError(err) {
						time.Sleep(constants.SQLiteRetrySleep)
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
				time.Sleep(constants.SQLiteRetrySleep)
				continue
			}
			return err
		}
		_, err := q.db.ExecContext(ctx, `INSERT INTO notes_fts (note_path, content, title, tags) VALUES (?, ?, ?, ?)`, notePath, content, title, tags)
		if err != nil && isFTSErr(err) {
			return nil
		}
		if err != nil {
			lastErr = err
			if isLockError(err) {
				time.Sleep(constants.SQLiteRetrySleep)
				continue
			}
			return err
		}
		return nil
	}
	return lastErr
}

func (q *Queue) DeleteFromIndex(ctx context.Context, notePath string) error {
	const maxRetries = constants.SQLiteMaxRetries
	var lastErr error

	for attempt := range maxRetries {
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
				time.Sleep(constants.SQLiteRetrySleep)
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

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
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
	for i := range n {
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
	`).Scan(&count)
	return count, err
}

func (q *Queue) CountNotesSince(ctx context.Context, since time.Time) (int, error) {
	var count int
	err := q.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT note_path) FROM chunks
		WHERE datetime(created_at) >= datetime(?)
	`, since.UTC().Format("2006-01-02T15:04:05")).Scan(&count)
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

	result := make([]TagCount, 0, limit)
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

	result := make([]PersonCount, 0, limit)
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
		SELECT DISTINCT DATE(datetime(created_at, 'localtime')) as capture_date
		FROM jobs
		WHERE status = 'done'
		ORDER BY capture_date DESC
		`)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	dates := make([]time.Time, 0, 100) // Pre-allocate with reasonable capacity
	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err != nil {
			return 0, 0, err
		}
		t, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
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

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

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

	if len(dates) > 0 && (dates[0].Equal(today) || dates[0].Equal(today.AddDate(0, 0, -1))) {
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

	tags := make([]string, 0, 10) // Typical max tags per note
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

// BatchGetNoteTags returns a map of note_path -> []tags for multiple notes in a single query.
func (q *Queue) BatchGetNoteTags(ctx context.Context, notePaths []string) (map[string][]string, error) {
	result := make(map[string][]string)
	if len(notePaths) == 0 {
		return result, nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(notePaths))
	args := make([]any, len(notePaths))
	for i, path := range notePaths {
		placeholders[i] = "?"
		args[i] = path
	}

	query := fmt.Sprintf(`
		SELECT note_path, entity_value
		FROM entities
		WHERE entity_type = 'tag' AND note_path IN (%s)
		ORDER BY note_path, entity_value`,
		strings.Join(placeholders, ","))

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var notePath, tag string
		if err := rows.Scan(&notePath, &tag); err != nil {
			return nil, err
		}
		result[notePath] = append(result[notePath], tag)
	}
	return result, rows.Err()
}

// BatchGetNoteTitles returns a map of note_path -> title for multiple notes in a single query.
func (q *Queue) BatchGetNoteTitles(ctx context.Context, notePaths []string) (map[string]string, error) {
	result := make(map[string]string)
	if len(notePaths) == 0 {
		return result, nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(notePaths))
	args := make([]any, len(notePaths))
	for i, path := range notePaths {
		placeholders[i] = "?"
		args[i] = path
	}

	query := fmt.Sprintf(`
		SELECT note_path, entity_value
		FROM entities
		WHERE entity_type = 'title' AND note_path IN (%s)`,
		strings.Join(placeholders, ","))

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var notePath, title string
		if err := rows.Scan(&notePath, &title); err != nil {
			return nil, err
		}
		result[notePath] = title
	}
	return result, rows.Err()
}

// ── New stats functions ──

func (q *Queue) GetHourlyBreakdown(ctx context.Context, dateStr string) ([24]int, error) {
	var result [24]int
	rows, err := q.db.QueryContext(ctx, `
		SELECT CAST(strftime('%H', datetime(created_at, 'localtime')) AS INTEGER) as hour,
		       COUNT(DISTINCT note_path)
		FROM chunks
		WHERE DATE(datetime(created_at, 'localtime')) = ?
		GROUP BY hour
	`, dateStr)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		var hour, count int
		if err := rows.Scan(&hour, &count); err != nil {
			return result, err
		}
		if hour >= 0 && hour < 24 {
			result[hour] = count
		}
	}
	return result, rows.Err()
}

func (q *Queue) GetLast7Days(ctx context.Context) ([7]int, error) {
	var result [7]int
	today := time.Now()
	startDate := today.AddDate(0, 0, -6).Format("2006-01-02")

	rows, err := q.db.QueryContext(ctx, `
		SELECT DATE(datetime(created_at, 'localtime')) as day,
		       COUNT(DISTINCT note_path)
		FROM chunks
		WHERE DATE(datetime(created_at, 'localtime')) >= ?
		GROUP BY day
	`, startDate)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	dayCounts := make(map[string]int)
	for rows.Next() {
		var day string
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			return result, err
		}
		dayCounts[day] = count
	}
	if err := rows.Err(); err != nil {
		return result, err
	}

	// Fill array from oldest (index 0) to today (index 6)
	for i := range 7 {
		day := today.AddDate(0, 0, i-6).Format("2006-01-02")
		result[i] = dayCounts[day]
	}
	return result, nil
}

func (q *Queue) GetLastCapture(ctx context.Context) (string, error) {
	var lastCapture sql.NullString
	err := q.db.QueryRowContext(ctx, `
		SELECT MAX(datetime(created_at, "localtime")) FROM chunks`).Scan(&lastCapture)
	if err != nil {
		return "", err
	}
	if lastCapture.Valid {
		return lastCapture.String, nil
	}
	return "", nil
}

func (q *Queue) GetAvgPerDay(ctx context.Context, days int) (float64, error) {
	if days <= 0 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	var count int
	err := q.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT note_path) FROM chunks
		WHERE DATE(datetime(created_at, 'localtime')) >= ?
	`, since).Scan(&count)
	if err != nil {
		return 0, err
	}
	return float64(count) / float64(days), nil
}

func (q *Queue) GetThisWeekStreak(ctx context.Context) ([7]bool, error) {
	var result [7]bool
	today := time.Now()
	startDate := today.AddDate(0, 0, -6).Format("2006-01-02")

	rows, err := q.db.QueryContext(ctx, `
		SELECT DISTINCT DATE(datetime(created_at, 'localtime')) as day
		FROM jobs
		WHERE status = 'done'
		AND DATE(datetime(created_at, 'localtime')) >= ?
	`, startDate)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	daysWithData := make(map[string]bool)
	for rows.Next() {
		var day string
		if err := rows.Scan(&day); err != nil {
			return result, err
		}
		daysWithData[day] = true
	}
	if err := rows.Err(); err != nil {
		return result, err
	}

	// Fill array from oldest (index 0) to today (index 6)
	for i := range 7 {
		day := today.AddDate(0, 0, i-6).Format("2006-01-02")
		result[i] = daysWithData[day]
	}
	return result, nil
}

func (q *Queue) GetBestStreak(ctx context.Context) (int, error) {
	var best int
	err := q.db.QueryRowContext(ctx, `
		SELECT value FROM stats_cache WHERE key = 'best_streak'
	`).Scan(&best)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return best, nil
}

func (q *Queue) UpdateBestStreak(ctx context.Context, current int) error {
	best, err := q.GetBestStreak(ctx)
	if err != nil {
		return err
	}
	if current > best {
		_, err = q.db.ExecContext(ctx, `
			INSERT OR REPLACE INTO stats_cache (key, value, updated_at)
			VALUES ('best_streak', ?, ?)
		`, fmt.Sprintf("%d", current), time.Now().UTC().Format(time.RFC3339))
		return err
	}
	return nil
}

// ── Stats cache functions ──

type StatsCacheEntry struct {
	Date      string `json:"date"`
	Stats     any    `json:"stats"`
	UpdatedAt string `json:"updated_at"`
}

func (q *Queue) SaveStatsCache(ctx context.Context, stats any) error {
	now := time.Now().UTC()
	entry := StatsCacheEntry{
		Date:      now.Format("2006-01-02"),
		Stats:     stats,
		UpdatedAt: now.Format(time.RFC3339),
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	const maxRetries = constants.SQLiteMaxRetries
	for attempt := range maxRetries {
		_, err = q.db.ExecContext(ctx, `
			INSERT OR REPLACE INTO stats_cache (key, value, updated_at)
			VALUES ('stats', ?, ?)
		`, string(data), now.Format(time.RFC3339))
		if err == nil {
			return nil
		}
		if isLockError(err) {
			q.logger.Warn("sqlite locked, retrying save stats cache",
				"attempt", attempt+1,
				"error", err,
			)
			time.Sleep(constants.SQLiteRetrySleep)
			continue
		}
		break
	}
	return err
}

func (q *Queue) LoadStatsCache(ctx context.Context) (string, error) {
	var value string
	err := q.db.QueryRowContext(ctx, `
		SELECT value FROM stats_cache WHERE key = 'stats'
	`).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}

	// Unwrap the cache entry
	var entry StatsCacheEntry
	if err := json.Unmarshal([]byte(value), &entry); err != nil {
		// Corrupted cache — delete
		if _, err := q.db.ExecContext(ctx, `DELETE FROM stats_cache WHERE key = 'stats'`); err != nil {
			q.logger.Warn("failed to delete corrupted cache", "error", err)
		}
		return "", nil
	}

	// Check date boundary
	today := time.Now().UTC().Format("2006-01-02")
	if entry.Date != today {
		return "", nil // stale date — triggers recompute
	}

	// Return just the stats portion, not the wrapper
	statsData, err := json.Marshal(entry.Stats)
	if err != nil {
		return "", nil
	}
	return string(statsData), nil
}

// ── Stats response types ──

type StreakStats struct {
	Current         int     `json:"current"`
	Best            int     `json:"best"`
	NextMilestone   int     `json:"next_milestone"`
	DaysToMilestone int     `json:"days_to_milestone"`
	ThisWeek        [7]bool `json:"this_week"`
}

type TodayStats struct {
	Count     int     `json:"count"`
	ByHour    [24]int `json:"by_hour"`
	AvgPerDay float64 `json:"avg_per_day"`
}

type VaultStats struct {
	TotalNotes    int    `json:"total_notes"`
	TodayDelta    int    `json:"today_delta"`
	LastCaptureAt string `json:"last_capture_at"`
	Last7Days     [7]int `json:"last_7_days"`
}

type StatsResponse struct {
	Streak StreakStats `json:"streak"`
	Today  TodayStats  `json:"today"`
	Vault  VaultStats  `json:"vault"`
}

func nextMilestone(current, best int) (int, int) {
	fixed := constants.Milestones

	milestones := []int{}
	if best > 0 {
		milestones = append(milestones, best)
	}
	for _, m := range fixed {
		if m > best {
			milestones = append(milestones, m)
		}
	}
	if best >= 365 {
		next100 := ((best / 100) + 1) * 100
		milestones = append(milestones, next100)
	}

	for _, m := range milestones {
		if m > current {
			return m, m - current
		}
	}

	next := ((current / 100) + 1) * 100
	return next, next - current
}

func (q *Queue) RecomputeStats(ctx context.Context) (*StatsResponse, error) {
	now := time.Now()
	todayStr := now.Format("2006-01-02")
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	midnightUtc := midnight.UTC()

	todayCount, err := q.CountNotesSince(ctx, midnightUtc)
	if err != nil {
		return nil, err
	}

	byHour, err := q.GetHourlyBreakdown(ctx, todayStr)
	if err != nil {
		return nil, err
	}

	avgPerDay, err := q.GetAvgPerDay(ctx, 30)
	if err != nil {
		return nil, err
	}

	totalNotes, err := q.CountNotes(ctx)
	if err != nil {
		return nil, err
	}

	lastCaptureAt, err := q.GetLastCapture(ctx)
	if err != nil {
		return nil, err
	}

	last7Days, err := q.GetLast7Days(ctx)
	if err != nil {
		return nil, err
	}

	currentStreak, _, err := q.GetStreaks(ctx)
	if err != nil {
		return nil, err
	}

	if err := q.UpdateBestStreak(ctx, currentStreak); err != nil {
		q.logger.Warn("failed to update best streak", "error", err)
	}

	bestStreak, err := q.GetBestStreak(ctx)
	if err != nil {
		return nil, err
	}

	thisWeek, err := q.GetThisWeekStreak(ctx)
	if err != nil {
		return nil, err
	}

	nextMs, daysToMs := nextMilestone(currentStreak, bestStreak)

	stats := &StatsResponse{
		Streak: StreakStats{
			Current:         currentStreak,
			Best:            bestStreak,
			NextMilestone:   nextMs,
			DaysToMilestone: daysToMs,
			ThisWeek:        thisWeek,
		},
		Today: TodayStats{
			Count:     todayCount,
			ByHour:    byHour,
			AvgPerDay: avgPerDay,
		},
		Vault: VaultStats{
			TotalNotes:    totalNotes,
			TodayDelta:    todayCount,
			LastCaptureAt: lastCaptureAt,
			Last7Days:     last7Days,
		},
	}

	if err := q.SaveStatsCache(ctx, stats); err != nil {
		q.logger.Warn("failed to save stats cache", "error", err)
	}

	return stats, nil
}

// EntityMatch is a note that shares an entity with the queried note.
type EntityMatch struct {
	NotePath  string
	CreatedAt time.Time
	Excerpt   string
}

// GetEntitiesByNote returns a note's stored values for one entity type.
func (q *Queue) GetEntitiesByNote(ctx context.Context, notePath, entityType string) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT entity_value FROM entities
		WHERE note_path = ? AND entity_type = ?
		ORDER BY id`,
		notePath, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			continue
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

// GetNotesByEntity returns notes (created before cutoff) that carry the
// given entity, most recent first. Only notes whose jobs exist are matched.
func (q *Queue) GetNotesByEntity(ctx context.Context, entityValue, entityType string, cutoff time.Time) ([]EntityMatch, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT e.note_path, j.created_at
		FROM entities e
		JOIN jobs j ON e.note_path = j.note_path
		WHERE LOWER(e.entity_value) = LOWER(?) AND e.entity_type = ? AND j.created_at < ?
		GROUP BY e.note_path
		ORDER BY j.created_at DESC
		LIMIT 10`,
		entityValue, entityType, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []EntityMatch
	for rows.Next() {
		var path, createdAt string
		if err := rows.Scan(&path, &createdAt); err != nil {
			continue
		}
		t, _ := time.Parse(time.RFC3339, createdAt)
		excerpt, _ := q.GetNoteExcerpt(ctx, path)
		matches = append(matches, EntityMatch{NotePath: path, CreatedAt: t, Excerpt: excerpt})
	}
	return matches, rows.Err()
}

// CountNotesByEntity counts OTHER in-window notes carrying the entity —
// used for "Alice also appears in N other notes" labels.
func (q *Queue) CountNotesByEntity(ctx context.Context, entityValue, entityType string, cutoff time.Time, excludePath string) (int, error) {
	var n int
	err := q.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT e.note_path)
		FROM entities e
		JOIN jobs j ON e.note_path = j.note_path
		WHERE LOWER(e.entity_value) = LOWER(?) AND e.entity_type = ? AND j.created_at < ? AND e.note_path != ?`,
		entityValue, entityType, cutoff.UTC().Format(time.RFC3339), excludePath).Scan(&n)
	return n, err
}

// GetChunkEmbeddingForNote returns the note's first stored chunk embedding.
// ok=false when the note has no chunks (e.g. captured before v1.1).
func (q *Queue) GetChunkEmbeddingForNote(ctx context.Context, notePath string) ([]float32, bool, error) {
	var blob []byte
	err := q.db.QueryRowContext(ctx,
		`SELECT embedding FROM chunks WHERE note_path = ? ORDER BY chunk_idx LIMIT 1`,
		notePath).Scan(&blob)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return decodeEmbedding(blob), true, nil
}

// GetNoteExcerpt returns the best available short text for a note:
// its lowest-index chunk content, else the head of the job content.
func (q *Queue) GetNoteExcerpt(ctx context.Context, notePath string) (string, error) {
	var content string
	err := q.db.QueryRowContext(ctx,
		`SELECT content FROM chunks WHERE note_path = ? ORDER BY chunk_idx LIMIT 1`,
		notePath).Scan(&content)
	if err == nil && content != "" {
		return truncateExcerpt(content), nil
	}
	err = q.db.QueryRowContext(ctx,
		`SELECT content FROM jobs WHERE note_path = ? ORDER BY created_at DESC LIMIT 1`,
		notePath).Scan(&content)
	if err != nil {
		return "", err
	}
	return truncateExcerpt(content), nil
}

func truncateExcerpt(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

// UpdateJobResult stores a JSON payload on a job row (connections output).
func (q *Queue) UpdateJobResult(ctx context.Context, jobID string, result json.RawMessage) error {
	const maxRetries = constants.SQLiteMaxRetries
	var lastErr error
	for range maxRetries {
		_, err := q.db.ExecContext(ctx,
			`UPDATE jobs SET result = ? WHERE id = ?`, string(result), jobID)
		if err == nil {
			return nil
		}
		lastErr = err
		if isLockError(err) {
			time.Sleep(constants.SQLiteRetrySleep)
			continue
		}
		return err
	}
	return lastErr
}

// LinkConnectionsJob records on the ingest job which connections job
// carries its results.
func (q *Queue) LinkConnectionsJob(ctx context.Context, ingestJobID, connJobID string) error {
	const maxRetries = constants.SQLiteMaxRetries
	var lastErr error
	for range maxRetries {
		_, err := q.db.ExecContext(ctx,
			`UPDATE jobs SET connections_job_id = ? WHERE id = ?`, connJobID, ingestJobID)
		if err == nil {
			return nil
		}
		lastErr = err
		if isLockError(err) {
			time.Sleep(constants.SQLiteRetrySleep)
			continue
		}
		return err
	}
	return lastErr
}

// RawChunkMatch is one older note's best chunk under a raw-cosine query,
// reported without any score normalization.
type RawChunkMatch struct {
	NotePath  string
	CreatedAt string
	Content   string
	Score     float64
}

// TopSimilarChunks finds notes created before cutoff whose best chunk has
// raw cosine similarity >= minScore to the query embedding, excluding the
// query note itself. Scores are true cosines — no rescaling — so callers
// can treat them as confidence values.
func (q *Queue) TopSimilarChunks(ctx context.Context, embedding []float32, limit int, minScore float64, cutoff time.Time, excludePath string) ([]RawChunkMatch, error) {
	if len(embedding) == 0 {
		return nil, nil
	}

	best := make(map[string]RawChunkMatch)
	offset := 0
	for {
		rows, err := q.db.QueryContext(ctx, `
			SELECT c.note_path, j.created_at, c.content, c.embedding
			FROM chunks c
			JOIN jobs j ON c.note_path = j.note_path
			WHERE c.embedding IS NOT NULL AND j.created_at < ? AND c.note_path != ?
			LIMIT ? OFFSET ?`,
			cutoff.UTC().Format(time.RFC3339), excludePath, batchSize, offset)
		if err != nil {
			return nil, err
		}

		hasRows := false
		for rows.Next() {
			hasRows = true
			var notePath, createdAt, content string
			var blob []byte
			if err := rows.Scan(&notePath, &createdAt, &content, &blob); err != nil {
				rows.Close()
				return nil, err
			}
			score := cosine(embedding, decodeEmbedding(blob))
			if score < minScore {
				continue
			}
			if cur, ok := best[notePath]; !ok || score > cur.Score {
				best[notePath] = RawChunkMatch{NotePath: notePath, CreatedAt: createdAt, Content: content, Score: score}
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

	out := make([]RawChunkMatch, 0, len(best))
	for _, m := range best {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// GetEntityGlossary returns distinct person/org values across the vault,
// most frequent first, capped — the "who does the user know" list.
func (q *Queue) GetEntityGlossary(ctx context.Context, limit int) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT entity_value, COUNT(*) AS n FROM entities
		WHERE entity_type IN ('person','org')
		GROUP BY LOWER(entity_value)
		ORDER BY n DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		var n int
		if err := rows.Scan(&v, &n); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetStat reads a stats_cache marker; ok=false when absent.
func (q *Queue) GetStat(ctx context.Context, key string) (string, bool, error) {
	var v string
	err := q.db.QueryRowContext(ctx,
		`SELECT value FROM stats_cache WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return v, err == nil, err
}

// SetStat writes a stats_cache marker.
func (q *Queue) SetStat(ctx context.Context, key, value string) error {
	_, err := q.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO stats_cache (key, value, updated_at) VALUES (?, ?, ?)`,
		key, value, time.Now().UTC().Format(time.RFC3339))
	return err
}

// CountPersonsSince counts distinct person entities created after t.
func (q *Queue) CountPersonsSince(ctx context.Context, t time.Time) (int, error) {
	var n int
	err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT entity_value) FROM entities
		 WHERE entity_type = 'person' AND created_at > ?`,
		t.UTC().Format(time.RFC3339)).Scan(&n)
	return n, err
}

// GetReferencedMedia returns the distinct media file paths referenced by
// captured notes (jobs.source_file).
func (q *Queue) GetReferencedMedia(ctx context.Context) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT DISTINCT source_file FROM jobs
		WHERE source_file IS NOT NULL AND source_file <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			continue
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
