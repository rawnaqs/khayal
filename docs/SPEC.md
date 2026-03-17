# Khayal — v1 Project Specification

> Your private treasury of thought. Local, secure, yours.

---

## Identity

| | |
|---|---|
| **Tool name** | Khayal |
| **CLI short** | `kl` (short for khayal, short for knowledge) |
| **Org** | Rawnaqs — the luster of craftsmanship |
| **Module path** | `github.com/rawnaqs/khayal` |
| **License** | AGPLv3 |
| **Language** | Go 1.22+ |
| **Theme** | `github.com/rawnaqs/theme` |

---

## What It Is

A local-first, privacy-focused second brain. Capture anything — text, images, articles, URLs. Process it locally with your own LLM. Search it semantically and by keyword. Your data never leaves your machine.

## What It Is Not

- Not a chat interface over your notes
- Not a graph database
- Not an Obsidian replacement
- Not a SaaS, subscription, or cloud service

---

## Core Philosophy

```
Capture  → zero friction, any device
Process  → immediate, local, private
Search   → fast, semantic + keyword
Store    → plain markdown, yours forever
```

---

## v1 Scope

### In v1
- Text / quick thought capture
- Image / screenshot capture
- Article / web URL capture
- Keyword + semantic hybrid search
- Search results with relevant excerpt
- PWA (Templ + HTMX, dark only, embedded in binary)
- CLI (`khayal` server + `kl` client)
- macOS + Linux
- Single binary + Docker Compose both supported
- Token auth on every request
- Request logging
- Ollama primary, Groq + OpenAI as fallbacks
- Installer checks dependencies and guides user
- GitHub releases + Homebrew formula
- AGPLv3

### Explicitly Out of v1
- Voice notes
- PDF ingestion
- YouTube / video ingestion
- Browser extension
- Raycast extension
- Mobile app
- iOS Shortcuts
- Graph connections / wikilinks
- Windows support
- Setup wizard UI (non-technical users)
- Multi-user
- Chat over vault (Open WebUI integration)

---

## Phases After v1

```
v1.1  → Voice notes + PDF ingestion
v1.2  → YouTube / video ingestion  
v1.3  → Browser extension (github.com/rawnaqs/khayal-browser)
v1.4  → Raycast extension (github.com/rawnaqs/khayal-raycast)
v1.5  → iOS Shortcuts (github.com/rawnaqs/khayal-ios)
v2.0  → Setup wizard UI for non-technical users
v2.1  → Graph connections, backlinks
v2.2  → Windows support
v2.3  → Mobile app (github.com/rawnaqs/khayal-mobile)
```

---

## Architecture

### Capture Interface Philosophy — Lego Model

All capture interfaces are independent, pluggable clients. They speak HTTP to the server. No interface implements capture logic — that lives exclusively in the server.

```
┌──────────────────────────────────────────────────────┐
│                 Capture Interfaces                   │
│  kl CLI  │  PWA  │  Browser  │  Raycast  │  Mobile  │
│          │ HTMX  │  Ext(v2)  │  Ext(v2)  │  (v3)   │
└──────────┴───────┴───────────┴───────────┴──────────┘
                    │ HTTP + X-Khayal-Token
                    │ POST /v1/capture
                    │ GET  /v1/search
                    │ GET  /v1/queue/:id
                    ▼
┌──────────────────────────────────────────────────────┐
│                  khayal server                       │
│   API layer → capture core → worker → ingest        │
└──────────────────────────────────────────────────────┘
```

### Interface Repos

```
github.com/rawnaqs/theme               ← shared design system (colors, typography, styles)
github.com/rawnaqs/khayal              ← core server + kl CLI + PWA (v1)
github.com/rawnaqs/khayal-browser      ← browser extension (v1.3)
github.com/rawnaqs/khayal-raycast      ← Raycast extension (v1.4)
github.com/rawnaqs/khayal-ios          ← iOS Shortcuts (v1.5)
github.com/rawnaqs/khayal-mobile       ← mobile app (v2.3)
```

