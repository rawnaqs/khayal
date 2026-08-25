package ingest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/rawnaqs/khayal/internal/chunk"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
)

func IngestText(ctx context.Context, job *queue.Job, v *vault.Writer, q *queue.Queue, llmClient llm.LLMExt, chunkOpts chunk.Options, memCfg config.MemoryConfig) (string, error) {
	var tags []string
	var summary string
	var keyIdeas []string

	defer setCallContext(llmClient, assembleMemoryContext(ctx, q, v, llmClient, memCfg, job.Content))()

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

	if err := g.Wait(); err != nil {
		return "", fmt.Errorf("llm extraction failed: %w", err)
	}

	// Sequential enrichment pass — not part of the errgroup above.
	rawEntities, err := llmClient.ExtractEntities(job.Content, llm.BucketText)
	if err != nil {
		return "", fmt.Errorf("failed to extract entities: %w", err)
	}
	entities := NormalizeEntities(rawEntities)

	title := extractTitle(job.Content)
	now := time.Now().UTC()
	entities.ResolveRelativeDates(now)

	note := &vault.Note{
		Metadata: vault.NoteMetadata{
			Created:  job.CreatedAt,
			Updated:  &now,
			Type:     "text",
			Status:   "done",
			Tags:     tags,
			Entities: entities.toVaultBlock(),
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

	if err := q.SaveEntities(ctx, notePath, entities.toQueue()); err != nil {
		return "", fmt.Errorf("failed to save entities: %w", err)
	}

	if err := q.IndexNote(ctx, notePath, title, job.Content, strings.Join(tags, ",")); err != nil {
		return "", fmt.Errorf("failed to index note: %w", err)
	}

	saveChunks(ctx, q, llmClient, notePath, job.Content, chunkOpts)

	return notePath, nil
}

func extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	firstLine := strings.TrimSpace(lines[0])
	if len(firstLine) > 100 {
		firstLine = firstLine[:100]
	}
	return firstLine
}
