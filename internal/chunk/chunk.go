package chunk

import (
	"strings"
	"unicode"
)

type Options struct {
	TargetWords  int
	MinWords     int
	OverlapWords int
}

type Chunk struct {
	Content   string
	WordCount int
}

func DefaultOptions() Options {
	return Options{
		TargetWords:  175,
		MinWords:     50,
		OverlapWords: 35,
	}
}

const paraSep = "\n\n"

func ChunkText(text string, opts Options) []Chunk {
	if opts.TargetWords <= 0 {
		opts = DefaultOptions()
	}

	paras := splitParas(text)

	var chunks []Chunk
	var cur []string
	curWords := 0
	overlapActive := false

	for _, para := range paras {
		pw := wordCount(para)

		if curWords+pw > opts.TargetWords && curWords >= opts.MinWords {
			content := strings.Join(cur, paraSep)
			chunks = append(chunks, Chunk{Content: content, WordCount: curWords})

			cur = nil
			overlapActive = false
			if op := joinWords(takeLastNWords(content, opts.OverlapWords)); op != "" {
				overlapActive = true
				cur = append(cur, op)
				curWords = wordCount(op)
			} else {
				curWords = 0
			}
		}

		cur = append(cur, para)
		curWords += pw
	}

	if curWords == 0 {
		return chunks
	}

	// Lone input: keep everything as one chunk so short captures stay
	// searchable — the minimum governs fragment merging, not existence.
	if len(chunks) == 0 {
		return append(chunks, Chunk{
			Content:   strings.Join(cur, paraSep),
			WordCount: curWords,
		})
	}

	// Compare the non-overlap remainder against the minimum: overlap words
	// are duplicated content and must not inflate a fragment past minimum.
	remWords := curWords
	remainder := cur
	if overlapActive && len(cur) > 0 {
		remWords -= wordCount(cur[0])
		remainder = cur[1:]
	}

	if remWords >= opts.MinWords {
		chunks = append(chunks, Chunk{
			Content:   strings.Join(cur, paraSep),
			WordCount: curWords,
		})
		return chunks
	}

	if remWords == 0 {
		return chunks
	}

	last := &chunks[len(chunks)-1]
	last.Content += paraSep + strings.Join(remainder, paraSep)
	last.WordCount += remWords

	return chunks
}

func splitParas(text string) []string {
	raw := strings.Split(text, paraSep)
	result := make([]string, 0, len(raw))
	for _, p := range raw {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func wordCount(s string) int {
	n := 0
	inWord := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			n++
			inWord = true
		}
	}
	return n
}

func takeLastNWords(text string, n int) []string {
	if n <= 0 {
		return nil
	}
	words := strings.Fields(text)
	if n >= len(words) {
		return words
	}
	return words[len(words)-n:]
}

func joinWords(words []string) string {
	return strings.Join(words, " ")
}
