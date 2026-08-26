# Repository Structure

> Complete file tree for Khayal v1. Updated: 2026-03-24

## Two Binaries

| Binary | Command | Description |
|--------|---------|-------------|
| `khayal` | `khayal init`, `khayal start` | Server + Worker + PWA |
| `kl` | `kl "thought"`, `kl search` | Thin HTTP client |

## File Tree

```
khayal/
├── cmd/
│   ├── khayal/                      # Server admin CLI
│   │   ├── main.go                  # Entry point: khayal
│   │   ├── internal/                # khayal-only utilities
│   │   │   ├── config.go           # Config loading/writing
│   │   │   ├── pid.go              # PID file management
│   │   │   ├── deps.go             # Dependency checking (ollama)
│   │   │   ├── output.go           # Styled output helpers
│   │   │   └── errors.go           # Error formatting + exit codes
│   │   └── commands/               # khayal subcommands
│   │       ├── init.go            # First-run setup
│   │       ├── start.go           # Start server + deps check
│   │       ├── stop.go            # Graceful shutdown
│   │       ├── restart.go          # Stop + start
│   │       ├── status.go          # Server status + update check
│   │       ├── reindex.go         # Progress bar reindex
│   │       ├── vault*.go         # Vault maintenance: health, fix-links, clean-media, show-duplicates
│   │       ├── backup.go         # Backup vault/db/config (--dest --encrypt --init-key)
│   │       └── restore.go        # Restore from backup (--from --date --overwrite)
│   │       ├── version.go         # Version info
│   │       └── config.go          # View config
│   │
│   └── kl/                          # Client CLI
│       ├── main.go                  # Entry point: kl
│       ├── internal/                # kl-only utilities
│       │   ├── config.go           # Config loading (KL_CONFIG env)
│       │   ├── output.go           # Styled output helpers
│       │   └── api/                # HTTP client
│       │       └── client.go      # API client for server
│       └── commands/               # kl subcommands
│           ├── root.go             # Default capture
│           ├── capture.go          # Text capture
│           ├── capture_url.go      # URL capture
│           ├── capture_image.go    # Image capture
│           ├── search.go           # Search vault
│           ├── delete.go           # Soft-delete a note (kl delete)
│           ├── recent.go           # Recent captures
│           ├── stats.go            # Vault statistics
│           ├── status.go           # Lightweight check
│           ├── init.go             # Huh wizard setup
│           └── config/             # Config subcommands
│               └── root.go        # View/set/get config
│
├── internal/
│   ├── api/
│   │   ├── server.go                # HTTP server, router, middleware
│   │   ├── capture.go               # POST /v1/capture
│   │   ├── search.go               # GET /v1/search
│   │   ├── overview.go             # AI answer: RAG synthesis + citation extraction
│   │   ├── notes.go                # GET /v1/notes/{path}, DELETE /v1/note
│   │   ├── overview.go             # AI answer: RAG synthesis + citation extraction
│   │   ├── health.go               # GET /v1/health
│   │   ├── queue.go                # GET /v1/queue, queue operations
│   │   ├── ws.go                   # GET /v1/queue/ws WebSocket stream
│   │   ├── static.go               # SPA static file serving
│   │   ├── client/                 # SHARED HTTP CLIENT
│   │   │   └── client.go           # Typed Go client for API
│   │   └── middleware/
│   │       ├── auth.go             # Token authentication
│   │       └── log.go              # Request logging
│   │
│   ├── constants/
│   │   └── constants.go            # Shared constants (retry, milestones, prompts, timeouts)
│   │
│   ├── worker/
│   │   └── worker.go               # Job processor, concurrency, retry
│   │
│   ├── ingest/
│   │   ├── text.go                 # Text processing (tags, summary)
│   │   ├── image.go                # Image processing (description, OCR)
│   │   ├── article.go              # Article scraping, summarization
│   │   ├── chunks.go               # Chunk + batch-embed + persist pipeline
│   │   └── entities.go             # Entity extraction + normalization
│   │
│   ├── llm/
│   │   ├── interface.go            # LLM interface definition
│   │   ├── ollama.go               # Ollama client
│   │   └── factory.go              # LLM factory
│   │
│   ├── vault/
│   │   └── writer.go               # Markdown writer, frontmatter
│   │
│   ├── chunk/
│   │   └── chunk.go                    # Paragraph-aligned text chunking (pure)
│   │
│   ├── queue/
│   │   └── queue.go                    # SQLite job queue, FTS5, semantic search
│   │                                   #   (keyword/semantic/hybrid), entity rows,
│   │                                   #   connections helpers
│   │
│   ├── connections/                    # Proactive connections (v1.1 phase 2)
│   │   └── connections.go              # Detectors (similar/person/amount), ranking, dedup
│   │
│   ├── config/
│   │   └── config.go                   # Config loader, validation
│   │
│   ├── version/
│   │   └── version.go                  # Version info (set by goreleaser)
│   │
│   └── updater/
│       └── check.go                    # GitHub release version check
│
├── .goreleaser.yml                      # Release config (2 binaries)
├── .github/
│   └── workflows/
│       ├── ci.yml                       # Test + lint on PRs
│       └── release.yml                  # GoReleaser on v* tags
├── install.sh                           # One-liner curl installer
├── Dockerfile                           # Go-only build
├── docker-compose.yml                   # Dev build (docker compose build)
├── docker.sh                            # Wrapper script for end users
├── .dockerignore                        # Exclude node_modules, .git, docs
├── config.example.yaml                  # Full config reference
├── README.md
├── CONTRIBUTING.md
│
├── external/
│   └── react/                          # Vite + React PWA project
│       ├── package.json
│       ├── vite.config.ts              # Vite + PWA plugin config
│       ├── vitest.config.ts            # Unit test config
│       ├── playwright.config.ts        # E2E test config
│       ├── tailwind.config.js
│       ├── postcss.config.js
│       ├── tsconfig.json
│       ├── tsconfig.node.json
│       ├── index.html
│       ├── components.json             # shadcn/ui config
│       ├── public/
│       │   ├── icon-192.png            # PWA icon (small)
│       │   ├── icon-512.png            # PWA icon (large)
│       │   ├── icon.png                # Source icon
│       │   └── icon.svg                # SVG icon
│       ├── src/
│       │   ├── main.tsx                # Entry point + SW registration
│       │   ├── App.tsx                 # Root component, tab routing
│       │   ├── index.css               # All styles (single CSS file)
│       │   ├── sw.ts                   # Service worker (Workbox + background sync)
│       │   ├── vite-env.d.ts
│       │   ├── components/
│       │   │   ├── capture/
│       │   │   │   ├── CaptureView.tsx      # Main capture screen
│       │   │   │   ├── TextCapture.tsx      # Text input
│       │   │   │   ├── UrlCapture.tsx       # URL input
│       │   │   │   ├── ImageCapture.tsx     # File upload
│       │   │   │   ├── CaptureResult.tsx    # Success/queued/offline/error tiles
│       │   │   │   ├── CaptureStats.tsx     # Bento grid stats
│       │   │   │   └── __tests__/
│       │   │   │       └── CaptureView.test.tsx
│       │   │   ├── search/
│       │   │   │   ├── SearchView.tsx       # Search with mode chips, filters
│       │   │   │   ├── AIAnswer.tsx         # Inline expanding AI answer row
│       │   │   │   ├── SearchInput.tsx      # Search bar
│       │   │   │   ├── ResultCard.tsx       # Generic result card
│       │   │   │   ├── ResultHero.tsx       # Hero result (high score)
│       │   │   │   ├── ResultCompact.tsx    # Compact result (rest)
│       │   │   │   └── __tests__/
│       │   │   │       └── SearchView.test.tsx
│       │   │   ├── queue/
│       │   │   │   ├── QueueView.tsx        # Queue with metrics
│       │   │   │   ├── QueueMetrics.tsx     # Queue stats
│       │   │   │   ├── ActiveJobCard.tsx    # Processing job
│       │   │   │   ├── FailedJobCard.tsx    # Failed job
│       │   │   │   ├── FailedJobExpanded.tsx # Expanded failed
│       │   │   │   ├── DoneItem.tsx         # Completed job
│       │   │   │   ├── OfflineSection.tsx   # Offline queue items
│       │   │   │   └── RetryAllBanner.tsx   # Retry all failed
│       │   │   ├── layout/
│       │   │   │   ├── BottomNav.tsx        # Tab navigation
│       │   │   │   └── Header.tsx           # Top bar (brand + security icon)
│       │   │   ├── lock/
│       │   │   │   ├── LockScreen.tsx       # Unlock gate (PRF) when locked
│       │   │   │   └── LockSetupPrompt.tsx  # One-time post-onboarding decision
│       │   │   ├── settings/
│       │   │   │   └── SecuritySheet.tsx    # Security drawer (enable/disable)
│       │   │   ├── ui/                      # shadcn/ui components
│       │   │   │   ├── button.tsx
│       │   │   │   ├── input.tsx
│       │   │   │   ├── textarea.tsx
│       │   │   │   ├── badge.tsx
│       │   │   │   ├── card.tsx
│       │   │   │   ├── separator.tsx
│       │   │   │   ├── toast.tsx
│       │   │   │   ├── toaster.tsx
│       │   │   │   ├── tabs.tsx
│       │   │   │   ├── skeleton.tsx
│       │   │   │   ├── sheet.tsx
│       │   │   │   ├── switch.tsx
│       │   │   │   └── dialog.tsx
│       │   │   ├── Onboarding.tsx           # First-run setup
│       │   │   └── ErrorBoundary.tsx        # Error catching
│       │   ├── hooks/
│       │   │   ├── useCapture.ts            # Capture with offline fallback
│       │   │   ├── useSearch.ts             # Search execution
│       │   │   ├── useAIAnswer.ts           # On-demand AI answer state machine
│       │   │   ├── useStats.ts              # Polling stats
│       │   │   ├── useQueue.ts              # Queue polling
│       │   │   ├── useQueueWS.ts            # Live job updates over WebSocket
│       │   │   ├── useServerStatus.ts       # Health polling
│       │   │   ├── useSubmitLock.ts         # Prevent double-submit
│       │   │   ├── useVaultLock.tsx          # App-lock state + token/key context
│       │   │   ├── use-toast.ts             # Toast notifications
│       │   │   └── __tests__/
│       │   │       ├── useCapture.test.tsx
│       │   │       ├── useSearch.test.tsx
│       │   │       └── useStats.test.tsx
│       │   ├── lib/
│       │   │   ├── api.ts                   # KhayalClient, type definitions
│       │   │   ├── offline.ts               # IndexedDB queue + background sync
│       │   │   ├── secureVault.ts           # WebAuthn PRF + AES-GCM primitives
│       │   │   ├── vaultStorage.ts          # IndexedDB vault record + queue
│       │   │   ├── constants.ts             # Shared constants (storage keys, limits, timeouts)
│       │   │   ├── utils.ts                 # Utility functions (cn, etc.)
│       │   │   └── __tests__/
│       │   │       ├── offline.test.ts
│       │   │       ├── api.test.ts
│       │   │       └── constants.test.ts
│       │   └── test/
│       │       ├── setup.ts                 # Vitest setup (mocks, jest-dom)
│       │       └── utils.tsx                # Render helper
│       └── e2e/
│           ├── helpers.ts                   # Playwright fixtures
│           ├── capture.spec.ts              # Capture flow E2E
│           ├── search.spec.ts               # Search flow E2E
│           ├── queue-open-note.spec.ts      # Queue -> note sheet E2E
│           └── offline.spec.ts              # Offline/PWA E2E
│
├── internal/api/ui/                         # Built PWA (generated)
│   └── static/
│       ├── index.html
│       ├── manifest.webmanifest             # PWA manifest (generated by VitePWA)
│       ├── registerSW.js                    # SW registration
│       ├── sw.js                            # Workbox service worker
│       ├── workbox-*.js                     # Workbox runtime
│       └── assets/
│           ├── index-*.css                  # Bundled CSS
│           └── index-*.js                   # Bundled JS
│
├── docs/
│   ├── SPEC.md                              # Master specification
│   ├── API/
│   │   ├── REFERENCE.md                     # API endpoint reference
│   │   ├── openapi.yaml                     # OpenAPI 3.0 spec
│   │   ├── AUTH.md                          # Authentication guide
│   │   └── PLUGINS.md                       # Plugin development
│   ├── BUILD.md                             # Build instructions
│   ├── ARCHITECTURE.md                      # System design
│   ├── TECH_STACK.md                        # Technology decisions
│   ├── PLAN.md                              # Implementation overview
│   ├── REPO_STRUCTURE.md                    # This file
│   ├── RULES.md                             # Memory management rules
│   ├── UI_SPEC.md                           # PWA implementation spec
│   ├── VAULT.md                             # Vault structure and safety
│   ├── CLI_RULES.md                         # CLI color rules
│   ├── MANUAL_TESTING.md                    # Manual testing guide
│   ├── RETROSPECTIVE.md                     # Decision history
│   ├── ui/                                  # HTML mockups
│   │   ├── khayal_search_improved.html
│   │   ├── khayal_bento_option_d.html
│   │   ├── khayal_status_tiles_final.html
│   │   ├── khayal_queue_states.html
│   │   ├── khayal_compose_boxes.html
│   │   └── khayal_pwa_2025.html
│   └── phases/
│       ├── phase-1-foundation.md
│       ├── phase-2-api.md
│       ├── phase-3-worker.md
│       ├── phase-4-llm.md
│       ├── phase-5-cli.md
│       ├── phase-6-pwa.md
│       └── phase-7-polish.md
│
├── go.mod
├── go.sum
├── .gitignore
├── LICENSE                                  # AGPLv3
└── config.example.yaml                      # Safe to commit, no secrets
```

