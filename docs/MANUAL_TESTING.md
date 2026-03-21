# Manual Testing Guide

> Step-by-step verification commands for Khayal implementation.
> Update after completing each phase.

**Current Phase:** Phase 5 (CLI) — khayal + kl commands implemented
**Last Updated:** 2026-03-20

---

## Prerequisites

```bash
cd /Users/armedev/Developer/Rawnaqs/khayal

# Config (testdata/config.yaml)
server.token: "abc"
server.port: 1133
vault.path: testdata/vault
vault.inbox_dir: khayal  # optional, defaults to "khayal"
db.path: testdata/khayal.db

# Logging config
logging:
  level: "info"
  worker_level: "debug"
  file: "logs/khayal.log"
  max_size_mb: 10
  max_backups: 5
  compress: true

# Ollama (for Phase 3+)
# Run: ollama list
# Required models: nomic-embed-text, qwen2.5:3b, moondream
```

---

## Start the Server

```bash
# Terminal 1: Start server
KHAYAL_CONFIG=./testdata/config.yaml go run ./cmd/khayal start

# Expected output:
# khayal v0.1.0
#
# loading config...
# checking dependencies...
#   ✓ ollama        http://localhost:11434
#
#   ✓ vault         /absolute/path/to/testdata/vault
#   ✓ db            /absolute/path/to/testdata/khayal.db
#   ✓ log           /absolute/path/to/testdata/logs/khayal.log
#   ✓ queue         ready
#   ✓ llm           ollama
#   ✓ worker        started
#   ✓ server        127.0.0.1:1133
#   ✓ pid           12345
#
# khayal is running.
# press ctrl+c to stop
```

---

## Logging Verification

```bash
# Check log file was created (logs go to file only, not stdout)
ls -la testdata/logs/

# Expected: khayal.log exists

# View log file
cat testdata/logs/khayal.log

# Expected: JSON formatted logs
# {"time":"2026-03-19T...","level":"INFO","msg":"server started",...}
# {"time":"2026-03-19T...","level":"DEBUG","msg":"worker processing",...}

# Check for rotation (after hitting max_size_mb)
ls -la testdata/logs/
# Expected: khayal.log.1.gz, khayal.log.2.gz, etc.
```

---

## Path Handling Verification

```bash
# Create a note (trash goes to inbox/.khayal-trash, not vault/.khayal-trash)
curl -s -X POST http://127.0.0.1:1133/v1/capture \
  -H "X-Khayal-Token: abc" \
  -H "Content-Type: application/json" \
  -d '{"type": "text", "content": "Test path handling"}' | jq

# Wait for processing...
sleep 5

# Check trash location (should be in inbox, not vault root)
ls -la testdata/vault/khayal/.khayal-trash/

# Verify NOT in vault root
ls testdata/vault/.khayal-trash/ 2>/dev/null || echo "Correct: No trash at vault root"
```

---

## Phase 5: CLI Commands

### Environment Variables

```bash
# khayal uses KHAYAL_CONFIG for config path
KHAYAL_CONFIG=./testdata/config.yaml

# kl uses KL_CONFIG for config path (defaults to ~/.config/khayal/kl.yaml)
KL_CONFIG=./testdata/kl.yaml
```

### khayal Commands

#### khayal config
```bash
# View current config (token redacted)
KHAYAL_CONFIG=./testdata/config.yaml go run ./cmd/khayal config

# Expected: formatted config output with token masked
```

#### khayal start
```bash
# Start server (already tested above)
KHAYAL_CONFIG=./testdata/config.yaml go run ./cmd/khayal start
```

#### khayal stop
```bash
# Graceful shutdown (run from another terminal while server is running)
KHAYAL_CONFIG=./testdata/config.yaml go run ./cmd/khayal stop

# Expected output:
# stopping worker...    ✓ (waited for current job to finish)
# stopping server...     ✓
# khayal stopped.
```

#### khayal status
```bash
# Bubble Tea TUI dashboard
KHAYAL_CONFIG=./testdata/config.yaml go run ./cmd/khayal status

# Expected: Interactive TUI with queue status
# Press 'q' to quit
```

#### khayal version
```bash
KHAYAL_CONFIG=./testdata/config.yaml go run ./cmd/khayal version

# Expected:
# khayal v0.1.0
# commit  a3f9c2e
# built   2024-03-20T10:00:00Z
```

