# Phase 1: Foundation

> Project setup, config, database, vault writer. Updated: 2026-03-17

## Goals

- [ ] Initialize Go module with dependencies
- [ ] Create directory structure
- [ ] Config loader with validation
- [ ] SQLite job queue schema
- [ ] Markdown + frontmatter writer

## Dependencies

Add to `go.mod`:

```go
require (
    github.com/google/uuid v1.6.0
    gopkg.in/yaml.v3 v3.0.1
    modernc.org/sqlite v1.45.0
)
```

**Note:** Using `modernc.org/sqlite` (pure Go, no CGO) - not `mattn/go-sqlite3`.

## Directory Structure

Create:

```
cmd/khayal/
internal/
├── config/
│   └── config.go
├── queue/
│   └── queue.go
└── vault/
    └── writer.go
```

## Step 1.1: Initialize Go Module

```bash
go mod init github.com/rawnaqs/khayal
go mod tidy
```

## Step 1.2: Config System

**File:** `internal/config/config.go`

### Requirements

- Load YAML from `~/.config/khayal/config.yaml`
- Validate required fields (fail hard)
- Generate token if empty
- Create directories if missing
- Set file permissions to 0600

### Config Struct

```go
type Config struct {
    Vault   VaultConfig   `yaml:"vault"`
    Server  ServerConfig  `yaml:"server"`
    LLM     LLMConfig    `yaml:"llm"`
    Worker  WorkerConfig `yaml:"worker"`
    DB      DBConfig     `yaml:"db"`
}

type VaultConfig struct {
    Path        string `yaml:"path"`
    InboxDir    string `yaml:"inbox_dir"`
    Media       MediaConfig `yaml:"media"`
}

type MediaConfig struct {
    DefaultDir string `yaml:"default_dir"`
    Strategy   StrategyConfig `yaml:"strategy"`
}

type StrategyConfig struct {
    Image string `yaml:"image"` // "vault" | "config"
    PDF   string `yaml:"pdf"`   // "vault" | "config"
    Audio string `yaml:"audio"` // "vault" | "config"
    Video string `yaml:"video"` // "vault" | "config"
}

type ServerConfig struct {
    Host    string `yaml:"host"`
    Port    int    `yaml:"port"`
    Token   string `yaml:"token"`
    LogFile string `yaml:"log_file"`
}

type LLMConfig struct {
    Provider          string `yaml:"provider"` // "ollama" | "groq" | "openai"
    OllamaHost        string `yaml:"ollama_host"`
    EmbedModel        string `yaml:"embed_model"`
    TextModel         string `yaml:"text_model"`
    VisionModel       string `yaml:"vision_model"`
    FallbackProvider  string `yaml:"fallback_provider"`
    FallbackAPIKey    string `yaml:"fallback_api_key"`
}

type WorkerConfig struct {
    MaxWorkers     int    `yaml:"max_workers"`
    MaxRetries    int    `yaml:"max_retries"`
    RetryBackoff  string `yaml:"retry_backoff"` // "immediate" | "fixed" | "exponential"
}

type DBConfig struct {
    Path string `yaml:"path"`
}
```

### Validation Rules

| Field | Rule |
|-------|------|
| `vault.path` | Required, must be absolute or start with ~ |
| `server.host` | Required |
| `server.port` | Required, 1-65535 |
| `server.token` | Auto-generate if empty (32-byte hex) |
| `db.path` | Required |

### Methods

```go
func Load() (*Config, error)
func (c *Config) Validate() error
func (c *Config) EnsureDirectories() error
func GenerateToken() string // 32-byte hex
```

### Example Config

```yaml
vault:
  path: ~/Documents/brain
  inbox_dir: khayal
  media:
    default_dir: media
    strategy:
      image: vault
      pdf: vault
      audio: config
      video: config

server:
  host: 127.0.0.1
  port: 1133
  token: ""
  log_file: ~/.config/khayal/logs/khayal.log

llm:
  provider: ollama
  ollama_host: http://localhost:11434
  embed_model: nomic-embed-text
  text_model: llama3.2:3b
  vision_model: moondream
  fallback_provider: ""
  fallback_api_key: ""

worker:
  max_workers: 1
  max_retries: 3
  retry_backoff: exponential

db:
  path: ~/.config/khayal/khayal.db
```

## Step 1.3: SQLite Queue

**File:** `internal/queue/queue.go`

### Schema