---

## Directory Purpose

### `cmd/`

Two separate CLI binaries:
- `cmd/khayal/` — Server admin CLI (khayal start, stop, status, etc.)
- `cmd/kl/` — Client CLI (kl capture, search, status, etc.)

### `internal/`

Private application code. Not importable by external packages.

| Directory | Purpose |
|-----------|---------|
| `api/` | HTTP handlers, middleware, routing |
| `constants/` | Shared constants (retry, milestones, prompts) |
| `worker/` | Background job processing |
| `ingest/` | Content processing (text, image, article) |
| `llm/` | AI integration |
| `vault/` | Markdown file writing |
| `queue/` | SQLite database operations |
| `queue/` | Job queue, FTS5 + semantic search, entity/chunk stores |
| `backup/` | Encrypted backup (age) and additive-merge restore |
| `events/` | In-process pub-sub hub for realtime job updates |
| `connections/` | Proactive connections |
| `config/` | Configuration management |
| `version/` | Version info |

### `external/react/`

Frontend PWA project. Built with Vite + React + Tailwind + shadcn/ui.

| Directory | Purpose |
|-----------|---------|
| `src/components/capture/` | Capture UI (text, url, image, result, stats) |
| `src/components/search/` | Search UI (view, input, results, AI answer row) |
| `src/components/queue/` | Queue display (jobs, metrics) |
| `src/components/layout/` | Navigation (bottom nav, header) |
| `src/components/ui/` | shadcn/ui components |
| `src/hooks/` | Custom React hooks |
| `src/lib/` | API client, offline queue, constants |
| `src/test/` | Vitest setup and utilities |
| `src/sw.ts` | Service worker (Workbox + background sync) |
| `e2e/` | Playwright E2E tests |

