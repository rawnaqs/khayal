package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractWikilinks(t *testing.T) {
	content := "see [[alice]] and [[Bob Smith]] but not [single] or [[]]"
	links := extractWikilinks(content)
	if len(links) != 2 || links[0] != "alice" || links[1] != "Bob Smith" {
		t.Errorf("got %v", links)
	}
}

func TestWordSimilarity(t *testing.T) {
	same, shared := wordSimilarity("the quick brown fox", "the quick brown fox")
	if same != 1 || shared != 4 {
		t.Errorf("identical: sim=%f shared=%d", same, shared)
	}
	disjoint, _ := wordSimilarity("alpha beta", "gamma delta")
	if disjoint != 0 {
		t.Errorf("disjoint: %f", disjoint)
	}
	half, shared2 := wordSimilarity("one two three four", "three four five six")
	if shared2 != 2 {
		t.Errorf("shared=%d", shared2)
	}
	if half <= 0 || half >= 1 {
		t.Errorf("partial similarity out of range: %f", half)
	}
}

func TestListInboxNotesExcludesTrashAndNonMd(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("---\n---\nbody"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("nope"), 0644)
	trash := filepath.Join(dir, ".khayal-trash")
	os.MkdirAll(trash, 0755)
	os.WriteFile(filepath.Join(trash, "dead.md"), []byte("x"), 0644)

	names, err := listInboxNotes(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "a.md" {
		t.Errorf("got %v, want only a.md", names)
	}
}

func TestFormatSize(t *testing.T) {
	if got := formatSize(500); got != "500 B" {
		t.Errorf("got %q", got)
	}
	if got := formatSize(2048); got != "2.0 KB" {
		t.Errorf("got %q", got)
	}
	if got := formatSize(3 * 1024 * 1024); got != "3.0 MB" {
		t.Errorf("got %q", got)
	}
}
