# Khayal Retrospective

> History of decisions, discoveries, and lessons learned. Updated: 2026-03-19

---

## Phase 1: Foundation Decisions

### 2026-03-17: SQLite Driver Selection

| Aspect | Decision |
|--------|----------|
| Initial choice | `mattn/go-sqlite3` |
| Final choice | `modernc.org/sqlite` |
| Reason | Pure Go, no CGO, no system dependencies |

#### Why Not mattn/go-sqlite3?

1. **CGO Required** - Must compile C code, complicates builds
2. **macOS Deprecation** - `sqlite3_auto_extension` deprecated on macOS
3. **Build warnings** - Users see warnings during compilation
4. **Deployment complexity** - Requires cross-compiler setup

#### Why modernc.org/sqlite?

1. **Pure Go** - No CGO, compiles like any Go package
2. **Zero dependencies** - No system sqlite3 needed
3. **FTS5 included** - Full-text search works out of the box
4. **Cross-platform** - Same binary works everywhere

#### Trade-offs Accepted

- Larger binary size (~15MB vs ~5MB)
- No access to SQLite C extensions
- Some SQLite PRAGMAs behave differently

---

### 2026-03-18: Vector Search Approach

| Aspect | Decision |
|--------|----------|
| Initial attempt | viant/sqlite-vec |
| Final choice | Pure Go cosine similarity |
| Reason | Virtual table deadlocks with modernc.org/sqlite |

#### Investigation Timeline

**Attempt 1: viant/sqlite-vec with single DB**
```
db, _ := engine.Open("test.db")
db.SetMaxOpenConns(1)
vec.Register(db)
db.Exec(`CREATE VIRTUAL TABLE vec_chunks USING vec(chunk_id)`)
```
**Result:** Module registration fails silently

**Attempt 2: SetMaxOpenConns after registration**
```
db.SetMaxOpenConns(1)
vec.Register(db)
db.SetMaxOpenConns(4)
db.Exec(`CREATE VIRTUAL TABLE vec_chunks USING vec(chunk_id)`)
```
**Result:** CREATE works, but queries deadlock

**Root cause identified from stack trace:**
```
database/sql.(*DB).query
  → vec.(*Cursor).Filter
    → vec.(*Table).ensureIndex
      → database/sql.(*DB).Exec  ← DEADLOCK
```
The virtual table holds a connection while executing more SQL on the same connection.

**Attempt 3: Separate databases**
```
mainDB, _ := engine.Open("main.db")  // App data
vecDB, _ := engine.Open("vec.db")     // Vec module
vec.Register(vecDB)
vecDB.Exec(`CREATE VIRTUAL TABLE vec_chunks USING vec(chunk_id, dbpath='main.db')`)
```
**Result:** Works! But adds complexity (two files)

**Attempt 4: WAL mode with more connections**
```
db.SetMaxOpenConns(4)
db.Exec(`PRAGMA journal_mode=WAL`)
```
**Result:** Single query works, concurrent queries deadlock

#### Final Decision: Pure Go Cosine Similarity

**Implementation:**
```go
func cosine(a, b []float32) float64 {
    dot := float64(0)
    for i := range a {
        dot += float64(a[i]) * float64(b[i])
    }
    return dot  // Already normalized
}
```

**Batch processing:**
```go
const batchSize = 1000
for offset := 0; offset < maxChunks; offset += batchSize {
    rows, _ := db.Query(`SELECT ... LIMIT ? OFFSET ?`, batchSize, offset)
    // Compute similarities
}
```

**Deduplication:**
```go
noteBest := make(map[string]scoredChunk)  // One result per note_path
for rows.Next() {
    score := cosine(query, embedding)
    if score > noteBest[notePath].score {
        noteBest[notePath] = scoredChunk{...}
    }
}
```

**Why this works:**
1. No virtual tables (no connection holding during exec)
2. Simple SQL queries with batching
3. All computation in Go memory
4. No external dependencies

**Limitations:**
- O(n) scan for each search
- Not suitable for millions of embeddings
- No pre-built index

**When to upgrade:**
- If embeddings exceed ~100K
- If search latency > 500ms
- If CPU usage becomes problematic

---

## Key Principles Established

### 1. Pure Go First

Always prefer pure Go dependencies when available:
- No CGO complications
- Reproducible builds
- Cross-platform compatibility
- No system dependency management

### 2. Test External Assumptions

Never assume an external library works as documented. Test:
- Single-threaded scenarios
- Concurrent access
- Edge cases
- Error conditions

### 3. Simplicity Over Optimization

Choose simpler solutions over optimized ones:
- Pure Go cosine vs sqlite-vec: simpler, works, fast enough
- Single DB vs separate DBs: simpler, one file to manage
- In-memory vs precomputed indexes: simpler until proven needed

### 4. Document Decisions

Record not just what was chosen, but why:
- What was tried
- What failed
- What trade-offs exist

---

## Future Considerations

### When to Re-evaluate

| Component | Trigger for re-evaluation |
|-----------|---------------------------|
| Vector search | >100K embeddings or >500ms latency |
| SQLite driver | major breaking change in modernc.org/sqlite |
| Single DB | User requests for separate vec database |

### Potential Upgrades