### `internal/api/ui/static/`

Built PWA output. Generated by `npm run build` in `external/react/`. Embedded into Go binary at compile time.

---

## File Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Go source | `snake_case.go` | `config.go`, `auth.go` |
| Go test | `*_test.go` | `config_test.go` |
| React components | `PascalCase.tsx` | `CaptureView.tsx`, `SearchView.tsx` |
| React hooks | `camelCase.ts` | `useCapture.ts`, `useSearch.ts` |
| React utilities | `camelCase.ts` | `api.ts`, `offline.ts`, `constants.ts` |
| Config | `kebab-case.yaml` | `config.example.yaml` |

---

## Key Interfaces

### LLM (internal/llm/interface.go)

```go
type LLM interface {
    Embed(text string) ([]float32, error)
    EmbedBatch(texts []string) ([][]float32, error)
    Generate(prompt string) (string, error)
    GenerateWithSystem(system, user string) (string, error)
    DescribeImage(path string) (string, error)
    Ping() error
    Type() string
}

// LLMExt adds the enrichment extractions.
type LLMExt interface {
    LLM
    ExtractTags(content, bucket string) ([]string, error)
    Summarize(content, bucket string) (string, error)
    ExtractKeyIdeas(content, bucket string) ([]string, error)
    ExtractEntities(content, bucket string) (EntityResult, error)
}
```

