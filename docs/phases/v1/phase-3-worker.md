# Phase 3: Worker

> Background job processing, ingest pipeline. Updated: 2026-03-17

## Goals

- [ ] Worker pool with configurable concurrency
- [ ] Crash recovery (reset stuck jobs)
- [ ] Text ingest (tags, summary)
- [ ] Image ingest (LLM description, OCR)
- [ ] Article ingest (scrape, summarize)
- [ ] Retry logic (exponential backoff)

## Directory Structure

```
internal/
├── worker/
│   └── worker.go
└── ingest/
    ├── text.go
    ├── image.go
    └── article.go
```

## Step 3.1: Worker Pool

**File:** `internal/worker/worker.go`

### Requirements

- Configurable concurrency (`worker.max_workers`)
- Single goroutine per worker, jobs processed serially
- Exponential backoff on failure
- Max 3 retries then permanently failed
- On permanent failure: move note to `.khayal-trash/`, move media to trash, mark failed in DB
- On startup: reset stuck `processing` jobs

### Config

```go
type WorkerConfig struct {
    MaxWorkers    int    `yaml:"max_workers"`    // default: 1
    MaxRetries   int    `yaml:"max_retries"`   // default: 3
    RetryBackoff string `yaml:"retry_backoff"` // "immediate" | "fixed" | "exponential"
}
```

### Structure

```go
type Worker struct {
    queue      *queue.Queue
    vault      *vault.Writer
    llm        llm.LLM
    config     WorkerConfig
    jobs       chan string // Job IDs to process
    wg         sync.WaitGroup
    running    atomic.Bool
}

### Job Status Flow

Jobs transition through these states:

```
pending → queued → processing → done/failed
   ↑                  ↓
   └──────────────────┘ (on retry)
```

- **pending**: Job created, waiting to be fetched
- **queued**: Fetched from database, waiting in channel for worker
- **processing**: Worker actively processing the job
- **done**: Job completed successfully
- **failed**: Job permanently failed (max retries exceeded)

### Atomic Fetch+Lock Pattern

To prevent duplicate job processing, we use an atomic fetch+lock operation:

```go
func (q *Queue) FetchAndLockPendingJobs(ctx context.Context, limit int) ([]Job, error) {
    // Single atomic operation: UPDATE + RETURNING
    // Prevents race conditions where multiple workers fetch the same job
    rows, err := q.db.QueryContext(ctx, `
        UPDATE jobs 
        SET status = 'queued'
        WHERE id IN (
            SELECT id FROM jobs 
            WHERE status = 'pending' 
            ORDER BY created_at ASC 
            LIMIT ?
        )
        RETURNING id, type, status, note_path, ...`,
        limit)
    // ... scan and return jobs
}
```

**Benefits:**
- Only one worker can fetch each job
- Status is immediately set to 'queued'
- No duplicate processing

**Crash Recovery:**
On startup, `ResetStuckJobs()` resets both `processing` and `queued` jobs to `pending`:

```go
func (q *Queue) ResetStuckJobs(ctx context.Context) error {
    _, err := q.db.ExecContext(ctx, 
        `UPDATE jobs SET status = 'pending' WHERE status = 'processing' OR status = 'queued'`)
    return err
}
```

### Worker Constructor

```go
func NewWorker(cfg WorkerConfig, q *queue.Queue, v *vault.Writer, l llm.LLM) *Worker {
    return &Worker{
        config: cfg,
        queue:  q,
        vault:  v,
        llm:    l,
        jobs:   make(chan string, 1000),  // Buffer for job IDs
    }
}
```

func (w *Worker) Start() {
    if w.running.Swap(true) {
        return
    }
    
    // Crash recovery: reset stuck jobs
    if err := w.queue.ResetStuckJobs(); err != nil {
        log.Error().Err(err).Msg("failed to reset stuck jobs")
    }
    
    // Start worker goroutines
    for i := 0; i < w.config.MaxWorkers; i++ {
        w.wg.Add(1)
        go w.workerLoop(i)
    }
    
    // Start job fetcher
    go w.jobFetcher()
    
    log.Info().Int("workers", w.config.MaxWorkers).Msg("worker pool started")
}

func (w *Worker) Stop() {
    w.running.Store(false)
    close(w.jobs)
    w.wg.Wait()
    log.Info().Msg("worker pool stopped")
}

func (w *Worker) jobFetcher() {
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    
    for w.running.Load() {
        select {
        case <-ticker.C:
            // Calculate how many jobs we can fetch based on channel capacity
            available := cap(w.jobs) - len(w.jobs)
            if available <= 0 {
                continue
            }
            
            // Fetch and lock jobs atomically - prevents duplicate processing
            // Uses UPDATE...SET status='queued' WHERE status='pending' RETURNING *
            jobs, err := w.queue.FetchAndLockPendingJobs(ctx, available)
            if err != nil {
                log.Error().Err(err).Msg("failed to fetch pending jobs")
                continue
            }
            for _, job := range jobs {
                w.jobs <- job.ID  // Safe - we only fetched what fits
            }
        }
    }
}

func (w *Worker) workerLoop(id int) {
    defer w.wg.Done()
    
    for jobID := range w.jobs {
        if !w.running.Load() {
            break
        }
        
        w.processJob(jobID)
    }
}
```