---

## Project Structure

```
khayal/
├── cmd/
│   └── khayal/
│       └── main.go              ← single binary entry point
├── internal/
│   ├── api/
│   │   ├── server.go            ← HTTP server, middleware, auth, logger
│   │   ├── capture.go           ← POST /v1/capture
│   │   ├── search.go            ← GET /v1/search
│   │   ├── health.go            ← GET /v1/health
│   │   └── queue.go             ← GET /v1/queue, GET /v1/queue/:id
│   ├── worker/
│   │   └── worker.go            ← job processor, configurable concurrency, exponential backoff
│   ├── ingest/
│   │   ├── text.go              ← text processing
│   │   ├── image.go             ← LLaVA/moondream + OCR
│   │   └── article.go           ← scrape + summarize
│   ├── llm/
│   │   ├── interface.go         ← LLM interface definition
│   │   ├── ollama.go            ← Ollama client (primary)
│   │   ├── groq.go              ← Groq fallback
│   │   └── openai.go            ← OpenAI fallback
│   ├── vault/
│   │   └── writer.go            ← markdown writer, vault-agnostic
│   ├── queue/
│   │   └── queue.go             ← SQLite job queue
│   ├── search/
│   │   ├── keyword.go           ← full-text search
│   │   └── semantic.go          ← vector similarity search
│   └── config/
│       └── config.go            ← config loader, fail hard on error
├── cli/
│   ├── root.go                  ← Cobra root command
│   ├── capture.go               ← kl "thought", --url, --image
│   ├── search.go                ← kl search (Glamour rendering)
│   ├── status.go                ← kl status (Bubble Tea TUI)
│   ├── init.go                  ← kl init (Huh wizard)
│   └── config.go                ← kl config set key value
├── ui/
│   ├── templates/               ← Templ files
│   │   ├── layout.templ
│   │   ├── capture.templ
│   │   └── search.templ
│   └── static/
│       ├── style.css            ← imports rawnaqs/theme generated/css/variables.css
│       └── offline.js           ← IndexedDB offline queue (~50 lines)
├── install/
│   └── check.go                 ← dependency checker + guidance
├── .github/
│   └── workflows/
│       ├── ci.yml               ← test + vet + lint on every PR
│       └── release.yml          ← goreleaser on tag push
├── docker-compose.yml
├── .goreleaser.yml
├── config.example.yaml          ← safe to commit, no secrets
├── .gitignore
├── LICENSE                      ← AGPLv3
├── README.md
└── CONTRIBUTING.md
```

---

## Design System

Khayal uses `github.com/rawnaqs/theme` — the shared Rawnaqs design system. Never define colors or typography directly in Khayal code.

```
Theme dependency flow:

rawnaqs/theme/tokens/tokens.json   ← source of truth
        │
        ▼ generate.py
rawnaqs/theme/generated/
  ├── go/theme.go        ← imported in CLI (Lip Gloss constants)
  ├── css/variables.css  ← imported in ui/static/style.css
  └── python/theme.py    ← imported in any Python tooling

rawnaqs/theme/custom/
  └── go/styles.go       ← pre-built Lip Gloss styles, used in cli/
```

### CLI color usage

```go
import (
    theme  "github.com/rawnaqs/theme/generated/go"
    styles "github.com/rawnaqs/theme/custom/go"
)

// Use pre-built styles
fmt.Println(styles.SuccessStyle.Render("✓ saved"))
fmt.Println(styles.CaptureOK("saved", "#react", "3ms"))
fmt.Println(styles.CaptureQueued("image", "abc123"))
```

### PWA CSS usage

```css
/* ui/static/style.css */
@import url('https://raw.githubusercontent.com/rawnaqs/theme/main/generated/css/variables.css');

body {
  background: var(--bg-base);
  color: var(--text-primary);
  font-family: var(--font-mono);
}
```

---

## Binary

Single binary — `khayal` — does everything:

```bash
khayal init       # first-run setup, generates config + token
khayal start      # starts server + worker
khayal version    # prints version
```

CLI client — `kl` — is a separate subpackage compiled into the same binary or as standalone:

```bash
kl "thought"
kl --url https://...
kl --image ~/file.png
kl search "query"
kl status
kl init
kl config set token abc123
kl config set host http://100.x.x.x:7766
```

---

## Data Directory

```
~/.config/khayal/
├── config.yaml          ← main config (permissions: 600)
├── khayal.db            ← SQLite: job queue + embeddings
└── logs/
    └── khayal.log       ← request + system logs
```

Vault lives wherever the user points it — completely separate from `~/.config/khayal/`.

---

## Config

Format: YAML only. Behavior on missing/malformed: fail hard with actionable error message.

```yaml
# config.yaml — never commit this file
# Copy from config.example.yaml and fill in

vault:
  path: ~/Documents/brain              # required — where markdown notes are written
  inbox_dir: inbox                     # relative to vault root
  media:
    default_dir: inbox/media           # fallback for unspecified types
    strategy:
      image: vault                     # saved inside vault, linked relatively
      pdf: vault                       # saved inside vault
      audio: config                    # saved in ~/.config/khayal/media/
      video: config                    # transcript goes to vault, raw file stays here

server:
  host: 127.0.0.1                      # never 0.0.0.0 by default
  port: 7766
  token: ""                            # auto-generated on first run if empty
  log_file: ~/.config/khayal/logs/khayal.log

llm:
  provider: ollama                     # ollama | groq | openai
  ollama_host: http://localhost:11434
  embed_model: nomic-embed-text
  text_model: llama3.2:3b
  vision_model: moondream
  fallback_provider: ""                # groq | openai | "" (none)
  fallback_api_key: ""

worker:
  max_workers: 1                       # configurable concurrency
  max_retries: 3                       # then mark permanently failed
  retry_backoff: exponential           # immediate | fixed | exponential

db:
  path: ~/.config/khayal/khayal.db
```

### First Run

```bash
$ khayal start
✗ Config file not found at ~/.config/khayal/config.yaml
  Run 'khayal init' to generate a default config

$ khayal init
Created ~/.config/khayal/config.yaml (permissions: 600)
Created ~/.config/khayal/logs/
Generated token: a3f9c2e1d7b4... (save this — shown once)

Edit ~/.config/khayal/config.yaml to set your vault path
Then run 'khayal start'
```

### Malformed Config

```bash
$ khayal start
✗ Config error: vault.path is required
  Line 4 in ~/.config/khayal/config.yaml
  Fix the error and restart
```

---

## Security Model

| Layer | Rule |
|---|---|
| Default bind | `127.0.0.1` — never `0.0.0.0` unless explicitly set |
| Token | Auto-generated 32-byte hex on first run, printed once, stored in config.yaml |
| Auth | `X-Khayal-Token` header required on every request |
| Logging | Timestamp, method, path, status, latency — never logs token or request body |
| Config permissions | Written as `600` on creation |
| `.gitignore` | `config.yaml`, `khayal.db`, `*.log` auto-ignored |
| Tailscale | User's responsibility — documented in README |
| Media outside vault | Raw audio/video stored in `~/.config/khayal/media/` — never in vault |

---

## API — v1

Base: `/v1/`
Auth: `X-Khayal-Token: <token>` on every request

### Endpoints

```
POST   /v1/capture          → capture anything
GET    /v1/search           → keyword + semantic search
GET    /v1/health           → dependency status + queue counts
GET    /v1/queue            → job list with pagination
GET    /v1/queue/:id        → single job status
```

Queue retry + delete → v2.

---

### POST /v1/capture

**Text / URL — JSON**

```json
POST /v1/capture
Content-Type: application/json

{ "type": "text", "content": "useEffect cleanup runs after every render" }
{ "type": "url",  "content": "https://blog.example.com/post" }
```

**Image — Multipart**

