package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
)

type CaptureRequest struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type CaptureResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	NotePath  string `json:"note_path,omitempty"`
	CreatedAt string `json:"created_at"`
}

func (s *Server) captureHandler(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		s.handleImageCapture(w, r)
		return
	}

	s.handleTextCapture(w, r)
}

func (s *Server) handleTextCapture(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > int64(s.config.Server.MaxTextBodyMB)<<20 {
		WriteError(w, "request body too large", "CAPTURE_BODY_TOO_LARGE", http.StatusRequestEntityTooLarge)
		return
	}

	var req CaptureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, "invalid request body", "CAPTURE_INVALID_BODY", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		WriteError(w, "missing required field: content", "CAPTURE_MISSING_CONTENT", http.StatusBadRequest)
		return
	}

	jobType := req.Type
	if jobType == "" {
		jobType = "text"
	}

	ctx := context.Background()
	now := time.Now()

	job := &queue.Job{
		ID:        uuid.New().String(),
		Type:      jobType,
		Status:    "pending",
		Content:   req.Content,
		CreatedAt: now,
	}

	if jobType == "text" {
		note := &vault.Note{
			Metadata: vault.NoteMetadata{
				Created: now,
				Type:    "text",
				Status:  "done",
			},
			Title: extractTitle(req.Content),
			Raw:   req.Content,
		}

		notePath, err := s.vault.WriteNote(note)
		if err != nil {
			WriteError(w, "failed to write note", "VAULT_WRITE_FAILED", http.StatusInternalServerError)
			return
		}

		if err := s.queue.IndexNote(ctx, notePath, note.Title, req.Content, ""); err != nil {
			WriteError(w, "failed to index note", "INDEX_FAILED", http.StatusInternalServerError)
			return
		}

		job.Status = "done"
		job.NotePath = notePath
		job.ProcessedAt = &now
	}

	if err := s.queue.CreateJob(ctx, job); err != nil {
		WriteError(w, "failed to create job", "QUEUE_CREATE_FAILED", http.StatusInternalServerError)
		return
	}

	WriteCreated(w, CaptureResponse{
		ID:        job.ID,
		Type:      job.Type,
		Status:    job.Status,
		NotePath:  job.NotePath,
		CreatedAt: job.CreatedAt.Format(time.RFC3339),
	})
}

func extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	firstLine := strings.TrimSpace(lines[0])
	if len(firstLine) > 100 {
		firstLine = firstLine[:100]
	}
	return firstLine
}

func (s *Server) handleImageCapture(w http.ResponseWriter, r *http.Request) {
	maxSize := int64(s.config.Server.MaxImageBodyMB) << 20

	if err := r.ParseMultipartForm(maxSize); err != nil {
		WriteError(w, "invalid multipart form or file too large", "CAPTURE_INVALID_FORM", http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		WriteError(w, "missing file", "CAPTURE_MISSING_FILE", http.StatusBadRequest)
		return
	}
	defer file.Close()

	_, err = io.ReadAll(io.LimitReader(file, maxSize))
	if err != nil {
		WriteError(w, "failed to read file", "CAPTURE_READ_FAILED", http.StatusInternalServerError)
		return
	}

	_ = header.Filename

	note := r.FormValue("note")

	ctx := context.Background()
	now := time.Now()

	job := &queue.Job{
		ID:          uuid.New().String(),
		Type:        "image",
		Status:      "pending",
		UserContext: note,
		CreatedAt:   now,
	}

	if err := s.queue.CreateJob(ctx, job); err != nil {
		WriteError(w, "failed to create job", "QUEUE_CREATE_FAILED", http.StatusInternalServerError)
		return
	}

	notePath := fmt.Sprintf("inbox/%s-image.md", now.Format("2006-01-02-")+job.ID[:8])

	WriteCreated(w, CaptureResponse{
		ID:        job.ID,
		Type:      "image",
		Status:    "processing",
		NotePath:  notePath,
		CreatedAt: job.CreatedAt.Format(time.RFC3339),
	})
}

func (s *Server) parseLimit(query string, defaultVal, maxVal int) int {
	if query == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(query)
	if err != nil || val <= 0 {
		return defaultVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}
