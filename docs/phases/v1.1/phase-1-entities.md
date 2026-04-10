# Phase 1: Entity Extraction

> Agent instruction file. Read all of it before writing any code.

---

## What This Phase Does

Adds structured entity extraction to the ingest pipeline. After tags and summary are
generated, a new LLM call extracts people, amounts, dates, places, orgs, and URLs from
the note content. Entities are written to:

1. The note's frontmatter (`entities:` block)
2. A new `entities` SQL table for fast lookup (used by proactive connections in Phase 3)

This is **purely additive** — no existing tables are modified, no migration runner
needed, no reindex required. Existing notes simply won't have entities until re-processed.

---

## Files To Read First

Before writing any code, read these files in full:

```
RULES.md          ← memory management rules
TECH_STACK.md     ← modernc.org/sqlite, pure Go, no CGO
RETROSPECTIVE.md  ← phase 3+4 section: ingest pipeline details, errgroup pattern,
                    fail-fast error handling, LLM concurrency semaphore
REPO_STRUCTURE.md ← DB schema, file tree
```

Key facts that affect this phase:

- Ingest paths: `internal/ingest/text.go`, `image.go`, `article.go`
- All three already use `golang.org/x/sync/errgroup` for parallel LLM calls
- Any LLM call failure → entire job fails → worker retries (fail-fast, no graceful degradation)
- `internal/queue/queue.go` owns all DB operations — ingest does NOT touch DB directly
- Vault writer is in `internal/vault/writer.go` — frontmatter is built manually (not via YAML marshal)
- LLM interface is in `internal/llm/interface.go`
- `internal/constants/constants.go` holds prompts — add the entity extraction prompt there

---

## New Files To Create

```
internal/ingest/entities.go       ← extraction logic + normalization
internal/ingest/entities_test.go  ← unit tests
```

## Files To Modify

```
internal/queue/queue.go           ← new entities table + SaveEntities + DeleteEntities
internal/llm/interface.go         ← new ExtractEntities method on LLMExt
internal/llm/ollama.go            ← implement ExtractEntities
internal/llm/factory.go           ← no change likely, verify
internal/constants/constants.go   ← add entity extraction prompt
internal/ingest/text.go           ← call ExtractEntities, write to frontmatter + DB
internal/ingest/image.go          ← same
internal/ingest/article.go        ← same
internal/vault/writer.go          ← add entities block to frontmatter rendering
```

---

## Step 1 — Database

Add to `internal/queue/queue.go`, in the schema init block alongside existing table
creation:

```sql
CREATE TABLE IF NOT EXISTS entities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_path TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_value TEXT NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_entities_note ON entities(note_path);
CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(entity_type);
CREATE INDEX IF NOT EXISTS idx_entities_value ON entities(entity_value);
```

Add two methods to `Queue`:

```go
// SaveEntities stores extracted entities for a note.
// Replaces any existing entities for that note_path (idempotent).
// entity_type values: "person", "amount", "date", "place", "org", "url"
func (q *Queue) SaveEntities(ctx context.Context, notePath string, entities Entities) error

// DeleteEntities removes all entities for a note_path.
func (q *Queue) DeleteEntities(ctx context.Context, notePath string) error
```

`Entities` is the struct defined in Step 2. `SaveEntities` runs in a single transaction:
delete existing rows for `notePath`, then insert one row per entity value per type.
Use the same lock retry pattern as other write methods in queue.go (see RETROSPECTIVE.md
"Lock Retry Logic" section).

---

## Step 2 — Entities Struct

Define in `internal/ingest/entities.go`:

```go
package ingest

// Entities holds structured entities extracted from note content.
// All fields are slices — empty slice means no entities of that type found.
type Entities struct {
    People  []string `json:"people"`
    Amounts []string `json:"amounts"`
    Dates   []string `json:"dates"`
    Places  []string `json:"places"`
    Orgs    []string `json:"orgs"`
    URLs    []string `json:"urls"`
}
```

