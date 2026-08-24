package ingest

import (
	"context"
	"log/slog"
	"strings"

	"github.com/rawnaqs/khayal/internal/chunk"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/queue"
)

// SaveChunksForNote splits content into chunks, embeds them in a single
// batch, and atomically replaces the note's stored chunks.
func SaveChunksForNote(ctx context.Context, q *queue.Queue, llmClient llm.LLMExt, notePath, content string, opts chunk.Options) error {
	chunks := chunk.ChunkText(content, opts)
	if len(chunks) == 0 {
		return nil
	}

	texts := make([]string, len(chunks))
	rows := make([]queue.ChunkRow, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Content
		rows[i] = queue.ChunkRow{Idx: i, Content: c.Content}
	}

	embeddings, err := llmClient.EmbedBatch(texts)
	if err != nil {
		return err
	}
	if len(embeddings) != len(chunks) {
		return ErrEmbeddingCountMismatch{Chunks: len(chunks), Embeddings: len(embeddings)}
	}
	for i := range rows {
		rows[i].Embedding = embeddings[i]
	}

	return q.ReplaceChunks(ctx, notePath, rows)
}

// ErrEmbeddingCountMismatch is returned when the embedding backend returns a
// different number of vectors than the texts submitted.
type ErrEmbeddingCountMismatch struct {
	Chunks     int
	Embeddings int
}

func (e ErrEmbeddingCountMismatch) Error() string {
	return "embedding count mismatch"
}

// saveChunks is the ingest-side wrapper: notes are already saved and
// FTS-indexed at this point, so chunk failures fail open and only the
// vectors are skipped.
func saveChunks(ctx context.Context, q *queue.Queue, llmClient llm.LLMExt, notePath, content string, opts chunk.Options) {
	chunks := chunk.ChunkText(content, opts)
	slog.Debug("chunk indexing",
		"note_path", notePath,
		"content_chars", len(content),
		"paragraphs", strings.Count(content, "\n\n")+1,
		"opts", opts,
		"chunks", len(chunks),
	)
	if err := SaveChunksForNote(ctx, q, llmClient, notePath, content, opts); err != nil {
		slog.Warn("chunk indexing failed", "note_path", notePath, "error", err)
	}
}

// StripFrontmatter removes a leading YAML frontmatter block (delimited by
// "---" lines) so chunking operates on note body text only.
func StripFrontmatter(content string) string {
	lines := strings.SplitAfter(content, "\n")
	if len(lines) < 2 || strings.TrimRight(lines[0], "\r\n") != "---" {
		return content
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r\n") == "---" {
			return strings.Join(lines[i+1:], "")
		}
	}
	return content
}
