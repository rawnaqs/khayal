# Phase 3: Proactive Connections

> Auto-discover related notes after capture. Updated: 2026-04-09

## Goals

- [ ] Create connections package
- [ ] Implement semantic similar (v1.1)
- [ ] Add GetEntitiesByNote to queue
- [ ] Add GetNotesByEntity to queue
- [ ] Add job type "connections" to worker
- [ ] Update capture response with connections_job_id
- [ ] Implement ranking logic
- [ ] Unit tests

## Existing Code (Don't Re-create)

- **SearchSemantic** — Already exists in `internal/queue/queue.go`
  - Uses cosine similarity, handles min score, date filtering
- **worker** — Already handles job types via switch/case

## Specification

Per SPEC.md (lines 1123-1296):

### Connection Types (v1.1)

| Type | Detection | Threshold | Min Age |
|------|----------|-----------|---------|
| Semantic Similar | sqlite-vec cosine | > 0.85 | 7 days |
| Same Person | SQL entity lookup | — | 7 days |
| Same Amount | SQL entity lookup | — | 7 days |

### Config

```yaml
connections:
  enabled: true
  min_age_days: 7
  max_per_capture: 3
  similarity_threshold: 0.85
  types:
    similar: true
    person: true
    amount: true
```

### Output

```json
{
  "id": "abc123",
  "status": "done",
  "note_path": "khayal/2024-03-16-thought.md",
  "connections_job_id": "def456"
}
```

### Ranking

Per SPEC.md (line 1223-1235):
- Max 3 connections per capture
- Type priority: person > amount > similar

## Step 3.1: Create Connections Package

**File:** `internal/connections/connections.go`

