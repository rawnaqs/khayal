package worker

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/ingest"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
)

type Worker struct {
	queue   *queue.Queue
	vault   *vault.Writer
	llm     llm.LLMExt
	config  config.WorkerConfig
	jobs    chan string
	wg      sync.WaitGroup
	running atomic.Bool
	logger  *slog.Logger
}

func NewWorker(cfg config.WorkerConfig, q *queue.Queue, v *vault.Writer, l llm.LLMExt, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		queue:  q,
		vault:  v,
		llm:    l,
		config: cfg,
		jobs:   make(chan string, 100),
		logger: logger,
	}
}

func (w *Worker) Start() {
	if w.running.Swap(true) {
		w.logger.Warn("worker already running")
		return
	}

	ctx := context.Background()
	if err := w.queue.ResetStuckJobs(ctx); err != nil {
		w.logger.Error("failed to reset stuck jobs", "error", err)
	}

	for i := 0; i < w.config.MaxWorkers; i++ {
		w.wg.Add(1)
		go w.workerLoop(i)
	}

	go w.jobFetcher()

	w.logger.Info("worker pool started", "workers", w.config.MaxWorkers)
}

func (w *Worker) Stop() {
	if !w.running.Swap(false) {
		return
	}

	w.logger.Info("stopping worker pool...")
	close(w.jobs)
	w.wg.Wait()
	w.logger.Info("worker pool stopped")
}

func (w *Worker) jobFetcher() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for w.running.Load() {
		select {
		case <-ticker.C:
			ctx := context.Background()
			jobs, err := w.queue.GetPendingJobs(ctx, w.config.MaxWorkers)
			if err != nil {
				w.logger.Error("failed to fetch pending jobs", "error", err)
				continue
			}
			for _, job := range jobs {
				select {
				case w.jobs <- job.ID:
				default:
					w.logger.Warn("job channel full, skipping", "job_id", job.ID)
				}
			}
		}
	}
}

func (w *Worker) workerLoop(id int) {
	defer w.wg.Done()

	w.logger.Debug("worker started", "worker_id", id)

	for jobID := range w.jobs {
		if !w.running.Load() {
			break
		}
		w.processJob(jobID)
	}

	w.logger.Debug("worker stopped", "worker_id", id)
}

func (w *Worker) processJob(jobID string) {
	ctx := context.Background()

	job, err := w.queue.GetJob(ctx, jobID)
	if err != nil {
		w.logger.Error("failed to get job", "job_id", jobID, "error", err)
		return
	}

	if err := w.queue.UpdateJobStatus(ctx, jobID, "processing"); err != nil {
		w.logger.Error("failed to update job status", "job_id", jobID, "error", err)
		return
	}

	var notePath string
	var processErr error

	switch job.Type {
	case "text":
		notePath, processErr = ingest.IngestText(ctx, job, w.vault, w.queue, w.llm)
	case "image":
		notePath, processErr = ingest.IngestImage(ctx, job, w.vault, w.queue, w.llm)
	case "article":
		notePath, processErr = ingest.IngestArticle(ctx, job, w.vault, w.queue, w.llm)
	default:
		processErr = fmt.Errorf("unknown job type: %s", job.Type)
	}

	if processErr != nil {
		w.handleFailure(job, processErr)
		return
	}

	now := time.Now()
	job.NotePath = notePath
	job.Status = "done"
	job.ProcessedAt = &now
	job.Error = ""

	if err := w.queue.UpdateJob(ctx, job); err != nil {
		w.logger.Error("failed to update job", "job_id", jobID, "error", err)
		return
	}

	w.logger.Info("job completed", "job_id", jobID, "type", job.Type, "note_path", notePath)
}

func (w *Worker) handleFailure(job *queue.Job, processErr error) {
	ctx := context.Background()

	job.Retries++
	job.Error = processErr.Error()

	if job.Retries >= w.config.MaxRetries {
		job.Status = "failed"
		w.logger.Error("job permanently failed",
			"job_id", job.ID,
			"type", job.Type,
			"retries", job.Retries,
			"error", processErr,
		)
	} else {
		job.Status = "pending"
		delay := w.calculateBackoff(job.Retries)

		w.logger.Warn("job failed, will retry",
			"job_id", job.ID,
			"retry", job.Retries,
			"max_retries", w.config.MaxRetries,
			"delay", delay,
			"error", processErr,
		)

		time.Sleep(delay)
	}

	if err := w.queue.UpdateJob(ctx, job); err != nil {
		w.logger.Error("failed to update job after failure", "job_id", job.ID, "error", err)
	}
}

func (w *Worker) calculateBackoff(retry int) time.Duration {
	switch w.config.RetryBackoff {
	case "immediate":
		return 0
	case "fixed":
		return 5 * time.Second
	case "exponential":
		fallthrough
	default:
		return time.Duration(math.Pow(2, float64(retry))) * time.Second
	}
}
