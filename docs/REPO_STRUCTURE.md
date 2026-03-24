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
│   │   ├── health.go               # GET /v1/health
│   │   ├── queue.go                # GET /v1/queue, queue operations
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
│   │   └── article.go             # Article scraping, summarization
│   │
│   ├── llm/
│   │   ├── interface.go            # LLM interface definition
│   │   ├── ollama.go              # Ollama client
│   │   ├── groq.go                # Groq client
│   │   ├── openai.go              # OpenAI client
│   │   └── factory.go             # LLM factory
│   │
│   ├── vault/
│   │   └── writer.go               # Markdown writer, frontmatter
│   │
│   ├── queue/
│   │   └── queue.go                    # SQLite job queue, FTS5, embeddings
│   │
│   ├── search/
│   │   ├── keyword.go                  # FTS5 + porter stemming + BM25
│   │   ├── semantic.go                 # Vector similarity search
│   │   ├── hybrid.go                  # RRF merge (k=60)
│   │   ├── date.go                     # Date range filtering
│   │   └── sync.go                     # mtime check + re-index stale
│   │
│   ├── connections/                    # Proactive connections (v1.1+)
│   │   ├── engine.go                  # Orchestrates all types
│   │   ├── similar.go                 # Semantic similarity
│   │   ├── entity.go                  # Person + amount lookup
│   │   ├── revisit.go                  # Revisit detection
│   │   ├── followup.go                 # Follow-up detection
│   │   └── contradiction.go            # LLM contradiction check
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
├── docker-compose.yml                   # khayal + ollama
├── config.example.yaml                  # Full config reference
├── Dockerfile                           # Go-only multi-stage
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
│       │   │   │   └── Header.tsx           # Top bar
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
│       │   │   │   └── dialog.tsx
│       │   │   ├── Onboarding.tsx           # First-run setup
│       │   │   └── ErrorBoundary.tsx        # Error catching
│       │   ├── hooks/
│       │   │   ├── useCapture.ts            # Capture with offline fallback
│       │   │   ├── useSearch.ts             # Search execution
│       │   │   ├── useStats.ts              # Polling stats
│       │   │   ├── useQueue.ts              # Queue polling
│       │   │   ├── useServerStatus.ts       # Health polling
│       │   │   ├── useSubmitLock.ts         # Prevent double-submit
│       │   │   ├── use-toast.ts             # Toast notifications
│       │   │   └── __tests__/
│       │   │       ├── useCapture.test.tsx
│       │   │       ├── useSearch.test.tsx
│       │   │       └── useStats.test.tsx
│       │   ├── lib/
│       │   │   ├── api.ts                   # KhayalClient, type definitions
│       │   │   ├── offline.ts               # IndexedDB queue + background sync
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
| `search/` | Search algorithms |
| `connections/` | Proactive connections |
| `config/` | Configuration management |
| `version/` | Version info |

### `external/react/`

Frontend PWA project. Built with Vite + React + Tailwind + shadcn/ui.

| Directory | Purpose |
|-----------|---------|
| `src/components/capture/` | Capture UI (text, url, image, result, stats) |
| `src/components/search/` | Search UI (view, input, results) |
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
