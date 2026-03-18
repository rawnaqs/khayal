package vault

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rawnaqs/khayal/internal/config"
	"gopkg.in/yaml.v3"
)

const (
	MaxTags        = 20
	MaxHistory     = 50
	MaxFilenameLen = 200
)

type Writer struct {
	basePath  string
	inboxPath string
	mediaPath string
}

type NoteMetadata struct {
	Created     time.Time      `yaml:"created"`
	Updated     *time.Time     `yaml:"updated,omitempty"`
	Type        string         `yaml:"type"`
	Status      string         `yaml:"status"`
	SourceURL   string         `yaml:"source_url,omitempty"`
	SourceFile  string         `yaml:"source_file,omitempty"`
	UserContext string         `yaml:"user_context,omitempty"`
	Tags        []string       `yaml:"tags,omitempty"`
	History     []HistoryEvent `yaml:"history,omitempty"`
	Entities    *EntitiesBlock `yaml:"entities,omitempty"`
}

type HistoryEvent struct {
	At    time.Time `yaml:"at"`
	Event string    `yaml:"event"`
}

type EntitiesBlock struct {
	People  []string `yaml:"people,omitempty"`
	Amounts []string `yaml:"amounts,omitempty"`
	Dates   []string `yaml:"dates,omitempty"`
	Places  []string `yaml:"places,omitempty"`
	Orgs    []string `yaml:"orgs,omitempty"`
	URLs    []string `yaml:"urls,omitempty"`
}

type Note struct {
	Metadata NoteMetadata
	Title    string
	Summary  string
	KeyIdeas []string
	Content  string
	Raw      string
}

func NewWriter(cfg *config.Config) (*Writer, error) {
	basePath := expandPath(cfg.Vault.Path)
	inboxPath := filepath.Join(basePath, cfg.Vault.InboxDir)
	mediaPath := filepath.Join(inboxPath, "media")

	if err := os.MkdirAll(inboxPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create inbox directory: %w", err)
	}
	if err := os.MkdirAll(mediaPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create media directory: %w", err)
	}

	return &Writer{
		basePath:  basePath,
		inboxPath: inboxPath,
		mediaPath: mediaPath,
	}, nil
}

func NewWriterWithPaths(vaultPath, inboxDir string) (*Writer, error) {
	basePath := expandPath(vaultPath)
	inboxPath := filepath.Join(basePath, inboxDir)
	mediaPath := filepath.Join(inboxPath, "media")

	if err := os.MkdirAll(inboxPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create inbox directory: %w", err)
	}
	if err := os.MkdirAll(mediaPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create media directory: %w", err)
	}

	return &Writer{
		basePath:  basePath,
		inboxPath: inboxPath,
		mediaPath: mediaPath,
	}, nil
}

func (w *Writer) BasePath() string {
	return w.basePath
}

func (w *Writer) InboxPath() string {
	return w.inboxPath
}

func (w *Writer) Exists() bool {
	_, err := os.Stat(w.basePath)
	return err == nil
}

func (w *Writer) MediaPath() string {
	return w.mediaPath
}

func (w *Writer) WriteNote(note *Note) (string, error) {
	sanitizedContent := sanitizeUTF8(note.Raw)
	if !utf8.ValidString(sanitizedContent) {
		return "", fmt.Errorf("invalid UTF-8 content")
	}

	if err := w.validateNote(note); err != nil {
		return "", fmt.Errorf("note validation failed: %w", err)
	}

	now := time.Now()
	if note.Metadata.Created.IsZero() {
		note.Metadata.Created = now
	}
	note.Metadata.Updated = &now

	if note.Metadata.History == nil {
		note.Metadata.History = []HistoryEvent{}
	}
	note.Metadata.History = append(note.Metadata.History, HistoryEvent{
		At:    now,
		Event: "created",
	})

	filename := w.generateFilename(note)
	notePath := filepath.Join(w.inboxPath, filename)
	relativePath := filepath.Join(filepath.Base(w.inboxPath), filename)

	content := w.renderNote(note)

	if err := w.writeFileAtomically(notePath, content); err != nil {
		return "", fmt.Errorf("failed to write note: %w", err)
	}

	return relativePath, nil
}

