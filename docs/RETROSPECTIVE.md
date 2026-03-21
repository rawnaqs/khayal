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

### FTS5 Tokenizer and BM25 Weighting (2026-03-22)

**Problem:** Search quality was poor - exact matches didn't rank higher than partial matches.

**Changes:**

1. **Added `porter unicode61` tokenizer:**
   - **Porter stemming**: "running" matches "run", "runs", "runner"
   - **Unicode61**: Better handling of Unicode characters
   ```sql
   CREATE VIRTUAL TABLE notes_fts USING fts5(
       note_path UNINDEXED,  -- path is metadata, not searchable
       content,
       title,
       tags,
       tokenize = 'porter unicode61'
   )
   ```

2. **Added BM25 weighting:**
   ```sql
   ORDER BY bm25(notes_fts, 0, 3.0, 1.0, 1.0)
   ```
   - `note_path`: 0 (UNINDEXED)
   - `title`: 3.0 (3x more important)
   - `content`: 1.0 (base weight)
   - `tags`: 1.0 (same as content)

3. **Auto-drop on startup:**
   - `DROP TABLE IF EXISTS notes_fts` runs on every startup
   - Ensures schema changes are applied
   - Requires `khayal reindex --force` to repopulate

**Result:** Better search relevance with stemming and weighted ranking.

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

## Phase 3 + Phase 4: LLM + Worker (2026-03-19)

### Implementation Summary

| Component | File | Status |
|-----------|------|--------|
| LLM Interface | `internal/llm/interface.go` | ✅ |
| Ollama Client | `internal/llm/ollama.go` | ✅ |
| LLM Factory | `internal/llm/factory.go` | ✅ |
| Worker Pool | `internal/worker/worker.go` | ✅ |
| Text Ingest | `internal/ingest/text.go` | ✅ |
| Image Ingest | `internal/ingest/image.go` | ✅ |
| Article Ingest | `internal/ingest/article.go` | ✅ |
| HTML Parsing | `github.com/PuerkitoBio/goquery` | ✅ Added |

### Architecture Changes

**New Flow: Text Capture (Async)**
```
Before (Phase 2): Capture → Write vault → Index → Create job (done) → Return
After (Phase 3):   Capture → Create job (pending) → Return immediately
                               ↓
Worker picks up → Process with LLM → Write vault → Index → Update job (done)
```

### LLM Interface

```go
type LLM interface {
    Embed(text string) ([]float32, error)
    Generate(prompt string) (string, error)
    DescribeImage(imagePath string) (string, error)
    Ping() error
    Type() string
}

type LLMExt interface {
    LLM
    ExtractTags(content string) ([]string, error)
    Summarize(content string) (string, error)
    ExtractKeyIdeas(content string) ([]string, error)
}
```

### Ollama Client Features

| Method | Description |
|--------|-------------|
| `Ping()` | Check Ollama availability |
| `Embed()` | Generate vector embeddings |
| `Generate()` | Text generation |
| `DescribeImage()` | Vision with base64 image |
| `ExtractTags()` | 3-5 relevant tags |
| `Summarize()` | 2-3 sentence summary |
| `ExtractKeyIdeas()` | Key ideas as bullet points |

### Smart Truncation

```go
func truncateForLLM(content string, maxTokens int) string {
    maxChars := maxTokens * 4
    // Truncate at sentence boundary
}
```

### Bug Fix: Mock Embeddings in Production Code

**Problem:** `search.go` was using `mockEmbeddings()` instead of real LLM embeddings.

**Impact:** Hybrid search returned wrong results - all queries matched all documents.

**Fix:** Replaced `mockEmbeddings()` with `s.llm.Embed()`:
```go
// Before (wrong)
embedding := mockEmbeddings(query)

// After (correct)
embedding, err := s.llm.Embed(query)
```

### Bug Fix: Empty Embedding Panic

**Problem:** `cosine()` and `normalize()` panicked on empty slices.

**Fix:** Added length checks:
```go
func normalize(v []float32) float64 {
    if len(v) == 0 {
        return 0
    }
    // ...
}

func cosine(a, b []float32) float64 {
    if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
        return 0
    }
    // ...
}
```

### Bug Fix: RRF Merge Logic Error

**Problem:** Hybrid search returned unrelated queries matched notes.

**Root Cause:** In `mergeResultsRRF()`, the semantic results loop was adding RRF rank-based scores instead of keeping cosine similarity.