```go
package connections

import (
    "context"
    "fmt"
    "time"

    "github.com/rawnaqs/khayal/internal/llm"
    "github.com/rawnaqs/khayal/internal/queue"
)

type Connection struct {
    Type      string  `json:"type"`
    NotePath  string  `json:"note_path"`
    Excerpt  string  `json:"excerpt"`
    Score    float32 `json:"score"`
    Label    string  `json:"label"`
}

type Config struct {
    Enabled             bool
    MinAgeDays         int
    MaxPerCapture     int
    SimilarityThreshold float32
    Types              struct {
        Similar bool
        Person  bool
        Amount  bool
    }
}

var DefaultConfig = Config{
    Enabled:             true,
    MinAgeDays:         7,
    MaxPerCapture:     3,
    SimilarityThreshold: 0.85,
    Types: struct {
        Similar bool
        Person  bool
        Amount  bool
    },
}

func FindConnections(ctx context.Context, notePath, content string, q *queue.Queue, cfg Config) ([]Connection, error) {
    if !cfg.Enabled {
        return nil, nil
    }

    var connections []Connection
    seen := make(map[string]bool)

    // Type 1: Semantic Similar
    if cfg.Types.Similar {
        similar, err := findSimilar(ctx, q, content, notePath, cfg.SimilarityThreshold, cfg.MinAgeDays)
        if err == nil {
            connections = append(connections, similar...)
            for _, c := range similar {
                seen[c.NotePath] = true
            }
        }
    }

    // Type 2: Same Person
    if cfg.Types.Person {
        persons, err := findByEntity(ctx, q, notePath, "person", cfg.MinAgeDays)
        if err == nil {
            for _, p := range persons {
                if !seen[p.NotePath] {
                    connections = append(connections, p)
                    seen[p.NotePath] = true
                }
            }
        }
    }

    // Type 3: Same Amount
    if cfg.Types.Amount {
        amounts, err := findByEntity(ctx, q, notePath, "amount", cfg.MinAgeDays)
        if err == nil {
            for _, a := range amounts {
                if !seen[a.NotePath] {
                    connections = append(connections, a)
                }
            }
        }
    }

    // Rank and limit
    connections = rankAndLimit(connections, cfg.MaxPerCapture)

    return connections, nil
}

func findSimilar(ctx context.Context, q *queue.Queue, content, excludePath string, threshold float32, minAgeDays int) ([]Connection, error) {
    embedding, err := q.ComputeEmbedding(ctx, content)
    if err != nil {
        return nil, err
    }

    results, err := q.SearchSemantic(ctx, embedding, 5)
    if err != nil {
        return nil, err
    }

    minDate := time.Now().AddDate(0, 0, -minAgeDays)
    var connections []Connection

    for _, r := range results {
        if r.NotePath == excludePath {
            continue
        }
        if r.Score < threshold {
            continue
        }
        if r.CreatedAt.Before(minDate) {
            continue
        }

        label := fmt.Sprintf("you thought about this %s", formatAge(r.CreatedAt))
        connections = append(connections, Connection{
            Type:     "similar",
            NotePath: r.NotePath,
            Excerpt: r.Excerpt,
            Score:   r.Score,
            Label:   label,
        })
    }

    return connections, nil
}

func findByEntity(ctx context.Context, q *queue.Queue, notePath, entityType string, minAgeDays int) ([]Connection, error) {
    entities, err := q.GetEntitiesByNote(ctx, notePath, entityType)
    if err != nil || len(entities) == 0 {
        return nil, err
    }

    minDate := time.Now().AddDate(0, 0, -minAgeDays)
    var connections []Connection

    for _, entity := range entities {
        related, err := q.GetNotesByEntity(ctx, entity, entityType, minDate)
        if err != nil {
            continue
        }

        for _, r := range related {
            if r.NotePath == notePath {
                continue
            }

            entityLabel := entity
            if entityType == "amount" {
                entityLabel = "this amount"
            }

            connections = append(connections, Connection{
                Type:     entityType,
                NotePath: r.NotePath,
                Excerpt: r.Excerpt,
                Score:   1.0,
                Label:   fmt.Sprintf("%s also appears in this note", entityLabel),
            })
        }
    }

    return connections, nil
}

func rankAndLimit(connections []Connection, max int) []Connection {
    type priority struct {
        string
        int
    }

    priorityMap := map[string]int{
        "person":  3,
        "amount":  2,
        "similar": 1,
    }

    scored := make([]struct {
        Connection
        priority int
    }, len(connections))

    for i, c := range connections {
        scored[i].Connection = c
        scored[i].priority = priorityMap[c.Type]
    }

    // Sort by priority desc, then score desc
    for i := 0; i < len(scored)-1; i++ {
        for j := i + 1; j < len(scored); j++ {
            if scored[j].priority > scored[i].priority ||
                (scored[j].priority == scored[i].priority && scored[j].Score > scored[i].Score) {
                scored[i], scored[j] = scored[j], scored[i]
            }
        }
    }

    result := make([]Connection, 0, max)
    for i := 0; i < len(scored) && len(result) < max; i++ {
        result = append(result, scored[i].Connection)
    }

    return result
}

func formatAge(t time.Time) string {
    now := time.Now()
    days := int(now.Sub(t).Hours() / 24

    if days < 30 {
        return fmt.Sprintf("%d days ago", days)
    }
    if days < 365 {
        return fmt.Sprintf("%d months ago", days/30)
    }
    return fmt.Sprintf("%d years ago", days/365)
}
```

## Step 3.2: Add Connection Methods to Queue

**File:** `internal/queue/queue.go`

Add helper methods:

```go
func (q *Queue) GetEntitiesByNote(ctx context.Context, notePath, entityType string) ([]string, error) {
    rows, err := q.db.QueryContext(ctx, `
        SELECT entity_value FROM entities
        WHERE note_path = ? AND entity_type = ?`,
        notePath, entityType)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var values []string
    for rows.Next() {
        var v string
        if err := rows.Scan(&v); err != nil {
            continue
        }
        values = append(values, v)
    }

    return values, nil
}

func (q *Queue) GetNotesByEntity(ctx context.Context, entityValue, entityType string, minDate time.Time) ([]SearchResult, error) {
    rows, err := q.db.QueryContext(ctx, `
        SELECT DISTINCT e.note_path, j.created_at, j.content
        FROM entities e
        JOIN jobs j ON e.note_path = j.note_path
        WHERE e.entity_value = ? AND e.entity_type = ? AND j.created_at > ?
        ORDER BY j.created_at DESC
        LIMIT 10`,
        entityValue, entityType, minDate.Format(time.RFC3339))
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []SearchResult
    for rows.Next() {
        var path, createdAt, content string
        if err := rows.Scan(&path, &createdAt, &content); err != nil {
            continue
        }

        t, _ := time.Parse(time.RFC3339, createdAt)
        excerpt := extractExcerpt(content, 100)

        results = append(results, SearchResult{
            NotePath: path,
            CreatedAt: t,
            Excerpt: excerpt,
        })
    }

    return results, nil
}
```

## Step 3.3: Add Connections Job Type

**File:** `internal/queue/queue.go`

Add new job type:

```go
const (
    JobTypeText      = "text"
    JobTypeImage     = "image"
    JobTypeArticle  = "article"
    JobTypeConnections = "connections"
)
```

Add new job creation:

```go
func (q *Queue) CreateConnectionsJob(ctx context.Context, notePath string) (string, error) {
    id := uuid.New().String()
    now := time.Now().UTC()

    _, err := q.db.ExecContext(ctx, `
        INSERT INTO jobs (id, type, note_path, status, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?)`,
        id, JobTypeConnections, notePath, "pending", now.Format(time.RFC3339), now.Format(time.RFC3339))

    return id, err
}
```

## Step 3.4: Update Capture Response

**File:** `internal/api/capture.go`

Update response struct:

```go
type CaptureResponse struct {
    ID               string `json:"id"`
    Status           string `json:"status"`
    NotePath        string `json:"note_path"`
    CreatedAt       string `json:"created_at"`
    ConnectionsJobID string `json:"connections_job_id,omitempty"`
}
```

After creating ingest job, create connections job:

```go
// ... after ingest job created ...

// Create connections job
connJobID, err := q.CreateConnectionsJob(ctx, notePath)
if err != nil {
    logger.Warn("failed to create connections job", "error", err)
}

// Return response with connections job ID
return CaptureResponse{
    ID:               job.ID,
    Status:           "done",
    NotePath:        notePath,
    CreatedAt:       job.CreatedAt,
    ConnectionsJobID: connJobID,
}, nil
```

## Step 3.5: Update Worker

**File:** `internal/worker/worker.go`

Add connections job processing:

```go
func (w *Worker) processJob(ctx context.Context, job *queue.Job) error {
    switch job.Type {
    case queue.JobTypeText:
        return w.processText(ctx, job)
    case queue.JobTypeImage:
        return w.processImage(ctx, job)
    case queue.JobTypeArticle:
        return w.processArticle(ctx, job)
    case queue.JobTypeConnections:
        return w.processConnections(ctx, job)
    default:
        return fmt.Errorf("unknown job type: %s", job.Type)
    }
}

func (w *Worker) processConnections(ctx context.Context, job *queue.Job) error {
    notePath := job.NotePath

    content, err := os.ReadFile(notePath)
    if err != nil {
        return err
    }

    connections, err := connections.FindConnections(
        ctx,
        notePath,
        string(content),
        w.queue,
        connections.DefaultConfig,
    )
    if err != nil {
        return err
    }

    // Save connections to job result
    return w.queue.UpdateJobResult(ctx, job.ID, map[string]interface{}{
        "connections": connections,
    })
}
```

## Checklist

- [ ] Connections package created
- [ ] GetEntitiesByNote in queue
- [ ] GetNotesByEntity in queue
- [ ] Semantic similar (reuse SearchSemantic)
- [ ] Same person via entity lookup
- [ ] Same amount via entity lookup
- [ ] Ranking by priority (person > amount > similar)
- [ ] Max 3 connections
- [ ] Min 7 days filter
- [ ] Job type "connections" in worker
- [ ] Capture response includes connections_job_id
- [ ] Worker processes connections job
- [ ] LLM tests passing
- [ ] Queue entity query tests passing

