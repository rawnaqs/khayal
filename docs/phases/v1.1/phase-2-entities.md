# Phase 2: Entity Extraction

> Extract structured entities from notes: people, amounts, dates, places, orgs, URLs. Updated: 2026-04-09

## Goals

- [ ] Add ExtractEntities to LLMExt interface
- [ ] Add ExtractEntities implementation (Ollama)
- [ ] Add SaveEntities to queue
- [ ] Update ingest pipeline
- [ ] Unit tests

## Existing Code (Don't Re-create)

- **Entities in vault metadata** — Already exists in `internal/vault/writer.go` as `EntitiesBlock` (lines 57-64)
  - Fields: people, amounts, dates, places, orgs, urls
  - Already in NoteMetadata as `Entities *EntitiesBlock`

## Specification

Per SPEC.md (lines 1357-1385):

### Frontmatter

```yaml
entities:
  people:  ["John Doe"]
  amounts: ["2000", "2k"]
  dates:   ["2019-03-03"]
  places:  []
  orgs:    []
  urls:    []
```

### Database

**Already exists in `internal/queue/queue.go`** (lines 137-146) — don't recreate:

```sql
CREATE TABLE entities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note_path TEXT NOT NULL,
    chunk_idx INTEGER,
    entity_type TEXT NOT NULL,
    entity_value TEXT NOT NULL,
    created_at TEXT NOT NULL
);
-- Indexes also exist
```

## Step 2.1: Add ExtractEntities to LLM

**File:** `internal/llm/llm.go`

Add method to interface:

```go
type Extractor interface {
    ExtractTags(content string, bucket BucketType) ([]string, error)
    Summarize(content string, bucket BucketType) (string, error)
    ExtractKeyIdeas(content string, bucket BucketType) ([]string, error)
    ExtractEntities(content string, bucket BucketType) (map[string][]string, error)
}
```

Add implementation in Ollama client:

```go
func (o *Ollama) ExtractEntities(content string, bucket BucketType) (map[string][]string, error) {
    prompt := fmt.Sprintf(`Extract entities from the following note.

Return ONLY valid JSON in this exact format:
{"people": [], "amounts": [], "dates": [], "places": [], "orgs": [], "urls": []}

Rules:
- people: Full names of people mentioned
- amounts: Dollar amounts, percentages, numbers (normalize "2k" to "2000")
- dates: Any date in ISO format (YYYY-MM-DD preferred)
- places: Cities, countries, locations
- orgs: Companies, organizations
- urls: Any URLs found

Note:
%s

JSON:`, content[:10000]) // Limit content size

    resp, err := o.Generate(ctx, prompt, "llama3.2:3b", 512, 0.1)
    if err != nil {
        return nil, err
    }

    return parseEntityResponse(resp)
}

func parseEntityResponse(text string) (map[string][]string, error) {
    text = strings.TrimSpace(text)

    start := strings.Index(text, "{")
    end := strings.LastIndex(text, "}")
    if start == -1 || end == -1 {
        return map[string][]string{
            "people":  {},
            "amounts": {},
            "dates":   {},
            "places":  {},
            "orgs":   {},
            "urls":    {},
        }, nil
    }

    text = text[start : end+1]

    var result map[string][]string
    if err := json.Unmarshal([]byte(text), &result); err != nil {
        return map[string][]string{
            "people":  {},
            "amounts": {},
            "dates":   {},
            "places":  {},
            "orgs":   {},
            "urls":    {},
        }, nil
    }

    return result, nil
}
```

## Step 2.2: Add SaveEntity to Queue

**File:** `internal/queue/queue.go`

Add method:

```go
func (q *Queue) SaveEntity(ctx context.Context, notePath string, chunkIdx int, entityType, entityValue string) error {
    _, err := q.db.ExecContext(ctx, `
        INSERT INTO entities (note_path, chunk_idx, entity_type, entity_value, created_at)
        VALUES (?, ?, ?, ?, ?)`,
        notePath, chunkIdx, entityType, entityValue, time.Now().UTC().Format(time.RFC3339))
    return err
}
```

Add batch method for efficiency:

```go
func (q *Queue) SaveEntities(ctx context.Context, notePath string, entities map[string][]string) error {
    tx, err := q.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO entities (note_path, chunk_idx, entity_type, entity_value, created_at)
        VALUES (?, ?, ?, ?, ?)`)
    if err != nil {
        return err
    }
    defer stmt.Close()

    for entityType, values := range entities {
        for _, value := range values {
            if _, err := stmt.ExecContext(ctx, notePath, 0, entityType, value, time.Now().UTC().Format(time.RFC3339)); err != nil {
                return err
            }
        }
    }

    return tx.Commit()
}
```

## Step 2.3: Add Entities to Vault Metadata

**File:** `internal/vault/writer.go`

Add to NoteMetadata:

```go
type NoteMetadata struct {
    Created time.Time  `yaml:"created"`
    Updated *time.Time  `yaml:"updated"`
    Type    string      `yaml:"type"`
    Status  string      `yaml:"status"`
    SourceURL string    `yaml:"source_url,omitempty"`
    SourceFile string   `yaml:"source_file,omitempty"`
    UserContext string  `yaml:"user_context,omitempty"`
    Tags    []string    `yaml:"tags"`
    Entities Entities   `yaml:"entities"`
    History []HistoryEvent `yaml:"history"`
}

type Entities struct {
    People  []string `yaml:"people,omitempty"`
    Amounts []string `yaml:"amounts,omitempty"`
    Dates   []string `yaml:"dates,omitempty"`
    Places []string `yaml:"places,omitempty"`
    Orgs    []string `yaml:"orgs,omitempty"`
    URLs    []string `yaml:"urls,omitempty"`
}
```

## Step 2.4: Update Ingest Pipeline

**File:** `internal/ingest/text.go`

Add entity extraction after tags/summary:

```go
func IngestText(ctx context.Context, job *queue.Job, v *vault.Writer, q *queue.Queue, llmClient llm.LLMExt) (string, error) {
    var tags []string
    var summary string
    var keyIdeas []string
    var entities map[string][]string

    g, _ := errgroup.WithContext(ctx)

    g.Go(func() error {
        var err error
        tags, err = llmClient.ExtractTags(job.Content, llm.BucketText)
        return err
    })

    g.Go(func() error {
        var err error
        summary, err = llmClient.Summarize(job.Content, llm.BucketText)
        return err
    })

    g.Go(func() error {
        var err error
        keyIdeas, err = llmClient.ExtractKeyIdeas(job.Content, llm.BucketText)
        return err
    })

    g.Go(func() error {
        var err error
        entities, err = llmClient.ExtractEntities(job.Content, llm.BucketText)
        return err
    })

    if err := g.Wait(); err != nil {
        return "", fmt.Errorf("llm extraction failed: %w", err)
    }

    title := extractTitle(job.Content)
    now := time.Now().UTC()

    note := &vault.Note{
        Metadata: vault.NoteMetadata{
            Created: job.CreatedAt,
            Updated: &now,
            Type:    "text",
            Status:  "done",
            Tags:    tags,
            Entities: vault.Entities{
                People:  entities["people"],
                Amounts: entities["amounts"],
                Dates:   entities["dates"],
                Places: entities["places"],
                Orgs:   entities["orgs"],
                URLs:   entities["urls"],
            },
            History: []vault.HistoryEvent{
                {At: now, Event: "processed"},
            },
        },
        Title:    title,
        Summary:  summary,
        KeyIdeas: keyIdeas,
        Raw:      job.Content,
    }

    notePath, err := v.WriteNote(note, job.ID)
    if err != nil {
        return "", fmt.Errorf("failed to write note: %w", err)
    }

    // Index for keyword search
    if err := q.IndexNote(ctx, notePath, title, job.Content, strings.Join(tags, ",")); err != nil {
        return "", fmt.Errorf("failed to index note: %w", err)
    }

    // Save entities to DB
    if err := q.SaveEntities(ctx, notePath, entities); err != nil {
        q.logger.Warn("failed to save entities", "error", err)
    }

    // Embed and save chunks
    chunks := chunker.SplitIntoChunks(job.Content)
    for i, chunk := range chunks {
        embedding, err := llmClient.Embed(chunk)
        if err != nil {
            continue
        }
        if err := q.SaveChunk(ctx, notePath, i, chunk, embedding); err != nil {
            return notePath, nil
        }
    }

    return notePath, nil
}
```

Apply same changes to `image.go` and `article.go`.

## Step 2.5: Unit Tests

**File:** `internal/llm/entities_test.go`

```go
package llm