### Process Job

```go
func (w *Worker) processJob(jobID string) {
    // 120-second timeout per job
    ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
    defer cancel()
    
    job, err := w.queue.GetJob(ctx, jobID)
    if err != nil {
        log.Error().Str("job", jobID).Err(err).Msg("failed to get job")
        return
    }
    
    // Mark as processing (job was previously 'queued')
    if err := w.queue.UpdateJobStatus(ctx, jobID, "processing"); err != nil {
        log.Error().Str("job", jobID).Err(err).Msg("failed to update job status")
        return
    }
    
    // CRITICAL: Process FIRST, write vault LAST
    // If any step fails, don't write anything to vault
    // Job stays pending for user to retry or discard
    
    var notePath string
    
    switch job.Type {
    case "text":
        notePath, err = w.ingestText(ctx, job)
    case "image":
        notePath, err = w.ingestImage(ctx, job)
    case "article":
        notePath, err = w.ingestArticle(job)
    default:
        err = fmt.Errorf("unknown job type: %s", job.Type)
    }
    
    if err != nil {
        // Processing failed - DON'T write to vault
        // Keep job pending for user to retry or discard
        w.handleFailure(job, err)
        return
    }
    
    // All processing succeeded - NOW write to vault
    job.NotePath = notePath
    job.Status = "done"
    
    now := time.Now()
    job.ProcessedAt = &now
    
    if err := w.queue.UpdateJob(job); err != nil {
        log.Error().Str("job", jobID).Err(err).Msg("failed to update job")
    }
    
    log.Info().Str("job", jobID).Str("type", job.Type).Msg("job completed")
}
```

### Retry Logic

```go
func (w *Worker) handleFailure(job *queue.Job, err error) {
    job.Retries++
    job.Error = err.Error()
    
    // CRITICAL: Don't write to vault on failure
    // Don't delete media - user might want to retry
    
    // After max retries, mark as failed but keep data
    // User can manually retry or discard
    if job.Retries >= w.config.MaxRetries {
        job.Status = "failed"
        
        log.Error().
            Str("job", job.ID).
            Int("retries", job.Retries).
            Err(err).
            Msg("job permanently failed - user can retry or discard")
    } else {
        // Keep pending for retry
        job.Status = "pending"
        delay := w.calculateBackoff(job.Retries)
        
        log.Warn().
            Str("job", job.ID).
            Int("retry", job.Retries).
            Dur("delay", delay).
            Err(err).
            Msg("job failed - will retry later")
        
        time.Sleep(delay)
    }
    
    // IMPORTANT: Don't write note_path - it doesn't exist yet
    // Only update error message and retry count
    w.queue.UpdateJob(job)
}

func (w *Worker) calculateBackoff(retry int) time.Duration {
    switch w.config.RetryBackoff {
    case "immediate":
        return 0
    case "fixed":
        return 5 * time.Second
    case "exponential":
        fallthrough
    default:
        return time.Duration(math.Pow(2, float64(retry))) * time.Second
    }
}
```

### Safety-First Principle

**Never write to vault until ALL processing succeeds.**

```
Job created → Process with LLM → All succeeded → Write to vault → Mark done
                 ↓ (any failure)
              Keep job pending → Don't write vault → User can retry/discard
```

**Why this approach?**
- No orphaned/incomplete notes in vault
- User can retry failed jobs manually
- User can discard when ready
- No data loss from failed processing
- Clear failure state for debugging

## Step 3.2: Text Ingest

**File:** `internal/ingest/text.go`

### Requirements

