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
- On permanent failure: delete note, delete media, mark failed
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

func NewWorker(cfg WorkerConfig, q *queue.Queue, v *vault.Writer, l llm.LLM) *Worker {
    return &Worker{
        config: cfg,
        queue:  q,
        vault:  v,
        llm:    l,
        jobs:   make(chan string, 100),
    }
}

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
            jobs, err := w.queue.GetPendingJobs(w.config.MaxWorkers)
            if err != nil {
                log.Error().Err(err).Msg("failed to fetch pending jobs")
                continue
            }
            for _, job := range jobs {
                select {
                case w.jobs <- job.ID:
                default:
                    // Channel full, will pick up next tick
                }
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
    job, err := w.queue.GetJob(jobID)
    if err != nil {
        log.Error().Str("job", jobID).Err(err).Msg("failed to get job")
        return
    }
    
    // Mark as processing
    if err := w.queue.UpdateJobStatus(jobID, "processing"); err != nil {
        log.Error().Str("job", jobID).Err(err).Msg("failed to update job status")
        return
    }
    
    var processErr error
    
    switch job.Type {
    case "text":
        processErr = w.ingestText(job)
    case "image":
        processErr = w.ingestImage(job)
    case "article":
        processErr = w.ingestArticle(job)
    default:
        processErr = fmt.Errorf("unknown job type: %s", job.Type)
    }
    
    if processErr != nil {
        w.handleFailure(job, processErr)
        return
    }
    
    // Mark as done
    now := time.Now()
    job.ProcessedAt = &now
    job.Status = "done"
    
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
    
    if job.Retries >= w.config.MaxRetries {
        // Permanent failure
        job.Status = "failed"
        
        // Cleanup: delete note and media
        if job.NotePath != "" {
            w.vault.DeleteNote(job.NotePath)
        }
        if job.SourceFile != "" {
            w.vault.DeleteMedia(job.SourceFile)
        }
        
        log.Error().Str("job", job.ID).Int("retries", job.Retries).Msg("job permanently failed")
    } else {
        // Schedule retry with backoff
        job.Status = "pending"
        delay := w.calculateBackoff(job.Retries)
        
        log.Warn().
            Str("job", job.ID).
            Int("retry", job.Retries).
            Dur("delay", delay).
            Err(err).
            Msg("job failed, scheduling retry")
        
        time.Sleep(delay) // In production, use a proper scheduler
    }
    
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

## Step 3.2: Text Ingest

**File:** `internal/ingest/text.go`

### Requirements

- Extract tags using LLM
- Generate summary
- Update note with results
- Generate embedding for search

### Process

1. Send content to LLM for tag extraction
2. Generate summary
3. Update note with tags, summary, key ideas
4. Generate embedding and save to DB

```go
func (w *Worker) ingestText(job *queue.Job) error {
    // Generate tags
    tags, err := w.llm.ExtractTags(job.Content)
    if err != nil {
        return fmt.Errorf("failed to extract tags: %w", err)
    }
    
    // Generate summary
    summary, err := w.llm.Summarize(job.Content)
    if err != nil {
        return fmt.Errorf("failed to generate summary: %w", err)
    }
    
    // Generate key ideas
    keyIdeas, err := w.llm.ExtractKeyIdeas(job.Content)
    if err != nil {
        return fmt.Errorf("failed to extract key ideas: %w", err)
    }
    
    // Build title from content (first line or first 50 chars)
    title := strings.SplitN(job.Content, "\n")[0]
    if len(title) > 50 {
        title = title[:50] + "..."
    }
    
    // Update note
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
    
    notePath, err := w.vault.WriteNote(note)
    if err != nil {
        return fmt.Errorf("failed to write note: %w", err)
    }
    
    job.NotePath = notePath
    
    // Generate embedding
    embedding, err := w.llm.Embed(job.Content)
    if err != nil {
        log.Warn().Err(err).Msg("failed to generate embedding")
        return nil // Non-fatal
    }
    
    return w.queue.SaveEmbedding(job.ID, w.config.EmbedModel, embedding)
}
```

## Step 3.3: Image Ingest

**File:** `internal/ingest/image.go`

### Requirements

- Describe image using LLM vision
- Run OCR for text extraction
- Update note with description + extracted text

### Process

1. Send image to LLM for description
2. Run OCR on image
3. Update note with results

```go
func (w *Worker) ingestImage(job *queue.Job) error {
    // LLM description
    description, err := w.llm.DescribeImage(job.SourceFile)
    if err != nil {
        return fmt.Errorf("failed to describe image: %w", err)
    }
    
    // OCR
    ocrText, err := w.ocrImage(job.SourceFile)
    if err != nil {
        log.Warn().Err(err).Msg("ocr failed, continuing without text")
        ocrText = ""
    }
    
    // Extract tags from description + user context
    context := job.UserContext
    if context != "" {
        context = context + "\n" + description
    } else {
        context = description
    }
    
    tags, err := w.llm.ExtractTags(context)
    if err != nil {
        tags = []string{"image"}
    }
    
    // Update note
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
        Title:    fmt.Sprintf("Image — %s", job.CreatedAt.Format("2006-01-02")),
        Summary:  description,
        Raw:      description,
    }
    
    notePath, err := w.vault.WriteNote(note)
    if err != nil {
        return fmt.Errorf("failed to write note: %w", err)
    }
    
    job.NotePath = notePath
    
    // Generate embedding from description
    embedContent := description
    if ocrText != "" {
        embedContent = description + "\n\nExtracted text:\n" + ocrText
    }
    
    embedding, err := w.llm.Embed(embedContent)
    if err != nil {
        log.Warn().Err(err).Msg("failed to generate embedding")
        return nil
    }
    
    return w.queue.SaveEmbedding(job.ID, w.config.EmbedModel, embedding)
}

func (w *Worker) ocrImage(path string) (string, error) {
    // Use system OCR or tesseract
    // For now, this is a placeholder - integrate with tesseract or cloud OCR
    return "", nil
}
```

## Step 3.4: Article Ingest

**File:** `internal/ingest/article.go`

### Requirements

- Scrape article content
- Extract title, main content
- Generate summary using LLM
- Update note

### Process

1. Fetch URL
2. Extract title, content (use Readability or similar)
3. Summarize using LLM
4. Extract tags
5. Update note

```go
func (w *Worker) ingestArticle(job *queue.Job) error {
    // Fetch article
    title, content, err := w.scrapeArticle(job.SourceURL)
    if err != nil {
        return fmt.Errorf("failed to scrape article: %w", err)
    }
    
    // Generate summary
    summary, err := w.llm.Summarize(content)
    if err != nil {
        return fmt.Errorf("failed to generate summary: %w", err)
    }
    
    // Extract key ideas
    keyIdeas, err := w.llm.ExtractKeyIdeas(content)
    if err != nil {
        keyIdeas = []string{}
    }
    
    // Extract tags
    tags, err := w.llm.ExtractTags(content)
    if err != nil {
        tags = []string{"article"}
    }
    
    // Update note
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
        return fmt.Errorf("failed to write note: %w", err)
    }
    
    job.NotePath = notePath
    
    // Generate embedding
    embedContent := title + "\n\n" + summary + "\n\n" + strings.Join(keyIdeas, "\n")
    
    embedding, err := w.llm.Embed(embedContent)
    if err != nil {
        log.Warn().Err(err).Msg("failed to generate embedding")
        return nil
    }
    
    return w.queue.SaveEmbedding(job.ID, w.config.EmbedModel, embedding)
}

func (w *Worker) scrapeArticle(url string) (title, content string, err error) {
    // Use chromedp or net/html + readability
    // This is a placeholder - implement proper scraping
    resp, err := http.Get(url)
    if err != nil {
        return "", "", err
    }
    defer resp.Body.ReadCloser()
    
    // Parse with readability-lite
    // Return title and main content
    return "Article Title", "Article content...", nil
}
```

## Testing

Write tests for:

- [ ] Worker pool startup/shutdown
- [ ] Crash recovery
- [ ] Job processing (text, image, article)
- [ ] Retry logic
- [ ] Permanent failure cleanup

```bash
go test ./internal/worker/... -v
go test ./internal/ingest/... -v
```

## Checklist

- [ ] Worker pool implementation
- [ ] Configurable concurrency
- [ ] Job fetcher loop
- [ ] Crash recovery
- [ ] Text ingest
- [ ] Image ingest
- [ ] Article ingest
- [ ] Retry with backoff
- [ ] Permanent failure cleanup
- [ ] Embedding generation
- [ ] Tests passing
- [ ] go vet clean

## Next Phase

[Phase 4: LLM](phase-4-llm.md)

## Notes

- Processing times (M2 Mac Air):
  - Text: ~3s
  - Image: ~10s
  - Article: ~15s
- Embedding model: nomic-embed-text
- Default workers: 1
- Max retries: 3
