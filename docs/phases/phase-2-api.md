# Phase 2: Core API

> HTTP server, auth, logging, endpoints. Updated: 2026-03-17

## Goals

- [ ] Chi router setup
- [ ] Auth middleware
- [ ] Logging middleware
- [ ] Health endpoint
- [ ] Capture endpoint
- [ ] Queue endpoints
- [ ] Search endpoint (keyword initially)

## Dependencies

Add to `go.mod`:

```go
require (
    github.com/go-chi/chi/v5 v5.0.12
    github.com/go-chi/cors v1.6.0
)
```

## Directory Structure

```
internal/api/
├── server.go
├── capture.go
├── search.go
├── health.go
├── queue.go
└── middleware/
    ├── auth.go
    └── log.go
```

## Step 2.1: Server Setup

**File:** `internal/api/server.go`

### Requirements

- Chi router with middleware chain
- Configurable host/port
- Graceful shutdown
- Serve embedded static files (Phase 6)

### Structure

```go
type Server struct {
    router   *chi.Mux
    config   *config.Config
    queue    *queue.Queue
    vault    *vault.Writer
    llm      llm.LLM
    worker   *worker.Worker
}

func NewServer(cfg *config.Config, q *queue.Queue, v *vault.Writer) *Server {
    s := &Server{
        config: cfg,
        queue:  q,
        vault:  v,
    }
    s.setupRouter()
    return s
}

func (s *Server) setupRouter() {
    s.router = chi.NewRouter()
    
    // Global middleware
    s.router.Use(middleware.RequestID)
    s.router.Use(middleware.RealIP)
    s.router.Use(middleware.Logger)
    s.router.Use(middleware.Recoverer)
    
    // API routes
    s.router.Route("/v1", func(r chi.Router) {
        r.Use(s.authMiddleware)
        
        r.Get("/health", s.healthHandler)
        r.Post("/capture", s.captureHandler)
        r.Get("/search", s.searchHandler)
        r.Get("/queue", s.queueListHandler)
        r.Get("/queue/{id}", s.queueGetHandler)
        r.Post("/queue/{id}/retry", s.queueRetryHandler)
        r.Post("/queue/{id}/discard", s.queueDiscardHandler)
    })
    
    // Static files (Phase 6)
    s.router.Get("/*", s.staticHandler)
}

func (s *Server) Start() error {
    addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
    return http.ListenAndServe(addr, s.router)
}
```

### Middleware Chain Order

1. RequestID (unique per request)
2. RealIP (client IP)
3. Logger (log request)
4. Recoverer (panic recovery)

## Step 2.2: Auth Middleware

**File:** `internal/api/middleware/auth.go`

### Requirements

- Validate `X-Khayal-Token` header
- Return 401 if missing/invalid
- Never log token

```go
func AuthMiddleware(token string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            clientToken := r.Header.Get("X-Khayal-Token")
            if clientToken == "" {
                writeError(w, "token missing", "AUTH_002", http.StatusUnauthorized)
                return
            }
            if clientToken != token {
                writeError(w, "invalid token", "AUTH_001", http.StatusUnauthorized)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

## Step 2.3: Logging Middleware

**File:** `internal/api/middleware/log.go`

### Requirements

- Log: timestamp, method, path, status, latency
- Never log token or request body
- Use zerolog

```go
func Logger(log zerolog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // Wrap response to capture status
            rr := &responseWriter{ResponseWriter: w, status: 200}
            next.ServeHTTP(rr, r)
            
            latency := time.Since(start)
            
            log.Info().
                Str("method", r.Method).
                Str("path", r.URL.Path).
                Int("status", rr.status).
                Dur("latency", latency).
                Str("ip", r.RemoteAddr).
                Msg("request")
        })
    }
}

type responseWriter struct {
    http.ResponseWriter
    status int
}