Also define the normalization functions here (Step 3).

---

## Step 3 — Normalization

### Amount normalization

Amounts from the LLM come in inconsistent forms. Normalize to plain integer strings
before storing. All normalization is pure Go — no LLM involved.

```go
// normalizeAmount converts an amount string to a plain integer string.
// Examples:
//   "2k"     → "2000"
//   "$2,000" → "2000"
//   "2.5k"   → "2500"
//   "£1m"    → "1000000"
//   "500"    → "500"
//   "weird"  → "" (discard)
func normalizeAmount(s string) string
```

Rules:
- Strip leading currency symbols: `$`, `£`, `€`, `¥`
- Strip commas
- Parse suffix multipliers: `k`/`K` → ×1000, `m`/`M` → ×1000000, `b`/`B` → ×1000000000
- Result must be a whole number — if fractional after multiplier, round to nearest integer
- If parsing fails entirely, return `""` — caller discards empty strings

### Name normalization (people)

Deduplicate when a short form and long form of the same name both appear.

```go
// normalizeNames deduplicates a list of person names.
// If a shorter name is a subset of a longer name, keep only the longer form.
// Examples:
//   ["John", "John Doe"] → ["John Doe"]
//   ["Sarah", "Sarah Connor", "James"] → ["Sarah Connor", "James"]
//   ["Alice", "Bob"] → ["Alice", "Bob"]  (no overlap)
func normalizeNames(names []string) []string
```

Rules:
- Case-insensitive substring check: if name A is contained within name B (word-boundary
  match, not just substring), discard A and keep B
- "John" inside "John Doe" → keep "John Doe"
- "Jo" inside "John" → do NOT discard "Jo" (not a word-boundary match)
- If two names are identical after trimming whitespace, keep one
- Preserve original casing of the kept name
- Order of output does not matter

### Apply normalization

In `ExtractEntities` (Step 4), after parsing the LLM JSON response:

```go
entities.People  = normalizeNames(raw.People)
entities.Amounts = normalizeAmounts(raw.Amounts)  // map normalizeAmount over slice, discard ""
entities.Dates   = raw.Dates                       // no normalization
entities.Places  = raw.Places                      // no normalization
entities.Orgs    = raw.Orgs                        // no normalization
entities.URLs    = raw.URLs                        // no normalization
```

`normalizeAmounts` is a simple helper:
```go
func normalizeAmounts(raw []string) []string {
    out := make([]string, 0, len(raw))
    for _, s := range raw {
        if n := normalizeAmount(s); n != "" {
            out = append(out, n)
        }
    }
    return out
}
```

---

## Step 4 — LLM Interface + Prompt

### Prompt

Add to `internal/constants/constants.go`:

```go
PromptExtractEntities = `Extract structured entities from the following content.
Return ONLY a JSON object with these exact keys. No markdown, no explanation.

{
  "people":  [],
  "amounts": [],
  "dates":   [],
  "places":  [],
  "orgs":    [],
  "urls":    []
}

Rules:
- people: full names of real people mentioned (not fictional, not the author)
- amounts: monetary or numerical amounts (e.g. "$2,000", "2k", "500 users")
- dates: specific dates or date ranges mentioned (e.g. "March 2024", "2019-03-03")
- places: cities, countries, regions, specific locations
- orgs: company names, organization names, institutions
- urls: any URLs mentioned in the content
- Return empty arrays if nothing found for a type
- Maximum 10 items per type
- Prefer full forms over abbreviations

Content:
%s`
```

### LLM interface

Add to `LLMExt` in `internal/llm/interface.go`:

```go
ExtractEntities(content string, bucket string) (Entities, error)
```

`Entities` here is `ingest.Entities` — import `internal/ingest` in the LLM package,
or define a local `EntityResult` struct in the LLM package and convert. **Prefer
defining a local struct in the LLM package to avoid a circular import** — `ingest`
already imports `llm`.

Define in `internal/llm/interface.go`:

```go
// EntityResult is the raw LLM output before normalization.
type EntityResult struct {
    People  []string `json:"people"`
    Amounts []string `json:"amounts"`
    Dates   []string `json:"dates"`
    Places  []string `json:"places"`
    Orgs    []string `json:"orgs"`
    URLs    []string `json:"urls"`
}
```

`ExtractEntities` returns `EntityResult`, not `ingest.Entities`. Normalization happens
in `internal/ingest/entities.go` after the LLM call returns.

Update the `LLMExt` interface signature accordingly:

```go
ExtractEntities(content string, bucket string) (EntityResult, error)
```

### Ollama implementation

Add to `internal/llm/ollama.go`:

```go
func (c *OllamaClient) ExtractEntities(content string, bucket string) (EntityResult, error) {
    truncated := c.truncateForBucket(content, bucket)
    prompt := fmt.Sprintf(constants.PromptExtractEntities, truncated)
    raw, err := c.Generate(prompt)
    if err != nil {
        return EntityResult{}, err
    }
    var result EntityResult
    if err := json.NewDecoder(strings.NewReader(raw)).Decode(&result); err != nil {
        // LLM returned non-JSON — return empty result, do not fail the job
        // Log at warn level with the raw response (truncated to 200 chars)
        slog.Warn("entity extraction: invalid JSON from LLM",
            "raw", truncate(raw, 200))
        return EntityResult{}, nil
    }
    return result, nil
}
```

**Note the exception to fail-fast**: entity extraction JSON parse failure returns an
empty result rather than an error. The note is still saved with empty entities. This is
acceptable because:
- Entities are enrichment, not core data
- LLMs occasionally return malformed JSON for complex prompts
- The job should not fail and retry just because entities couldn't be parsed

A `Generate()` error (network, timeout, Ollama down) still propagates as an error and
fails the job normally.

---

## Step 5 — Ingest Integration

All three ingest paths follow the same pattern. Entity extraction runs **after** the
existing parallel LLM calls complete, before vault write. It is a **sequential** call,
not added to the errgroup — because:

1. The errgroup already runs 3 concurrent LLM calls (tags, summary, key ideas)
2. Adding a 4th increases Ollama pressure during the already-loaded parallel phase
3. Entity extraction depends on no other result, but also blocks nothing — sequential
   is fine

### Pattern for all three ingest paths

```
existing: errgroup { ExtractTags, Summarize, ExtractKeyIdeas } — unchanged
NEW:      rawEntities, err := llm.ExtractEntities(content, bucket)
          if err != nil { return err }  // fail job on LLM error
NEW:      entities := ingest.NormalizeEntities(rawEntities)
existing: vault.WriteNote(note) — updated to include entities in frontmatter
NEW:      queue.SaveEntities(ctx, notePath, entities)
existing: queue.IndexNote(ctx, ...) — unchanged
```

### Per-path content to extract entities from

| Ingest path | Content passed to ExtractEntities |
|-------------|----------------------------------|
| `text.go`   | `req.Content` (raw user text) |
| `image.go`  | LLM-generated description (the string returned by `DescribeImage`) |
| `article.go`| Full article content (`note.Raw`) |

Same bucket values as existing LLM calls: `BucketText`, `BucketImage`, `BucketArticle`.

### NormalizeEntities

Add to `internal/ingest/entities.go`:

```go
// NormalizeEntities applies normalization to raw LLM entity output.
func NormalizeEntities(raw llm.EntityResult) Entities {
    return Entities{
        People:  normalizeNames(raw.People),
        Amounts: normalizeAmounts(raw.Amounts),
        Dates:   raw.Dates,
        Places:  raw.Places,
        Orgs:    raw.Orgs,
        URLs:    raw.URLs,
    }
}
```

---

## Step 6 — Frontmatter

File: `internal/vault/writer.go`