```sql
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,           -- "text" | "image" | "article"
    status TEXT NOT NULL,          -- "pending" | "processing" | "done" | "failed"
    note_path TEXT,
    source_url TEXT,
    source_file TEXT,
    content TEXT,
    user_context TEXT,
    created_at TEXT NOT NULL,
    processed_at TEXT,
    error TEXT,
    retries INTEGER DEFAULT 0
);

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_created ON jobs(created_at);

-- Full-text search
CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
    note_path,
    content,
    title,
    tags
);

-- Embeddings storage
CREATE TABLE IF NOT EXISTS embeddings (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL,
    vector BLOB NOT NULL,
    model TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (job_id) REFERENCES jobs(id)
);

CREATE INDEX idx_embeddings_job ON embeddings(job_id);

-- Entities table (v1.1+)
CREATE TABLE IF NOT EXISTS entities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_path TEXT NOT NULL,
    chunk_idx INTEGER,
    entity_type TEXT NOT NULL, -- "person" | "amount" | "date" | "place" | "org" | "url"
    entity_value TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX idx_entities_note ON entities(note_path);
CREATE INDEX idx_entities_type ON entities(entity_type);

-- Chunks table for chunk-level embeddings (v1.1+)
CREATE TABLE IF NOT EXISTS chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_path TEXT NOT NULL,
    chunk_idx INTEGER NOT NULL,
    content TEXT NOT NULL,
    embedding BLOB NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX idx_chunks_note ON chunks(note_path);
```

### Job Struct

```go
type Job struct {
    ID          string    `json:"id"`
    Type        string    `json:"type"` // "text" | "image" | "article"
    Status      string    `json:"status"`
    NotePath    string    `json:"note_path,omitempty"`
    SourceURL   string    `json:"source_url,omitempty"`
    SourceFile  string    `json:"source_file,omitempty"`
    Content     string    `json:"content,omitempty"`
    UserContext string    `json:"user_context,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    ProcessedAt *time.Time `json:"processed_at,omitempty"`
    Error       string    `json:"error,omitempty"`
    Retries     int       `json:"retries"`
}
```

### Methods

```go
func NewQueue(dbPath string) (*Queue, error)
func (q *Queue) CreateJob(job *Job) error
func (q *Queue) GetJob(id string) (*Job, error)
func (q *Queue) UpdateJobStatus(id, status string) error
func (q *Queue) ListJobs(status string, limit, offset int) ([]Job, int, error)
func (q *Queue) GetPendingJobs(limit int) ([]Job, error)
func (q *Queue) ResetStuckJobs() error // For crash recovery

// Search
func (q *Queue) SearchKeyword(query string, limit int) ([]SearchResult, error)
func (q *Queue) SearchSemantic(queryEmbedding []float32, limit int) ([]SearchResult, error)
func (q *Queue) SaveEmbedding(jobID, model string, vector []float32) error

type SearchResult struct {
    JobID     string  `json:"id"`
    NotePath  string  `json:"note_path"`
    Title     string  `json:"title"`
    Excerpt   string  `json:"excerpt"`
    Score     float64 `json:"score"`
    Type      string  `json:"type"`
    CreatedAt string `json:"created_at"`
}
```

## Step 1.4: Vault Writer

**File:** `internal/vault/writer.go`

### Requirements

- Write markdown with YAML frontmatter
- All frontmatter keys: snake_case
- Support text, image, article types
- Media file management
- Path resolution (relative + absolute)

### Note Structures

#### Text (done)

```markdown
---
created: 2024-03-16T14:23:00
updated: 2024-03-16T14:23:04
type: text
status: done
tags:
  - react
  - performance
history:
  - at: 2024-03-16T14:23:04
    event: processed
---

# useEffect cleanup runs after every render

## Summary
A brief summary of the thought...

## Key Ideas
- useEffect cleanup prevents memory leaks
- Dependency array controls when effect runs

## Raw
useEffect cleanup runs after every render
```

#### Image (processing)

```markdown
---
created: 2024-03-16T14:23:00
type: image
status: processing
source_file: "media/2024-03-16-image.png"
user_context: "optional note user attached"
---

# Image — 2024-03-16

optional note user attached

![image](media/2024-03-16-image.png)

_Processing — content will appear here shortly_
```

#### Image (done)

```markdown
---
created: 2024-03-16T14:23:00
updated: 2024-03-16T14:23:12
type: image
status: done
source_file: "media/2024-03-16-image.png"
user_context: "optional note user attached"
tags:
  - system-design
history:
  - at: 2024-03-16T14:23:12
    event: processed
---

# Image — 2024-03-16

optional note user attached

![image](media/2024-03-16-image.png)

## Description
LLM generated description of the image...

## Extracted Text
OCR output here...
```

#### Article

```markdown
---
created: 2024-03-16T14:23:00
updated: 2024-03-16T14:23:18
type: article
status: done
source_url: "https://blog.example.com/post"
tags:
  - distributed-systems
history:
  - at: 2024-03-16T14:23:18
    event: processed
---

# Article Title

## Summary
A concise summary of the article...

