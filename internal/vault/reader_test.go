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