**Fix:** Simplified merge - keyword takes priority, semantic only fills gaps:
```go
for _, r := range keywordResults {
    r.Score = 1.0
    scoreMap[r.NotePath] = scoredResult{result: r, score: 1.0}
}

for _, r := range semanticResults {
    if _, exists := scoreMap[r.NotePath]; exists {
        continue  // Skip if already in keyword results
    }
    // Keep r.Score (cosine similarity), don't overwrite
    scoreMap[r.NotePath] = scoredResult{result: r, score: r.Score}
}
```

### Bug Fix: Semantic Search Threshold

**Problem:** Semantic search returned unrelated results even when cosine similarity was near zero.

**Root Cause:** All chunks with any non-zero similarity were returned, including random matches.

**Fix:** Added configurable minimum similarity threshold (default: 0.5):
```go
// In SearchSemantic():
if score < minScore {
    continue  // Skip results below threshold
}
```

**Config:**
```yaml
search:
  min_semantic_score: 0.5  # Minimum cosine similarity
```

**Result:** Semantic search only returns results with meaningful similarity (>50%).

### Bug Fix: Cosine Similarity Not Normalized

**Problem:** Cosine similarity scores were extremely high (215+) instead of -1 to 1 range.

**Root Cause:** `cosine()` returned raw dot product without dividing by vector norms.

**Fix:** Added proper normalization:
```go
func cosine(a, b []float32) float64 {
    // ...
    dot := float64(0)
    for i := range a {
        dot += float64(a[i]) * float64(b[i])
    }
    
    normA := normalize(a)
    normB := normalize(b)
    
    if normA == 0 || normB == 0 {
        return 0
    }
    
    return dot / (normA * normB)  // Proper cosine similarity
}
```

**Result:** Scores now in valid range (-1 to 1), threshold filtering works correctly.

### Bug Fix: RRF Merge Overwrote Semantic Scores

**Problem:** Hybrid search showed same score (0.016) for all results regardless of relevance.

**Root Cause:** `mergeResultsRRF()` overwrote semantic cosine similarity scores with RRF rank-based scores.

**Fix:** Semantic results now keep their original cosine similarity score:
```go
for _, r := range semanticResults {
    if _, exists := scoreMap[r.NotePath]; exists {
        continue
    }
    // r.Score is already cosine similarity - keep it
    scoreMap[r.NotePath] = scoredResult{result: r, score: r.Score}
}
```

**Result:** Hybrid search now shows meaningful similarity scores (0.0-1.0).

### Configurable Semantic Threshold

**Added:** `search.min_semantic_score` config option (default: 0.5)

```yaml
search:
  max_results: 50
  max_excerpt: 500
  rrf_k: 60
  min_semantic_score: 0.5  # Minimum cosine similarity to return semantic results
```

**Behavior:**
- Threshold 0.1: Returns more results, includes borderline matches
- Threshold 0.5: Stricter filtering, only high-similarity matches

### Worker Pool

- Configurable concurrency (`worker.max_workers`)
- Exponential backoff retry (2^retry seconds)
- Max retries: 3 (then job marked "failed")
- Crash recovery: Reset stuck "processing" jobs

### Code Quality Checklist

| Check | Status |
|-------|--------|
| `go vet` | ✅ Pass |
| Tests | ✅ All passing (Phase 1-4) |
| Context support | ✅ All DB operations |
| Error handling | ✅ No ignored errors |
| Named constants | ✅ No magic numbers |
| Interface defined | ✅ `JobStore` |
| Graceful shutdown | ✅ 30s timeout |
| LLM integration | ✅ Ollama client |
| Worker pool | ✅ Background processing |

---

## Performance Optimization: Concurrent LLM Calls (2026-03-19)

### Problem

Capture operations were slow due to sequential LLM calls:
- `IngestText`: ~15s (4 sequential calls)
- `IngestImage`: ~8s (3 sequential calls)
- `IngestArticle`: ~20s (4 sequential calls)

### Solution

Restructured LLM calls to run concurrently using `golang.org/x/sync/errgroup`:

#### IngestText
- **Before**: ExtractTags → Summarize → ExtractKeyIdeas → Embed (sequential)
- **After**: ExtractTags + Summarize + ExtractKeyIdeas (parallel) → Embed (sequential)
- **Speedup**: ~3x

#### IngestImage
- **Before**: DescribeImage → ExtractTags → Embed (sequential)
- **After**: DescribeImage → ExtractTags + Embed (parallel)
- **Speedup**: ~1.6x

