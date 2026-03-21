# Khayal API Reference

> Complete API reference for Khayal v1. Updated: 2026-03-17

## Base URL

```
http://localhost:1133/v1
```

## Authentication

All requests require the `X-Khayal-Token` header:

```bash
curl -H "X-Khayal-Token: your-token-here" \
     http://localhost:1133/v1/health
```

See [AUTH.md](AUTH.md) for detailed authentication guide.

## Endpoints

---

### GET /stats

Get vault statistics.

**Response:**
```json
{
  "total": 2847,
  "this_week": 23,
  "this_month": 94,
  "by_type": {
    "text": 1420,
    "article": 890,
    "image": 537
  },
  "top_tags": [
    { "name": "react", "count": 142 },
    { "name": "go", "count": 98 },
    { "name": "work", "count": 87 }
  ],
  "top_people": [
    { "name": "John Doe", "count": 34 },
    { "name": "Sarah Chen", "count": 18 }
  ],
  "capture_streak": 12,
  "longest_streak": 34
}
```

---

### GET /queue

List jobs in queue.

**Parameters:**

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| status | No | all | Filter: all, pending, queued, processing, done, failed |
| limit | No | 20 | Max results (max 100) |
| offset | No | 0 | Pagination offset |

**Example:**
```bash
curl "http://localhost:1133/v1/queue?status=failed&limit=10" \
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
curl http://localhost:1133/v1/queue/abc123 \
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
curl -X POST http://localhost:1133/v1/queue/abc123/retry \
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
curl -X POST http://localhost:1133/v1/queue/abc123/discard \
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

All errors return `{"error": "...", "code": "..."}`. See SPEC.md Error Taxonomy for full list.

| Code | HTTP Status | Description |
|------|-------------|-------------|
| AUTH_TOKEN_MISSING | 401 | Token missing |
| AUTH_TOKEN_INVALID | 401 | Invalid token |
| CAPTURE_BODY_TOO_LARGE | 413 | Request body too large |
| CAPTURE_INVALID_BODY | 400 | Invalid request body |
| CAPTURE_MISSING_CONTENT | 400 | Missing required field: content |
| CAPTURE_INVALID_FORM | 413 | Invalid multipart form |
| CAPTURE_MISSING_FILE | 400 | Missing file |
| CAPTURE_READ_FAILED | 500 | Failed to read file |
| QUEUE_CREATE_FAILED | 500 | Failed to create job |
| QUEUE_LIST_FAILED | 500 | Failed to list jobs |
| QUEUE_JOB_NOT_FOUND | 404 | Job not found |
| QUEUE_INVALID_STATE | 400 | Invalid job state for operation |
| QUEUE_UPDATE_FAILED | 500 | Failed to update job |
| QUEUE_DELETE_FAILED | 500 | Failed to delete job |
| VAULT_MEDIA_FAILED | 500 | Failed to save media |
| SEARCH_MISSING_QUERY | 400 | Missing query parameter |
| SEARCH_INVALID_MODE | 400 | Invalid search mode |
| SEARCH_FAILED | 500 | Search operation failed |
| COUNT_ERROR | 500 | Database error |

### Examples

```json
400 { "error": "missing required field: content", "code": "CAPTURE_MISSING_CONTENT" }
401 { "error": "invalid token", "code": "AUTH_TOKEN_INVALID" }
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
