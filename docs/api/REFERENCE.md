# Khayal API Reference

> Complete API reference for Khayal v1. Updated: 2026-03-17

## Base URL

```
http://localhost:7766/v1
```

## Authentication

All requests require the `X-Khayal-Token` header:

```bash
curl -H "X-Khayal-Token: your-token-here" \
     http://localhost:7766/v1/health
```

See [AUTH.md](AUTH.md) for detailed authentication guide.

## Endpoints

---

### GET /health

Get system health status, dependency checks, and queue statistics.

**Response:**
```json
{
  "status": "ok",
  "version": "0.1.0",
  "dependencies": {
    "ollama": { "status": "ok", "host": "http://localhost:11434" },
    "vault": { "status": "ok", "path": "~/Documents/brain" },
    "db": { "status": "ok", "path": "~/.config/khayal/khayal.db" }
  },
  "queue": {
    "pending": 2,
    "processing": 1,
    "done": 147,
    "failed": 0
  }
}
```

---

### POST /capture

Capture text, URL, or image for processing.

#### Text Capture (JSON)

```bash
curl -X POST http://localhost:7766/v1/capture \
  -H "Content-Type: application/json" \
  -H "X-Khayal-Token: your-token" \
  -d '{"type": "text", "content": "useEffect cleanup runs after every render"}'
```

**Response:**
```json
{
  "id": "abc123",
  "type": "text",
  "status": "done",
  "note_path": "inbox/2024-03-16-thought.md",
  "created_at": "2024-03-16T14:23:00Z"
}
```

#### URL Capture (JSON)

```bash
curl -X POST http://localhost:7766/v1/capture \
  -H "Content-Type: application/json" \
  -H "X-Khayal-Token: your-token" \
  -d '{"type": "url", "content": "https://blog.example.com/post"}'
```

**Response:**
```json
{
  "id": "def456",
  "type": "article",
  "status": "processing",
  "note_path": null,
  "created_at": "2024-03-16T14:23:00Z"
}
```

#### Image Capture (Multipart)

```bash
curl -X POST http://localhost:7766/v1/capture \
  -H "X-Khayal-Token: your-token" \
  -F "type=image" \
  -F "file=@screenshot.png" \
  -F "note=optional context"
```

**Response:**
```json
{
  "id": "ghi789",
  "type": "image",
  "status": "processing",
  "note_path": null,
  "created_at": "2024-03-16T14:23:00Z"
}
```

---

### GET /search

Search knowledge base.

**Parameters:**

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| q | Yes | - | Search query |
| limit | No | 10 | Max results (max 50) |
| mode | No | hybrid | Search mode: keyword, semantic, hybrid |
| excerpt_length | No | 200 | Max excerpt chars (max 500) |
| from | No | - | Filter: notes created after this date (ISO) |
| to | No | - | Filter: notes created before this date (ISO) |
| connections | No | false | Include related connections (v1.1+) |

**Example:**
```bash
# Basic search
curl "http://localhost:7766/v1/search?q=distributed+systems&limit=5&mode=hybrid" \
  -H "X-Khayal-Token: your-token"

# Search with date filter
curl "http://localhost:7766/v1/search?q=react&from=2024-01-01&to=2024-12-31" \
  -H "X-Khayal-Token: your-token"

# Search with connections (v1.1+)
curl "http://localhost:7766/v1/search?q=react&connections=true" \
  -H "X-Khayal-Token: your-token"
```

**Response:**
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

### GET /queue

List jobs in queue.

**Parameters:**

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| status | No | all | Filter: all, pending, processing, done, failed |
| limit | No | 20 | Max results (max 100) |
| offset | No | 0 | Pagination offset |

**Example:**
```bash
curl "http://localhost:7766/v1/queue?status=failed&limit=10" \
  -H "X-Khayal-Token: your-token"
```

