# Phase 2.5: Capture Intelligence

> Agent instruction file. Read all of it before writing any code.
> Two sub-features: relative-date resolution (2.5a) and LLM context memory (2.5b).

---

## What This Phase Does

### 2.5a — Relative date resolution

At capture time, resolve relative date references in extracted entities
("today", "tomorrow", "in 3 days", "next friday") into absolute dates using
capture time as the reference point. No extra LLM call — pure Go.

Resolved dates are stored in two places:

1. Frontmatter `dates:` entries gain their resolved form:
   `- tomorrow → 2026-08-27` (raw string preserved for readability)
2. The `entities` table gains a nullable `resolved_date DATETIME` column so
   proactive connection Type 5 (follow-up detection) can query real dates.

This is the project's **first schema migration** on an existing populated
table. Use `ALTER TABLE entities ADD COLUMN resolved_date DATETIME` and
tolerate the "duplicate column name" error as success (no migration runner
exists; keep it that way for one column).

### 2.5b — LLM context memory

Before enrichment runs, build a compact context block from what khayal
already knows and inject it into all four enrichment prompts (tags,
summary, key ideas, entities), so naming and topics stay consistent across
months of captures.

Context block sources (strictly local, no new stores):

1. **Entity glossary** — distinct known `person` and `org` values from the
   entities table
2. **Topical recall** — top-3 semantically similar past note summaries

Hard character cap (~1200) on the assembled block to protect prompt size.
Config-gated via new `memory.enabled` (default: true).

---

## Files To Read First

```
RULES.md              ← memory management rules
RETROSPECTIVE.md      ← ingest pipeline patterns, errgroup, lock retry
internal/chunk/       ← existing pure-function test style to imitate
docs/adr/0001-chunk-coverage-asymmetry-capture-vs-reindex.md
```

Key facts affecting this phase:

- Ingest paths: `internal/ingest/text.go`, `image.go`, `article.go`
- Entity extraction already runs sequentially after the errgroup;
  date resolution slots into `NormalizeEntities`'s neighborhood
- `queue.SaveEntities` writes enrichment rows; extend its input struct,
  not the table writer shape
- All four enrichment prompts flow through
  `OllamaClient.getSystemPrompt` / per-bucket templates in constants

---

## New Files To Create

```
internal/dates/dates.go            ← ResolveRelative(ref string, now) (time.Time, bool)
internal/dates/dates_test.go       ← table-driven, fixed `now`, no mocks
internal/memory/memory.go          ← BuildContextBlock(...) pure builder
internal/memory/memory_test.go     ← canned queue data, assert cap + content
```

## Files To Modify

```
internal/queue/queue.go            ← NoteEntities gains ResolvedDates; ALTER TABLE
internal/ingest/entities.go        ← resolve after normalize; frontmatter arrow form
internal/vault/writer.go           ← render `- raw → YYYY-MM-DD` entries
internal/config/config.go          ← MemoryConfig{Enabled bool} + default true
internal/constants/constants.go    ← memory context prompt suffix template
internal/llm/ollama.go             ← call-context injection hook
internal/api/ui/                   ← none (this phase is capture-side only)
```

---

## Step 1 — Date resolver (pure, TDD)

```go
func ResolveRelative(ref string, now time.Time) (time.Time, bool)
```

Case-insensitive, trimmed. v1 grammar:

| Input | Resolution |
|-------|------------|
| today / tonight | now's date |
| tomorrow | +1 day |
| yesterday | −1 day |
| in N days / weeks / months | now + N units |
| next week / month | +7 days / +1 calendar month |
| <weekday> / next <weekday> | next occurrence (strictly future for `next`) |

Everything else → `(zero, false)`. Ambiguous forms stay unresolved rather
than guessed.

Tests: full table with fixed `now := time.Date(2026, 8, 25, ...)` covering
every rule plus boundary cases ("in 0 days" → false or today? pick: false —
not a meaningful date), "Friday" when today IS Friday (→ today for bare
weekday, +7 for "next Friday").

## Step 2 — Storage

- `NoteEntities` gains `ResolvedDates []string` parallel to `Dates`
  (index-aligned; empty string = unresolved). `rows()` emits them into the
  new column by index.
- Schema init: attempt `ALTER TABLE entities ADD COLUMN resolved_date DATETIME`;
  swallow duplicate-column errors only.
- `SaveEntities` inserts resolved values where present.

## Step 3 — Frontmatter

In `writeEntitiesBlock`, when a date has a resolved form render:

```yaml
dates:
  - tomorrow → 2026-08-27
```

The arrow character is UTF-8 — vault safety rule 3 requires confirming
valid UTF-8 output (it is). YAML-safe quoting rules apply unchanged.

## Step 4 — Memory context builder (pure, TDD)

```go
type Sources struct {
    Glossary []string            // distinct person+org values
    Recall   []string            // top similar summaries, best first
}
func BuildContextBlock(s Sources, maxChars int) string
```

Deterministic output: glossary line first (`Known people & orgs: A, B, C`),
then up to 3 recall bullets. Truncates at maxChars mid-bullet with `…`.
Empty sources → empty string (callers skip injection entirely).

## Step 5 — Retrieval wiring

At ingest start (before errgroup):

1. If `memory.enabled`: `emb := llm.Embed(job.Content)` — failure degrades
   to empty context (fail-open), never blocks capture
2. Retrieve top-3 past summaries via existing semantic machinery
3. Assemble block; stash on the Ollama client for this job
   (`SetCallContext(block)`), cleared after enrichment completes
4. Injection point: append block to each system prompt inside
   `GenerateWithSystemTemp` calls for the four enrichment ops only —
   never to image description or search overview prompts

## Step 6 — Config

```yaml
memory:
  enabled: true        # default true; false skips retrieval AND injection
```

ApplyDefaults sets Enabled=true only when the section is absent — use a
pointer or sentinel since false must remain expressible.

---

## Tests

- `dates_test.go`: every grammar row + boundaries (fixed clock)
- `memory_test.go`: glossary-only, recall-only, both, overflow truncation,
  empty → ""
- `entities_test.go`: resolution applied post-normalize; unresolved pass through
- `writer_test.go`: arrow-form rendering; quoting still applies
- `queue_test.go`: SaveEntities persists/resolves round-trip; ALTER idempotent
  (open DB twice)
- `ingest_test.go`: end-to-end — mock LLM returns "tomorrow"; note file shows
  arrow line; memory disabled → no Embed call counted

## Checklist

- [ ] Read referenced files first
- [ ] dates resolver red-green
- [ ] memory builder red-green
- [ ] schema ALTER + SaveEntities round-trip green
- [ ] frontmatter arrow rendering green
- [ ] ingest wiring + config gate green
- [ ] go build ./... && go vet ./... && golangci-lint run ./... clean
- [ ] go test ./... green
- [ ] Manual: capture "meet Sarah tomorrow about $500" → frontmatter shows
      `tomorrow → <real date>`; sqlite resolved_date populated

## Hard Rules

1. Resolver is deterministic and pure — no LLM, no clock reads (take `now`).
2. Memory failures NEVER fail capture. Fail-open everywhere in this path.
3. One ALTER statement, tolerated-duplicate pattern — do not introduce a
   migration runner for one column.
4. Context injection touches ONLY the four enrichment prompts.
5. `memory.enabled=false` must result in zero extra embed calls.
