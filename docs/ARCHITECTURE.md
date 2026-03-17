# Khayal Architecture

> High-level system design. Updated: 2026-03-17

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
- Static file serving (PWA)

### Worker (`internal/worker/`)

- Job processing (concurrent, configurable)
- Retry logic (exponential backoff, max 3)
- Crash recovery (reset stuck jobs)
- Error handling (mark failed, cleanup)

### Ingest (`internal/ingest/`)

- Text: Tag extraction, summarization
- Image: LLM description, OCR
- Article: Scraping, summarization

### LLM (`internal/llm/`)

- Interface definition
- Ollama (primary)
- Groq (fallback)
- OpenAI (fallback)
- Graceful degradation

### Vault (`internal/vault/`)

- Markdown file writing
- Frontmatter generation
- Media file management
- Path resolution

### Queue (`internal/queue/`)

- SQLite-based job queue
- FTS5 search index
- Embedding storage

### CLI (`cli/`)

- User interface
- Command handling
- Output formatting

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
├── inbox/               ← New captures
└── inbox/media/         ← Images
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

| Error | Code | HTTP Status |
|-------|------|-------------|
| Missing/invalid token | UNAUTHORIZED | 401 |
| Invalid request | INVALID_REQUEST | 400 |
| Vault write failed | VAULT_ERROR | 500 |
| LLM unavailable | LLM_ERROR | 503 |
| Job not found | NOT_FOUND | 404 |

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
