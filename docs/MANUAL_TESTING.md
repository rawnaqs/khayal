# Manual Testing Guide

> Step-by-step verification commands for Khayal implementation.
> Update after completing each phase.

**Current Phase:** Phase 1 (Foundation) + Phase 2 (Core API)
**Last Updated:** 2026-03-19

---

## Prerequisites

```bash
cd /Users/armedev/Developer/Rawnaqs/khayal

# Config (testdata/config.yaml)
server.token: "abc"
server.port: 7766
vault.path: testdata/vault
db.path: testdata/khayal.db
```

---

## Start the Server

```bash
# Terminal 1: Start server
go run ./cmd/khayal

# Expected output:
# Khayal v0.1.0
#
# Config:       testdata/config.yaml
# Vault path:   testdata/vault
# DB path:      testdata/khayal.db
# Server:       127.0.0.1:7766
# LLM provider: ollama
#
# All directories ready.
# Database ready.
# Vault ready.
# Server listening on 127.0.0.1:7766
# Press Ctrl+C to stop
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

## Phase 2: Core API Endpoints

All endpoints require header: `X-Khayal-Token: abc`

---

### 1. Health Check

```bash
# Valid request
curl -s http://127.0.0.1:7766/v1/health \
  -H "X-Khayal-Token: abc" | jq

# Expected response:
{
  "status": "ok",
  "version": "0.1.0"
}
```

```bash
# Invalid token (should fail)
curl -s http://127.0.0.1:7766/v1/health \
  -H "X-Khayal-Token: wrong"

# Expected: 401 Unauthorized
```

```bash
# Missing token (should fail)
curl -s http://127.0.0.1:7766/v1/health

# Expected: 401 Unauthorized
```

---

### 2. Capture Text

```bash
# Capture text note
curl -s -X POST http://127.0.0.1:7766/v1/capture \
  -H "X-Khayal-Token: abc" \
  -H "Content-Type: application/json" \
  -d '{"type": "text", "content": "Test note for manual verification"}' | jq

# Expected response:
{
  "id": "uuid-here",
  "type": "text",
  "status": "done",
  "note_path": "inbox/2026-03-19-test-note-for-manual-verification.md",
  "created_at": "2026-03-19T..."
}
```

```bash
# Missing content (should fail)
curl -s -X POST http://127.0.0.1:7766/v1/capture \
  -H "X-Khayal-Token: abc" \
  -H "Content-Type: application/json" \
  -d '{"type": "text"}'

# Expected: 400 Bad Request
```

```bash
# Invalid JSON (should fail)
curl -s -X POST http://127.0.0.1:7766/v1/capture \
  -H "X-Khayal-Token: abc" \
  -H "Content-Type: application/json" \
  -d 'not valid json'

# Expected: 400 Bad Request
```

---

### 3. Verify Note Saved

```bash
# Check note exists in vault
ls -la testdata/vault/inbox/

# View note content
cat testdata/vault/inbox/*test-note*.md
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

```bash
# Keyword search
curl -s "http://127.0.0.1:7766/v1/search?q=test" \
  -H "X-Khayal-Token: abc" | jq

# Expected response:
{
  "query": "test",
  "mode": "hybrid",
  "results": [...],
  "total": 1,
  "took_ms": ...
}
```

```bash
# Search with mode=keyword only
curl -s "http://127.0.0.1:7766/v1/search?q=test&mode=keyword" \
  -H "X-Khayal-Token: abc" | jq
```

```bash
# Missing query (should fail)
curl -s "http://127.0.0.1:7766/v1/search" \
  -H "X-Khayal-Token: abc"

# Expected: 400 Bad Request
```

---

### 5. Queue Operations

```bash
# List all jobs
curl -s http://127.0.0.1:7766/v1/queue \
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
      "note_path": "inbox/...",
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
curl -s "http://127.0.0.1:7766/v1/queue?status=done" \
  -H "X-Khayal-Token: abc" | jq
```

```bash
# Get single job (use ID from previous response)
curl -s http://127.0.0.1:7766/v1/queue/{job_id} \
  -H "X-Khayal-Token: abc" | jq
```

```bash
# Get non-existent job (should fail)
curl -s http://127.0.0.1:7766/v1/queue/nonexistent \
  -H "X-Khayal-Token: abc"

# Expected: 404 Not Found
```

---

### 6. Queue Retry

```bash
# Create a job to test retry
curl -s -X POST http://127.0.0.1:7766/v1/capture \
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
| Server starts | `go run ./cmd/khayal` | Listens on 7766 |
| Health endpoint | `curl /v1/health` | 200 + status ok |
| Auth - valid token | `curl -H "Token: abc" ...` | 200 |
| Auth - invalid token | `curl -H "Token: x" ...` | 401 |
| Auth - missing token | `curl ...` | 401 |
| Capture text | `curl -X POST /v1/capture ...` | 201 + job |
| Capture - missing content | `curl ... -d '{"type":"text"}'` | 400 |
| Capture - invalid JSON | `curl ... -d 'invalid'` | 400 |
| Note saved to vault | `cat testdata/vault/inbox/*.md` | File exists |
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
rm -f testdata/vault/inbox/*.md

# Reset test state
go clean -testcache
```

---

## Known Limitations (Phase 3+ Required)

- URL capture creates job but doesn't process (no worker)
- Image capture creates job but doesn't process (no worker)
- Semantic search uses mock embeddings (no LLM integration)
- Queue retry/discard limited to pending/failed jobs (no worker to create them)

---

## Troubleshooting

### Server won't start
```bash
# Check port in use
lsof -i :7766

# Kill existing process
kill $(lsof -t -i :7766)
```

### Database locked
```bash
# Remove stale DB
rm -f testdata/khayal.db
```

### Test notes accumulate
```bash
# Clean vault
rm -f testdata/vault/inbox/*.md
```
