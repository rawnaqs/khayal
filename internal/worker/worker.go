package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rawnaqs/khayal/internal/chunk"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/connections"
	"github.com/rawnaqs/khayal/internal/constants"
	"github.com/rawnaqs/khayal/internal/ingest"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/rawnaqs/khayal/internal/vault"
)

type Worker struct {
	queue     *queue.Queue
	vault     *vault.Writer
	llm       llm.LLMExt
	config    config.WorkerConfig
	chunkOpts chunk.Options
	connCfg   config.ConnectionsConfig
	jobs      chan string
	wg        sync.WaitGroup
	running   atomic.Bool
	logger    *slog.Logger
}

func NewWorker(cfg config.WorkerConfig, chunkOpts chunk.Options, connCfg config.ConnectionsConfig, q *queue.Queue, v *vault.Writer, l llm.LLMExt, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		queue:     q,
		vault:     v,
		llm:       l,
		config:    cfg,
		chunkOpts: chunkOpts,
		connCfg:   connCfg,
		jobs:      make(chan string, 1000),
		logger:    logger,
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
	ticker := time.NewTicker(constants.WorkerTickerInterval)
	defer ticker.Stop()

	for w.running.Load() {
		<-ticker.C

		ctx := context.Background()

		// Calculate how many jobs we can fetch based on channel capacity
		available := cap(w.jobs) - len(w.jobs)
		if available <= 0 {
			continue
		}

		// Fetch and lock jobs atomically - prevents duplicate processing
		jobs, err := w.queue.FetchAndLockPendingJobs(ctx, available)
		if err != nil {
			w.logger.Error("failed to fetch pending jobs", "error", err)
			continue
		}
		for _, job := range jobs {
			w.jobs <- job.ID // Safe - we only fetched what fits
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
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	start := time.Now()

	job, err := w.queue.GetJob(ctx, jobID)
	if err != nil {
		w.logger.Error("failed to get job", "job_id", jobID, "error", err)
		return
	}

	w.logger.Info("job started",
		"type", job.Type,
		"job_id", job.ID,
	)

	if err := w.queue.UpdateJobStatus(ctx, jobID, "processing"); err != nil {
		w.logger.Error("failed to update job status", "job_id", jobID, "error", err)
		return
	}

	var notePath string
	var processErr error

	switch job.Type {
	case "text":
		notePath, processErr = ingest.IngestText(ctx, job, w.vault, w.queue, w.llm, w.chunkOpts)
	case "image":
		notePath, processErr = ingest.IngestImage(ctx, job, w.vault, w.queue, w.llm, w.chunkOpts)
	case "article":
		notePath, processErr = ingest.IngestArticle(ctx, job, w.vault, w.queue, w.llm, w.chunkOpts)
	case "connections":
		processErr = w.processConnections(ctx, job)
	default:
		processErr = fmt.Errorf("unknown job type: %s", job.Type)
	}

	if processErr == nil && job.Type != "connections" {
		w.chainConnections(ctx, jobID, notePath)
	}

	if processErr != nil {
		w.handleFailure(job, processErr)
		return
	}

	now := time.Now().UTC()
	job.NotePath = notePath
	job.Status = "done"
	job.ProcessedAt = &now
	job.Error = ""

	if err := w.queue.UpdateJob(ctx, job); err != nil {
		w.logger.Error("failed to update job", "job_id", jobID, "error", err)
		return
	}

	// Recompute stats cache after successful capture
	if _, err := w.queue.RecomputeStats(ctx); err != nil {
		w.logger.Warn("stats recompute failed", "error", err)
	}

	w.logger.Info("job processed",
		"type", job.Type,
		"job_id", job.ID,
		"note_path", notePath,
		"duration_ms", time.Since(start).Milliseconds(),
	)
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
		return constants.WorkerRetryBackoff
	case "exponential":
		fallthrough
	default:
		return time.Duration(math.Pow(2, float64(retry))) * time.Second
	}
}

// chainConnections enqueues a connections job for a successfully ingested
// note and records its id on the ingest job so polling clients can find
// the results. Chaining failures are logged, never fatal — connections are
// enrichment.
func (w *Worker) chainConnections(ctx context.Context, ingestJobID, notePath string) {
	if !config.IsOn(w.connCfg.Enabled) || notePath == "" {
		return
	}
	connJob := &queue.Job{
		Type:     "connections",
		Status:   "pending",
		NotePath: notePath,
	}
	if err := w.queue.CreateJob(ctx, connJob); err != nil {
		w.logger.Warn("failed to enqueue connections job",
			"job_id", ingestJobID, "error", err)
		return
	}
	if err := w.queue.LinkConnectionsJob(ctx, ingestJobID, connJob.ID); err != nil {
		w.logger.Warn("failed to link connections job",
			"job_id", ingestJobID, "connections_job_id", connJob.ID, "error", err)
	}
	w.logger.Info("connections job queued",
		"job_id", ingestJobID,
		"connections_job_id", connJob.ID,
		"note_path", notePath,
	)
}

// processConnections runs the connection engine for one note, writes the
// ranked results onto the job row, and mirrors verified connections into
// the note's frontmatter as Obsidian wikilinks. Only targets that exist on
// disk become links (vault safety: never write broken wikilinks).
func (w *Worker) processConnections(ctx context.Context, job *queue.Job) error {
	conns, err := connections.Find(ctx, w.queue, job.NotePath, w.connCfg)
	if err != nil {
		return fmt.Errorf("connection engine failed: %w", err)
	}
	payload, err := json.Marshal(map[string]any{"connections": conns})
	if err != nil {
		return fmt.Errorf("marshal connections: %w", err)
	}
	if err := w.queue.UpdateJobResult(ctx, job.ID, payload); err != nil {
		return fmt.Errorf("store connections result: %w", err)
	}

	links := make([]string, 0, len(conns))
	for _, c := range conns {
		if w.vault.NoteExists(c.NotePath) {
			links = append(links, c.NotePath)
		} else {
			w.logger.Warn("connection target missing on disk, skipping link",
				"note_path", job.NotePath, "target", c.NotePath)
		}
	}
	if len(links) == 0 && len(conns) == 0 {
		return nil // nothing found; also clears any stale block
	}
	return w.vault.SetConnections(job.NotePath, links)
}