```
POST /v1/capture
Content-Type: multipart/form-data

type=image
file=<binary>
note="optional context"         ← becomes frontmatter context + first paragraph
```

**Response — text (fast, done immediately)**

```json
{
  "id": "abc123",
  "type": "text",
  "status": "done",
  "note_path": "inbox/2024-03-16-thought.md",
  "created_at": "2024-03-16T14:23:00Z"
}
```

**Response — image / url (queued)**

```json
{
  "id": "def456",
  "type": "image",
  "status": "processing",
  "note_path": "inbox/2024-03-16-image.md",
  "created_at": "2024-03-16T14:23:00Z"
}
```

---

### GET /v1/search

```
GET /v1/search?q=distributed+systems&limit=10&mode=hybrid&excerpt_length=200
```

Parameters:

| Param | Required | Default | Options |
|---|---|---|---|
| `q` | yes | — | any string |
| `limit` | no | 10 | max 50 |
| `mode` | no | hybrid | hybrid \| keyword \| semantic |
| `excerpt_length` | no | 200 | max 500 chars |

Response:

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

---

### GET /v1/health

```json
{
  "status": "ok",
  "version": "0.1.0",
  "dependencies": {
    "ollama": { "status": "ok", "host": "http://localhost:11434" },
    "vault":  { "status": "ok", "path": "~/Documents/brain" },
    "db":     { "status": "ok", "path": "~/.config/khayal/khayal.db" }
  },
  "queue": {
    "pending":    2,
    "processing": 1,
    "done":       147,
    "failed":     0
  }
}
```

---

### GET /v1/queue

```
GET /v1/queue?status=pending&limit=20&offset=0
```

Parameters:

| Param | Default | Options |
|---|---|---|
| `status` | all | all \| pending \| processing \| done \| failed |
| `limit` | 20 | max 100 |
| `offset` | 0 | — |

Response:

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

---

### GET /v1/queue/:id

```json
{
  "id": "abc123",
  "type": "image",
  "status": "done",
  "note_path": "inbox/2024-03-16-image.md",
  "created_at": "2024-03-16T14:23:00Z",
  "processed_at": "2024-03-16T14:23:12Z",
  "error": null
}
```

---

### Error Responses

```json
400 { "error": "missing required field: content", "code": "INVALID_REQUEST" }
401 { "error": "invalid or missing token",        "code": "UNAUTHORIZED" }
500 { "error": "failed to write note to vault",   "code": "VAULT_ERROR" }
```

---

## Worker

- Configurable concurrency via `worker.max_workers` in config
- Single goroutine per worker, jobs processed serially within each worker
- Exponential backoff on failure
- Max 3 retries then permanently failed
- On permanent failure: note deleted from vault, media file deleted, job marked failed in DB
- On startup: reset any jobs stuck in `processing` state (crash recovery)

### Processing Times (M2 Mac Air)

| Type | Time |
|---|---|
| Text | ~3s |
| Image | ~10s |
| Article / URL | ~15s |

---

## Note Structures

All frontmatter keys: `snake_case`

### Text — done

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

### Article / URL — done

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

### Image — processing (before worker)

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

### Image — done (after worker)

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

### Permanent failure

Note and media file deleted from vault. Failure tracked in `khayal.db` only, visible via `GET /v1/queue?status=failed`.

---

## LLM Interface

```go
type LLM interface {
    Embed(text string) ([]float32, error)
    Generate(prompt string) (string, error)
    DescribeImage(path string) (string, error)
}
```

Implementations: `OllamaClient` (primary), `GroqClient` (fallback), `OpenAIClient` (fallback).

Fallback activates if Ollama is unreachable. If no fallback configured and Ollama is down — raw note is saved, job queued and retried when Ollama recovers.

### Default Models (Ollama)

| Task | Model | Size |
|---|---|---|
| Embeddings | nomic-embed-text | 274MB |
| Text extraction / tagging | llama3.2:3b | 2GB |
| Vision / image description | moondream | 1.8GB |

---

## CLI — kl

### Libraries