#### IngestArticle
- **Before**: Summarize → ExtractKeyIdeas → ExtractTags → Embed (sequential)
- **After**: Summarize + ExtractKeyIdeas + ExtractTags (parallel) → Embed (sequential)
- **Speedup**: ~3x

### Ollama Batch Embeddings API

Added `EmbedBatch` to the LLM interface to support batch embedding when multiple embeddings are needed:

```json
POST /api/embeddings
{
  "model": "nomic-embed-text",
  "prompts": [
    {"prompt": "text 1"},
    {"prompt": "text 2"}
  ]
}
```

Response:
```json
{
  "embeddings": [
    [0.1, 0.2, ...],
    [0.3, 0.4, ...]
  ]
}
```

**Note:** `EmbedBatch` is available in the interface but current ingest flow uses single `Embed` calls. Batch embeddings will be useful for future chunking support.

### Error Handling Change

**Before:** Graceful degradation (e.g., `keyIdeas = []string{}` on failure)
**After:** Fail-fast — any LLM call failure returns error, job marked as failed

This is consistent with the worker retry flow:
1. Job fails → marked as `failed`
2. User can retry via `POST /queue/{id}/retry`
3. Worker picks up and retries with exponential backoff

### Dependencies Added

```
golang.org/x/sync v0.20.0
```

### Implementation

| File | Change |
|------|--------|
| `internal/llm/interface.go` | Added `EmbedBatch(texts []string) ([][]float32, error)` |
| `internal/llm/ollama.go` | Implemented `EmbedBatch` and `EmbedBatchWithModel` |
| `internal/ingest/text.go` | Restructured with `errgroup` |
| `internal/ingest/image.go` | Restructured with `errgroup` |
| `internal/ingest/article.go` | Restructured with `errgroup` |
| `internal/ingest/ingest_test.go` | New tests for concurrency and fail-fast |
| `internal/llm/ollama_test.go` | New tests for batch embeddings |

### Tests Added

- `TestIngestText_BasicSuccess` — Verifies basic success case
- `TestIngestText_ConcurrentExecution` — Verifies parallel execution (takes ~50ms vs ~150ms sequential)
- `TestIngestText_FailFastOnError` — Verifies error propagation
- `TestOllamaClient_EmbedBatch*` — 6 tests for batch embedding API

### Estimated Performance Improvement

| Function | Before | After | Speedup |
|----------|--------|-------|---------|
| `IngestText` | ~15s | ~5s | **~3x** |
| `IngestImage` | ~8s | ~5s | **~1.6x** |
| `IngestArticle` | ~20s | ~7s | **~3x** |

---

## Comprehensive Logging System (2026-03-19)

### Implementation Summary

| Component | File | Status |
|-----------|------|--------|
| Rotating log handler | `internal/log/rotating.go` | ✅ |
| Multi-handler setup | `internal/log/setup.go` | ✅ |
| LogConfig struct | `internal/config/config.go` | ✅ |
| Wired up in main.go | `cmd/khayal/main.go` | ✅ |

### Features

1. **Dual output**: File + stdout simultaneously
2. **JSON format**: Structured logs for machine parsing
3. **Size-based rotation**: Configurable max file size with gzip compression
4. **Worker-specific levels**: Main log level + separate worker log level
5. **Panic recovery**: Full stack traces logged to file

### Config

```yaml
logging:
  level: "info"           # Main log level (debug, info, warn, error)
  worker_level: "debug"  # Worker log level (separate for noisy workers)
  file: "logs/khayal.log"
  max_size_mb: 10
  max_backups: 5
  compress: true
```

### Design Decisions

1. **Standard library only**: No external dependencies (log/slog)
2. **Built-in rotation**: Uses `slog.Handler` with custom RotateHandler
3. **Gzip compression**: Compresses rotated files to save space
4. **Context propagation**: `_context` field for request tracing

### Files Changed

| File | Change |
|------|--------|
| `internal/log/rotating.go` | New - file rotation with gzip |
| `internal/log/setup.go` | New - multi-handler for file + stdout |
| `internal/config/config.go` | Added LogConfig struct |
| `cmd/khayal/main.go` | Wired up logging with config |
| `testdata/config.yaml` | Added log settings |

---

## Path Handling Improvements (2026-03-19)

### Problem

1. Config paths were resolved relative to CWD, not config.yaml location
2. No `~` expansion in log file paths
3. Trash location was at vault root instead of inbox directory
4. Duplicate `expandPath` implementations
5. No validation that paths were within vault boundaries

