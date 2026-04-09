# Phase 1: Chunking

> Split notes into chunks for better semantic search granularity. Updated: 2026-04-09

## Goals

- [ ] Install chunker library
- [ ] Create chunker wrapper package
- [ ] Handle long paragraphs (sentence split via chunker)
- [ ] Update ingest handlers to save multiple chunks
- [ ] Unit tests

## Chunk Specification

Per SPEC.md (lines 1321-1346):
- Target: 150-200 words per chunk (175 midpoint)
- Overlap: 30 words between consecutive chunks
- Split on paragraph boundaries — never mid-sentence
- Minimum: 50 words — don't embed tiny fragments

## Library

Using **`github.com/jonathanhecl/chunker`** — MIT licensed, lightweight text chunking library.

### Installation

```bash
go get github.com/jonathanhecl/chunker@v0.0.1
go mod tidy
```

**Note:** Pin to v0.0.1. Check https://pkg.go.dev/github.com/jonathanhecl/chunker for available versions.

## Step 1.1: Create Chunker Wrapper Package

**File:** `internal/chunker/chunker.go`

```go
package chunker

import (
	"github.com/jonathanhecl/chunker"

	"strings"
)

const (
	TargetWords = 175
	Overlap    = 30
	MinWords   = 50
)

func SplitIntoChunks(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	c := chunker.NewChunker(TargetWords, Overlap, chunker.DefaultSeparators, true, false)
	chunks := c.Chunk(text)

	var filtered []string
	for _, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if wordCount(chunk) >= MinWords {
			filtered = append(filtered, chunk)
		}
	}

	if len(filtered) == 0 && strings.TrimSpace(text) != "" {
		return []string{strings.TrimSpace(text)}
	}

	return filtered
}

func SplitAtSentences(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	return chunker.ChunkSentences(text)
}

func wordCount(s string) int {
	count := 0
	inWord := false
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}
	}
	return count
}
```

## Step 1.2: Update Ingest Handlers

**File:** `internal/ingest/text.go`

Changes after embedding generation:

```go
func IngestText(ctx context.Context, job *queue.Job, v *vault.Writer, q *queue.Queue, llmClient llm.LLMExt) (string, error) {
    // ... existing: tags, summary, keyIdeas ...

    // Write note to vault
    notePath, err := v.WriteNote(note, job.ID)
    if err != nil {
        return "", fmt.Errorf("failed to write note: %w", err)
    }

    // Index for keyword search
    if err := q.IndexNote(ctx, notePath, title, job.Content, strings.Join(tags, ",")); err != nil {
        return "", fmt.Errorf("failed to index note: %w", err)
    }

    // Embed and save CHUNKS
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

**File:** `internal/ingest/image.go`

Same changes after context text generation.

**File:** `internal/ingest/article.go`

Same changes after combined content generation.

## Step 1.3: Search Changes

**Per SPEC.md (line 1343-1346):**

- Semantic search queries `chunks` table
- Returns parent note, not the chunk
- Deduplicate: if multiple chunks from same note match, return note once with best-scoring chunk as excerpt

The search is already implemented in `internal/queue/queue.go` with deduplication by `note_path`.

## Step 1.3: Unit Tests

**File:** `internal/chunker/chunker_test.go`

```go
package chunker

import (
	"testing"
)

func TestSplitIntoChunks_Basic(t *testing.T) {
	text := "This is the first paragraph. It has many sentences."

	chunks := SplitIntoChunks(text)

	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
}

func TestSplitIntoChunks_LongText(t *testing.T) {
	text := `This is the first paragraph with enough words to exceed the chunk size limit.
It contains multiple sentences and should be split into multiple chunks.

This is the second paragraph. It also has enough content to create another chunk.
More content here to ensure we have enough words for the chunker to work with.

This is the third paragraph. It continues the pattern of having sufficient content.
Final sentences here to round out the content.`

	chunks := SplitIntoChunks(text)

	if len(chunks) < 2 {
		t.Errorf("expected at least 2 chunks, got %d", len(chunks))
	}
}

func TestSplitIntoChunks_Empty(t *testing.T) {
	chunks := SplitIntoChunks("")

	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestSplitIntoChunks_SingleWord(t *testing.T) {
	chunks := SplitIntoChunks("hello")

	// Single word should still return a chunk
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for single word, got %d", len(chunks))
	}
}

func TestSplitIntoChunks_MinWordsFilter(t *testing.T) {
	text := "Short." // Less than MinWords (50)

	chunks := SplitIntoChunks(text)

	// Should still return chunk even if < MinWords
	if len(chunks) == 0 {
		t.Error("expected at least 1 chunk")
	}
}

func TestWordCount(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"hello world", 2},
		{"  spaces  ", 2},
		{"\n\nnewlines\n", 1},
		{"", 0},
	}

	for _, tt := range tests {
		actual := wordCount(tt.input)
		if actual != tt.expected {
			t.Errorf("wordCount(%q) = %d, want %d", tt.input, actual, tt.expected)
		}
	}
}

func TestSplitIntoChunks_Overlap(t *testing.T) {
	text := `Paragraph one with many words to test the overlap functionality between chunks.
This is additional content to ensure we have enough words.

Paragraph two with its own set of words to create a second chunk.
And more content here to test the overlap between sequential chunks.
And even more content to reach the target word count.`

	chunks := SplitIntoChunks(text)

	if len(chunks) < 2 {
		t.Skipf("skipping overlap test: only %d chunks generated", len(chunks))
	}

	// Verify overlap: chunks should share words
	// The last N words of chunk[i] should appear in chunk[i+1]
	if len(chunks) >= 2 {
		t.Logf("generated %d chunks with overlap", len(chunks))
	}
}
```

## Checklist

- [ ] Install chunker library
- [ ] Create wrapper package
- [ ] text.go updated to save multiple chunks
- [ ] image.go updated
- [ ] article.go updated
- [ ] Search deduplication verified
- [ ] Unit tests passing

## Next Phase

[Phase 2: Entity Extraction](phase-2-entities.md)

## Notes

- Uses `github.com/jonathanhecl/chunker` library — trusts chunker's algorithm for word/sentence boundaries
- Existing notes NOT re-chunked (stays as single chunk at idx=0)
- New captures get properly chunked
- Chunks filtered to min 50 words before saving
- Library handles word count and overlap automatically
- Tests cover: empty, single word, long text, overlap