package vault

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Reader struct {
	vaultPath string
	inboxPath string
}

func NewReader(vaultPath, inboxDir string) *Reader {
	return &Reader{
		vaultPath: filepath.Clean(vaultPath),
		inboxPath: filepath.Join(vaultPath, inboxDir),
	}
}

type NoteContent struct {
	// Frontmatter fields
	Created     string                 `yaml:"created"`
	Updated     string                 `yaml:"updated"`
	Type        string                 `yaml:"type"`
	Status      string                 `yaml:"status"`
	Tags        []string               `yaml:"tags"`
	SourceURL   string                 `yaml:"source_url,omitempty"`
	SourceFile  string                 `yaml:"source_file,omitempty"`
	UserContext string                 `yaml:"user_context,omitempty"`
	Entities    map[string]interface{} `yaml:"entities,omitempty"`
	Related     []string               `yaml:"related,omitempty"`

	// Sections (parsed from markdown body)
	Title      string
	Summary    string
	KeyIdeas   []string
	Raw        string
	Description string
	Source      string
}

func (r *Reader) ReadNote(notePath string) (*NoteContent, error) {
	// Clean the input path
	notePath = filepath.Clean(notePath)

	// Build full path — notePath is always relative to vault (e.g., "inbox/test.md")
	fullPath := filepath.Join(r.vaultPath, notePath)

	// Ensure the resolved path is within the inbox
	cleanFullPath := filepath.Clean(fullPath)
	cleanInboxPath := filepath.Clean(r.inboxPath)
	if !strings.HasPrefix(cleanFullPath, cleanInboxPath+string(filepath.Separator)) && cleanFullPath != cleanInboxPath {
		return nil, fmt.Errorf("path outside inbox: %s", notePath)
	}

	// Read file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read note: %w", err)
	}

	// Parse frontmatter + body
	return parseMarkdown(content)
}

func parseMarkdown(content []byte) (*NoteContent, error) {
	note := &NoteContent{}

	// Split frontmatter and body
	parts := bytes.SplitN(content, []byte("---"), 3)
	if len(parts) < 3 {
		// No frontmatter - treat entire content as raw
		note.Raw = string(content)
		// Try to extract title from first heading
		scanner := bufio.NewScanner(bytes.NewReader(content))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "# ") {
				note.Title = strings.TrimPrefix(line, "# ")
				break
			}
		}
		return note, nil
	}

	// Parse YAML frontmatter
	if err := yaml.Unmarshal(parts[1], note); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Parse markdown body sections
	body := string(parts[2])
	parseSections(note, body)

	return note, nil
}

func parseSections(note *NoteContent, body string) {
	scanner := bufio.NewScanner(strings.NewReader(body))

	var currentSection string
	var buffer strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Detect section headers (## Section Name)
		if strings.HasPrefix(line, "## ") {
			// Save previous section
			saveSection(note, currentSection, buffer.String())
			buffer.Reset()

			// Start new section
			currentSection = strings.TrimPrefix(line, "## ")
			continue
		}

		// Accumulate section content
		if currentSection != "" {
			buffer.WriteString(line)
			buffer.WriteString("\n")
		} else {
			// Content before any ## heading - could be title or intro
			trimmed := strings.TrimSpace(line)
			if note.Title == "" && strings.HasPrefix(trimmed, "# ") {
				note.Title = strings.TrimPrefix(trimmed, "# ")
			}
		}
	}

	// Save final section
	saveSection(note, currentSection, buffer.String())
}

func saveSection(note *NoteContent, name, content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}

	switch name {
	case "Summary":
		note.Summary = content
	case "Key Ideas":
		note.KeyIdeas = parseListItems(content)
	case "Raw":
		note.Raw = content
	case "Description":
		note.Description = content
	case "Source":
		note.Source = content
	}
}

func parseListItems(content string) []string {
	var items []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "- ") {
			items = append(items, strings.TrimPrefix(line, "- "))
		} else if strings.HasPrefix(line, "• ") {
			items = append(items, strings.TrimPrefix(line, "• "))
		}
	}

	return items
}