func (w *Writer) UpdateNote(notePath string, note *Note) error {
	absolutePath := w.resolvePath(notePath)

	if !w.NoteExists(notePath) {
		return fmt.Errorf("note does not exist: %s", notePath)
	}

	info, err := os.Stat(absolutePath)
	if err != nil {
		return fmt.Errorf("failed to stat note: %w", err)
	}

	sanitizedContent := sanitizeUTF8(note.Raw)
	if !utf8.ValidString(sanitizedContent) {
		return fmt.Errorf("invalid UTF-8 content")
	}

	now := time.Now()
	note.Metadata.Updated = &now

	if note.Metadata.History == nil {
		note.Metadata.History = []HistoryEvent{}
	}
	note.Metadata.History = append(note.Metadata.History, HistoryEvent{
		At:    now,
		Event: "updated",
	})

	content := w.renderNote(note)

	if err := w.writeFileAtomically(absolutePath, content); err != nil {
		return fmt.Errorf("failed to update note: %w", err)
	}

	os.Chtimes(absolutePath, info.ModTime(), info.ModTime())

	return nil
}

func (w *Writer) DeleteNote(notePath string) error {
	absolutePath := w.resolvePath(notePath)

	trashPath := filepath.Join(w.basePath, ".khayal-trash")
	if err := os.MkdirAll(trashPath, 0755); err != nil {
		return fmt.Errorf("failed to create trash directory: %w", err)
	}

	filename := filepath.Base(absolutePath)
	destPath := filepath.Join(trashPath, fmt.Sprintf("%s.%d", filename, time.Now().Unix()))

	if err := os.Rename(absolutePath, destPath); err != nil {
		return fmt.Errorf("failed to move note to trash: %w", err)
	}

	return nil
}