## Next Phase

[Phase 4: Vault Commands](phase-4-vault.md)

## Notes

- Connections run async after capture
- If extraction fails, job marked as failed, user can retry
- Results stored in job result column
- Min 7 days filters out recent notes (per SPEC.md)
- Priority: person > amount > similar
- SearchSemantic already handles semantic similar (reuse in connections package)
- Tests cover: entity query, ranking, max limit

## Step 3.X: Unit Tests

**File:** `internal/connections/connections_test.go`

```go
package connections

import (
	"context"
	"testing"
	"time"

	"github.com/rawnaqs/khayal/internal/queue"
)

func TestFindConnections_Semantic(t *testing.T) {
	q, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save some notes with embeddings
	notePath1 := "khayal/2024-01-01-old-note.md"
	notePath2 := "khayal/2024-01-02-similar-note.md"
	notePath3 := "khayal/2024-01-03-different.md"

	// Notes older than 7 days
	oldDate := time.Now().AddDate(0, 0, -10)

	// Save chunks with embeddings (mock data)
	emb1 := []float32{0.1, 0.2, 0.3}
	emb2 := []float32{0.1, 0.2, 0.31} // similar
	emb3 := []float32{0.9, 0.8, 0.7}

	q.SaveChunk(ctx, notePath1, 0, "content about Go programming", emb1)
	q.SaveChunk(ctx, notePath2, 0, "content about Go and Rust", emb2)
	q.SaveChunk(ctx, notePath3, 0, "content about cooking", emb3)
}

func TestFindConnections_ByPerson(t *testing.T) {
	q, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	notePath := "khayal/2024-01-01-test.md"

	// Save entities
	entities := map[string][]string{
		"people": {"Alice", "Bob"},
	}
	q.SaveEntities(ctx, notePath, entities)

	// Retrieve
	people, err := q.GetEntitiesByNote(ctx, notePath, "people")
	if err != nil {
		t.Fatal(err)
	}
	if len(people) != 2 {
		t.Errorf("expected 2 people, got %d", len(people))
	}
}

func TestRankAndLimit(t *testing.T) {
	connections := []Connection{
		{Type: "similar", Score: 0.9},
		{Type: "person", Score: 1.0},
		{Type: "amount", Score: 1.0},
		{Type: "similar", Score: 0.95},
		{Type: "person", Score: 1.0},
	}

	limited := rankAndLimit(connections, 3)

	if len(limited) != 3 {
		t.Errorf("expected 3 connections, got %d", len(limited))
	}

	// First should be person (highest priority)
	if limited[0].Type != "person" {
		t.Errorf("expected person first, got %s", limited[0].Type)
	}
}
```

**File:** `internal/queue/connections_test.go`

```go
package queue

import (
	"context"
	"testing"
	"time"
)

func TestGetEntitiesByNote(t *testing.T) {
	q, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	notePath := "test-note.md"

	// Save entities
	err := q.SaveEntities(ctx, notePath, map[string][]string{
		"people": {"Alice", "Bob"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve
	people, err := q.GetEntitiesByNote(ctx, notePath, "people")
	if err != nil {
		t.Fatal(err)
	}

	if len(people) != 2 {
		t.Errorf("expected 2 people, got %d", len(people))
	}
}

func TestGetNotesByEntity(t *testing.T) {
	q, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Save multiple notes with same person
	notePath1 := "khayal/2024-01-01-note1.md"
	notePath2 := "khayal/2024-01-02-note2.md"

	err := q.SaveEntities(ctx, notePath1, map[string][]string{
		"people": {"Alice"},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = q.SaveEntities(ctx, notePath2, map[string][]string{
		"people": {"Alice"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve notes with "Alice"
	minDate := time.Now().AddDate(0, 0, -30)
	notes, err := q.GetNotesByEntity(ctx, "Alice", "people", minDate)
	if err != nil {
		t.Fatal(err)
	}

	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
	}
}
```