### Solution

#### 1. MakeAbsolute with Config Path

```go
func MakeAbsolute(path, configPath string) (string, error) {
    if filepath.IsAbs(path) {
        return path, nil
    }
    if strings.HasPrefix(path, "~") {
        home, err := os.UserHomeDir()
        // ...
    }
    if strings.HasPrefix(path, "$") {
        // Expand env vars
    }
    // Resolve relative to config location
    configDir := filepath.Dir(configPath)
    return filepath.Join(configDir, path), nil
}
```

#### 2. LoadFromPath Returns Config Path

```go
func LoadFromPath(path string) (*Config, string, error) {
    // ...
    return cfg, path, nil  // Returns the config path for relative resolution
}
```

#### 3. Vault Path Validation

Helper functions for path validation:
- `isPathInVault(path)` - Check if path is within vault
- `isPathInInbox(path)` - Check if path is within inbox
- `isPathInMedia(path)` - Check if path is within media directory
- `ensurePathInVault(path)` - Validate + error if outside
- `ensurePathInInbox(path)` - Validate + error if outside

#### 4. Sentinel Errors

```go
var (
    ErrVaultPathNotAbsolute    = errors.New("VAULT_003: path must be absolute")
    ErrVaultPathOutsideVault  = errors.New("VAULT_004: path outside vault")
    ErrVaultPathOutsideInbox  = errors.New("VAULT_005: path outside inbox")
    ErrVaultNoteNotFound      = errors.New("VAULT_006: note not found")
)
```

#### 5. Trash Location Fix

**Before:** `<vault>/.khayal-trash`
**After:** `<inbox>/.khayal-trash`

Now consistent with Rule #9: "Never write outside <inbox_dir>/"

### Files Changed

| File | Change |
|------|--------|
| `internal/config/config.go` | MakeAbsolute, LoadFromPath returns path, ValidateVaultSubPath |
| `internal/vault/writer.go` | Removed duplicate expandPath, added path validation |
| `internal/ingest/image.go` | Fixed image path: `v.ResolvePath(job.SourceFile)` |
| `testdata/config.yaml` | Relative paths, inbox/media instead of inbox/media/images |

### Key Discoveries

1. **No stdlib for `~` expansion** - Must use `os.UserHomeDir()` manually
2. **os.ReadFile doesn't expand `~`** - Need to expand before OS operations
3. **Config paths need base dir** - Cannot just use filepath.Abs() on relative paths

---

## Article Content Fix + Configurable Truncation (2026-03-19)

### Problem

Article scraping was stripping content:
- Hard limit of 20 paragraphs
- Minimum paragraph length of 20 chars (too restrictive)
- Limited element types (p, h2, h3, h4, li only)
- Content truncated BEFORE storing in Raw field

### Solution

#### 1. Full Content Storage
- `scrapeArticle()` now returns full extracted content
- `note.Raw = combinedContent` stores everything (title + content)
- Raw field contains complete article text

#### 2. Configurable Truncation Limits

Added per-capture-type truncation limits to config:

```yaml
llm:
  truncate_text_tokens: 2000    # ~8k chars for text captures
  truncate_image_tokens: 3000    # ~12k chars for image descriptions
  truncate_article_tokens: 12000  # ~48k chars for articles
```

#### 3. Updated LLM Interface

LLM methods now accept a bucket parameter to determine truncation limits:

```go
type LLMExt interface {
    ExtractTags(content string, bucket string) ([]string, error)
    Summarize(content string, bucket string) (string, error)
    ExtractKeyIdeas(content string, bucket string) ([]string, error)
}

const (
    BucketText    = "text"
    BucketImage   = "image"
    BucketArticle = "article"
)
```

#### 4. Ingest Functions Use Correct Buckets

| Function | LLM Methods | Bucket Used |
|----------|------------|-------------|
| `IngestText` | ExtractTags, Summarize, ExtractKeyIdeas | `text` |
| `IngestImage` | ExtractTags | `image` |
| `IngestArticle` | ExtractTags, Summarize, ExtractKeyIdeas | `article` |

#### 5. Improved Article Scraping

- Removed hard 20-paragraph limit
- Expanded element types: `blockquote, pre, code, figure, figcaption`
- Expanded selectors: `.article-content, .post-content, .story-body`
- Lowered paragraph minimum: 10 chars (was 20)