### Chunking (internal/chunk/chunk.go)

```go
type Options struct {
    TargetWords   int // ~175
    MinWords      int // fragments below this merge into the previous chunk
    OverlapWords  int // words carried between consecutive chunks
}

type Chunk struct {
    Content   string
    WordCount int
}

func DefaultOptions() Options
func ChunkText(text string, opts Options) []Chunk
```

### Queue (internal/queue/queue.go)

```go
type JobStore interface {
    CreateJob(ctx context.Context, job *Job) error
    GetJob(ctx context.Context, id string) (*Job, error)
    UpdateJob(ctx context.Context, job *Job) error
    UpdateJobStatus(ctx context.Context, id, status string) error
    SearchKeyword(...) ([]SearchResult, error)
    SearchSemantic(...) ([]SearchResult, error)
    SaveChunk(ctx context.Context, notePath string, chunkIdx int,
        content string, embedding []float32) error
    DeleteChunksByNote(ctx context.Context, notePath string) error
    CountChunks(ctx context.Context, notePath string) (int, error)
    SaveEntities(ctx context.Context, notePath string, ents NoteEntities) error
    DeleteEntities(ctx context.Context, notePath string) error

    // Connections (phase 2)
    GetEntitiesByNote(ctx context.Context, notePath, entityType string) ([]string, error)
    GetNotesByEntity(ctx context.Context, entityValue, entityType string,
        cutoff time.Time) ([]EntityMatch, error)
    CountNotesByEntity(ctx context.Context, entityValue, entityType string,
        cutoff time.Time, excludePath string) (int, error)
    TopSimilarChunks(ctx context.Context, embedding []float32, limit int,
        minScore float64, cutoff time.Time, excludePath string) ([]RawChunkMatch, error)
    UpdateJobResult(ctx context.Context, jobID string, result json.RawMessage) error
    LinkConnectionsJob(ctx context.Context, ingestJobID, connJobID string) error
}
```

