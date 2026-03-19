package api

import (
	"bytes"
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

	if jobType == "url" {
		jobType = "article"
	}

	ctx := context.Background()
	now := time.Now()

	job := &queue.Job{
		ID:        uuid.New().String(),
		Type:      jobType,
		Status:    "pending",
		CreatedAt: now,
	}

	if jobType == "article" && req.Type == "url" {
		job.SourceURL = req.Content
		job.Content = ""
	} else {
		job.Content = req.Content
	}

	if err := s.queue.CreateJob(ctx, job); err != nil {
		WriteError(w, "failed to create job", "QUEUE_CREATE_FAILED", http.StatusInternalServerError)
		return
	}

	WriteCreated(w, CaptureResponse{
		ID:        job.ID,
		Type:      job.Type,
		Status:    job.Status,
		NotePath:  "",
		CreatedAt: job.CreatedAt.Format(time.RFC3339),
	})
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

	limitedReader := io.LimitReader(file, maxSize)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		WriteError(w, "failed to read file", "CAPTURE_READ_FAILED", http.StatusInternalServerError)
		return
	}

	mediaPath, err := s.vault.CopyMediaFromReader(bytes.NewReader(content), header.Filename)
	if err != nil {
		WriteError(w, "failed to save media", "VAULT_MEDIA_FAILED", http.StatusInternalServerError)
		return
	}

	note := r.FormValue("note")

	ctx := context.Background()
	now := time.Now()

	job := &queue.Job{
		ID:          uuid.New().String(),
		Type:        "image",
		Status:      "pending",
		SourceFile:  mediaPath,
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
		Status:    job.Status,
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