| Library | Purpose |
|---|---|
| Cobra | Command structure, flags, help |
| Lip Gloss | Output styling — **use `rawnaqs/theme/custom/go/styles.go`, never define colors directly** |
| Glamour | Markdown rendering in terminal |
| Huh | `kl init` wizard, `kl config set` prompts |
| Bubble Tea | `kl status` live dashboard |
| rawnaqs/theme | Shared design system — colors, typography, pre-built styles |

### Commands

```bash
kl "thought"                     # capture text → ✓ saved · #tag · 3ms
kl --url https://...             # capture URL  → ⏳ queued · article · id: abc123
kl --image ~/screenshot.png      # capture image → ⏳ queued · image · id: def456
kl search "distributed systems"  # search → Glamour renders excerpts
kl status                        # Bubble Tea live dashboard
kl init                          # Huh wizard → writes ~/.config/khayal/kl.yaml
kl config set token abc123       # update single config value
kl config set host http://...
```

### kl.yaml

```yaml
# ~/.config/khayal/kl.yaml
host: http://127.0.0.1:7766
token: your-token-here
```

---

## PWA

- Stack: Templ + HTMX, embedded in binary via `embed.FS`
- Theme: `rawnaqs/theme` — `generated/css/variables.css` imported in `ui/static/style.css`
- No Node.js, no build pipeline, no npm
- Served at `http://<host>:<port>/`

### Features

- Text input
- URL input
- Image upload
- Camera capture
- Offline queue (IndexedDB, ~50 lines JS)
- Search with excerpts

### Offline Behavior

```
No connection → thought saved to IndexedDB
               → UI shows "N items pending sync"
Connection restored → auto-flushes queue silently
```

---

## Dependency Checker

Runs on `khayal start`. Missing dependencies degrade gracefully — text capture always works.

```
$ khayal start

Checking dependencies...
  ✓ Ollama        found at localhost:11434
  ✓ ffmpeg        found at /usr/local/bin/ffmpeg
  ✗ yt-dlp        not found
    → Install: brew install yt-dlp
    → Or: pip install yt-dlp
    → Video ingestion unavailable until installed
    → Continue anyway? [Y/n]
```

---

## CI / CD

### On every PR

```yaml
- go test ./...
- go vet ./...
- staticcheck ./...
- verify generated/ is in sync with tokens.json   ← theme integrity check
```

### On tag push (v*)

```yaml
- goreleaser
  - build for: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
  - create GitHub release with binaries
  - update Homebrew tap (github.com/rawnaqs/homebrew-tap)
  - publish Docker image to ghcr.io/rawnaqs/khayal
```

### Versioning

Semantic versioning: `v0.1.0`

---

## Distribution

```bash
# Homebrew
brew install rawnaqs/tap/khayal

# Direct download
curl -fsSL https://github.com/rawnaqs/khayal/releases/latest/download/install.sh | sh

# Docker
docker pull ghcr.io/rawnaqs/khayal
docker compose up
```

---

## Org Repos Required Before v1 Launch

```
github.com/rawnaqs/theme            ← must exist, khayal depends on it
github.com/rawnaqs/.github          ← org profile (avatar, banner, README)
github.com/rawnaqs/homebrew-tap     ← for brew install to work
github.com/rawnaqs/khayal           ← this repo
```

---

## Vault Compatibility

Khayal writes plain markdown with YAML frontmatter. It makes no assumptions about what the user opens it with. Tested compatible with:

- Obsidian
- Logseq
- Foam
- Any text editor

Wikilinks, graph view, backlinks — user's choice, not enforced.

---

## What We Are Not Building in v1

To be explicit — these are out of scope and will not be reconsidered for v1:

- No ambient/always-on voice capture
- No Telegram bot (transits third-party servers)
- No iCloud sync (Apple in the middle of private data)
- No Open WebUI integration
- No graph database (Neo4j, SurrealDB)
- No multi-user support
- No Windows support
- No browser extension
- No custom color/font definitions in Khayal — always import from rawnaqs/theme