import (
	"testing"
)

func TestParseEntityResponse_Valid(t *testing.T) {
	json := `{"people": ["John Doe"], "amounts": ["2000"], "dates": ["2024-01-01"], "places": ["NYC"], "orgs": ["Acme"], "urls": ["https://example.com"]}`

	result, err := parseEntityResponse(json)
	if err != nil {
		t.Fatalf("parseEntityResponse() error = %v", err)
	}

	if len(result["people"]) != 1 || result["people"][0] != "John Doe" {
		t.Errorf("people = %v, want [John Doe]", result["people"])
	}
}

func TestParseEntityResponse_Empty(t *testing.T) {
	json := `{"people": [], "amounts": [], "dates": [], "places": [], "orgs": [], "urls": []}`

	result, err := parseEntityResponse(json)
	if err != nil {
		t.Fatalf("parseEntityResponse() error = %v", err)
	}

	if len(result["people"]) != 0 {
		t.Errorf("expected empty people, got %v", result["people"])
	}
}

func TestParseEntityResponse_Malformed(t *testing.T) {
	// Should return empty map, not error
	result, err := parseEntityResponse("not valid json")
	if err != nil {
		t.Errorf("expected no error for malformed JSON, got %v", err)
	}
	if len(result) == 0 {
		t.Error("expected empty map result")
	}
}

func TestParseEntityResponse_WithPrefix(t *testing.T) {
	// LLM might add text before/after JSON
	text := `Here is the extracted information: {"people": ["Alice"], "amounts": ["500"]} end of response.`

	result, err := parseEntityResponse(text)
	if err != nil {
		t.Fatalf("parseEntityResponse() error = %v", err)
	}

	if len(result["people"]) != 1 || result["people"][0] != "Alice" {
		t.Errorf("people = %v, want [Alice]", result["people"])
	}
}
```

**File:** `internal/queue/entities_test.go`

```go
package queue

import (
	"context"
	"testing"
)

func TestSaveEntities(t *testing.T) {
	q, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	entities := map[string][]string{
		"people":  {"John Doe", "Jane Smith"},
		"amounts": {"2000", "500"},
		"dates":   {"2024-01-01"},
	}

	err := q.SaveEntities(ctx, "test-note.md", entities)
	if err != nil {
		t.Fatalf("SaveEntities() error = %v", err)
	}

	// Verify stored
	rows, err := q.db.QueryContext(ctx, "SELECT entity_type, entity_value FROM entities WHERE note_path = ?", "test-note.md")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}
	if count != 4 {
		t.Errorf("expected 4 entities, got %d", count)
	}
}

func TestGetEntitiesByNote(t *testing.T) {
	q, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save entities first
	q.SaveEntities(ctx, "test-note.md", map[string][]string{
		"people": {"Alice", "Bob"},
	})

	// Retrieve
	people, err := q.GetEntitiesByNote(ctx, "test-note.md", "people")
	if err != nil {
		t.Fatal(err)
	}

	if len(people) != 2 {
		t.Errorf("expected 2 people, got %d", len(people))
	}
}
```

## Checklist

- [ ] ExtractEntities method in LLMExt interface
- [ ] ExtractEntities implementation in Ollama
- [ ] SaveEntities in queue (batch insert)
- [ ] GetEntitiesByNote in queue (for connections)
- [ ] text.go extracts entities
- [ ] image.go extracts entities
- [ ] article.go extracts entities
- [ ] LLM entity tests passing
- [ ] Queue entity tests passing

## Next Phase

[Phase 3: Proactive Connections](phase-3-connections.md)

## Notes

- Entity extraction runs in parallel with tags/summary/keyIdeas (via errgroup)
- If extraction fails, note is still saved (warning only)
- Entities stored in both: DB (for search) + frontmatter (for display)
- Vault metadata already has EntitiesBlock — don't re-create
- Tests cover: JSON parsing, DB save/retrieve