The frontmatter is built manually (not via YAML marshal — see RETROSPECTIVE.md "Duplicate
History Frontmatter" bug fix section for why). Add the `entities:` block after `tags:`.

### Frontmatter output format

```yaml
---
created: 2026-03-16T14:23:00Z
updated: 2026-03-16T14:23:04Z
type: text
status: done
tags:
  - react
  - performance
entities:
  people:  []
  amounts: []
  dates:   []
  places:  []
  urls:    []
  orgs:    []
history:
  - at: 2026-03-16T14:23:04Z
    event: processed
---
```

Rules for rendering the entities block:

- Always write the `entities:` block, even if all fields are empty
- Empty field renders as `  people:  []` (two spaces indent, two spaces before `[]`)
- Non-empty field renders as a YAML list:
  ```yaml
    people:
      - John Doe
      - Jane Smith
  ```
- Field order: people, amounts, dates, places, urls, orgs (match frontmatter example in SPEC.md)
- String values must be YAML-safe: if a value contains `:`, `#`, `[`, `]`, `{`, `}`,
  or leading/trailing whitespace, wrap in double quotes
- Maximum 10 items per field (enforce in writer, not just in prompt)

### NoteMetadata struct update

Add `Entities` field to `NoteMetadata` in `internal/vault/writer.go` (or wherever
the struct is defined):

```go
type NoteMetadata struct {
    // ... existing fields ...
    Entities ingest.Entities
}
```

Check for circular import — if `vault` already imports `ingest` or vice versa, define
a `VaultEntities` struct in the vault package mirroring the fields, and convert at the
call site in ingest.

---

## Step 7 — Tests

### `internal/ingest/entities_test.go`

**Amount normalization — table-driven:**

```go
var amountCases = []struct {
    input string
    want  string
}{
    {"$2,000",   "2000"},
    {"2k",       "2000"},
    {"2.5k",     "2500"},
    {"£1m",      "1000000"},
    {"500",      "500"},
    {"10B",      "10000000000"},
    {"€3.2m",    "3200000"},
    {"weird",    ""},
    {"",         ""},
    {"$",        ""},
    {"1,234.56", "1235"},  // no suffix, fractional → round
}
```

**Name normalization — table-driven:**

```go
var nameCases = []struct {
    input []string
    want  []string
}{
    {[]string{"John", "John Doe"}, []string{"John Doe"}},
    {[]string{"Sarah Connor", "Sarah"}, []string{"Sarah Connor"}},
    {[]string{"Alice", "Bob"}, []string{"Alice", "Bob"}},
    {[]string{"Jo", "John"}, []string{"Jo", "John"}},  // not a word-boundary match
    {[]string{"John Doe", "John Doe"}, []string{"John Doe"}},  // dedup identical
    {[]string{}, []string{}},
}
```

**NormalizeEntities — integration:**

```go
func TestNormalizeEntities(t *testing.T) {
    raw := llm.EntityResult{
        People:  []string{"Sarah", "Sarah Connor", "James"},
        Amounts: []string{"$2,000", "weird", "3k"},
        Dates:   []string{"March 2024"},
        Places:  []string{"London"},
        Orgs:    []string{},
        URLs:    []string{"https://example.com"},
    }
    got := NormalizeEntities(raw)
    assert.Equal(t, []string{"Sarah Connor", "James"}, got.People)
    assert.Equal(t, []string{"2000", "3000"}, got.Amounts)
    assert.Equal(t, []string{"March 2024"}, got.Dates)
    assert.Equal(t, []string{"London"}, got.Places)
    assert.Equal(t, []string{}, got.Orgs)
    assert.Equal(t, []string{"https://example.com"}, got.URLs)
}
```

### `internal/queue/queue_test.go` additions

- `TestSaveEntities_Basic` — save entities for a note, verify row count per type
- `TestSaveEntities_Idempotent` — save twice, verify only latest entities remain
- `TestDeleteEntities` — verify deletion
- `TestSaveEntities_Empty` — save with all empty slices, verify no rows inserted, no error

### `internal/llm/ollama_test.go` additions

- `TestExtractEntities_ValidJSON` — mock Generate returns valid JSON, verify parsing
- `TestExtractEntities_InvalidJSON` — mock Generate returns garbage, verify empty result returned (not error)
- `TestExtractEntities_GenerateError` — mock Generate returns error, verify error propagated

### Frontmatter rendering

Add cases to existing vault writer tests:

- `TestRenderNote_WithEntities` — entities block appears after tags, before history
- `TestRenderNote_EmptyEntities` — all fields render as `[]`
- `TestRenderNote_EntitiesWithSpecialChars` — value containing `:` is quoted

---

## Checklist

Work through this in order. Do not skip ahead.

- [ ] Read all referenced files before writing code
- [ ] Add `entities` table + indexes to queue.go schema init
- [ ] Add `SaveEntities` and `DeleteEntities` to queue.go
- [ ] Write queue tests for new methods — all pass
- [ ] Define `Entities` struct in `internal/ingest/entities.go`
- [ ] Implement `normalizeAmount`, `normalizeAmounts`, `normalizeNames`
- [ ] Implement `NormalizeEntities`
- [ ] Write entities_test.go — all normalization tests pass
- [ ] Define `EntityResult` in `internal/llm/interface.go`
- [ ] Add `ExtractEntities` to `LLMExt` interface
- [ ] Add prompt to `internal/constants/constants.go`
- [ ] Implement `ExtractEntities` in `internal/llm/ollama.go`
- [ ] Write ollama entity extraction tests — all pass
- [ ] Update `NoteMetadata` in vault/writer.go (check for circular imports first)
- [ ] Add entities block rendering to frontmatter builder in vault/writer.go
- [ ] Write vault writer tests for entities rendering — all pass
- [ ] Update `internal/ingest/text.go` — call ExtractEntities, pass entities to WriteNote, SaveEntities
- [ ] Update `internal/ingest/image.go` — same
- [ ] Update `internal/ingest/article.go` — same
- [ ] `go build ./...` — zero errors
- [ ] `go test ./...` — all tests pass
- [ ] `go vet ./...` — no warnings
- [ ] Manual test: capture a text note mentioning a person and an amount, verify entities in frontmatter
- [ ] Manual test: check entities table in sqlite3: `SELECT * FROM entities LIMIT 20`
- [ ] Manual test: capture an article URL, verify entities extracted from article content

---

## Hard Rules

1. **Fail fast on LLM errors, not on JSON parse errors.** `Generate()` failure → return
   error, job fails, worker retries. JSON parse failure → log warn, return empty
   `EntityResult{}`, job continues. This is the one exception to fail-fast in this codebase.

2. **No circular imports.** `ingest` imports `llm`. `llm` must not import `ingest`.
   `vault` must not import `ingest`. Define mirror structs or move shared types to a
   neutral package if needed. Check with `go build ./...` before proceeding.

3. **SaveEntities is idempotent.** Delete then insert in one transaction. Re-processing
   a note must not create duplicate entity rows.

4. **Always write the entities block in frontmatter.** Even if all fields are empty.
   Absence of the block would mean older notes and newer notes have different frontmatter
   shapes, which breaks tooling that parses the vault.

5. **10 items per field maximum.** Enforced in the vault writer (truncate slice to 10
   before rendering), not just in the prompt. LLMs ignore prompt limits.

6. **No YAML marshal for frontmatter.** The frontmatter builder is manual string
   construction. Do not introduce `gopkg.in/yaml.v3` or any YAML library for this.
   See RETROSPECTIVE.md "Duplicate History Frontmatter" for why.

7. **Entity extraction is sequential, not added to errgroup.** Do not add it to the
   parallel LLM call group in any ingest path.

8. **Module path** — `github.com/rawnaqs/khayal`. All new files use this prefix.

9. **No magic numbers.** The 10-item cap must be a named constant:
   ```go
   const maxEntitiesPerType = 10
   ```
   defined in `internal/constants/constants.go`.