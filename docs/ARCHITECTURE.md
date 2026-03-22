# Khayal Architecture

> High-level system design. Updated: 2026-03-17

## Two Binaries

Khayal is distributed as two separate binaries:

| Binary | Purpose | Dependencies |
|--------|---------|--------------|
| `khayal` | Server + Worker + PWA | Ollama, SQLite |
| `kl` | Thin HTTP client | None (just calls khayal) |

### khayal

Full server binary with:
- API server
- Worker pool (job processing)
- Ingest pipeline (text, image, article)
- LLM clients (Ollama, Groq, OpenAI)
- PWA (embedded via embed.FS)
- Vault writer
- Queue (SQLite)
- Config loader
- Dependency checker

```bash
khayal init       # First-run setup
khayal start      # Start server + worker
```

### kl

Lightweight HTTP client (no server, no database):

```bash
kl init           # Setup kl.yaml with token
kl "thought"      # Capture text
kl --url ...     # Capture URL
kl --image ...   # Capture image
kl search ...    # Search
kl status        # Show queue
```

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Khayal System                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │   kl CLI    │    │   PWA        │    │   Future     │       │
│  │  (Cobra)    │    │  (React)     │    │  Interfaces  │       │
│  └──────┬───────┘    └──────┬───────┘    └──────────────┘       │
│         │                   │                                    │
│         │  X-Khayal-Token   │                                    │
│         ▼                   ▼                                    │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │                    API Server                             │    │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐         │    │
│  │  │  Capture   │  │   Search   │  │   Queue    │         │    │
│  │  └─────┬──────┘  └─────┬──────┘  └────────────┘         │    │
│  │        │               │                                 │    │
│  │        ▼               ▼                                 │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │              Worker Pool                         │    │    │
│  │  │   ┌─────────┐  ┌─────────┐  ┌─────────┐        │    │    │
│  │  │   │Worker 1│  │Worker 2│  │Worker N │        │    │    │
│  │  │   └────┬────┘  └────┬────┘  └────┬────┘        │    │    │
│  │  └────────│───────────┼───────────┼──────────────┘    │    │
│  │           │           │          │                    │    │
│  │           ▼           ▼          ▼                    │    │
│  │  ┌──────────────────────────────────────────────┐    │    │
│  │  │              Ingest Pipeline                 │    │    │
│  │  │   ┌─────────┐  ┌─────────┐  ┌─────────┐     │    │    │
│  │  │   │  Text   │  │  Image  │  │ Article │     │    │    │
│  │  │   └────┬────┘  └────┬────┘  └────┬────┘     │    │    │
│  │  │        │            │            │           │    │    │
│  │  │        └────────────┼────────────┘           │    │    │
│  │  │                     ▼                        │    │    │
│  │  │              ┌────────────┐                   │    │    │
│  │  │              │    LLM     │                   │    │    │
│  │  │              │ (Ollama)   │                   │    │    │
│  │  │              └────────────┘                   │    │    │
│  │  └──────────────────────────────────────────────┘    │    │
│  │                          │                              │    │
│  │                          ▼                              │    │
│  │  ┌──────────────────────────────────────────────┐    │    │
│  │  │              Vault (Markdown)                │    │    │
│  │  │         ~/Documents/brain (user config)      │    │    │
│  │  └──────────────────────────────────────────────┘    │    │
│  │                                                           │    │
│  │  ┌──────────────────────────────────────────────┐    │    │
│  │  │              SQLite DB                        │    │    │
│  │  │    Job Queue + Embeddings + Metadata         │    │    │
│  │  └──────────────────────────────────────────────┘    │    │
│  │                                                           │    │
│  └──────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## Core Principles

```
Capture  → zero friction, any device
Process  → immediate, local, private
Search   → fast, semantic + keyword
Store    → plain markdown, yours forever
```

## Data Flow

### Capture (Text - Synchronous)

```
User Input → API Server → Vault Writer → Search Index → Response
              (3ms)        (done)        (immediate)
```

### Capture (Image/URL - Asynchronous)

```
User Input → API Server → Job Queue → Worker → Ingest → Vault
                (queued)              (async)   (10-15s) (done)
```

### Search

```
Query → API Server → Keyword Search (FTS5)
                  → Semantic Search (embeddings)
                  → Hybrid Combine → Results
```

## Component Responsibilities

### API Server (`internal/api/`)

- HTTP request handling
- Authentication (token validation)
- Request logging
- Job queue management
- Search orchestration
- **Static file serving** (PWA via `embed.FS`)
- **SPA fallback** (non-API routes serve index.html)

### API Client (`internal/api/client/`)

Shared typed Go client used by:
- `kl` CLI
- Future interfaces (browser extension, iOS, Android)

```go
import "github.com/rawnaqs/khayal/internal/api/client"

c := client.New("http://localhost:1133", "your-token")

// Capture
resp, _ := c.Capture(ctx, client.CaptureRequest{
    Type:    "text",
    Content: "my thought",
})

// Search
results, _ := c.Search(ctx, "query", client.SearchOptions{Limit: 10})

// Queue
jobs, _ := c.ListQueue(ctx, client.QueueFilter{Status: "pending"})
```

