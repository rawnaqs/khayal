package ingest

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/rawnaqs/khayal/internal/chunk"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/queue"
	"golang.org/x/sync/errgroup"

	"github.com/rawnaqs/khayal/internal/vault"
)

// IngestPDF processes a captured PDF: the text layer was extracted at
// capture time and rides in job.Content. Enrichment mirrors IngestText
// (tags, summary, key ideas, entities, chunks); the stored PDF lives in
// the media dir and is linked via source_file.
func IngestPDF(ctx context.Context, job *queue.Job, v *vault.Writer, q *queue.Queue, llmClient llm.LLMExt, chunkOpts chunk.Options, memCfg config.MemoryConfig) (string, error) {
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

	rawEntities, err := llmClient.ExtractEntities(job.Content, llm.BucketText)
	if err != nil {
		return "", fmt.Errorf("failed to extract entities: %w", err)
	}
	entities := NormalizeEntities(rawEntities)

	title := pdfTitle(job.SourceFile, job.Content)
	now := time.Now().UTC()
	entities.ResolveRelativeDates(now)
	rescuePeople(ctx, q, &entities, job.Content)

	note := &vault.Note{
		Metadata: vault.NoteMetadata{
			Created:    job.CreatedAt,
			Updated:    &now,
			Type:       "pdf",
			Status:     "done",
			Tags:       tags,
			SourceFile: job.SourceFile,
			Entities:   entities.toVaultBlock(),
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

// pdfTitle prefers the uploaded filename ("report.pdf" -> "Report").
// Media storage renames uploads to timestamps, so digit-only basenames
// carry no meaning and the first content line is used instead.
func pdfTitle(sourceFile, content string) string {
	if sourceFile != "" {
		base := filepath.Base(sourceFile)
		base = strings.TrimSuffix(base, filepath.Ext(base))
		base = strings.ReplaceAll(base, "_", " ")
		base = strings.TrimSpace(base)
		if base != "" && !isAllDigits(base) {
			return strings.ToUpper(base[:1]) + base[1:]
		}
	}
	return extractTitle(content)
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
