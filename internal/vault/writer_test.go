package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	fullPath := writer.ResolvePath(notePath)

	if err := writer.UpdateNote(fullPath, note); err != nil {
		t.Fatalf("UpdateNote() error = %v", err)
	}

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

	fullPath := writer.ResolvePath(notePath)

	if err := writer.DeleteNote(fullPath); err != nil {
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

	fullMediaPath := writer.ResolveMediaPath(mediaPath)
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

func TestRenderNote_WithEntities(t *testing.T) {
	w := &Writer{}
	note := &Note{
		Metadata: NoteMetadata{
			Created: time.Date(2026, 3, 16, 14, 23, 0, 0, time.UTC),
			Type:    "text",
			Status:  "done",
			Entities: &EntitiesBlock{
				People:  []string{"John Doe", "Jane Smith"},
				Amounts: []string{"2000"},
			},
		},
	}

	out := w.renderNote(note)

	if !strings.Contains(out, "entities:\n") {
		t.Fatalf("entities block missing:\n%s", out)
	}
	if !strings.Contains(out, "  people:\n    - John Doe\n    - Jane Smith\n") {
		t.Errorf("people rendering wrong:\n%s", out)
	}
	if !strings.Contains(out, "  amounts:\n    - 2000\n") {
		t.Errorf("amounts rendering wrong:\n%s", out)
	}
	if !strings.Contains(out, "  dates:  []\n") {
		t.Errorf("empty dates should render as []:\n%s", out)
	}

	// Order: entities block must come before history when history exists.
	entIdx := strings.Index(out, "entities:")
	histIdx := strings.Index(out, "history:")
	if entIdx == -1 {
		t.Fatalf("entities block missing:\n%s", out)
	}
	if histIdx != -1 && entIdx > histIdx {
		t.Errorf("entities block must precede history:\n%s", out)
	}
}

func TestRenderNote_EmptyEntitiesStillWritten(t *testing.T) {
	w := &Writer{}
	note := &Note{
		Metadata: NoteMetadata{
			Created:  time.Date(2026, 3, 16, 14, 23, 0, 0, time.UTC),
			Type:     "text",
			Status:   "done",
			Entities: &EntitiesBlock{},
		},
	}

	out := w.renderNote(note)

	for _, field := range []string{"people", "amounts", "dates", "places", "urls", "orgs"} {
		want := "  " + field + ":  []\n"
		if !strings.Contains(out, want) {
			t.Errorf("missing %s:\n%s", want, out)
		}
	}
}

func TestRenderNote_NilEntitiesStillWritten(t *testing.T) {
	w := &Writer{}
	note := &Note{
		Metadata: NoteMetadata{
			Created: time.Date(2026, 3, 16, 14, 23, 0, 0, time.UTC),
			Type:    "text",
			Status:  "done",
		},
	}

	out := w.renderNote(note)
	if !strings.Contains(out, "entities:\n") {
		t.Errorf("nil Entities must still render an empty block:\n%s", out)
	}
}

func TestRenderNote_EntitiesCappedAtTen(t *testing.T) {
	w := &Writer{}
	people := make([]string, 15)
	for i := range people {
		people[i] = fmt.Sprintf("person-%d", i)
	}
	note := &Note{
		Metadata: NoteMetadata{
			Created:  time.Date(2026, 3, 16, 14, 23, 0, 0, time.UTC),
			Type:     "text",
			Status:   "done",
			Entities: &EntitiesBlock{People: people},
		},
	}

	out := w.renderNote(note)
	if strings.Contains(out, "person-10") {
		t.Errorf("more than 10 entities rendered:\n%s", out)
	}
	if !strings.Contains(out, "person-9") {
		t.Error("expected first 10 entities to render")
	}
}

func TestRenderNote_EntitiesWithSpecialChars(t *testing.T) {
	w := &Writer{}
	note := &Note{
		Metadata: NoteMetadata{
			Created: time.Date(2026, 3, 16, 14, 23, 0, 0, time.UTC),
			Type:    "article",
			Status:  "done",
			Entities: &EntitiesBlock{
				URLs: []string{"https://example.com/a:b"},
				Orgs: []string{"Acme [Inc]"},
			},
		},
	}

	out := w.renderNote(note)
	if !strings.Contains(out, `- "https://example.com/a:b"`) {
		t.Errorf("URL containing colon must be quoted:\n%s", out)
	}
	if !strings.Contains(out, `- "Acme [Inc]"`) {
		t.Errorf("value containing brackets must be quoted:\n%s", out)
	}
}

func writeFixtureNote(t *testing.T, w *Writer, body string) string {
	t.Helper()
	note := &Note{
		Metadata: NoteMetadata{
			Created: time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC),
			Type:    "text", Status: "done",
		},
		Title: "Fixture",
		Raw:   body,
	}
	p, err := w.WriteNote(note, "fixturejob1")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSetConnections_InsertsBlockAndKeepsBody(t *testing.T) {
	w := newTestWriter(t)
	p := writeFixtureNote(t, w, "first paragraph\n\nsecond paragraph")

	if err := w.SetConnections(p, []string{"khayal/old-alice.md", "khayal/old-bob.md"}); err != nil {
		t.Fatalf("SetConnections: %v", err)
	}

	full := readNoteFile(t, w, p)
	if !strings.Contains(full, "- \"[[old-alice]]\"") || !strings.Contains(full, "- \"[[old-bob]]\"") {
		t.Errorf("wikilinks missing:\n%s", full)
	}
	if !strings.Contains(full, "second paragraph") {
		t.Error("body was mutated")
	}
}

func TestSetConnections_IdempotentAndReplace(t *testing.T) {
	w := newTestWriter(t)
	p := writeFixtureNote(t, w, "content here")

	if err := w.SetConnections(p, []string{"khayal/a.md"}); err != nil {
		t.Fatal(err)
	}
	first := readNoteFile(t, w, p)

	// Same links again → byte-identical file (no write churn).
	if err := w.SetConnections(p, []string{"khayal/a.md"}); err != nil {
		t.Fatal(err)
	}
	if readNoteFile(t, w, p) != first {
		t.Error("identical SetConnections must not change the file")
	}

	// Replace with different links → old entries gone.
	if err := w.SetConnections(p, []string{"khayal/b.md", "khayal/c.md"}); err != nil {
		t.Fatal(err)
	}
	updated := readNoteFile(t, w, p)
	if strings.Contains(updated, "[[a]]") {
		t.Errorf("stale link survived:\n%s", updated)
	}
	if !strings.Contains(updated, "[[b]]") || !strings.Contains(updated, "[[c]]") {
		t.Errorf("new links missing:\n%s", updated)
	}
}

func TestSetConnections_ClearsWhenEmpty(t *testing.T) {
	w := newTestWriter(t)
	p := writeFixtureNote(t, w, "body")

	if err := w.SetConnections(p, []string{"khayal/a.md"}); err != nil {
		t.Fatal(err)
	}
	if err := w.SetConnections(p, nil); err != nil {
		t.Fatal(err)
	}
	out := readNoteFile(t, w, p)
	if strings.Contains(out, "connections:") {
		t.Errorf("block should be removed when no targets:\n%s", out)
	}
	if !strings.Contains(out, "body") {
		t.Error("body lost")
	}
}

func TestSetConnections_MissingNoteFails(t *testing.T) {
	w := newTestWriter(t)
	if err := w.SetConnections("khayal/nope.md", []string{"khayal/a.md"}); err == nil {
		t.Fatal("expected error for missing note")
	}
}

func newTestWriter(t *testing.T) *Writer {
	t.Helper()
	cfg := &config.Config{
		Vault: config.VaultConfig{Path: t.TempDir(), InboxDir: "khayal"},
	}
	w, err := NewWriter(cfg, filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func readNoteFile(t *testing.T, w *Writer, p string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(w.BasePath(), p))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRenderNote_DateResolutionsArrow(t *testing.T) {
	w := &Writer{}
	note := &Note{
		Metadata: NoteMetadata{
			Created: time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC),
			Type:    "text", Status: "done",
			Entities: &EntitiesBlock{
				Dates:           []string{"tomorrow", "March 2024"},
				DateResolutions: []string{"2026-08-26", ""},
			},
		},
	}
	out := w.renderNote(note)
	if !strings.Contains(out, "    - tomorrow \u2192 2026-08-26\n") {
		t.Errorf("arrow rendering missing:\n%s", out)
	}
	if !strings.Contains(out, "    - March 2024\n") {
		t.Errorf("unresolved date must render plain:\n%s", out)
	}
	if strings.Contains(out, "\u2192\n") || strings.Count(out, "\u2192") != 1 {
		t.Errorf("unexpected arrow count:\n%s", out)
	}
}