func (w *responseWriter) WriteHeader(code int) {
    w.status = code
    w.ResponseWriter.WriteHeader(code)
}
```

## Step 2.4: Health Endpoint

**File:** `internal/api/health.go`

**Route:** `GET /v1/health`

### Response

```json
{
  "status": "ok",
  "version": "0.1.0",
  "dependencies": {
    "ollama": { "status": "ok", "host": "http://localhost:11434" },
    "vault": { "status": "ok", "path": "~/Documents/brain" },
    "db": { "status": "ok", "path": "~/.config/khayal/khayal.db" }
  },
  "queue": {
    "pending": 2,
    "processing": 1,
    "done": 147,
    "failed": 0
  }
}
```

### Handler

```go
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
    // Check Ollama
    ollamaStatus := "ok"
    if err := s.llm.Ping(); err != nil {
        ollamaStatus = "error"
    }
    
    // Check vault
    vaultStatus := "ok"
    if !s.vault.Exists(s.config.Vault.Path) {
        vaultStatus = "error"
    }
    
    // Queue stats
    pending, _ := s.queue.CountByStatus("pending")
    processing, _ := s.queue.CountByStatus("processing")
    done, _ := s.queue.CountByStatus("done")
    failed, _ := s.queue.CountByStatus("failed")
    
    writeJSON(w, HealthResponse{
        Status: "ok",
        Version: "0.1.0",
        Dependencies: Dependencies{
            Ollama: Dependency{Status: ollamaStatus, Host: s.config.LLM.OllamaHost},
            Vault: Dependency{Status: vaultStatus, Path: s.config.Vault.Path},
            DB: Dependency{Status: "ok", Path: s.config.DB.Path},
        },
        Queue: QueueStats{
            Pending: pending,
            Processing: processing,
            Done: done,
            Failed: failed,
        },
    })
}
```

## Step 2.5: Capture Endpoint

**File:** `internal/api/capture.go`

**Route:** `POST /v1/capture`

### Text Capture (JSON)

**Request:**
```json
{
  "type": "text",
  "content": "useEffect cleanup runs after every render"
}
```

**Response (sync - done immediately):**
```json
{
  "id": "abc123",
  "type": "text",
  "status": "done",
  "note_path": "inbox/2024-03-16-thought.md",
  "created_at": "2024-03-16T14:23:00Z"
}
```

### URL Capture (JSON)

**Request:**
```json
{
  "type": "url",
  "content": "https://blog.example.com/post"
}
```

**Response (queued):**
```json
{
  "id": "def456",
  "type": "article",
  "status": "processing",
  "note_path": "inbox/2024-03-16-url.md",
  "created_at": "2024-03-16T14:23:00Z"
}
```

### Image Capture (Multipart)

**Request:**
```
POST /v1/capture
Content-Type: multipart/form-data

type=image
file=<binary>
note="optional context"
```

**Response (queued):**
```json
{
  "id": "ghi789",
  "type": "image",
  "status": "processing",
  "note_path": "inbox/2024-03-16-image.md",
  "created_at": "2024-03-16T14:23:00Z"
}
```

### Handler

```go
func (s *Server) captureHandler(w http.ResponseWriter, r *http.Request) {
    contentType := r.Header.Get("Content-Type")
    
    if strings.Contains(contentType, "multipart/form-data") {
        s.handleImageCapture(w, r)
    } else {
        s.handleTextCapture(w, r)
    }
}

func (s *Server) handleTextCapture(w http.ResponseWriter, r *http.Request) {
    var req CaptureRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, "invalid request body", "CAPTURE_004", http.StatusBadRequest)
        return
    }
    
    if req.Content == "" {
        writeError(w, "missing required field: content", "CAPTURE_004", http.StatusBadRequest)
        return
    }
    
    job := &queue.Job{
        ID:        uuid.New().String(),
        Type:      req.Type,
        Status:    "pending",
        Content:   req.Content,
        CreatedAt: time.Now(),
    }
    
    // Text is synchronous
    if job.Type == "text" {
        job.Status = "done"
        notePath, err := s.processTextJob(job)
        if err != nil {
            writeError(w, "failed to write note", "VAULT_002", http.StatusInternalServerError)
            return
        }
        job.NotePath = notePath
        now := time.Now()
        job.ProcessedAt = &now
    } else {
        // URL/Article queued
        job.Type = "article"
    }
    
    // Save to queue
    if err := s.queue.CreateJob(job); err != nil {
        writeError(w, "failed to create job", "QUEUE_ERROR", http.StatusInternalServerError)
        return
    }
    
    writeJSON(w, CaptureResponse{
        ID:         job.ID,
        Type:       job.Type,
        Status:     job.Status,
        NotePath:   job.NotePath,
        CreatedAt:  job.CreatedAt.Format(time.RFC3339),
    })
}

