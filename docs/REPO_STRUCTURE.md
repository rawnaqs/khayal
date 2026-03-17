# Repository Structure

> Complete file tree for Khayal v1. Updated: 2026-03-17

```
khayal/
├── cmd/
│   └── khayal/
│       └── main.go                    # Single binary entry point
│
├── internal/
│   ├── api/
│   │   ├── server.go                   # HTTP server, router, middleware
│   │   ├── capture.go                  # POST /v1/capture
│   │   ├── search.go                   # GET /v1/search
│   │   ├── health.go                   # GET /v1/health
│   │   ├── queue.go                    # GET /v1/queue, GET /v1/queue/:id
│   │   ├── static.go                   # SPA static file serving
│   │   └── middleware/
│   │       ├── auth.go                 # Token authentication
│   │       └── log.go                  # Request logging
│   │
│   ├── worker/
│   │   └── worker.go                   # Job processor, concurrency, retry
│   │
│   ├── ingest/
│   │   ├── text.go                     # Text processing (tags, summary)
│   │   ├── image.go                    # Image processing (description, OCR)
│   │   └── article.go                  # Article scraping, summarization
│   │
│   ├── llm/
│   │   ├── interface.go                # LLM interface definition
│   │   ├── ollama.go                   # Ollama client (primary)
│   │   ├── groq.go                     # Groq fallback
│   │   ├── openai.go                   # OpenAI fallback
│   │   └── factory.go                  # LLM factory with fallback
│   │
│   ├── vault/
│   │   └── writer.go                   # Markdown writer, frontmatter
│   │
│   ├── queue/
│   │   └── queue.go                    # SQLite job queue, FTS5, embeddings
│   │
│   ├── search/
│   │   ├── keyword.go                  # FTS5 keyword search
│   │   └── semantic.go                 # Vector similarity search
│   │
│   ├── config/
│   │   └── config.go                   # Config loader, validation
│   │
│   └── version/
│       └── version.go                  # Version info (set by goreleaser)
│
├── cli/
│   ├── main.go                         # CLI entry point
│   ├── root.go                         # Cobra root command
│   ├── capture.go                      # kl "text", --url, --image
│   ├── search.go                       # kl search (Glamour)
│   ├── status.go                       # kl status (Bubble Tea)
│   ├── init.go                         # kl init (Huh wizard)
│   └── config.go                       # kl config set/get/view
│
├── ui/
│   ├── react/                          # Vite + React project
│   │   ├── package.json
│   │   ├── vite.config.ts
│   │   ├── index.html
│   │   ├── tsconfig.json
│   │   ├── src/
│   │   │   ├── main.tsx                # Entry point
│   │   │   ├── App.tsx                 # Routes
│   │   │   ├── components/
│   │   │   │   ├── Layout.tsx          # Main layout
│   │   │   │   ├── Capture.tsx         # Capture form
│   │   │   │   ├── Search.tsx          # Search UI
│   │   │   │   ├── Queue.tsx           # Queue display
│   │   │   │   └── OfflineIndicator.tsx
│   │   │   ├── lib/
│   │   │   │   ├── api.ts              # API client
│   │   │   │   ├── offline.ts          # IndexedDB queue
│   │   │   │   └── store.ts            # Zustand store
│   │   │   └── styles/
│   │   │       └── global.css          # Global styles
│   │   └── public/
│   │       └── manifest.json           # PWA manifest
│   │
│   └── static/                         # Built React app (generated)
│       ├── index.html
│       ├── assets/
│       └── ...
│
├── install/
│   └── check.go                        # Dependency checker
│
├── .github/
│   └── workflows/
│       ├── ci.yml                      # Test, vet, lint
│       └── release.yml                  # GoReleaser
│
├── docs/
│   ├── khayal-spec.md                  # Master specification
│   ├── TECH_STACK.md                    # Technology decisions
│   ├── ARCHITECTURE.md                  # System design
│   ├── PLAN.md                          # Implementation overview
│   ├── REPO_STRUCTURE.md                # This file
│   └── phases/
│       ├── phase-1-foundation.md
│       ├── phase-2-api.md
│       ├── phase-3-worker.md
│       ├── phase-4-llm.md
│       ├── phase-5-cli.md
│       ├── phase-6-pwa.md
│       └── phase-7-polish.md
│
├── ui/react/                            # npm dependencies for PWA
│
├── docker-compose.yml                   # Local development
├── Dockerfile                           # Docker build
├── .goreleaser.yml                       # Release config
├── .gitignore
├── LICENSE                              # AGPLv3
├── README.md
├── CONTRIBUTING.md
├── config.example.yaml                  # Safe to commit, no secrets
├── go.mod
└── go.sum
```