### Worker (`internal/worker/`)

- Job processing (concurrent, configurable)
- **Atomic fetch+lock pattern** prevents duplicate job processing
- **120-second timeout per job** prevents hung jobs
- **Job status flow**: `pending → queued → processing → done/failed`
- Retry logic (exponential backoff, max 3)
- Crash recovery (reset stuck jobs to pending)
- Error handling (mark failed, cleanup)

### Ingest (`internal/ingest/`)

- Text: Tag extraction, summarization
- Image: LLM description, OCR
- Article: Scraping, summarization

### LLM (`internal/llm/`)

- Interface definition
- Ollama (primary) with **concurrency semaphore** (default 4)
- Groq (fallback)
- OpenAI (fallback)
- Graceful degradation
- **Token truncation** per content type (text/image/article)

### Vault (`internal/vault/`)

- Markdown file writing
- Frontmatter generation (YAML validated before write)
- Media file management
- Path resolution
- **Safety features:**
  - Atomic writes (temp file + rename)
  - File locking to prevent race conditions with Obsidian
  - mtime check to detect external edits
  - UTF-8 validation on all LLM output
  - Wikilink verification before writing
  - Hard caps on frontmatter list fields

### Queue (`internal/queue/`)

- SQLite-based job queue with **WAL mode** and **busy_timeout**
- **Atomic fetch+lock** with `UPDATE...RETURNING`
- **Lock retry logic** for concurrent writes
- FTS5 search index
- Embedding storage

### CLI (`cli/`)

- User interface
- Command handling
- Output formatting

## LLM Adapter Pattern

The LLM layer uses the **adapter pattern** for extensibility:

```
┌─────────────────────┐
│   LLM Interface     │  (internal/llm/interface.go)
│   ─────────────     │
│ Embed()            │
│ Generate()         │
│ DescribeImage()    │
│ Ping()             │
│ Type()             │
└──────────┬──────────┘
           │
  ┌────────┼────────┐
  │        │        │
  ▼        ▼        ▼
Ollama   Groq    OpenAI
(primary)(fallback)(fallback)
```

**Adding a new provider:**
1. Create `internal/llm/<provider>.go`
2. Implement the `LLM` interface
3. Add to factory

No other code changes required. See `docs/phases/phase-4-llm.md` for details.

## Security Model

| Layer | Rule |
|-------|------|
| Bind | 127.0.0.1 (never 0.0.0.0 by default) |
| Auth | X-Khayal-Token header on every request |
| Token | 32-byte hex, auto-generated on first run |
| Config | 600 permissions |
| Logging | Never log token or request body |
| Media | Audio/video stored in ~/.config/khayal/media/ |

## Data Locations

```
~/.config/khayal/
├── config.yaml          ← Main config (600 permissions)
├── khayal.db            ← SQLite: queue + embeddings
├── logs/
│   └── khayal.log       ← Request + system logs
└── media/               ← Raw audio/video (not in vault)

<user-vault-path>/       ← User's markdown vault
├── khayal/             ← New captures
└── khayal/media/       ← Images
```

## API Base

```
Base URL: http://<host>:<port>/v1/
Auth: X-Khayal-Token: <token>
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | /v1/capture | Capture text, URL, or image |
| GET | /v1/search | Keyword + semantic search |
| GET | /v1/health | Dependency status + queue counts |
| GET | /v1/queue | Job list with pagination |
| GET | /v1/queue/:id | Single job status |

## Capture Interface Model

All capture interfaces are independent, pluggable clients:

```
┌──────────────────────────────────────┐
│        Capture Interfaces            │
│   kl CLI   │   PWA   │   Future     │
└────────────┴─────────┴──────────────┘
              │ HTTP + Token
              ▼
┌──────────────────────────────────────┐
│          khayal server               │
│   API → Worker → Ingest → Vault     │
└──────────────────────────────────────┘
```

No interface implements capture logic - that lives exclusively in the server.

## Error Handling

See SPEC.md Error Taxonomy for full list. Common codes:

| Error | Code | HTTP Status |
|-------|------|-------------|
| Missing/invalid token | AUTH_001 / AUTH_002 | 401 |
| Invalid request | CAPTURE_004 / SEARCH_003 | 400 |
| Vault write failed | VAULT_002 | 500 |
| Ollama unavailable | LLM_001 | 500 |
| Job not found | QUEUE_002 | 404 |

## Dependencies

### Required

- Ollama (for full LLM functionality)
- Go 1.22+

### Optional

- ffmpeg (v1.1+ for video)
- yt-dlp (v1.2+ for YouTube)

## Design System

All UI uses `rawnaqs/theme`:

- CLI: `github.com/rawnaqs/theme/theme.go` + `custom/go/styles.go`
- Web: `github.com/rawnaqs/theme/theme.css`

Never define colors or typography directly in Khayal code.