- Extract tags using LLM
- Generate summary
- Return note data (don't write to vault)
- Generate embedding for search

### Process

1. Send content to LLM for tag extraction, summary, and key ideas (concurrent)
2. Return note data (vault write happens in worker)
3. Generate embedding and save to DB

### IMPORTANT: Don't Write Vault Here

The ingest functions return note data. The worker writes to vault ONLY after ALL processing succeeds.

### Dependency: golang.org/x/sync

Add to `go.mod`:
```
require golang.org/x/sync v0.10.0
```

```go
import "golang.org/x/sync/errgroup"

// Concurrent LLM calls using errgroup
func (w *Worker) ingestText(job *queue.Job) (string, error) {
    g, _ := errgroup.WithContext(context.Background())
    
    var tags []string
    var summary string
    var keyIdeas []string
    
    // Run all three LLM calls concurrently
    g.Go(func() error {
        var err error
        tags, err = w.llm.ExtractTags(job.Content)
        return err
    })
    
    g.Go(func() error {
        var err error
        summary, err = w.llm.Summarize(job.Content)
        return err
    })
    
    g.Go(func() error {
        var err error
        keyIdeas, err = w.llm.ExtractKeyIdeas(job.Content)
        return err
    })
    
    // Fail fast - if any LLM call fails, the whole job fails
    if err := g.Wait(); err != nil {
        return "", fmt.Errorf("llm extraction failed: %w", err)
    }
    
    // Build title from content (first line or first 100 chars)
    title := extractTitle(job.Content)
    
    // Build note
    now := time.Now()
    note := &vault.Note{
        Metadata: vault.NoteMetadata{
            Created:  job.CreatedAt,
            Updated:  &now,
            Type:     "text",
            Status:   "done",
            Tags:     tags,
            History: []vault.HistoryEvent{
                {At: now, Event: "processed"},
            },
        },
        Title:    title,
        Summary:  summary,
        KeyIdeas: keyIdeas,
        Raw:      job.Content,
    }
    
    // Write note to vault
    notePath, err := w.vault.WriteNote(note)
    if err != nil {
        return "", fmt.Errorf("failed to write note: %w", err)
    }
    
    // Index for FTS5 search
    if err := w.queue.IndexNote(ctx, notePath, title, job.Content, strings.Join(tags, ",")); err != nil {
        return "", fmt.Errorf("failed to index note: %w", err)
    }
    
    // Generate embedding (non-fatal if it fails)
    embedding, embedErr := w.llm.Embed(job.Content)
    if embedErr != nil {
        log.Warn().Err(embedErr).Msg("failed to generate embedding")
        return notePath, nil
    }
    
    if err := w.queue.SaveChunk(ctx, notePath, 0, job.Content, embedding); err != nil {
        return notePath, nil
    }
    
    return notePath, nil
}

func extractTitle(content string) string {
    lines := strings.Split(content, "\n")
    firstLine := strings.TrimSpace(lines[0])
    if len(firstLine) > 100 {
        firstLine = firstLine[:100]
    }
    return firstLine
}
```

## Step 3.3: Image Ingest

**File:** `internal/ingest/image.go`

### Requirements

- Describe image using LLM vision
- Extract tags from description + user context
- Generate embedding for search

### Process

1. Send image to LLM for description (sequential - must happen first)
2. Extract tags from description (concurrent with embedding)
3. Write note and save embedding

```go
func (w *Worker) ingestImage(job *queue.Job) (string, error) {
    // LLM description (must happen first)
    description, err := w.llm.DescribeImage(job.SourceFile)
    if err != nil {
        return "", fmt.Errorf("failed to describe image: %w", err)
    }
    
    // Build context text
    contextText := description
    if job.UserContext != "" {
        contextText = job.UserContext + "\n\n" + description
    }
    
    // Extract tags (can run after description)
    g, _ := errgroup.WithContext(context.Background())
    
    var tags []string
    
    g.Go(func() error {
        var err error
        tags, err = w.llm.ExtractTags(contextText)
        return err
    })
    
    if err := g.Wait(); err != nil {
        return "", fmt.Errorf("failed to extract tags: %w", err)
    }
    
    if tags == nil {
        tags = []string{"image"}
    }
    
    // Build note
    now := time.Now()
    note := &vault.Note{
        Metadata: vault.NoteMetadata{
            Created:     job.CreatedAt,
            Updated:     &now,
            Type:        "image",
            Status:      "done",
            SourceFile:  job.SourceFile,
            UserContext: job.UserContext,
            Tags:        tags,
            History: []vault.HistoryEvent{
                {At: now, Event: "processed"},
            },
        },
        Title: fmt.Sprintf("Image — %s", job.CreatedAt.Format("2006-01-02")),
        Raw:   description,
    }
    
    notePath, err := w.vault.WriteNote(note)
    if err != nil {
        return "", fmt.Errorf("failed to write note: %w", err)
    }
    
    // Index for FTS5 search
    if err := w.queue.IndexNote(ctx, notePath, note.Title, contextText, strings.Join(tags, ",")); err != nil {
        return "", fmt.Errorf("failed to index note: %w", err)
    }
    
    // Generate embedding (non-fatal if it fails)
    embedding, embedErr := w.llm.Embed(contextText)
    if embedErr != nil {
        log.Warn().Err(embedErr).Msg("failed to generate embedding")
        return notePath, nil
    }
    
    if err := w.queue.SaveChunk(ctx, notePath, 0, contextText, embedding); err != nil {
        return notePath, nil
    }
    
    return notePath, nil
}
```

## Step 3.4: Article Ingest

**File:** `internal/ingest/article.go`

### Requirements

- Scrape article content
- Extract title, main content
- Generate summary using LLM (concurrent)
- Update note

### Process

1. Fetch URL
2. Extract title, content
3. Run Summarize, ExtractKeyIdeas, ExtractTags concurrently
4. Write note and generate embedding

```go
func (w *Worker) ingestArticle(job *queue.Job) (string, error) {
    // Fetch article
    title, content, err := w.scrapeArticle(job.SourceURL)
    if err != nil {
        return "", fmt.Errorf("failed to scrape article: %w", err)
    }
    
    combinedContent := title + "\n\n" + content
    
    // Run all LLM calls concurrently
    g, _ := errgroup.WithContext(context.Background())
    
    var summary string
    var keyIdeas []string
    var tags []string
    
    g.Go(func() error {
        var err error
        summary, err = w.llm.Summarize(combinedContent)
        return err
    })
    
    g.Go(func() error {
        var err error
        keyIdeas, err = w.llm.ExtractKeyIdeas(combinedContent)
        return err
    })
    
    g.Go(func() error {
        var err error
        tags, err = w.llm.ExtractTags(combinedContent)
        return err
    })
    
    // Fail fast if any LLM call fails
    if err := g.Wait(); err != nil {
        return "", fmt.Errorf("llm extraction failed: %w", err)
    }
    
    if tags == nil {
        tags = []string{"article"}
    }
    
    // Build note
    now := time.Now()
    note := &vault.Note{
        Metadata: vault.NoteMetadata{
            Created:   job.CreatedAt,
            Updated:   &now,
            Type:      "article",
            Status:    "done",
            SourceURL: job.SourceURL,
            Tags:      tags,
            History: []vault.HistoryEvent{
                {At: now, Event: "processed"},
            },
        },
        Title:    title,
        Summary:  summary,
        KeyIdeas: keyIdeas,
        Raw:      content,
    }
    
    notePath, err := w.vault.WriteNote(note)
    if err != nil {
        return "", fmt.Errorf("failed to write note: %w", err)
    }
    
    // Index for FTS5 search
    if err := w.queue.IndexNote(ctx, notePath, title, combinedContent, strings.Join(tags, ",")); err != nil {
        return "", fmt.Errorf("failed to index note: %w", err)
    }
    
    // Generate embedding from summary + key ideas
    embedContent := title + "\n\n" + summary + "\n\n" + strings.Join(keyIdeas, "\n")
    embedding, embedErr := w.llm.Embed(embedContent)
    if embedErr != nil {
        log.Warn().Err(embedErr).Msg("failed to generate embedding")
        return notePath, nil
    }
    
    if err := w.queue.SaveChunk(ctx, notePath, 0, combinedContent, embedding); err != nil {
        return notePath, nil
    }
    
    return notePath, nil
}
```

## Testing

Write tests for:

- [x] Worker pool startup/shutdown
- [x] Crash recovery
- [x] Job processing (text, image, article)
- [x] Retry logic
- [x] Permanent failure cleanup
- [x] Concurrent LLM calls

```bash
go test ./internal/worker/... -v
go test ./internal/ingest/... -v
```

## Checklist

- [x] Worker pool implementation
- [x] Configurable concurrency
- [x] Job fetcher loop
- [x] Crash recovery
- [x] Text ingest (concurrent LLM calls)
- [x] Image ingest (concurrent LLM calls)
- [x] Article ingest (concurrent LLM calls)
- [x] Vault write happens AFTER all processing succeeds
- [x] Retry with backoff (keep job pending, don't fail)
- [x] Max retries reached → mark failed, user can retry/discard
- [x] Embedding generation (non-fatal)
- [x] Concurrent LLM calls using errgroup
- [x] Tests passing
- [x] go vet clean

## Next Phase

[Phase 4: LLM](phase-4-llm.md)

## Notes

- **Safety-first**: Never write to vault until ALL processing succeeds
- **Concurrent LLM calls**: Use `golang.org/x/sync/errgroup` for parallel execution
- **Fail-fast**: Any LLM call failure fails the entire job (user can retry)
- Processing times (M2 Mac Air):
  - Text: ~5s (was ~15s sequential)
  - Image: ~5s (was ~8s sequential)
  - Article: ~7s (was ~20s sequential)
- Embedding model: nomic-embed-text
- Default workers: 1
- Max retries: 3
- Failed jobs stay in queue - user can retry or discard