### kl Commands

First, create a test config for kl:
```bash
mkdir -p ./testdata
cat > ./testdata/kl.yaml << 'EOF'
host: http://127.0.0.1:1133
token: abc
EOF
```

#### kl init
```bash
# Interactive wizard (run without test config first to test wizard)
KL_CONFIG=./testdata/kl.yaml go run ./cmd/kl init

# Expected: Huh form with host + token inputs
```

#### kl config
```bash
# View config
KL_CONFIG=./testdata/kl.yaml go run ./cmd/kl config view

# Expected: shows host and token (token masked)

# Set config values
KL_CONFIG=./testdata/kl.yaml go run ./cmd/kl config set host http://127.0.0.1:1133
KL_CONFIG=./testdata/kl.yaml go run ./cmd/kl config set token abc

# Get config value
KL_CONFIG=./testdata/kl.yaml go run ./cmd/kl config get host
```

#### kl status
```bash
# Lightweight server check (requires server running)
KL_CONFIG=./testdata/kl.yaml go run ./cmd/kl status

# Expected (when server up):
# ✓ khayal v0.1.0 · http://127.0.0.1:1133
#   queue
#     processing   1   image
#     pending      2
#
# Expected (when server down):
# ✗ cannot reach khayal at http://127.0.0.1:1133
```

#### kl capture
```bash
# Capture text (default command)
KL_CONFIG=./testdata/kl.yaml go run ./cmd/kl "test thought from cli"

# Expected: capture response with note path or queued status
```

#### kl search
```bash
# Search vault
KL_CONFIG=./testdata/kl.yaml go run ./cmd/kl search "test"

# Expected: search results with scores and excerpts
```

#### kl recent
```bash
# Recent captures
KL_CONFIG=./testdata/kl.yaml go run ./cmd/kl recent

# Expected: grouped list of recent captures by day
```

---

## Run Tests

```bash
# All tests
go test ./... -v

# API tests only
go test ./internal/api/... -v

# Vault tests only
go test ./internal/vault/... -v

# Queue tests only
go test ./internal/queue/... -v

# Code quality
go vet ./...
```

---

## Phase 2 + Phase 3: Core API Endpoints

All endpoints require header: `X-Khayal-Token: abc`

---

### 1. Health Check

```bash
# Valid request
curl -s http://127.0.0.1:1133/v1/health \
  -H "X-Khayal-Token: abc" | jq

# Expected response (includes LLM status):
{
  "status": "ok",
  "version": "0.1.0",
  "dependencies": {
    "db": { "status": "ok" },
    "vault": { "status": "ok" },
    "llm": { "status": "ok", "host": "http://localhost:11434" }
  },
  "queue": { "pending": 0, "queued": 0, "processing": 0, "done": 0, "failed": 0 }
}
```

```bash
# Invalid token (should fail)
curl -s http://127.0.0.1:1133/v1/health \
  -H "X-Khayal-Token: wrong"

# Expected: 401 Unauthorized
```

```bash
# Missing token (should fail)
curl -s http://127.0.0.1:1133/v1/health

# Expected: 401 Unauthorized
```

---

### 2. Capture Text (Async - Phase 3)

Text capture is now **async** - the job is queued and processed by the worker.

```bash
# Capture text note
curl -s -X POST http://127.0.0.1:1133/v1/capture \
  -H "X-Khayal-Token: abc" \
  -H "Content-Type: application/json" \
  -d '{"type": "text", "content": "Go is a programming language"}' | jq

# Expected response (immediate - job queued):
{
  "id": "uuid-here",
  "type": "text",
  "status": "pending",
  "note_path": "",
  "created_at": "2026-03-19T..."
}
```

```bash
# Missing content (should fail)
curl -s -X POST http://127.0.0.1:1133/v1/capture \
  -H "X-Khayal-Token: abc" \
  -H "Content-Type: application/json" \
  -d '{"type": "text"}'

# Expected: 400 Bad Request
```

```bash
# Check queue status
curl -s http://127.0.0.1:1133/v1/queue \
  -H "X-Khayal-Token: abc" | jq

# After worker processes (5-10 seconds):
# - status changes from "pending" → "queued" → "processing" → "done"
# - note_path is populated
```