## Key Ideas
- First key idea
- Second key idea

## Source
https://blog.example.com/post
```

### Structs

```go
type NoteMetadata struct {
    Created      time.Time         `yaml:"created"`
    Updated      *time.Time        `yaml:"updated,omitempty"`
    Type         string            `yaml:"type"`
    Status       string            `yaml:"status"`
    SourceURL    string            `yaml:"source_url,omitempty"`
    SourceFile   string            `yaml:"source_file,omitempty"`
    UserContext  string            `yaml:"user_context,omitempty"`
    Tags         []string          `yaml:"tags,omitempty"`
    History      []HistoryEvent    `yaml:"history,omitempty"`
}

type HistoryEvent struct {
    At    time.Time `yaml:"at"`
    Event string    `yaml:"event"`
}

type Note struct {
    Metadata NoteMetadata
    Title    string
    Summary  string
    KeyIdeas []string
    Content  string
    Raw      string
}
```

### Methods

```go
func NewWriter(vaultPath string) (*Writer, error)
func (w *Writer) WriteNote(note *Note) (string, error) // Returns note_path
func (w *Writer) UpdateNote(notePath string, note *Note) error
func (w *Writer) DeleteNote(notePath string) error
func (w *Writer) CopyMediaFile(srcPath string) (string, error)
func (w *Writer) ResolvePath(relative string) string
func (w *Writer) NoteExists(notePath string) bool

// Frontmatter generation
func GenerateFrontmatter(meta NoteMetadata) string
func ParseFrontmatter(content string) (*NoteMetadata, error)
```

## Step 1.5: Main Entry Point

**File:** `cmd/khayal/main.go`

```go
package main

import (
    "log"
    "github.com/rawnaqs/khayal/internal/config"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("Config loaded:", cfg.Vault.Path)
}
```

## Testing

Write unit tests for:

- [ ] Config validation (missing fields, invalid values)
- [ ] Token generation (length, uniqueness)
- [ ] Queue CRUD operations
- [ ] Vault frontmatter generation/parsing
- [ ] Path resolution

```bash
go test ./internal/config/...
go test ./internal/queue/...
go test ./internal/vault/...
```

## Checklist

- [ ] Go module initialized
- [ ] Dependencies added to go.mod
- [ ] Directory structure created
- [ ] Config loader with validation
- [ ] Token auto-generation
- [ ] SQLite schema
- [ ] Job queue operations
- [ ] Search index setup
- [ ] Vault markdown writer
- [ ] Frontmatter generation
- [ ] Media file handling
- [ ] Basic main.go
- [ ] Unit tests passing
- [ ] `go vet` clean
- [ ] `golangci-lint` clean

## Next Phase

[Phase 2: Core API](phase-2-api.md)

## Notes

- Config path: `~/.config/khayal/config.yaml`
- DB path: `~/.config/khayal/khayal.db`
- Vault: User-defined, separate from config dir
- Default bind: `127.0.0.1:1133`
- Token: 32-byte hex, shown once on init

---

## Learnings & Retrospectives

### SQLite Driver Decision (2026-03-17)

**Initial choice:** `mattn/go-sqlite3` (CGO-based)

**Problem discovered:**
- Requires CGO compilation
- `sqlite3_auto_extension` deprecated on macOS (causes warnings)
- Incompatible with pure Go deployment goals

**Solution:** `modernc.org/sqlite`

**Benefits:**
- 100% pure Go, no CGO
- No system dependencies
- FTS5 included by default
- Works out of the box on all platforms

**Trade-offs:**
- Slightly larger binary (includes SQLite implementation)
- No access to SQLite C extensions

### Vector Search Investigation (2026-03-18)

**Attempted:** `viant/sqlite-vec` (pure Go vector search)

**Problem discovered:**
- Requires `SetMaxOpenConns(1)` for module registration
- Internally calls `db.Exec()` during query execution (for index building)
- Causes deadlock when combined with single connection limit
- WAL mode doesn't resolve the issue
- Separate databases required for reliable operation

**Root cause:**
```
database/sql.(*DB).exec → vec.(*Table).ensureShadow → vec.(*Table).ensureIndex
```
The virtual table holds a connection while trying to execute more SQL on the same connection.

**Solution:** Pure Go cosine similarity

**Implementation:**
- Batch processing (1000 chunks per query)
- In-memory computation with deduplication
- `cosine(a, b) = dot(a, b)` for normalized vectors
- Results deduplicated by `note_path` (best score per note)

**Performance characteristics:**
- Memory: O(batch_size)
- CPU: O(n * d) where n=chunks, d=dimensions
- Suitable for <100K chunks

**Future optimizations (if needed):**
- Precomputed norms stored in column
- HNSW or IVF index in Go
- Separate worker for async indexing
