package ingest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
)

func IngestImage(ctx context.Context, job *queue.Job, v *vault.Writer, q *queue.Queue, llmClient llm.LLMExt) (string, error) {
	description, err := llmClient.DescribeImage(job.SourceFile)
	if err != nil {
		return "", fmt.Errorf("failed to describe image: %w", err)
	}

	contextText := description
	if job.UserContext != "" {
		contextText = job.UserContext + "\n\n" + description
	}

	tags, err := llmClient.ExtractTags(contextText)
	if err != nil {
		tags = []string{"image"}
	}

	now := time.Now()
	note := &vault.Note{
		Metadata: vault.NoteMetadata{
			Created:     job.CreatedAt,
			Updated:     &now,
			Type:        "image",
			Status:      "done",
			SourceFile:  job.SourceFile,
			UserContext: job.UserContext,
			Tags:        tags,
			History: []vault.HistoryEvent{
				{At: now, Event: "processed"},
			},
		},
		Title: fmt.Sprintf("Image — %s", job.CreatedAt.Format("2006-01-02")),
		Raw:   description,
	}

	notePath, err := v.WriteNote(note)
	if err != nil {
		return "", fmt.Errorf("failed to write note: %w", err)
	}

	if err := q.IndexNote(ctx, notePath, note.Title, contextText, strings.Join(tags, ",")); err != nil {
		return "", fmt.Errorf("failed to index note: %w", err)
	}

	embedding, err := llmClient.Embed(contextText)
	if err != nil {
		return notePath, nil
	}

	if err := q.SaveChunk(ctx, notePath, 0, contextText, embedding); err != nil {
		return notePath, nil
	}

	return notePath, nil
}
