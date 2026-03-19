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

func IngestText(ctx context.Context, job *queue.Job, v *vault.Writer, q *queue.Queue, llmClient llm.LLMExt) (string, error) {
	tags, err := llmClient.ExtractTags(job.Content)
	if err != nil {
		return "", fmt.Errorf("failed to extract tags: %w", err)
	}

	summary, err := llmClient.Summarize(job.Content)
	if err != nil {
		return "", fmt.Errorf("failed to summarize: %w", err)
	}

	keyIdeas, err := llmClient.ExtractKeyIdeas(job.Content)
	if err != nil {
		keyIdeas = []string{}
	}

	title := extractTitle(job.Content)
	now := time.Now()

	note := &vault.Note{
		Metadata: vault.NoteMetadata{
			Created: job.CreatedAt,
			Updated: &now,
			Type:    "text",
			Status:  "done",
			Tags:    tags,
			History: []vault.HistoryEvent{
				{At: now, Event: "processed"},
			},
		},
		Title:    title,
		Summary:  summary,
		KeyIdeas: keyIdeas,
		Raw:      job.Content,
	}

	notePath, err := v.WriteNote(note)
	if err != nil {
		return "", fmt.Errorf("failed to write note: %w", err)
	}

	if err := q.IndexNote(ctx, notePath, title, job.Content, strings.Join(tags, ",")); err != nil {
		return "", fmt.Errorf("failed to index note: %w", err)
	}

	embedding, err := llmClient.Embed(job.Content)
	if err != nil {
		return notePath, nil
	}

	if err := q.SaveChunk(ctx, notePath, 0, job.Content, embedding); err != nil {
		return notePath, nil
	}

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