---

## Directory Purpose

### `cmd/`

Single entry point for the binary. All other code is under `internal/`.

### `internal/`

Private application code. Not importable by external packages.

| Directory | Purpose |
|-----------|---------|
| `api/` | HTTP handlers, middleware, routing |
| `worker/` | Background job processing |
| `ingest/` | Content processing (text, image, article) |
| `llm/` | AI integration |
| `vault/` | Markdown file writing |
| `queue/` | SQLite database operations |
| `search/` | Search algorithms |
| `config/` | Configuration management |
| `version/` | Version info |

### `cli/`

Cobra-based CLI (`kl` command). Separated from server for clarity.

### `ui/`

Frontend code. `react/` is source, `static/` is built output.

### `install/`

Installation helpers (dependency checker).

### `docs/`

Documentation. See individual phase files for detailed implementation guides.

---

## File Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Go source | `snake_case.go` | `config.go`, `auth.go` |
| Go test | `*_test.go` | `config_test.go` |
| React components | `PascalCase.tsx` | `Capture.tsx`, `Search.tsx` |
| React utilities | `camelCase.ts` | `api.ts`, `offline.ts` |
| Config | `kebab-case.yaml` | `config.example.yaml` |

---

## Key Interfaces

### LLM (internal/llm/interface.go)

```go
type LLM interface {
    Embed(text string) ([]float32, error)
    Generate(prompt string) (string, error)
    DescribeImage(path string) (string, error)
    Ping() error
    Type() string
}
```

### Queue (internal/queue/queue.go)

```go
type Queue interface {
    CreateJob(job *Job) error
    GetJob(id string) (*Job, error)
    UpdateJob(job *Job) error
    GetPendingJobs(limit int) ([]Job, error)
    SearchKeyword(query string, limit int) ([]SearchResult, error)
    SearchSemantic(queryEmbedding []float32, limit int) ([]SearchResult, error)
    SaveEmbedding(jobID, model string, vector []float32) error
}
```

### Vault (internal/vault/writer.go)

```go
type Writer interface {
    WriteNote(note *Note) (string, error)
    UpdateNote(notePath string, note *Note) error
    DeleteNote(notePath string) error
    CopyMediaFile(srcPath string) (string, error)
}
```

---

## API Endpoints

| Method | Path | Handler |
|--------|------|---------|
| POST | /v1/capture | capture.go |
| GET | /v1/search | search.go |
| GET | /v1/health | health.go |
| GET | /v1/queue | queue.go |
| GET | /v1/queue/:id | queue.go |
| GET | /\* | static.go (SPA) |

---

## CLI Commands

| Command | File | Description |
|---------|------|-------------|
| `kl` | root.go | Root (capture) |
| `kl capture` | capture.go | Capture text/url/image |
| `kl search` | search.go | Search knowledge base |
| `kl status` | status.go | Queue dashboard |
| `kl init` | init.go | Setup wizard |
| `kl config` | config.go | Config management |

---

## Database Schema

### jobs table

```sql
CREATE TABLE jobs (
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
);
```

### notes_fts (FTS5)

```sql
CREATE VIRTUAL TABLE notes_fts USING fts5(
    note_path,
    content,
    title,
    tags
);
```

### embeddings table

```sql
CREATE TABLE embeddings (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL,
    vector BLOB NOT NULL,
    model TEXT NOT NULL,
    created_at TEXT NOT NULL
);
```

---

## Build Output

```
khayal              # Linux amd64
khayal_darwin_amd64 # macOS Intel
khayal_darwin_arm64 # macOS Apple Silicon
khayal_linux_arm64  # Linux ARM
```

---

## Environment

- Go: 1.22+
- Node: 18+ (for PWA build)
- SQLite: modernc.org/sqlite (pure Go)
