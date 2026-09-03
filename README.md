# Khayal

![Go](https://img.shields.io/github/go-mod/go-version/rawnaqs/khayal?label=go)
![Release](https://img.shields.io/github/v/release/rawnaqs/khayal)
![License](https://img.shields.io/github/license/rawnaqs/khayal)

> Your private treasury of thought. Local, secure, yours.

<!-- TODO: add asciinema recording or terminal screenshot here -->
<img src="charm-vhs-tape/khayal.gif" alt="Khayal demo" />

A local-first, privacy-focused second brain. Capture anything — text, images, URLs, PDFs, voice. Process locally with your own LLM: tags, summaries, entities, and proactive connections that resurface what you've forgotten. Search semantically and by keyword, or ask AI questions over your own notes. Your data never leaves your machine.

## How It Works

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ Capture  │ →  │ Process  │ →  │  Index   │ →  │  Search  │
│ kl / PWA │    │ Ollama   │    │ SQLite   │    │ FTS5+Vec │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
                                  ↓
                              ┌──────────┐
                              │  Vault   │
                              │ Markdown │
                              └──────────┘
```

Capture via CLI (`kl`) or web UI (PWA). The server processes content with a local LLM, extracts tags/summaries/key ideas, indexes everything in SQLite (FTS5 + embeddings), and stores plain markdown in your vault.

## Why Khayal?

| | Notion | Obsidian | Mem.ai | Khayal |
|---|---|---|---|---|
| Data | Cloud-hosted | Local files | Cloud-only | Local files |
| AI | API calls (OpenAI) | Plugins only | API calls | Local LLM (Ollama) |
| Search | Server-side | Graph/keyword | Server-side | FTS5 + semantic |
| Capture layer | Full editor | Manual | Full editor | CLI + PWA |
| Cost | Subscription | Free | $20+/mo | Free |
| Vault format | Proprietary | `.md` | Proprietary | Plain `.md` |

Khayal and Obsidian are complementary. Khayal is a capture and retrieval layer — your vault remains plain markdown that Obsidian can open directly. `khayal init` optionally installs the Front Matter Title plugin for better note titles in Obsidian.

## Features

- **Capture** — Text, images, URLs, articles with zero friction (voice notes and PDF ingestion on the roadmap)
- **Process** — Tags, summaries, key ideas, entities (people, amounts, dates, places, orgs, URLs) via local LLM
- **Proactive connections** — after every capture, khayal resurfaces related thoughts, shared people, matching amounts, **contradictions of things you wrote**, unfinished follow-ups, and ideas you keep revisiting
- **Capture intelligence** — relative dates resolved at capture; an LLM-maintained memory file keeps naming consistent across months
- **Search** — Keyword (FTS5) + semantic (chunk-level embeddings) hybrid, with passage-level excerpts
- **AI answers** — on-demand answers above search results, grounded in your own notes with `[n]` citations; explicit CTA, never automatic, never breaks search
- **Store** — Plain markdown in your vault, yours forever (Obsidian-friendly, with connection wikilinks written into frontmatter)
- **Vault care** — health report, broken-link repair, orphaned-media cleanup, duplicate detection, soft-delete with trash
- **Backup** — encrypted (age) vault/database/config backups with additive-merge restore
- **PWA** — Web interface, works offline, update notifications
  - Live queue over WebSocket (job status streams in; polling fallback)
  - AI answer with skeleton loading, connection flares on the queue, linked notes with reasons, entity chips, image previews
  - Offline capture queue (syncs when server is back)
  - Works as installable PWA on iOS and desktop
  - Optional Face ID / Touch ID app lock (WebAuthn PRF) that encrypts the token at rest
- **CLI** — Full client (`kl`) + server admin (`khayal`)
- **Updates** — Built-in update checker via GitHub releases

## Requirements

- [Ollama](https://ollama.com) (for LLM features)
- macOS or Linux

## Quick Start

### 1. Install Ollama

```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.com/install.sh | sh

# Pull required models
ollama pull nomic-embed-text
ollama pull qwen2.5:3b
ollama pull moondream

# optional: a larger model just for memory consolidation (recommended)
ollama pull qwen2.5:7b
```

### 2. Install Khayal

```bash
# Homebrew (includes khayal + kl)
brew install rawnaqs/tap/khayal

# Or one-liner
curl -fsSL https://raw.githubusercontent.com/rawnaqs/khayal/main/install.sh | sh
```

### 3. Initialize + Start

```bash
# Initialize server (creates config + token)
khayal init

# Start server
khayal start

# Or run as background service
brew services start khayal
```

### 4. Connect Client

```bash
# From same machine
kl init --token <token>

# Capture a thought
kl "my first thought"

# Search
kl search "distributed systems"

# Open the web UI
# http://127.0.0.1:1133
```

## Installation

### Homebrew

```bash
brew install rawnaqs/tap/khayal
```

### One-liner

```bash
curl -fsSL https://raw.githubusercontent.com/rawnaqs/khayal/main/install.sh | sh
```

### Docker

**Prerequisites:** Run [Ollama](https://ollama.com) locally for GPU acceleration.

```bash
docker run \
  --add-host host.docker.internal:host-gateway \
  -v ~/Documents/brain:/vault \
  -v ~/.config/khayal:/root/.config/khayal \
  -p 1133:1133 \
  ghcr.io/rawnaqs/khayal
```

## Configuration

Config file: `~/.config/khayal/config.yaml`

```yaml
vault:
  path: ~/Documents/brain       # your notes directory
  inbox_dir: khayal              # subdirectory for khayal captures

server:
  host: 127.0.0.1
  port: 1133
  token: <auto-generated>

llm:
  provider: ollama
  ollama_host: http://localhost:11434
  embed_model: nomic-embed-text
  text_model: qwen2.5:3b
  vision_model: moondream

worker:
  max_workers: 1
  max_retries: 3

db:
  path: khayal.db

search:
  chunk_target_words: 175    # semantic search indexes paragraph-aligned chunks
  chunk_min_words: 50
  chunk_overlap_words: 35

connections:
  enabled: true              # proactive connections after every capture
  similarity_threshold: 0.72
  types:                     # toggle each detector independently
    similar: true
    person: true
    amount: true
    contradiction: true
    follow_up: true
    revisit: true

memory:
  enabled: true              # LLM context memory + memory.md consolidation

log:
  level: info
  file: logs/khayal.log
```

Edit with: `vim ~/.config/khayal/config.yaml`

See [config.example.yaml](config.example.yaml) for all options.

## Where Is My Data?

| Location | Content |
|---|---|
| `~/Documents/brain/khayal/` | Your notes (plain markdown + media) |
| `~/Documents/brain/khayal/memory.md` | LLM-maintained memory (editable by hand) |
| `~/Documents/brain/khayal/.khayal-trash/` | Soft-deleted notes (recoverable) |
| `~/.config/khayal/khayal.db` | Search index + embeddings |
| `~/.config/khayal/config.yaml` | Server configuration |
| `~/.config/khayal/logs/` | Server logs |

All on your machine. Back up the vault directory — it's plain markdown (or use `khayal backup --encrypt`).

## Commands

### Server (`khayal`)

| Command | Description |
|---------|-------------|
| `khayal init` | First-run setup (config + token) |
| `khayal start` | Start server + worker |
| `khayal stop` | Graceful shutdown |
| `khayal restart` | Stop + start |
| `khayal status` | Server status + update check |
| `khayal reindex` | Rebuild search index (FTS + chunk embeddings) |
| `khayal config` | View config (token redacted) |
| `khayal vault health` | Vault health report (notes, indexed %, orphans, broken links) |
| `khayal vault fix-links` | Remove broken wikilinks (dry-run by default) |
| `khayal vault clean-media` | Move orphaned media files to trash |
| `khayal vault show-duplicates` | Show potential duplicate notes |
| `khayal backup --dest <path>` | Backup vault, database, config (`--encrypt` for age encryption) |
| `khayal restore --from <path>` | Restore from backup (additive merge, refuses while running) |

### Client (`kl`)

| Command | Description |
|---------|-------------|
| `kl "text"` | Capture text |
| `kl url "https://..."` | Capture URL |
| `kl image <path>` | Capture image |
| `kl search "query"` | Search vault (`--answer` adds a grounded AI answer) |
| `kl delete <path-or-id>` | Soft-delete a note (moved to `.khayal-trash/`) |
| `kl recent` | Recent captures |
| `kl stats` | Vault statistics |
| `kl status` | Server status + update check |

### Homebrew Service (macOS)

```bash
brew services start khayal   # background service
brew services stop khayal    # stop service
tail -f ~/.config/khayal/logs/khayal.log   # view logs
```

## Environment Variables

- `KHAYAL_CONFIG` — Config file path (default: `~/.config/khayal/config.yaml`)
- `KL_CONFIG` — Client config path (default: `~/.config/khayal/kl.yaml`)

## PWA

Web interface at `http://127.0.0.1:1133`

- Capture text, URLs, images
- Search with excerpts + on-demand **AI answers** with citations
- **Live queue** — job status streams over WebSocket, connection flares on finished captures
- **Note reader** — image previews, entity chips that jump to search, linked notes with reasons, copy-as-markdown
- **Delete** — two-step confirm, recoverable from trash
- Offline queue (IndexedDB)
- Update notification icon

## Development

```bash
git clone github.com/rawnaqs/khayal
cd khayal
go mod download
go build -o khayal ./cmd/khayal
go build -o kl ./cmd/kl
go test ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## Roadmap

- **v1.0** ✅ — Core capture, search, CLI, PWA
- **v1.1** ✅ — Chunking, entity extraction, proactive connections, capture intelligence, AI answers, delete, vault commands, encrypted backups
- **v1.2** 🚧 — Contradiction / follow-up / revisit connections ✅ · voice notes · PDF ingestion
- **v1.3** — Graph connections, backlinks
- **v1.4** — YouTube / video ingestion
- **v1.5** — Browser extension
- **v2.0** — Setup wizard UI

See [SPEC.md](docs/SPEC.md) for the full roadmap.

## License

AGPLv3 — See [LICENSE](LICENSE)

---

Built by [Rawnaqs](https://rawnaqs.io) · Open source tools, local first.
