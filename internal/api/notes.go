package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/rawnaqs/khayal/internal/vault"
)

type NoteResponse struct {
	NotePath       string   `json:"note_path"`
	Title          string   `json:"title,omitempty"`
	Type           string   `json:"type,omitempty"`
	Status         string   `json:"status,omitempty"`
	CreatedAt      string   `json:"created_at,omitempty"`
	UpdatedAt      string   `json:"updated_at,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	KeyIdeas       []string `json:"key_ideas,omitempty"`
	Raw            string   `json:"raw"`
	SourceURL      string   `json:"source_url,omitempty"`
	SourceFile     string   `json:"source_file,omitempty"`
	Description    string   `json:"description,omitempty"`
	Related        []string `json:"related,omitempty"`
	Excerpt        string   `json:"excerpt,omitempty"`
	SearchQuery    string   `json:"search_query,omitempty"`
	ExcerptSection string   `json:"excerpt_section,omitempty"`
}

func (s *Server) noteHandler(w http.ResponseWriter, r *http.Request) {
	// Get raw path parameter (may contain URL-encoded characters like %2F)
	rawPath := chi.URLParam(r, "path")

	// URL-decode the path (convert %2F back to /)
	notePath, err := url.PathUnescape(rawPath)
	if err != nil {
		WriteError(w, "invalid path encoding", "NOTE_INVALID_PATH", http.StatusBadRequest)
		return
	}

	// Read note using vault reader
	note, err := s.vaultReader.ReadNote(notePath)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "failed to read") {
			s.logger.Error("note not found",
				"path", notePath,
				"error", err,
			)
			WriteError(w, "note not found", "NOTE_NOT_FOUND", http.StatusNotFound)
			return
		}
		s.logger.Error("failed to read note",
			"path", notePath,
			"error", err,
		)
		WriteError(w, "failed to read note", "NOTE_READ_ERROR", http.StatusInternalServerError)
		return
	}

	s.logger.Info("note read", "path", notePath)

	// Build response
	resp := NoteResponse{
		NotePath:    notePath,
		Title:       note.Title,
		Type:        note.Type,
		Status:      note.Status,
		Tags:        note.Tags,
		Summary:     note.Summary,
		KeyIdeas:    note.KeyIdeas,
		Raw:         note.Raw,
		SourceURL:   note.SourceURL,
		SourceFile:  note.SourceFile,
		Description: note.Description,
		Related:     note.Related,
	}

	// Extract excerpt context if query provided
	query := r.URL.Query().Get("q")
	if query != "" {
		section, excerpt := extractExcerptContext(note, query)
		resp.ExcerptSection = section
		resp.Excerpt = excerpt
		resp.SearchQuery = query
	}

	WriteJSON(w, http.StatusOK, resp)
}

func extractExcerptContext(note *vault.NoteContent, query string) (section string, excerpt string) {
	// Search for query terms in each section
	sections := map[string]string{
		"Summary":     note.Summary,
		"Key Ideas":   strings.Join(note.KeyIdeas, " "),
		"Raw":         note.Raw,
		"Description": note.Description,
		"Source":      note.Source,
	}

	queryLower := strings.ToLower(query)

	for name, content := range sections {
		if content == "" {
			continue
		}

		contentLower := strings.ToLower(content)
		index := strings.Index(contentLower, queryLower)

		if index != -1 {
			// Found match - extract context (~200 chars)
			start := maxInt(0, index-100)
			end := minInt(len(content), index+len(query)+100)

			excerpt = content[start:end]
			if start > 0 {
				excerpt = "..." + excerpt
			}
			if end < len(content) {
				excerpt = excerpt + "..."
			}

			return name, excerpt
		}
	}

	// Fallback: return first 200 chars of Raw
	if len(note.Raw) > 200 {
		return "Raw", note.Raw[:200] + "..."
	}

	return "Raw", note.Raw
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Server) deleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	rawPath := chi.URLParam(r, "path")

	notePath, err := url.PathUnescape(rawPath)
	if err != nil {
		WriteError(w, "invalid path encoding", "NOTE_INVALID_PATH", http.StatusBadRequest)
		return
	}

	if err := s.vault.DeleteNote(notePath); err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "no such file") {
			s.logger.Error("note not found for deletion",
				"path", notePath,
				"error", err,
			)
			WriteError(w, "note not found", "NOTE_NOT_FOUND", http.StatusNotFound)
			return
		}
		if errors.Is(err, vault.ErrVaultPathOutsideInbox) ||
			errors.Is(err, vault.ErrVaultPathOutsideVault) ||
			strings.Contains(err.Error(), "path must be within") {
			s.logger.Error("invalid path for deletion",
				"path", notePath,
				"error", err,
			)
			WriteError(w, "invalid path", "NOTE_INVALID_PATH", http.StatusBadRequest)
			return
		}
		s.logger.Error("failed to delete note from vault",
			"path", notePath,
			"error", err,
		)
		WriteError(w, "failed to delete note", "NOTE_DELETE_FAILED", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.queue.DeleteFromIndex(ctx, notePath); err != nil {
		s.logger.Warn("failed to remove note from search indices after vault delete",
			"path", notePath,
			"error", err,
		)
	}

	if err := s.queue.InvalidateStatsCache(ctx); err != nil {
		s.logger.Warn("failed to invalidate stats cache after delete",
			"path", notePath,
			"error", err,
		)
	}

	s.logger.Info("note deleted", "path", notePath)
	WriteNoContent(w)
}