func (s *Server) handleImageCapture(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form
    err := r.ParseMultipartForm(10 << 20) // 10MB
    if err != nil {
        writeError(w, "invalid multipart form", "CAPTURE_004", http.StatusBadRequest)
        return
    }
    
    file, header, err := r.FormFile("file")
    if err != nil {
        writeError(w, "missing file", "CAPTURE_004", http.StatusBadRequest)
        return
    }
    defer file.Close()
    
    note := r.FormValue("note")
    
    // Copy to media directory
    mediaPath, err := s.vault.CopyMediaFile(file, header.Filename)
    if err != nil {
        writeError(w, "failed to save media", "VAULT_002", http.StatusInternalServerError)
        return
    }
    
    job := &queue.Job{
        ID:          uuid.New().String(),
        Type:        "image",
        Status:      "pending",
        SourceFile:  mediaPath,
        UserContext: note,
        CreatedAt:   time.Now(),
    }
    
    if err := s.queue.CreateJob(job); err != nil {
        writeError(w, "failed to create job", "QUEUE_ERROR", http.StatusInternalServerError)
        return
    }
    
    // Write initial note
    notePath := s.vault.InboxPath(fmt.Sprintf("inbox/%s-image.md", job.ID[:8]))
    // Write processing note...
    
    writeJSON(w, CaptureResponse{
        ID:        job.ID,
        Type:      "image",
        Status:    "processing",
        NotePath:  notePath,
        CreatedAt: job.CreatedAt.Format(time.RFC3339),
    })
}
```

## Step 2.6: Queue Endpoints

**File:** `internal/api/queue.go`

### List Jobs

**Route:** `GET /v1/queue`

**Query Params:**
| Param | Default | Options |
|-------|---------|---------|
| status | all | all, pending, processing, done, failed |
| limit | 20 | max 100 |
| offset | 0 | - |

**Response:**
```json
{
  "total": 3,
  "limit": 20,
  "offset": 0,
  "jobs": [
    {
      "id": "abc123",
      "type": "image",
      "status": "processing",
      "note_path": "inbox/2024-03-16-image.md",
      "created_at": "2024-03-16T14:23:00Z",
      "processed_at": null,
      "error": null
    }
  ]
}
```

### Get Single Job

**Route:** `GET /v1/queue/:id`

**Response:**
```json
{
  "id": "abc123",
  "type": "image",
  "status": "failed",
  "note_path": null,
  "created_at": "2024-03-16T14:23:00Z",
  "processed_at": null,
  "error": "failed to describe image: ollama timeout",
  "retries": 3
}
```

### Retry Failed Job

**Route:** `POST /v1/queue/:id/retry`

Retry a failed or pending job. Resets retry count and sets status to pending.

**Response:**
```json
{
  "id": "abc123",
  "type": "image",
  "status": "pending",
  "note_path": null,
  "created_at": "2024-03-16T14:23:00Z",
  "processed_at": null,
  "error": null,
  "retries": 0
}
```

### Discard Job

**Route:** `POST /v1/queue/:id/discard`

Permanently delete a failed or pending job. Also deletes associated media file if exists.

**Response:**
```json
{
  "success": true,
  "id": "abc123",
  "message": "job discarded"
}
```

### Queue Handler Implementation

```go
func (s *Server) queueRetryHandler(w http.ResponseWriter, r *http.Request) {
    jobID := chi.URLParam(r, "id")
    
    job, err := s.queue.GetJob(jobID)
    if err != nil {
        writeError(w, "job not found", "NOT_FOUND", http.StatusNotFound)
        return
    }
    
    // Can only retry pending or failed jobs
    if job.Status != "pending" && job.Status != "failed" {
        writeError(w, "can only retry pending or failed jobs", "INVALID_STATE", http.StatusBadRequest)
        return
    }
    
    // Reset for retry
    job.Status = "pending"
    job.Error = ""
    job.Retries = 0
    
    if err := s.queue.UpdateJob(job); err != nil {
        writeError(w, "failed to update job", "QUEUE_ERROR", http.StatusInternalServerError)
        return
    }
    
    writeJSON(w, job)
}

