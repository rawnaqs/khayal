# Khayal Implementation Plan

> Master implementation guide. Updated: 2026-03-18

## Overview

Khayal is a local-first, privacy-focused second brain. This plan guides implementation.

**Philosophy:**
```
Capture  → zero friction, any device
Process  → immediate, local, private
Search   → fast, semantic + keyword
Store    → plain markdown, yours forever
Connect  → proactive discovery of related thoughts
```

## Two Binaries

| Binary | Command | Purpose |
|--------|---------|---------|
| `khayal` | `khayal init`, `khayal start` | Server + Worker + PWA |
| `kl` | `kl "thought"`, `kl search` | Thin HTTP client |

**Distribution:**
- `brew install rawnaqs/tap/khayal` → Full server
- `brew install rawnaqs/tap/kl` → Client only

**Shared Package:**
- `internal/api/client/` → Typed Go client for all interfaces

## Quick Reference

| Phase | Name | Key Deliverables |
|-------|------|------------------|
| 1 | Foundation | config, db, vault |
| 2 | Core API | server, endpoints, search |
| 3 | Worker | worker, ingest |
| 4 | LLM | ollama, groq, openai |
| 5 | CLI + Client | kl commands, api/client |
| 6 | PWA | React app |
| 7 | Polish | ci, release |

## Phase Summary

### Phase 1: Foundation
Project setup, config, database, vault writer.

**Goals:**
- [ ] Initialize Go module
- [ ] Create directory structure
- [ ] Config loader with validation
- [ ] SQLite job queue (modernc.org/sqlite + pure Go cosine similarity)
- [ ] Markdown frontmatter writer

**Files Created:** ~5
**Tests:** Unit tests for config, vault

### Phase 2: Core API
HTTP server, auth, logging, endpoints, search.

**Goals:**
- [ ] Chi router setup
- [ ] Auth middleware
- [ ] Logging middleware
- [ ] Health endpoint
- [ ] Capture endpoint (text sync, image/url queued)
- [ ] Queue endpoints (list, get, retry, discard)
- [ ] Search endpoint with:
  - [ ] FTS5 keyword search (porter stemming + BM25)
  - [ ] sqlite-vec semantic search
  - [ ] RRF hybrid merge (k=60)
  - [ ] Date filtering (from/to params)

**Files Created:** ~10
**Tests:** Integration tests for endpoints

### Phase 3: Worker
Background job processing, ingest pipeline.

**Goals:**
- [ ] Worker pool with configurable concurrency
- [ ] Crash recovery (reset stuck jobs)
- [ ] Text ingest (tags, summary)
- [ ] Image ingest (LLM description, OCR)
- [ ] Article ingest (scrape, summarize)
- [ ] Retry logic (exponential backoff)
- [ ] Safety-first: vault write only after ALL processing succeeds

**Files Created:** ~5
**Tests:** Worker pool tests

### Phase 4: LLM
Local AI integration.

**Goals:**
- [ ] LLM interface
- [ ] Ollama client (embed, generate, vision)
- [ ] Groq client (optional)
- [ ] OpenAI client (optional)
- [ ] No auto-fallback (job stays pending for user retry)

**Files Created:** ~5
**Tests:** Mock LLM tests

### Phase 5: CLI + Client
`kl` command and shared API client.

**Goals:**
- [x] `cmd/kl/main.go` entry point
- [x] `cmd/kl/internal/api/client.go` package
  - [x] Capture methods (text, url, image)
  - [x] Search methods
  - [x] Queue methods
  - [x] Health/Stats methods
  - [x] Types
- [x] khayal commands:
  - [x] `khayal init` - First-run setup
  - [x] `khayal start` - Start server + worker
  - [x] `khayal stop` - Graceful shutdown
  - [x] `khayal restart` - Stop + start
  - [x] `khayal status` - Bubble Tea TUI dashboard
  - [x] `khayal reindex` - Progress bar reindex
  - [x] `khayal version` - Version info
  - [x] `khayal logs` - Log tail
  - [x] `khayal config` - View config (token redacted)
