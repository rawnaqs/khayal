package chunk

import (
	"strings"
	"testing"
)

func TestChunkText_Basic(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		opts    Options
		wantLen int
		wantWC  []int
	}{
		{
			name:    "empty input returns no chunks",
			text:    "",
			opts:    DefaultOptions(),
			wantLen: 0,
			wantWC:  nil,
		},
		{
			name:    "single short paragraph emits as one chunk",
			text:    "a b c d e",
			opts:    Options{TargetWords: 3, MinWords: 2, OverlapWords: 1},
			wantLen: 1,
			wantWC:  []int{5},
		},
		{
			name:    "single paragraph at target emits as one chunk",
			text:    "a b c",
			opts:    Options{TargetWords: 3, MinWords: 2, OverlapWords: 1},
			wantLen: 1,
			wantWC:  []int{3},
		},
		{
			name:    "overflow at minimum emits and carries overlap",
			text:    "a b\n\nc d",
			opts:    Options{TargetWords: 3, MinWords: 2, OverlapWords: 1},
			wantLen: 2,
			wantWC:  []int{2, 3},
		},
		{
			name:    "three paragraphs produce three chunks via overlap carry",
			text:    "one two three\n\nfour five six seven\n\neight nine ten eleven",
			opts:    Options{TargetWords: 4, MinWords: 3, OverlapWords: 2},
			wantLen: 3,
			wantWC:  []int{3, 6, 6},
		},
		{
			name:    "many single-word paragraphs with one-word overlap",
			text:    "one\n\ntwo\n\nthree\n\nfour\n\nfive\n\nsix\n\nseven",
			opts:    Options{TargetWords: 3, MinWords: 2, OverlapWords: 1},
			wantLen: 3,
			wantWC:  []int{3, 3, 3},
		},
		{
			name:    "sub-minimum final fragment merged into previous",
			text:    "one two three\n\nfour five\n\nsix",
			opts:    Options{TargetWords: 3, MinWords: 2, OverlapWords: 1},
			wantLen: 2,
			wantWC:  []int{3, 4},
		},
		{
			name:    "lone input below minimum kept as single chunk",
			text:    "x",
			opts:    Options{TargetWords: 3, MinWords: 2, OverlapWords: 1},
			wantLen: 1,
			wantWC:  []int{1},
		},
		{
			name:    "paragraph longer than target kept whole",
			text:    "a b c d e f g h",
			opts:    Options{TargetWords: 3, MinWords: 2, OverlapWords: 1},
			wantLen: 1,
			wantWC:  []int{8},
		},
		{
			name:    "long paragraph kept whole and short para joins overlap tail",
			text:    "a b c d e f\n\ng h",
			opts:    Options{TargetWords: 4, MinWords: 2, OverlapWords: 2},
			wantLen: 2,
			wantWC:  []int{6, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ChunkText(tt.text, tt.opts)
			if len(got) != tt.wantLen {
				t.Errorf("ChunkText len = %d, want %d", len(got), tt.wantLen)
				for i, c := range got {
					t.Logf("  chunk[%d]: %q (%d words)", i, c.Content, c.WordCount)
				}
				return
			}
			for i := range got {
				if got[i].WordCount != tt.wantWC[i] {
					t.Errorf("chunk[%d] WordCount = %d, want %d", i, got[i].WordCount, tt.wantWC[i])
				}
			}
		})
	}
}

func TestChunkText_OverlapContents(t *testing.T) {
	text := "one two three\n\nfour five six seven\n\neight nine ten eleven"
	opts := Options{TargetWords: 4, MinWords: 3, OverlapWords: 2}
	chunks := ChunkText(text, opts)

	want := []string{
		"one two three",
		"two three\n\nfour five six seven",
		"six seven\n\neight nine ten eleven",
	}
	if len(chunks) != len(want) {
		t.Fatalf("len = %d, want %d", len(chunks), len(want))
	}
	for i, w := range want {
		if chunks[i].Content != w {
			t.Errorf("chunk[%d] = %q, want %q", i, chunks[i].Content, w)
		}
	}
}

func TestChunkText_FinalFragmentMerged(t *testing.T) {
	text := "one two three\n\nfour five\n\nsix"
	opts := Options{TargetWords: 3, MinWords: 2, OverlapWords: 1}
	chunks := ChunkText(text, opts)

	if len(chunks) != 2 {
		t.Fatalf("len = %d, want 2", len(chunks))
	}
	if chunks[1].Content != "three\n\nfour five\n\nsix" {
		t.Errorf("chunk[1] = %q, want %q", chunks[1].Content, "three\n\nfour five\n\nsix")
	}
}

func TestChunkText_WallOfTextStaysWhole(t *testing.T) {
	// Deliberate decision: a single paragraph is never split, even when far
	// past the target — chunk boundaries only fall on paragraph breaks
	// ("never mid-sentence"). The target governs multi-paragraph notes.
	text := strings.Repeat("word ", 5000)
	chunks := ChunkText(text, DefaultOptions())

	if len(chunks) != 1 {
		t.Fatalf("len = %d, want 1", len(chunks))
	}
	if chunks[0].WordCount != 5000 {
		t.Errorf("WordCount = %d, want 5000", chunks[0].WordCount)
	}
}

func TestChunkText_ZeroOverlapPreservesContent(t *testing.T) {
	// Regression: with OverlapWords=0, no paragraph may be dropped or
	// misclassified as an overlap prefix during finalize.
	text := "one two three four five\n\nsix seven eight\n\nnine ten eleven twelve\n\nthirteen fourteen fifteen sixteen"
	opts := Options{TargetWords: 8, MinWords: 4, OverlapWords: 0}
	chunks := ChunkText(text, opts)

	joined := ""
	for i, c := range chunks {
		if i > 0 {
			joined += "\n\n"
		}
		joined += c.Content
	}

	for _, para := range []string{
		"one two three four five",
		"six seven eight",
		"nine ten eleven twelve",
		"thirteen fourteen fifteen sixteen",
	} {
		if !strings.Contains(joined, para) {
			t.Errorf("paragraph %q missing from output:\n%s", para, joined)
		}
	}
}

func TestChunkText_WordCount(t *testing.T) {
	chunks := ChunkText(
		"hello world this is a test note with many words here",
		DefaultOptions(),
	)

	if len(chunks) != 1 {
		t.Fatalf("len = %d, want 1", len(chunks))
	}
	if chunks[0].WordCount != 11 {
		t.Errorf("WordCount = %d, want 11", chunks[0].WordCount)
	}
}