func (s *Server) queueDiscardHandler(w http.ResponseWriter, r *http.Request) {
    jobID := chi.URLParam(r, "id")
    
    job, err := s.queue.GetJob(jobID)
    if err != nil {
        writeError(w, "job not found", "NOT_FOUND", http.StatusNotFound)
        return
    }
    
    // Can only discard pending or failed jobs
    if job.Status == "done" {
        writeError(w, "cannot discard completed jobs", "INVALID_STATE", http.StatusBadRequest)
        return
    }
    
    // Delete media file if exists
    if job.SourceFile != "" {
        s.vault.DeleteMedia(job.SourceFile)
    }
    
    // Delete job from queue
    if err := s.queue.DeleteJob(jobID); err != nil {
        writeError(w, "failed to delete job", "QUEUE_ERROR", http.StatusInternalServerError)
        return
    }
    
    writeJSON(w, map[string]interface{}{
        "success": true,
        "id":      jobID,
        "message": "job discarded",
    })
}
```

**Note:** Failed jobs are NOT automatically deleted. User must explicitly discard them to:
- Prevent accidental data loss
- Allow debugging why it failed
- Keep source media for re-processing

## Step 2.7: Search Endpoint

**File:** `internal/api/search.go`

**Route:** `GET /v1/search`

**Query Params:**
| Param | Required | Default | Max |
|-------|----------|---------|-----|
| q | yes | - | - |
| limit | no | 10 | 50 |
| mode | no | hybrid | hybrid/keyword/semantic |
| excerpt_length | no | 200 | 500 |

**Response:**
```json
{
  "query": "distributed systems",
  "mode": "hybrid",
  "results": [
    {
      "id": "abc123",
      "note_path": "inbox/2024-03-10-cap-theorem.md",
      "title": "CAP Theorem Notes",
      "excerpt": "...consistency and availability cannot both be guaranteed...",
      "score": 0.94,
      "type": "text",
      "created_at": "2024-03-10T09:00:00Z"
    }
  ],
  "total": 3,
  "took_ms": 42
}
```

### Handler

```go
func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    if query == "" {
        writeError(w, "missing required parameter: q", "SEARCH_001", http.StatusBadRequest)
        return
    }
    
    limit := 10
    if l := r.URL.Query().Get("limit"); l != "" {
        limit, _ = strconv.Atoi(l)
        if limit > 50 {
            limit = 50
        }
    }
    
    mode := r.URL.Query().Get("mode")
    if mode == "" {
        mode = "hybrid"
    }
    
    excerptLen := 200
    if e := r.URL.Query().Get("excerpt_length"); e != "" {
        excerptLen, _ = strconv.Atoi(e)
        if excerptLen > 500 {
            excerptLen = 500
        }
    }
    
    start := time.Now()
    
    var results []queue.SearchResult
    var err error
    
    switch mode {
    case "keyword":
        results, err = s.queue.SearchKeyword(query, limit)
    case "semantic":
        // Phase 4 - LLM needed
        results, err = s.queue.SearchSemanticSimple(query, limit)
    case "hybrid":
        fallthrough
    default:
        // Combine keyword + semantic
        results, err = s.queue.SearchHybrid(query, limit)
    }
    
    if err != nil {
        writeError(w, "search failed", "SEARCH_ERROR", http.StatusInternalServerError)
        return
    }
    
    // Generate excerpts
    for i := range results {
        results[i].Excerpt = s.generateExcerpt(results[i].NotePath, excerptLen)
    }
    
    took := time.Since(start).Milliseconds()
    
    writeJSON(w, SearchResponse{
        Query:    query,
        Mode:     mode,
        Results:  results,
        Total:    len(results),
        TookMs:   took,
    })
}
```

## Testing

Write integration tests for:

- [ ] Auth middleware (valid/invalid token)
- [ ] Health endpoint
- [ ] Capture text (sync)
- [ ] Capture URL (queued)
- [ ] Capture image (multipart)
- [ ] Queue list
- [ ] Queue get
- [ ] Search keyword
- [ ] Search semantic (Phase 4)
- [ ] Error responses

```bash
go test ./internal/api/... -v
```

## Checklist

- [ ] Chi router setup
- [ ] Global middleware chain
- [ ] Auth middleware
- [ ] Logging middleware
- [ ] Health endpoint
- [ ] Capture text handler
- [ ] Capture URL handler
- [ ] Capture image handler
- [ ] Queue list endpoint
- [ ] Queue get endpoint
- [ ] Queue retry endpoint
- [ ] Queue discard endpoint
- [ ] Search keyword endpoint
- [ ] Search semantic stub
- [ ] Error responses standardized
- [ ] Integration tests passing
- [ ] go vet clean
- [ ] golangci-lint clean

## Next Phase

[Phase 3: Worker](phase-3-worker.md)

## Notes

- All times in RFC3339 format
- Job IDs: UUID v4
- Note paths: relative to vault root
- Failed jobs stay in queue - user can retry or discard
- Excerpts: max 500 chars
- Limit max: 50 for search, 100 for queue
