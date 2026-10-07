// Command seedtestdata populates a Khayal database + vault with a small,
// deterministic fixture for the Playwright e2e suite. The testdata DB and
// vault notes are gitignored, so CI has no data unless this runs first.
//
//	go run ./cmd/seedtestdata -db testdata/khayal.db -vault testdata/vault/khayal
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/rawnaqs/khayal/internal/queue"
)

func main() {
	dbPath := flag.String("db", "testdata/khayal.db", "sqlite database path")
	vaultDir := flag.String("vault", "testdata/vault/khayal", "inbox directory for note files")
	flag.Parse()

	if err := os.MkdirAll(*vaultDir, 0o755); err != nil {
		log.Fatalf("mkdir vault: %v", err)
	}

	q, err := queue.NewQueue(*dbPath)
	if err != nil {
		log.Fatalf("open queue: %v", err)
	}
	defer q.Close()

	ctx := context.Background()
	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

	const notes = 12
	paths := make([]string, notes)
	for i := 0; i < notes; i++ {
		rel := fmt.Sprintf("khayal/seed-note-%02d.md", i+1)
		paths[i] = rel
		created := base.Add(time.Duration(i) * time.Hour)
		title := fmt.Sprintf("Seed Note %02d", i+1)
		body := fmt.Sprintf("This is deterministic seed note %d about topic %d.", i+1, i%4)

		if err := writeNote(*vaultDir, rel, title, body, created); err != nil {
			log.Fatalf("write note %s: %v", rel, err)
		}
		if err := q.IndexNote(ctx, rel, title, body, "seed"); err != nil {
			log.Fatalf("index note %s: %v", rel, err)
		}
		// a completed capture job so the queue has "done" items and the
		// graph can resolve a note's type + created time
		if err := q.CreateJob(ctx, &queue.Job{
			ID:        fmt.Sprintf("seed-text-%02d", i+1),
			Type:      "text",
			Status:    "done",
			NotePath:  rel,
			CreatedAt: created,
		}); err != nil {
			log.Fatalf("create text job %s: %v", rel, err)
		}
	}

	// people so the graph has person nodes
	people := []string{"Seed Alice", "Seed Bob", "Seed Carol"}
	for i, name := range people {
		if err := q.SaveEntities(ctx, paths[i], queue.NoteEntities{People: []string{name}}); err != nil {
			log.Fatalf("save entities: %v", err)
		}
	}

	// connection results so notes have typed edges (note 01 links to 02-04)
	conns := []map[string]any{
		{"type": "similar", "note_path": paths[1], "excerpt": "x", "score": 0.9, "label": "similar"},
		{"type": "similar", "note_path": paths[2], "excerpt": "x", "score": 0.8, "label": "similar"},
		{"type": "contradiction", "note_path": paths[3], "excerpt": "x", "score": 0.7, "label": "contradicts"},
	}
	payload, _ := json.Marshal(map[string]any{"connections": conns})
	if err := q.CreateJob(ctx, &queue.Job{
		ID:        "seed-conn-01",
		Type:      "connections",
		Status:    "done",
		NotePath:  paths[0],
		CreatedAt: base,
	}); err != nil {
		log.Fatalf("create conn job: %v", err)
	}
	if err := q.UpdateJobResult(ctx, "seed-conn-01", payload); err != nil {
		log.Fatalf("update conn result: %v", err)
	}

	log.Printf("seeded %d notes + %d people into %s", notes, len(people), *dbPath)
}

func writeNote(vaultDir, rel, title, body string, created time.Time) error {
	ts := created.Format(time.RFC3339)
	content := fmt.Sprintf(`---
created: %s
type: text
status: done
tags:
  - seed
---

# %s

## Summary
%s

## Raw
%s
`, ts, title, body, body)

	return os.WriteFile(filepath.Join(vaultDir, filepath.Base(rel)), []byte(content), 0o644)
}
