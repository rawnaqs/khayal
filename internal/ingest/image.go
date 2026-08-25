package ingest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/rawnaqs/khayal/internal/chunk"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
)

func IngestImage(ctx context.Context, job *queue.Job, v *vault.Writer, q *queue.Queue, llmClient llm.LLMExt, chunkOpts chunk.Options) (string, error) {
	imagePath := v.ResolveMediaPath(job.SourceFile)
	description, err := llmClient.DescribeImage(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to describe image: %w", err)
	}

	contextText := description
	if job.UserContext != "" {
		contextText = job.UserContext + "\n\n" + description
	}

	g, _ := errgroup.WithContext(ctx)

	var tags []string

	g.Go(func() error {
		var err error
		tags, err = llmClient.ExtractTags(contextText, llm.BucketImage)
		return err
	})

	if err := g.Wait(); err != nil {
		return "", fmt.Errorf("failed to extract tags: %w", err)
	}

	if tags == nil {
		tags = []string{"image"}
	}

	// Sequential enrichment pass — not part of the errgroup above.
	rawEntities, err := llmClient.ExtractEntities(contextText, llm.BucketImage)
	if err != nil {
		return "", fmt.Errorf("failed to extract entities: %w", err)
	}
	entities := NormalizeEntities(rawEntities)

	now := time.Now().UTC()
	entities.ResolveRelativeDates(now)
	note := &vault.Note{
		Metadata: vault.NoteMetadata{
			Created:     job.CreatedAt,
			Updated:     &now,
			Type:        "image",
			Status:      "done",
			SourceFile:  job.SourceFile,
			UserContext: job.UserContext,
			Tags:        tags,
			Entities:    entities.toVaultBlock(),
			History: []vault.HistoryEvent{
				{At: now, Event: "processed"},
			},
		},
		Title: func() string {
			if job.UserContext != "" {
				return job.UserContext
			} else {
				return "Image"
			}
		}(),
		Raw: description,
	}

	notePath, err := v.WriteNote(note, job.ID)
	if err != nil {
		return "", fmt.Errorf("failed to write note: %w", err)
	}

	if err := q.SaveEntities(ctx, notePath, entities.toQueue()); err != nil {
		return "", fmt.Errorf("failed to save entities: %w", err)
	}

	if err := q.IndexNote(ctx, notePath, note.Title, contextText, strings.Join(tags, ",")); err != nil {
		return "", fmt.Errorf("failed to index note: %w", err)
	}

	saveChunks(ctx, q, llmClient, notePath, contextText, chunkOpts)

	return notePath, nil
}
