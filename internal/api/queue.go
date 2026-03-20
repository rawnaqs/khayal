package api

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rawnaqs/khayal/internal/queue"
)

type QueueListResponse struct {
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
	Jobs   []queue.Job `json:"jobs"`
}

type QueueJobResponse struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	NotePath    string     `json:"note_path,omitempty"`
	SourceURL   string     `json:"source_url,omitempty"`
	SourceFile  string     `json:"source_file,omitempty"`
	Content     string     `json:"content,omitempty"`
	UserContext string     `json:"user_context,omitempty"`
	CreatedAt   string     `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
	Retries     int        `json:"retries"`
}

type QueueDiscardResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Message string `json:"message"`
}

func (s *Server) queueListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	status := r.URL.Query().Get("status")
	if status == "" {
		status = "all"
	}

	limit := s.parseLimit(r.URL.Query().Get("limit"), 20, 100)
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		offset = s.parseLimit(o, 0, 10000)
	}

	jobs, total, err := s.queue.ListJobs(ctx, status, limit, offset)
	if err != nil {
		s.logger.Error("queue operation failed",
			"code", "QUEUE_LIST_FAILED",
			"operation", "list",
			"status_filter", status,
			"error", err,
		)
		WriteError(w, "failed to list jobs", "QUEUE_LIST_FAILED", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, QueueListResponse{
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Jobs:   jobs,
	})
}

func (s *Server) queueGetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	jobID := chi.URLParam(r, "id")

	job, err := s.queue.GetJob(ctx, jobID)
	if err != nil {
		WriteError(w, "job not found", "QUEUE_JOB_NOT_FOUND", http.StatusNotFound)
		return
	}

	WriteJSON(w, http.StatusOK, job)
}

func (s *Server) queueRetryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	jobID := chi.URLParam(r, "id")

	job, err := s.queue.GetJob(ctx, jobID)
	if err != nil {
		WriteError(w, "job not found", "QUEUE_JOB_NOT_FOUND", http.StatusNotFound)
		return
	}

	if job.Status != "pending" && job.Status != "failed" {
		s.logger.Warn("queue operation failed",
			"code", "QUEUE_INVALID_STATE",
			"operation", "retry",
			"job_id", job.ID,
			"current_status", job.Status,
		)
		WriteError(w, "can only retry pending or failed jobs", "QUEUE_INVALID_STATE", http.StatusBadRequest)
		return
	}

	job.Status = "pending"
	job.Error = ""
	job.Retries = 0

	if err := s.queue.UpdateJob(ctx, job); err != nil {
		s.logger.Error("queue operation failed",
			"code", "QUEUE_UPDATE_FAILED",
			"operation", "retry",
			"job_id", job.ID,
			"error", err,
		)
		WriteError(w, "failed to update job", "QUEUE_UPDATE_FAILED", http.StatusInternalServerError)
		return
	}

	s.logger.Info("job retry",
		"job_id", job.ID,
	)

	WriteJSON(w, http.StatusOK, job)
}

func (s *Server) queueDiscardHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	jobID := chi.URLParam(r, "id")

	job, err := s.queue.GetJob(ctx, jobID)
	if err != nil {
		WriteError(w, "job not found", "QUEUE_JOB_NOT_FOUND", http.StatusNotFound)
		return
	}

	if job.Status == "done" {
		s.logger.Warn("queue operation failed",
			"code", "QUEUE_INVALID_STATE",
			"operation", "discard",
			"job_id", jobID,
			"current_status", job.Status,
		)
		WriteError(w, "cannot discard completed jobs", "QUEUE_INVALID_STATE", http.StatusBadRequest)
		return
	}

	if err := s.queue.DeleteJob(ctx, jobID); err != nil {
		s.logger.Error("queue operation failed",
			"code", "QUEUE_DELETE_FAILED",
			"operation", "discard",
			"job_id", jobID,
			"error", err,
		)
		WriteError(w, "failed to delete job", "QUEUE_DELETE_FAILED", http.StatusInternalServerError)
		return
	}

	s.logger.Info("job discarded",
		"job_id", jobID,
	)

	WriteJSON(w, http.StatusOK, QueueDiscardResponse{
		Success: true,
		ID:      jobID,
		Message: "job discarded",
	})
}