```bash
# Check note was saved (after processing)
cat testdata/vault/khayal/*.md | head -30

# Note will have:
# - LLM-generated tags
# - LLM-generated summary
# - history entry
```
```

```bash
# Invalid JSON (should fail)
curl -s -X POST http://127.0.0.1:1133/v1/capture \
  -H "X-Khayal-Token: abc" \
  -H "Content-Type: application/json" \
  -d 'not valid json'

# Expected: 400 Bad Request
```

---

### 2b. Capture URL

URLs are queued as articles and processed by the worker.

```bash
# Capture URL (becomes article job)
curl -s -X POST http://127.0.0.1:1133/v1/capture \
  -H "X-Khayal-Token: abc" \
  -H "Content-Type: application/json" \
  -d '{"type": "url", "content": "https://example.com/article"}' | jq

# Expected response:
{
  "id": "uuid-here",
  "type": "article",
  "status": "pending",
  "note_path": "",
  "created_at": "2026-03-19T..."
}
```

```bash
# Check queue - job should have source_url set
curl -s http://127.0.0.1:1133/v1/queue \
  -H "X-Khayal-Token: abc" | jq '.jobs[] | select(.type=="article")'
```

---

### 2c. Capture Image

Images are saved to media directory and processed by the worker.

```bash
# Capture image (multipart form)
curl -s -X POST http://127.0.0.1:1133/v1/capture \
  -H "X-Khayal-Token: abc" \
  -F "type=image" \
  -F "file=@testdata/vault/khayal/media/test.png" \
  -F "note=optional context" | jq

# Expected response:
{
  "id": "uuid-here",
  "type": "image",
  "status": "pending",
  "note_path": "khayal/2026-03-19-*.md",
  "created_at": "2026-03-19T..."
}
```

```bash
# Check media was saved
ls -la testdata/vault/khayal/media/

# Check queue - job should have source_file set
curl -s http://127.0.0.1:1133/v1/queue \
  -H "X-Khayal-Token: abc" | jq '.jobs[] | select(.type=="image")'
```

---

### 3. Verify Note Saved

```bash
# Check note exists in vault
ls -la testdata/vault/khayal/