`ReplaceChunks(notePath, rows)` atomically replaces a note's whole chunk
set (used by ingest and `khayal reindex`). The `entities` table has two
independent writers: IndexNote/UpdateNoteIndex own `title`/`tag` rows,
SaveEntities owns the six enrichment types (`person`, `amount`, `date`,
`place`, `org`, `url`) — each deletes only its own types.

### Vault (internal/vault/writer.go)

```go
func (w *Writer) WriteNote(note *Note, jobID string) (string, error)
func (w *Writer) UpdateNote(notePath string, note *Note) error
func (w *Writer) DeleteNote(notePath string) error
func (w *Writer) SetConnections(notePath string, targets []string) error // wikilink frontmatter
```

### API Client (internal/api/client/client.go)

```go
type Client struct {
    // opaque
}

func New(baseURL, token string) *Client

// Capture
func (c *Client) CaptureText(ctx context.Context, content string) (*CaptureResponse, error)
func (c *Client) CaptureURL(ctx context.Context, url string) (*CaptureResponse, error)
func (c *Client) CaptureImage(ctx context.Context, path, note string) (*CaptureResponse, error)

// Search
func (c *Client) Search(ctx context.Context, query string, opts ...SearchOptions) (*SearchResponse, error)

// Queue
func (c *Client) ListQueue(ctx context.Context, filter QueueFilter) (*QueueListResponse, error)
func (c *Client) GetJob(ctx context.Context, id string) (*Job, error)
func (c *Client) RetryJob(ctx context.Context, id string) (*Job, error)
func (c *Client) DiscardJob(ctx context.Context, id string) error

// Health
func (c *Client) Health(ctx context.Context) (*HealthResponse, error)
```