- [x] kl commands:
  - [x] `kl "text"` - capture text (default)
  - [x] `kl capture url` - capture URL
  - [x] `kl capture image` - capture image
  - [x] `kl search` - search vault
  - [x] `kl recent` - recent captures
  - [x] `kl browse` - browse by tag/person/amount
  - [x] `kl stats` - vault statistics
  - [x] `kl status` - lightweight check
  - [x] `kl init` - Huh wizard
  - [x] `kl config` - config management (view/set/get)

**Files Created:** ~25 (cmd/khayal + cmd/kl)
**Tests:** CLI integration

### Phase 6: PWA
Web interface.

**Goals:**
- [ ] Vite + React setup
- [ ] Capture form
- [ ] Search UI
- [ ] Offline queue (IndexedDB)
- [ ] Go static serving
- [ ] SPA fallback

**Files Created:** ~20 (React components)
**Tests:** Component + E2E tests

### Phase 7: Polish
Release preparation.

**Goals:**
- [ ] Dependency checker
- [ ] CI workflow
- [ ] GoReleaser config (two binaries)
- [ ] Docker Compose
- [ ] README, CONTRIBUTING
- [ ] Example config

**Files Created:** ~5
**Tests:** Full integration

## v1.1 Scope (Post-Release)

See [SPEC.md](./SPEC.md) for full details.

### Chunking
- 150-200 words per chunk
- 30-50 word overlap
- Paragraph boundary splits only
- Minimum 50 words per chunk

### Entity Extraction
- People, amounts, dates, places, orgs, URLs
- Name normalization
- Frontmatter + entities table

### Proactive Connections
- Async delivery after capture
- Types: similar, person, amount (v1.1)
- Types: contradiction, follow_up, revisit (v1.2)

---

## Build

No build tags required! See [BUILD.md](BUILD.md) for details.

```bash
go build -o khayal ./cmd/khayal
go test ./...
```

---

## Getting Started

```bash
# Clone and enter directory
git clone github.com/rawnaqs/khayal
cd khayal

# Initialize Go module
go mod init github.com/rawnaqs/khayal
go mod tidy

# Run tests
go test ./...

# Build
go build -o khayal ./cmd/khayal
go build -o kl ./cmd/kl
```

## Per-Phase Instructions

Each phase has a detailed document:

- [Phase 1: Foundation](phases/phase-1-foundation.md)
- [Phase 2: Core API](phases/phase-2-api.md)
- [Phase 3: Worker](phases/phase-3-worker.md)
- [Phase 4: LLM](phases/phase-4-llm.md)
- [Phase 5: CLI](phases/phase-5-cli.md)
- [Phase 6: PWA](phases/phase-6-pwa.md)
- [Phase 7: Polish](phases/phase-7-polish.md)

## Tech Stack

See [TECH_STACK.md](TECH_STACK.md) for complete technology decisions.

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for system design.

## Repository Structure

See [REPO_STRUCTURE.md](REPO_STRUCTURE.md) for file tree reference.

## Running Locally

```bash
# After Phase 1+
khayal init
khayal start

# After Phase 5+
kl "my first thought"
kl search "thoughts"

# After Phase 6+
# Visit http://127.0.0.1:1133
```

## Contributing

See CONTRIBUTING.md after Phase 7 setup.

## Notes for AI Agents

1. **Always check SPEC.md first** - It's the source of truth
2. **Use TECH_STACK.md** - For dependency/import questions
3. **Check ARCHITECTURE.md** - For system design context
4. **Check BUILD.md** - No build tags required (uses modernc.org/sqlite)
5. **Phase files are checklists** - Follow them in order
6. **Tests are required** - Don't skip testing
7. **Run lint before commit** - `golangci-lint run`
8. **Never log tokens** - Security requirement

## Version

This plan covers **Khayal v1** and **v1.1**.

- **v1**: Core capture, search (FTS5 + sqlite-vec), CLI, PWA
- **v1.1**: Chunking, entity extraction, proactive connections
