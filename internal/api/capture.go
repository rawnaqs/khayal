package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rawnaqs/khayal/internal/ingest"
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
		s.handleFileCapture(w, r)
		return
	}

	s.handleTextCapture(w, r)
}

func (s *Server) handleTextCapture(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > int64(s.config.Server.MaxTextBodyMB)<<20 {
		s.logger.Error("capture failed",
			"code", "CAPTURE_BODY_TOO_LARGE",
			"type", "text",
		)
		WriteError(w, "request body too large", "CAPTURE_BODY_TOO_LARGE", http.StatusRequestEntityTooLarge)
		return
	}

	var req CaptureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Error("capture failed",
			"code", "CAPTURE_INVALID_BODY",
			"type", "text",
			"error", err,
		)
		WriteError(w, "invalid request body", "CAPTURE_INVALID_BODY", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		s.logger.Warn("capture failed",
			"code", "CAPTURE_MISSING_CONTENT",
			"type", "text",
		)
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
	now := time.Now().UTC()

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
		s.logger.Error("capture failed",
			"code", "QUEUE_CREATE_FAILED",
			"type", "text",
			"job_id", job.ID,
			"error", err,
		)
		WriteError(w, "failed to create job", "QUEUE_CREATE_FAILED", http.StatusInternalServerError)
		return
	}

	s.logger.Info("capture",
		"type", "text",
		"job_id", job.ID,
	)

	WriteCreated(w, CaptureResponse{
		ID:        job.ID,
		Type:      job.Type,
		Status:    job.Status,
		NotePath:  "",
		CreatedAt: job.CreatedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleFileCapture(w http.ResponseWriter, r *http.Request) {
	// PDFs go down their own pipeline (text extraction at capture time);
	// everything else stays on the image path. The uploaded filename
	// lives in the multipart body, so parse the form before routing.
	maxSize := int64(s.config.Server.MaxImageBodyMB) << 20
	if err := r.ParseMultipartForm(maxSize); err != nil {
		WriteError(w, "invalid multipart form or file too large", "CAPTURE_INVALID_FORM", http.StatusRequestEntityTooLarge)
		return
	}
	_, header, err := r.FormFile("file")
	if err != nil {
		WriteError(w, "missing file", "CAPTURE_MISSING_FILE", http.StatusBadRequest)
		return
	}
	if strings.EqualFold(filepath.Ext(header.Filename), ".pdf") {
		s.handlePDFCapture(w, r)
		return
	}
	s.handleImageCapture(w, r)
}

// handlePDFCapture stores the PDF in the media dir, extracts its text
// layer, and enqueues a pdf job that rides the normal enrichment
// pipeline (tags, summary, entities, chunks).
func (s *Server) handlePDFCapture(w http.ResponseWriter, r *http.Request) {
	maxSize := int64(s.config.Server.MaxImageBodyMB) << 20

	file, header, err := r.FormFile("file")
	if err != nil {
		WriteError(w, "missing file", "CAPTURE_MISSING_FILE", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !strings.EqualFold(filepath.Ext(header.Filename), ".pdf") {
		WriteError(w, "only pdf files are accepted here", "CAPTURE_NOT_PDF", http.StatusBadRequest)
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, maxSize))
	if err != nil {
		WriteError(w, "failed to read file", "CAPTURE_READ_FAILED", http.StatusInternalServerError)
		return
	}

	text, err := ingest.ExtractPDFText(data)
	if err != nil {
		s.logger.Warn("pdf capture failed",
			"code", "PDF_EXTRACT_FAILED",
			"error", err,
		)
		WriteError(w, "no extractable text (scanned pdf?)", "PDF_EXTRACT_FAILED", http.StatusBadRequest)
		return
	}

	mediaPath, err := s.vault.CopyMediaFromReader(bytes.NewReader(data), header.Filename)
	if err != nil {
		s.logger.Error("pdf capture failed",
			"code", "VAULT_MEDIA_FAILED",
			"error", err,
		)
		WriteError(w, "failed to save media", "VAULT_MEDIA_FAILED", http.StatusInternalServerError)
		return
	}

	note := r.FormValue("note")
	ctx := context.Background()
	now := time.Now().UTC()

	job := &queue.Job{
		ID:          uuid.New().String(),
		Type:        "pdf",
		Status:      "pending",
		SourceFile:  mediaPath,
		Content:     text,
		UserContext: note,
		CreatedAt:   now,
	}
	if err := s.queue.CreateJob(ctx, job); err != nil {
		s.logger.Error("pdf capture failed",
			"code", "QUEUE_CREATE_FAILED",
			"job_id", job.ID,
			"error", err,
		)
		WriteError(w, "failed to create job", "QUEUE_CREATE_FAILED", http.StatusInternalServerError)
		return
	}

	s.logger.Info("capture",
		"type", "pdf",
		"job_id", job.ID,
		"chars", len(text),
	)

	notePath := fmt.Sprintf("%s/%s-pdf.md", s.config.Vault.InboxDir, now.Format("2006-01-02-")+job.ID[:8])
	WriteCreated(w, CaptureResponse{
		ID:        job.ID,
		Type:      "pdf",
		Status:    job.Status,
		NotePath:  notePath,
		CreatedAt: job.CreatedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleImageCapture(w http.ResponseWriter, r *http.Request) {
	maxSize := int64(s.config.Server.MaxImageBodyMB) << 20

	if err := r.ParseMultipartForm(maxSize); err != nil {
		s.logger.Error("capture failed",
			"code", "CAPTURE_INVALID_FORM",
			"type", "image",
			"error", err,
		)
		WriteError(w, "invalid multipart form or file too large", "CAPTURE_INVALID_FORM", http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.logger.Warn("capture failed",
			"code", "CAPTURE_MISSING_FILE",
			"type", "image",
		)
		WriteError(w, "missing file", "CAPTURE_MISSING_FILE", http.StatusBadRequest)
		return
	}
	defer file.Close()

	limitedReader := io.LimitReader(file, maxSize)
	mediaPath, err := s.vault.CopyMediaFromReader(limitedReader, header.Filename)
	if err != nil {
		s.logger.Error("capture failed",
			"code", "VAULT_MEDIA_FAILED",
			"type", "image",
			"error", err,
		)
		WriteError(w, "failed to save media", "VAULT_MEDIA_FAILED", http.StatusInternalServerError)
		return
	}

	note := r.FormValue("note")

	ctx := context.Background()
	now := time.Now().UTC()

	job := &queue.Job{
		ID:          uuid.New().String(),
		Type:        "image",
		Status:      "pending",
		SourceFile:  mediaPath,
		UserContext: note,
		CreatedAt:   now,
	}

	if err := s.queue.CreateJob(ctx, job); err != nil {
		s.logger.Error("capture failed",
			"code", "QUEUE_CREATE_FAILED",
			"type", "image",
			"job_id", job.ID,
			"error", err,
		)
		WriteError(w, "failed to create job", "QUEUE_CREATE_FAILED", http.StatusInternalServerError)
		return
	}

	s.logger.Info("capture",
		"type", "image",
		"job_id", job.ID,
	)

	notePath := fmt.Sprintf("%s/%s-image.md", s.config.Vault.InboxDir, now.Format("2006-01-02-")+job.ID[:8])

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