# View note content
cat testdata/vault/khayal/*test-note*.md
```

**Expected frontmatter format:**
```yaml
---
created: 2026-03-19T...
updated: 2026-03-19T...
type: text
status: done
history:
  - at: 2026-03-19T...
    event: created
---

# Test note for manual verification

## Raw
Test note for manual verification
```

**Verification checks:**
- [ ] `---` at start and end of frontmatter
- [ ] `created:` field present
- [ ] `type: text` present
- [ ] `status: done` present
- [ ] Exactly ONE `history:` block
- [ ] `history:` has proper indentation (`  - at:`, `    event:`)
- [ ] No duplicate keys

---

### 4. Search

Search uses hybrid mode (keyword + semantic) with real Ollama embeddings.

```bash
# Default: hybrid search (keyword + semantic)
curl -s "http://127.0.0.1:1133/v1/search?q=golang" \
  -H "X-Khayal-Token: abc" | jq

# Expected: matches notes containing "golang" or semantically similar
```

```bash
# Search with mode=keyword only
curl -s "http://127.0.0.1:1133/v1/search?q=test&mode=keyword" \
  -H "X-Khayal-Token: abc" | jq
```

```bash
# Missing query (should fail)
curl -s "http://127.0.0.1:1133/v1/search" \
  -H "X-Khayal-Token: abc"

# Expected: 400 Bad Request
```

---

### 5. Queue Operations

```bash
# List all jobs
curl -s http://127.0.0.1:1133/v1/queue \
  -H "X-Khayal-Token: abc" | jq

# Expected response:
{
  "total": 1,
  "limit": 20,
  "offset": 0,
  "jobs": [
    {
      "id": "...",
      "type": "text",
      "status": "done",
      "note_path": "khayal/...",
      "created_at": "2026-03-19T...",
      "processed_at": "2026-03-19T...",
      "error": null,
      "retries": 0
    }
  ]
}
```

```bash
# List by status
curl -s "http://127.0.0.1:1133/v1/queue?status=done" \
  -H "X-Khayal-Token: abc" | jq
```

```bash
# Get single job (use ID from previous response)
curl -s http://127.0.0.1:1133/v1/queue/{job_id} \
  -H "X-Khayal-Token: abc" | jq
```

```bash
# Get non-existent job (should fail)
curl -s http://127.0.0.1:1133/v1/queue/nonexistent \
  -H "X-Khayal-Token: abc"

# Expected: 404 Not Found
```

---

### 6. Queue Retry

```bash
# Create a job to test retry
curl -s -X POST http://127.0.0.1:1133/v1/capture \
  -H "X-Khayal-Token: abc" \
  -H "Content-Type: application/json" \
  -d '{"type": "text", "content": "Job for retry test"}' | jq

# Note: Currently text jobs complete immediately (status=done)
# To test retry, would need a failed job (Phase 3: Worker)
```

---

### 7. Queue Discard

```bash
# Note: Cannot discard completed jobs
# This would require a pending/failed job (Phase 3: Worker)
```

---

## Phase 1: Foundation Verification

### Config Loading

```bash
# Server starts with testdata/config.yaml
# Check output shows correct paths
```

### SQLite Database

```bash
# DB created at testdata/khayal.db
ls -la testdata/khayal.db

# Tables exist
sqlite3 testdata/khayal.db ".tables"
# Expected: chunks embeddings entities jobs notes_fts

# Schema
sqlite3 testdata/khayal.db ".schema jobs"
```

### FTS5 Search

```bash
# Index exists
sqlite3 testdata/khayal.db "SELECT * FROM notes_fts"
```

---

## Verification Checklist

### Phase 2: Core API

| Feature | Test | Expected |
|---------|------|----------|
| Server starts | `go run ./cmd/khayal` | Listens on 1133 |
| Health endpoint | `curl /v1/health` | 200 + status ok |
| Auth - valid token | `curl -H "Token: abc" ...` | 200 |
| Auth - invalid token | `curl -H "Token: x" ...` | 401 |
| Auth - missing token | `curl ...` | 401 |
| Capture text | `curl -X POST /v1/capture ...` | 201 + job |
| Capture URL | `curl ... '{"type":"url"...}'` | 201 + type=article |
| Capture image | `curl -F "file=@..." ...` | 201 + note_path |
| Capture - missing content | `curl ... -d '{"type":"text"}'` | 400 |
| Capture - invalid JSON | `curl ... -d 'invalid'` | 400 |
| Note saved to vault | `cat testdata/vault/khayal/*.md` | File exists |
| Note format valid | Check frontmatter | Valid YAML |
| Note - single history | Search for `history:` | Exactly 1 |
| Search keyword | `curl /v1/search?q=test` | Returns results |
| Search - missing query | `curl /v1/search` | 400 |
| Queue list | `curl /v1/queue` | Array of jobs |
| Queue get | `curl /v1/queue/{id}` | Single job |
| Queue get - not found | `curl /v1/queue/invalid` | 404 |

### Code Quality

| Check | Command | Expected |
|-------|---------|----------|
| Tests pass | `go test ./...` | All pass |
| go vet | `go vet ./...` | No output |
| Build | `go build ./...` | Success |

---

## Cleanup

```bash
# Remove test data
rm -f testdata/khayal.db
rm -f testdata/vault/khayal/*.md

# Reset test state
go clean -testcache
```

---

## Known Limitations

- Queue retry/discard limited to pending/failed jobs

---

## Troubleshooting

### Ollama not running or models missing
```bash
# Check Ollama status
ollama list

# Required models:
# - nomic-embed-text (for embeddings)
# - qwen2.5:3b (for text processing)
# - moondream (for image description)

# Pull missing models
ollama pull qwen2.5:3b
ollama pull nomic-embed-text
ollama pull moondream
```

### Server won't start
```bash
# Check port in use
lsof -i :1133

# Kill existing process
kill $(lsof -t -i :1133)
```

### Database locked
```bash
# Remove stale DB
rm -f testdata/khayal.db
```

### Test notes accumulate
```bash
# Clean vault
rm -f testdata/vault/khayal/*.md
```

### Job processing fails (404 error)
```bash
# Check config matches installed models
grep text_model testdata/config.yaml
ollama list

# If mismatch, update config.yaml or pull the model
```