func (w *Writer) CopyMediaFile(srcPath string) (string, error) {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	ext := filepath.Ext(srcPath)
	if ext == "" {
		ext = ".bin"
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	destPath := filepath.Join(w.mediaPath, filename)

	destFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := destFile.ReadFrom(srcFile); err != nil {
		os.Remove(destPath)
		return "", fmt.Errorf("failed to copy media file: %w", err)
	}

	relativePath := filepath.Join(filepath.Base(w.inboxPath), "media", filename)
	return relativePath, nil
}

func (w *Writer) ResolvePath(relative string) string {
	return w.resolvePath(relative)
}

func (w *Writer) NoteExists(notePath string) bool {
	absolutePath := w.resolvePath(notePath)
	_, err := os.Stat(absolutePath)
	return err == nil
}

func (w *Writer) resolvePath(relative string) string {
	if filepath.IsAbs(relative) {
		return relative
	}
	if strings.HasPrefix(relative, w.inboxPath) {
		return relative
	}
	if strings.HasPrefix(relative, w.basePath) {
		return relative
	}
	return filepath.Join(w.basePath, relative)
}

func (w *Writer) generateFilename(note *Note) string {
	date := note.Metadata.Created.Format("2006-01-02")

	title := cleanFilename(note.Title)
	if title == "" {
		title = "note"
	}

	title = strings.ReplaceAll(title, "/", "-")
	title = strings.ReplaceAll(title, "\\", "-")

	if len(title) > 50 {
		title = title[:50]
	}

	filename := fmt.Sprintf("%s-%s.md", date, title)

	if len(filename) > MaxFilenameLen {
		filename = filename[:MaxFilenameLen-3] + "..."
	}

	filename = sanitizeFilename(filename)

	return filename
}

func (w *Writer) renderNote(note *Note) string {
	var buf bytes.Buffer

	buf.WriteString("---\n")
	buf.WriteString(fmt.Sprintf("created: %s\n", note.Metadata.Created.Format(time.RFC3339)))

	if note.Metadata.Updated != nil {
		buf.WriteString(fmt.Sprintf("updated: %s\n", note.Metadata.Updated.Format(time.RFC3339)))
	}

	buf.WriteString(fmt.Sprintf("type: %s\n", note.Metadata.Type))
	buf.WriteString(fmt.Sprintf("status: %s\n", note.Metadata.Status))

	if note.Metadata.SourceURL != "" {
		buf.WriteString(fmt.Sprintf("source_url: %s\n", note.Metadata.SourceURL))
	}
	if note.Metadata.SourceFile != "" {
		buf.WriteString(fmt.Sprintf("source_file: %s\n", note.Metadata.SourceFile))
	}
	if note.Metadata.UserContext != "" {
		buf.WriteString(fmt.Sprintf("user_context: %s\n", note.Metadata.UserContext))
	}

	if len(note.Metadata.Tags) > 0 {
		buf.WriteString("tags:\n")
		for _, tag := range note.Metadata.Tags[:min(len(note.Metadata.Tags), MaxTags)] {
			buf.WriteString(fmt.Sprintf("  - %s\n", tag))
		}
	}

	if len(note.Metadata.History) > 0 {
		buf.WriteString("history:\n")
		for _, h := range note.Metadata.History[:min(len(note.Metadata.History), MaxHistory)] {
			buf.WriteString(fmt.Sprintf("  - at: %s\n", h.At.Format(time.RFC3339)))
			buf.WriteString(fmt.Sprintf("    event: %s\n", h.Event))
		}
	}

	buf.WriteString("---\n\n")

	if note.Title != "" {
		buf.WriteString("# " + note.Title + "\n\n")
	}

	if note.Summary != "" {
		buf.WriteString("## Summary\n")
		buf.WriteString(note.Summary + "\n\n")
	}

	if len(note.KeyIdeas) > 0 {
		buf.WriteString("## Key Ideas\n")
		for _, idea := range note.KeyIdeas {
			buf.WriteString("- " + idea + "\n")
		}
		buf.WriteString("\n")
	}

	if note.Content != "" {
		buf.WriteString(note.Content + "\n\n")
	}

	if note.Raw != "" {
		buf.WriteString("## Raw\n")
		buf.WriteString(note.Raw + "\n")
	}

	return buf.String()
}

func (w *Writer) writeFileAtomically(path, content string) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, "khayal-*.md")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	defer func() {
		os.Remove(tmpPath)
	}()

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0644); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

func (w *Writer) validateNote(note *Note) error {
	if note.Title != "" && len(note.Title) > 500 {
		return fmt.Errorf("title exceeds maximum length of 500 characters")
	}

	if len(note.Metadata.Tags) > MaxTags {
		return fmt.Errorf("exceeds maximum tag count of %d", MaxTags)
	}

	for _, tag := range note.Metadata.Tags {
		if strings.HasPrefix(tag, "#") {
			return fmt.Errorf("tags should not start with #")
		}
	}

	wikilinkPattern := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	matches := wikilinkPattern.FindAllString(note.Raw, -1)
	for _, match := range matches {
		linkTarget := matches[0][2 : len(matches[0])-2]
		if !w.NoteExists(linkTarget) && !w.NoteExists(linkTarget+".md") {
			return fmt.Errorf("broken wikilink: %s", match)
		}
	}

	return nil
}

func ParseFrontmatter(content string) (*NoteMetadata, error) {
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("no frontmatter found")
	}

	endIdx := strings.Index(content[3:], "---")
	if endIdx == -1 {
		return nil, fmt.Errorf("frontmatter not closed")
	}

	frontmatter := content[3 : endIdx+3]

	var meta NoteMetadata
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	return &meta, nil
}

func cleanFilename(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	reg := regexp.MustCompile(`[^a-z0-9\s\-]`)
	s = reg.ReplaceAllString(s, "")

	s = strings.Join(strings.Fields(s), "-")

	return s
}

func sanitizeFilename(filename string) string {
	reg := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	return reg.ReplaceAllString(filename, "_")
}

func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	var buf bytes.Buffer
	for _, r := range s {
		if r == 0xFFFD {
			continue
		}
		buf.WriteRune(r)
	}

	return buf.String()
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return os.ExpandEnv(path)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var wikilinkRegex = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
