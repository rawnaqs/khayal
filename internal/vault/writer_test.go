package vault

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rawnaqs/khayal/internal/config"
)

func TestWriter(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Vault: config.VaultConfig{
			Path:     tmpDir,
			InboxDir: "inbox",
		},
	}

	writer, err := NewWriter(cfg, filepath.Join(tmpDir, "config.yaml"))
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	if writer.BasePath() != tmpDir {
		t.Errorf("expected base path %s, got %s", tmpDir, writer.BasePath())
	}

	expectedInbox := filepath.Join(tmpDir, "inbox")
	if writer.InboxPath() != expectedInbox {
		t.Errorf("expected inbox path %s, got %s", expectedInbox, writer.InboxPath())
	}
}

func TestWriteNote(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Vault: config.VaultConfig{
			Path:     tmpDir,
			InboxDir: "inbox",
			Media: config.MediaConfig{
				DefaultDir: "media",
			},
		},
	}
	writer, err := NewWriter(cfg, filepath.Join(tmpDir, "config.yaml"))
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	now := time.Now()
	note := &Note{
		Metadata: NoteMetadata{
			Created: now,
			Type:    "text",
			Status:  "done",
			Tags:    []string{"test", "golang"},
		},
		Title:   "Test Note",
		Summary: "This is a test note",
		KeyIdeas: []string{
			"First idea",
			"Second idea",
		},
		Raw: "Original raw content",
	}

	notePath, err := writer.WriteNote(note, "test-job-001")
	if err != nil {
		t.Fatalf("WriteNote() error = %v", err)
	}

	if !writer.NoteExists(notePath) {
		t.Error("expected note to exist after writing")
	}

	fullPath := writer.ResolvePath(notePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read note: %v", err)
	}

	if !contains(string(content), "Test Note") {
		t.Error("expected note to contain title")
	}
	if !contains(string(content), "test") {
		t.Error("expected note to contain tag 'test'")
	}
}

func TestUpdateNote(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Vault: config.VaultConfig{
			Path:     tmpDir,
			InboxDir: "inbox",
			Media: config.MediaConfig{
				DefaultDir: "media",
			},
		},
	}
	writer, err := NewWriter(cfg, filepath.Join(tmpDir, "config.yaml"))
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	note := &Note{
		Metadata: NoteMetadata{
			Created: time.Now(),
			Type:    "text",
			Status:  "done",
		},
		Title: "Original Title",
		Raw:   "Original content",
	}

	notePath, err := writer.WriteNote(note, "test-job-002")
	if err != nil {
		t.Fatalf("WriteNote() error = %v", err)
	}

	note.Title = "Updated Title"
	note.Summary = "Updated summary"

	if err := writer.UpdateNote(notePath, note); err != nil {
		t.Fatalf("UpdateNote() error = %v", err)
	}

	fullPath := writer.ResolvePath(notePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read note: %v", err)
	}

	if !contains(string(content), "Updated Title") {
		t.Error("expected note to contain updated title")
	}
	if !contains(string(content), "Updated summary") {
		t.Error("expected note to contain updated summary")
	}
}

func TestDeleteNote(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Vault: config.VaultConfig{
			Path:     tmpDir,
			InboxDir: "inbox",
			Media: config.MediaConfig{
				DefaultDir: "media",
			},
		},
	}
	writer, err := NewWriter(cfg, filepath.Join(tmpDir, "config.yaml"))
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	note := &Note{
		Metadata: NoteMetadata{
			Created: time.Now(),
			Type:    "text",
			Status:  "done",
		},
		Title: "To Delete",
		Raw:   "Content",
	}

	notePath, err := writer.WriteNote(note, "test-job-003")
	if err != nil {
		t.Fatalf("WriteNote() error = %v", err)
	}

	if err := writer.DeleteNote(notePath); err != nil {
		t.Fatalf("DeleteNote() error = %v", err)
	}

	if writer.NoteExists(notePath) {
		t.Error("expected note to not exist after deletion")
	}

	trashPath := filepath.Join(writer.InboxPath(), ".khayal-trash")
	if _, err := os.Stat(trashPath); os.IsNotExist(err) {
		t.Error("expected trash directory to exist")
	}
}

func TestCleanFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Test@#$%File", "testfile"},
		{"already-clean", "already-clean"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"UPPERCASE", "uppercase"},
	}

	for _, tt := range tests {
		result := cleanFilename(tt.input)
		if result != tt.expected {
			t.Errorf("cleanFilename(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	result := sanitizeFilename("test<>:\"/\\|.md")
	if contains(result, "<") || contains(result, ">") {
		t.Error("sanitizeFilename should remove < and >")
	}
}

func TestParseFrontmatter(t *testing.T) {
	content := `---
created: 2026-03-18T10:00:00Z
type: text
status: done
tags:
  - golang
  - testing
---

# Title

Content here
`

	meta, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatalf("ParseFrontmatter() error = %v", err)
	}

	if meta.Type != "text" {
		t.Errorf("expected type 'text', got %s", meta.Type)
	}
	if meta.Status != "done" {
		t.Errorf("expected status 'done', got %s", meta.Status)
	}
	if len(meta.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(meta.Tags))
	}
}

func TestParseFrontmatterNoFrontmatter(t *testing.T) {
	content := "# Just a title"

	_, err := ParseFrontmatter(content)
	if err == nil {
		t.Error("expected error for content without frontmatter")
	}
}

func TestCopyMediaFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Vault: config.VaultConfig{
			Path:     tmpDir,
			InboxDir: "inbox",
			Media: config.MediaConfig{
				DefaultDir: "media",
			},
		},
	}
	writer, err := NewWriter(cfg, filepath.Join(tmpDir, "config.yaml"))
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	tmpFile := filepath.Join(writer.InboxPath(), "test-image.png")
	if err := os.WriteFile(tmpFile, []byte("fake image data"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mediaPath, err := writer.CopyMediaFile(tmpFile)
	if err != nil {
		t.Fatalf("CopyMediaFile() error = %v", err)
	}

	if !contains(mediaPath, "inbox/media/") {
		t.Error("expected media path to contain inbox/media/")
	}

	fullMediaPath := writer.ResolvePath(mediaPath)
	if _, err := os.Stat(fullMediaPath); os.IsNotExist(err) {
		t.Error("expected media file to exist after copy")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