### Implementation

| File | Change |
|------|--------|
| `internal/config/config.go` | Added `TruncateTextTokens`, `TruncateImageTokens`, `TruncateArticleTokens` |
| `testdata/config.yaml` | Added truncation settings |
| `internal/llm/interface.go` | Added bucket parameter to LLM methods, added bucket constants |
| `internal/llm/ollama.go` | Added `truncateLimit()` method, updated methods to use bucket |
| `internal/llm/factory.go` | Pass truncation config to client |
| `internal/ingest/article.go` | Store full content in Raw, improved scraping |
| `internal/ingest/text.go` | Pass `BucketText` to LLM methods |
| `internal/ingest/image.go` | Pass `BucketImage` to LLM methods |

---

## v1.1: Smart Chunking (Future)

### Concept

For very long articles, split into chunks and process concurrently:

```
Article (50 paragraphs)
       ↓
┌──────┴──────┐
│   Chunk 1    │   Chunk 2    │   Chunk 3
│ (paragraphs  │ (paragraphs  │ (paragraphs
│   1-20)     │   21-40)     │   41-50)
       ↓             ↓             ↓
   Summarize     Summarize     Summarize
       ↓             ↓             ↓
   chunk1Sum    chunk2Sum    chunk3Sum
       └─────────────┴─────────────┘
                     ↓
              Final Summary
```

### Planned Interface Changes

```go
type LLMChunkedOperations interface {
    LLMExt
    SummarizeChunks(chunks []string) (string, error)
    ExtractTagsFromChunks(chunks []string) ([]string, error)
    ExtractKeyIdeasFromChunks(chunks []string) ([]string, error)
}

func chunkArticle(content string, chunkSize int) []string {
    // Split by paragraphs, target ~8000 tokens per chunk
}
```

### Benefits
- Handles arbitrarily long articles
- Better quality (each chunk processed fully)
- Parallel processing via existing `errgroup` pattern

---

## SQLite Lock Contention Fixes (2026-03-22)

### Problem

Stress test with 20 concurrent captures was failing:
- `QUEUE_CREATE_FAILED` errors appearing
- Jobs getting stuck in 'processing' state
- Workers not completing jobs

### Root Cause Analysis

1. **SQLite default settings**: No WAL mode, no busy_timeout
   - Only ONE writer allowed at a time
   - 20 concurrent requests → 19 fail immediately

2. **LLM overload**: 30 workers × 4 LLM calls = 120 concurrent requests to Ollama
   - Ollama can only handle ~4 concurrent requests
   - Jobs hung forever waiting for LLM responses

3. **Race condition**: Multiple workers could fetch same job
   - `GetPendingJobs()` SELECT + later `UpdateJobStatus()` UPDATE not atomic
   - Duplicate job processing

### Solutions Implemented

#### 1. SQLite PRAGMAs

```go
// Enable WAL mode for concurrent reads/writes
db.Exec("PRAGMA journal_mode=WAL")

// Set busy timeout for automatic lock retry
db.Exec("PRAGMA busy_timeout=5000")

// No connection limit - let SQLite handle contention via busy_timeout
// WAL mode supports concurrent readers + 1 writer
db.SetMaxOpenConns(0)
db.SetMaxIdleConns(2)
db.SetConnMaxLifetime(time.Hour)
```

**WAL Mode Benefits:**
- Concurrent readers allowed during writes
- Better performance under load
- No read/write blocking

**Busy Timeout:**
- Retries lock acquisition up to 5 seconds
- Handles transient lock contention

**Connection Pool:**
- Unlimited connections (`SetMaxOpenConns(0)`)
- SQLite handles writer serialization via busy_timeout
- Search queries don't block when workers are writing

#### 2. Lock Retry Logic

Critical write operations include retry logic:

```go
const maxRetries = 3
for attempt := 0; attempt < maxRetries; attempt++ {
    _, err = q.db.ExecContext(ctx, `INSERT INTO jobs ...`)
    if err == nil {
        return nil
    }
    if isLockError(err) {
        time.Sleep(100 * time.Millisecond)
        continue
    }
    break  // Non-lock error
}
return err
```

Applied to: `CreateJob`, `UpdateJob`, `UpdateJobStatus`, `IndexNote`, `UpdateNoteIndex`, `DeleteFromIndex`

#### 3. LLM Concurrency Limit

Added semaphore to prevent Ollama overload:

```go
type OllamaClient struct {
    semaphore chan struct{}  // e.g., make(chan struct{}, 4)
}

func (c *OllamaClient) acquire(ctx context.Context) error {
    select {
    case c.semaphore <- struct{}{}:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (c *OllamaClient) release() {
    <-c.semaphore
}
```

**Default**: 4 concurrent LLM requests (configurable via `llm.max_llm_concurrency`)

#### 4. Worker Job Timeout

Added 120-second timeout per job:

```go
func (w *Worker) processJob(jobID string) {
    ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
    defer cancel()
    // ... process job
}
```

**Benefits:**
- Jobs fail fast instead of hanging forever
- Worker slots freed up for next job
- Visible error messages

#### 5. Atomic Fetch+Lock Pattern

Replaced separate SELECT + UPDATE with single atomic operation:

```go
func (q *Queue) FetchAndLockPendingJobs(ctx context.Context, limit int) ([]Job, error) {
    // Single atomic operation: UPDATE + RETURNING
    rows, err := q.db.QueryContext(ctx, `
        UPDATE jobs 
        SET status = 'queued'
        WHERE id IN (
            SELECT id FROM jobs 
            WHERE status = 'pending' 
            ORDER BY created_at ASC 
            LIMIT ?
        )
        RETURNING id, type, status, note_path, ...`,
        limit)
    // ... scan and return jobs
}
```

**Benefits:**
- Only one worker can fetch each job
- Status immediately set to 'queued'
- No duplicate processing

#### 6. Job Status Flow

Added 'queued' status between 'pending' and 'processing':

```
pending → queued → processing → done/failed
   ↑                  ↓
   └──────────────────┘ (on retry)
```

**Status Meanings:**
- **pending**: Job created, waiting to be fetched
- **queued**: Fetched from database, waiting in channel for worker
- **processing**: Worker actively processing the job
- **done**: Job completed successfully
- **failed**: Job permanently failed (max retries exceeded)

**Crash Recovery:**
`ResetStuckJobs()` now resets both 'processing' and 'queued' jobs to 'pending':

```go
UPDATE jobs SET status = 'pending' WHERE status = 'processing' OR status = 'queued'
```

### Files Changed

| File | Changes |
|------|---------|
| `internal/queue/queue.go` | SQLite PRAGMAs, lock retry, atomic fetch+lock |
| `internal/llm/ollama.go` | Concurrency semaphore, acquire/release |
| `internal/worker/worker.go` | 120s timeout, atomic fetch, queued status |
| `internal/api/health.go` | Added queued count |
| `internal/config/config.go` | Added max_llm_concurrency config |
| `cmd/khayal/commands/status.go` | Added queued display |
| `cmd/kl/commands/status.go` | Added queued display |
| `tests/stress/main.go` | Updated wait logic for queued status |

### Lessons Learned

1. **SQLite defaults are not production-ready**: Always set WAL mode and busy_timeout for concurrent access
2. **Worker pools need backpressure**: Don't create more workers than the underlying system can handle
3. **Atomic operations prevent races**: Use UPDATE...RETURNING for fetch+lock patterns
4. **Timeouts are essential**: Never let jobs hang forever - fail fast and retry
5. **Status tracking helps debugging**: 'queued' status provides visibility into job lifecycle

---

## Logging Improvements (2026-03-22)

### Problem

API handlers were not logging errors or warnings, making debugging difficult.

### Solution

Added comprehensive error/warn logging to all API handlers:

| Handler | Error Logging | Warn Logging |
|---------|---------------|--------------|
| capture.go | 9 error points | 2 warn points |
| queue.go | 3 error points | 2 invalid state warnings |
| search.go | 1 error point | - |
| stats.go | 7 error points | - |
| health.go | - | 2 degraded state warnings |
| middleware.go | - | 4xx status warnings |

### Log Fields

All logs include:
- `code`: Error code (e.g., `QUEUE_CREATE_FAILED`)
- `query`: For search (truncated to 50 chars)
- `error`: Actual error message
- `job_id`: For job operations
- `type`: For capture operations

### Files Changed

| File | Changes |
|------|---------|
| `internal/api/search.go` | Error logging with code, query, mode, error |
| `internal/api/capture.go` | 9 error/warn logging points |
| `internal/api/queue.go` | 5 logging points |
| `internal/api/stats.go` | 7 error logging points |
| `internal/api/health.go` | Warn logging for degraded dependencies |
| `internal/api/middleware/middleware.go` | WARN level for 4xx status