1. **Vector index (when needed)**
   - HNSW implementation in Go
   - Or try sqlite-vec with separate DBs
   - Or use a dedicated vector DB (Chroma, Qdrant)

2. **Batch async indexing**
   - Background worker for embedding computation
   - Precomputed norms column
   - Incremental index updates

3. **Query optimization**
   - Annoy or FAISS bindings if Go implementation insufficient
   - PostgreSQL with pgvector for server deployments

---

## Phase 2: Core API (2026-03-19)

### Implementation Summary

| Component | Status | Notes |
|-----------|--------|-------|
| Chi router | ✅ Complete | With graceful shutdown (30s timeout) |
| Auth middleware | ✅ Complete | Token via `X-Khayal-Token` header |
| Request logger | ✅ Complete | Uses `log/slog`, panic recovery |
| Health endpoint | ✅ Complete | `/v1/health` |
| Capture endpoint | ✅ Complete | Text sync, URL/image queued |
| Queue endpoints | ✅ Complete | List, get, retry, discard |
| Search endpoint | ✅ Complete | Keyword + semantic + hybrid (RRF) |
| Tests | ✅ 16 passing | All handlers tested |

### Key Design Decisions

1. **Single middleware file** - Combined auth and logging into `middleware.go` instead of separate files
2. **Chi router in tests** - Use chi router for path param tests to ensure handlers work in routing context
3. **JSON helpers** - `WriteJSON`, `WriteError`, `WriteCreated`, `WriteNoContent` for consistent responses
4. **Mock embeddings** - Semantic search uses deterministic mock for testing; real embeddings in Phase 4

### Config Additions

```go
ServerConfig {
    MaxTextBodyMB    int  // default: 1
    MaxImageBodyMB   int  // default: 10  
    ShutdownTimeoutS int  // default: 30
}
SearchConfig {
    MaxResults int  // default: 50
    MaxExcerpt int  // default: 500
    RRFK       int  // default: 60 (Reciprocal Rank Fusion constant)
}
```

### Testing Pattern

Handlers tested via `httptest.NewRecorder`:
```go
req := httptest.NewRequest(http.MethodGet, "/v1/queue", nil)
rec := httptest.NewRecorder()
ts.Server.queueListHandler(rec, req)
```

For path params, wrap with chi router:
```go
r := chi.NewRouter()
r.Get("/v1/queue/{id}", ts.Server.queueGetHandler)
r.ServeHTTP(rec, req)
```

---

## Bug Fix (2026-03-19)

### Issue: Capture Not Saving Notes

**Problem:** Text captures created jobs in the database but never wrote notes to the vault.

**Root Cause:** `handleTextCapture()` set `NotePath` but never called `vault.WriteNote()`.

**Fix:** Added proper note writing flow:
```go
if jobType == "text" {
    note := &vault.Note{
        Metadata: vault.NoteMetadata{...},
        Title: extractTitle(req.Content),
        Raw:   req.Content,
    }
    notePath, err := s.vault.WriteNote(note)
    // Index for FTS5 search
    s.queue.IndexNote(ctx, notePath, note.Title, req.Content, "")
}
```

**Test Added:** `TestCaptureText` now verifies `ts.Vault.NoteExists(resp.NotePath)`.

### Issue: Search Returns Empty Results (FTS5 Contentless Mode)

**Problem:** Captured notes were saved to vault but search returned no results.

**Root Cause:** FTS5 table was created with `content=''` (contentless mode):
```sql
CREATE VIRTUAL TABLE notes_fts USING fts5(
    note_path, content, title, tags,
    content='',           -- <-- Contentless mode
    contentless_delete=1
)
```

Contentless FTS5 requires external content table and `rowid` matching. Direct inserts fail silently.

**Fix:** Removed contentless mode parameters:
```sql
CREATE VIRTUAL TABLE notes_fts USING fts5(
    note_path, content, title, tags
)
```

**Note:** Required deleting `testdata/khayal.db` to recreate with correct schema.

### Issue: Duplicate History Frontmatter

**Problem:** Notes had duplicate `history:` keys in frontmatter or malformed YAML.

**Root Cause:** 
1. YAML marshaling outputs `history:` followed by indented list items
2. Initial fix only skipped the `history:` line but not the following list items
3. Then explicit history block was added, causing duplicates or malformed YAML

**Fix:** Replaced YAML marshaling with manual frontmatter construction:
```go
func (w *Writer) renderNote(note *Note) string {
    buf.WriteString("---\n")
    buf.WriteString(fmt.Sprintf("created: %s\n", note.Metadata.Created.Format(time.RFC3339)))
    // ... explicit field-by-field construction
    buf.WriteString("---\n\n")
}
```

**Result:** Clean, predictable YAML output with exactly one `history:` block.

---

## Code Quality Checklist

| Check | Status |
|-------|--------|
| `go vet` | ✅ Pass |
| Tests | ✅ 24 passing (Phase 1) + 16 passing (Phase 2) = 40 total |
| Context support | ✅ All DB operations |
| Error handling | ✅ No ignored errors |
| Named constants | ✅ No magic numbers |
| Interface defined | ✅ `JobStore` |
| Graceful shutdown | ✅ 30s timeout |
