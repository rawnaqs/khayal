package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReader_ReadNote(t *testing.T) {
	// Create temp vault with test note
	vaultPath := t.TempDir()
	inboxPath := filepath.Join(vaultPath, "inbox")
	os.MkdirAll(inboxPath, 0755)

	testNote := `---
created: "2024-03-16T14:23:00Z"
updated: "2024-03-16T14:23:04Z"
type: text
status: done
tags:
  - react
  - performance
---

# Test Note

## Summary
A brief summary of the note.

## Key Ideas
- First idea about performance
- Second idea about optimization

## Raw
Original content here with more details.
This is the raw body of the note.
`

	notePath := filepath.Join(inboxPath, "test.md")
	os.WriteFile(notePath, []byte(testNote), 0644)

	// Test reading
	reader := NewReader(vaultPath, "inbox")
	note, err := reader.ReadNote("inbox/test.md")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if note.Title != "Test Note" {
		t.Errorf("expected title 'Test Note', got %q", note.Title)
	}
	if note.Summary != "A brief summary of the note." {
		t.Errorf("expected summary 'A brief summary of the note.', got %q", note.Summary)
	}
	if len(note.KeyIdeas) != 2 {
		t.Errorf("expected 2 key ideas, got %d", len(note.KeyIdeas))
	}
	if note.Raw != "Original content here with more details.\nThis is the raw body of the note." {
		t.Errorf("unexpected raw content: %q", note.Raw)
	}
	if note.Type != "text" {
		t.Errorf("expected type 'text', got %q", note.Type)
	}
	if len(note.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(note.Tags))
	}
}

func TestReader_ReadNote_NoFrontmatter(t *testing.T) {
	vaultPath := t.TempDir()
	inboxPath := filepath.Join(vaultPath, "inbox")
	os.MkdirAll(inboxPath, 0755)

	testNote := `# Simple Note

This is a simple note without frontmatter.
Just plain markdown.
`

	notePath := filepath.Join(inboxPath, "simple.md")
	os.WriteFile(notePath, []byte(testNote), 0644)

	reader := NewReader(vaultPath, "inbox")
	note, err := reader.ReadNote("inbox/simple.md")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if note.Title != "Simple Note" {
		t.Errorf("expected title 'Simple Note', got %q", note.Title)
	}
	if note.Raw == "" {
		t.Error("expected raw content to be populated")
	}
}

func TestReader_ReadNote_PathTraversal(t *testing.T) {
	vaultPath := t.TempDir()
	inboxPath := filepath.Join(vaultPath, "inbox")
	os.MkdirAll(inboxPath, 0755)

	reader := NewReader(vaultPath, "inbox")

	// Try to read file outside inbox
	_, err := reader.ReadNote("../../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal attempt")
	}
}

func TestReader_ReadNote_NotFound(t *testing.T) {
	vaultPath := t.TempDir()
	reader := NewReader(vaultPath, "inbox")

	_, err := reader.ReadNote("inbox/nonexistent.md")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestReader_ReadNote_Subdir(t *testing.T) {
	vaultPath := t.TempDir()
	inboxPath := filepath.Join(vaultPath, "inbox")
	subdir := filepath.Join(inboxPath, "khayal")
	os.MkdirAll(subdir, 0755)

	testNote := `---
type: text
---

# Subdir Note
Content in subdirectory.
`

	notePath := filepath.Join(subdir, "note.md")
	os.WriteFile(notePath, []byte(testNote), 0644)

	reader := NewReader(vaultPath, "inbox")
	note, err := reader.ReadNote("inbox/khayal/note.md")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if note.Title != "Subdir Note" {
		t.Errorf("expected title 'Subdir Note', got %q", note.Title)
	}
}

// The proactive-connections block is written as `connections:` by
// SetConnections; the reader must surface it through Related so the API
// (and PWA note view) can render linked notes.
func TestReader_ReadNote_ConnectionsFoldIntoRelated(t *testing.T) {
	vaultPath := t.TempDir()
	inboxPath := filepath.Join(vaultPath, "inbox")
	os.MkdirAll(inboxPath, 0755)

	testNote := `---
created: "2024-03-16T14:23:00Z"
type: text
connections:
  - "[[2024-03-10-old-note]]"
  - "[[2024-03-12-other-note]]"
---

# Connected Note

## Summary
x
`
	os.WriteFile(filepath.Join(inboxPath, "connected.md"), []byte(testNote), 0644)

	r := NewReader(vaultPath, "inbox")
	note, err := r.ReadNote("inbox/connected.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(note.Related) != 2 {
		t.Fatalf("expected 2 related links, got %v", note.Related)
	}
	if note.Related[0] != "[[2024-03-10-old-note]]" {
		t.Errorf("link content mismatch: %v", note.Related)
	}
}

// Backlinks: which inbox notes reference a target in their connections
// block. Sources are returned as inbox-relative note paths.
func TestReader_ScanBacklinks(t *testing.T) {
	vaultPath := t.TempDir()
	inboxPath := filepath.Join(vaultPath, "khayal")
	os.MkdirAll(inboxPath, 0755)

	note := func(name, connections string) {
		content := "---\ntype: text\n" + connections + "---\n\n# " + name + "\n"
		os.WriteFile(filepath.Join(inboxPath, name), []byte(content), 0644)
	}
	note("a.md", "connections:\n  - \"[[b]]\"\n")
	note("b.md", "")
	note("c.md", "connections:\n  - \"[[a]]\"\n  - \"[[b]]\"\n")
	note("d.md", "") // not .md? keep all md
	// a non-md file must be ignored
	os.WriteFile(filepath.Join(inboxPath, "ignore.txt"), []byte("x"), 0644)

	r := NewReader(vaultPath, "khayal")
	scan, err := r.ScanBacklinks()
	if err != nil {
		t.Fatal(err)
	}

	if got := scan["b"]; len(got) != 2 {
		t.Errorf("b: expected 2 backlinks (a, c), got %v", got)
	} else {
		for _, want := range []string{"khayal/a.md", "khayal/c.md"} {
			found := false
			for _, p := range got {
				if p == want {
					found = true
				}
			}
			if !found {
				t.Errorf("b: missing %s in %v", want, got)
			}
		}
	}
	if got := scan["a"]; len(got) != 1 || got[0] != "khayal/c.md" {
		t.Errorf("a: expected [khayal/c.md], got %v", got)
	}
	if _, ok := scan["nonexistent"]; ok {
		t.Error("phantom key")
	}
}