**Response:**
```json
{
  "total": 2,
  "limit": 10,
  "offset": 0,
  "jobs": [
    {
      "id": "abc123",
      "type": "image",
      "status": "failed",
      "note_path": null,
      "source_file": "media/2024-03-16-image.png",
      "created_at": "2024-03-16T14:23:00Z",
      "processed_at": null,
      "error": "failed to describe image: ollama timeout",
      "retries": 3
    }
  ]
}
```

---

### GET /queue/:id

Get single job by ID.

**Example:**
```bash
curl http://localhost:7766/v1/queue/abc123 \
  -H "X-Khayal-Token: your-token"
```

**Response:**
```json
{
  "id": "abc123",
  "type": "image",
  "status": "failed",
  "note_path": null,
  "source_file": "media/2024-03-16-image.png",
  "created_at": "2024-03-16T14:23:00Z",
  "processed_at": null,
  "error": "failed to describe image: ollama timeout",
  "retries": 3
}
```

---

### POST /queue/:id/retry

Retry a pending or failed job.

**Example:**
```bash
curl -X POST http://localhost:7766/v1/queue/abc123/retry \
  -H "X-Khayal-Token: your-token"
```

**Response:**
```json
{
  "id": "abc123",
  "type": "image",
  "status": "pending",
  "note_path": null,
  "source_file": "media/2024-03-16-image.png",
  "created_at": "2024-03-16T14:23:00Z",
  "processed_at": null,
  "error": null,
  "retries": 0
}
```

---

### POST /queue/:id/discard

Permanently delete a pending or failed job.

**Example:**
```bash
curl -X POST http://localhost:7766/v1/queue/abc123/discard \
  -H "X-Khayal-Token: your-token"
```

**Response:**
```json
{
  "success": true,
  "id": "abc123",
  "message": "job discarded"
}
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": "human readable message",
  "code": "ERROR_CODE"
}
```

### Error Codes

All errors return `{"error": "...", "code": "...", "hint": "..."}`. See SPEC.md Error Taxonomy for full list.

| Code | HTTP Status | Description |
|------|-------------|-------------|
| AUTH_001 | 401 | Invalid token |
| AUTH_002 | 401 | Token missing |
| CAPTURE_004 | 400 | Missing required field |
| SEARCH_001 | 400 | Query too short |
| SEARCH_003 | 400 | Invalid mode |
| VAULT_001 | 500 | Vault path not found |
| VAULT_002 | 500 | Vault write failed |
| LLM_001 | 500 | Ollama unreachable |
| LLM_002 | 500 | Model not found |
| QUEUE_002 | 404 | Job not found |
| SYS_001 | 500 | Database error |

### Examples

```json
400 { "error": "missing required field: content", "code": "CAPTURE_004" }
401 { "error": "invalid token", "code": "AUTH_001" }
500 { "error": "failed to write note to vault", "code": "VAULT_002" }
```

---

## Processing Times

| Type | Time | Status |
|------|------|--------|
| Text | ~3s | Synchronous (done immediately) |
| Image | ~10s | Asynchronous (polling required) |
| Article/URL | ~15s | Asynchronous (polling required) |

---

## Job Lifecycle

```
pending → processing → done (success)
                    → pending (failure, retry)
                    → failed (max retries reached)
```

**Failed jobs:**
- Stay in queue (not deleted)
- User can retry: `POST /queue/:id/retry`
- User can discard: `POST /queue/:id/discard`

---

## Rate Limits

No rate limits for local use. For production/remote access, consider implementing rate limiting at the proxy level.

---

## OpenAPI Spec

For machine-readable API spec, see [openapi.yaml](openapi.yaml).

Can generate client code:

```bash
# Generate Go client
openapi-generator generate -i openapi.yaml -g go -o client/go

# Generate TypeScript client
openapi-generator generate -i openapi.yaml -g typescript-axios -o client/ts

# Generate Swift client
openapi-generator generate -i openapi.yaml -g swift5 -o client/swift

# Generate Kotlin client
openapi-generator generate -i openapi.yaml -g kotlin -o client/kotlin
```