---

## API Endpoints

| Method | Path | Handler |
|--------|------|---------|
| POST | /v1/capture | capture.go |
| GET | /v1/search | search.go |
| GET | /v1/health | health.go |
| GET | /v1/stats | stats.go |
| GET | /v1/queue | queue.go |
| GET | /v1/queue/:id | queue.go |
| POST | /v1/queue/:id/retry | queue.go |
| POST | /v1/queue/:id/discard | queue.go |
| GET | /\* | static.go (SPA) |

---

## CLI Commands

| Command | File | Description |
|---------|------|-------------|
| `kl` | root.go | Root (capture) |
| `kl capture` | capture.go | Capture text/url/image |
| `kl search` | search.go | Search knowledge base |
| `kl recent` | recent.go | Recent captures |
| `kl stats` | stats.go | Vault statistics |
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
    retries INTEGER DEFAULT 0,
    result TEXT,
    connections_job_id TEXT
);
```

`result` stores JSON payloads for enrichment job types (currently the
connections output); `connections_job_id` links an ingest job to the
connections job chained after it.

### notes_fts (FTS5)

```sql
CREATE VIRTUAL TABLE notes_fts USING fts5(
    note_path,
    content,
    title,
    tags
);
```

### embeddings table (removed in v1.1)

The legacy per-job `embeddings` table was removed in v1.1. The `chunks`
table below is the canonical vector store, and the table itself is dropped
from existing databases on startup (`DROP TABLE IF EXISTS embeddings`).

### chunks table

```sql
CREATE TABLE chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_path TEXT NOT NULL,
    chunk_idx INTEGER NOT NULL,
    content TEXT NOT NULL,
    embedding BLOB NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_chunks_note ON chunks(note_path);
```

One row per paragraph-aligned chunk (~175 words, 35-word overlap by
default — see the `search:` config section). Semantic search scores each
chunk and returns its parent note with the best-scoring chunk as the
excerpt. Writes go through transactional `ReplaceChunks`, so a note's
chunk set is replaced atomically.

Note: capture-time ingest chunks only the raw submitted content, while
`khayal reindex` chunks the full note body (including generated Summary /
Key Ideas sections) — see [ADR-0001](adr/0001-chunk-coverage-asymmetry-capture-vs-reindex.md).

### entities table

```sql
CREATE TABLE entities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_path TEXT NOT NULL,
    chunk_idx INTEGER,
    entity_type TEXT NOT NULL,
    entity_value TEXT NOT NULL,
    created_at TEXT NOT NULL
);
```

Two independent writers own disjoint `entity_type` sets:
IndexNote/UpdateNoteIndex write `title` and `tag` rows; SaveEntities
writes the six enrichment types (`person`, `amount`, `date`, `place`,
`org`, `url`). Each deletes only its own types, so re-indexing never
wipes enrichment and vice versa.

### stats_cache table

```sql
CREATE TABLE stats_cache (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL
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
- Ollama: Required for LLM features
- No CGO required (uses modernc.org/sqlite)
