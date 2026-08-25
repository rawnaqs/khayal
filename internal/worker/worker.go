package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rawnaqs/khayal/internal/chunk"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/connections"
	"github.com/rawnaqs/khayal/internal/constants"
	"github.com/rawnaqs/khayal/internal/ingest"
	"github.com/rawnaqs/khayal/internal/llm"
	"github.com/rawnaqs/khayal/internal/memory"
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
	memCfg    config.MemoryConfig
	jobs      chan string
	wg        sync.WaitGroup
	running   atomic.Bool
	logger    *slog.Logger
}

func NewWorker(cfg config.WorkerConfig, chunkOpts chunk.Options, connCfg config.ConnectionsConfig, memCfg config.MemoryConfig, q *queue.Queue, v *vault.Writer, l llm.LLMExt, logger *slog.Logger) *Worker {
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
		memCfg:    memCfg,
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
		notePath, processErr = ingest.IngestText(ctx, job, w.vault, w.queue, w.llm, w.chunkOpts, w.memCfg)
	case "image":
		notePath, processErr = ingest.IngestImage(ctx, job, w.vault, w.queue, w.llm, w.chunkOpts, w.memCfg)
	case "article":
		notePath, processErr = ingest.IngestArticle(ctx, job, w.vault, w.queue, w.llm, w.chunkOpts, w.memCfg)
	case "connections":
		processErr = w.processConnections(ctx, job)
	case "memory":
		processErr = w.processMemory(ctx, job)
	default:
		processErr = fmt.Errorf("unknown job type: %s", job.Type)
	}

	if processErr == nil && job.Type != "connections" && job.Type != "memory" {
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

	w.chainMemoryConsolidation(ingestJobID)
}

// chainMemoryConsolidation enqueues a memory job only when the throttle
// window has elapsed or enough new persons accumulated since last run.
func (w *Worker) chainMemoryConsolidation(ingestJobID string) {
	if !config.IsOn(w.memCfg.Enabled) {
		return
	}
	ctx := context.Background()
	lastRunStr, ok, _ := w.queue.GetStat(ctx, "memory_last_consolidation")
	var lastRun time.Time
	if ok {
		if t, err := time.Parse(time.RFC3339, lastRunStr); err == nil {
			lastRun = t
		}
	}
	personsSince := 0
	if !lastRun.IsZero() {
		personsSince, _ = w.queue.CountPersonsSince(ctx, lastRun)
	}

	due := lastRun.IsZero() ||
		time.Since(lastRun) >= w.memCfg.ConsolidationInterval() ||
		personsSince >= w.memCfg.PersonsThreshold()
	if !due {
		return
	}

	memJob := &queue.Job{Type: "memory", Status: "pending"}
	if err := w.queue.CreateJob(ctx, memJob); err != nil {
		w.logger.Warn("failed to enqueue memory job", "error", err)
		return
	}
	if err := w.queue.LinkConnectionsJob(ctx, ingestJobID, memJob.ID); err == nil {
		_ = w.queue.LinkConnectionsJob(ctx, memJob.ID, ingestJobID)
	}
	w.logger.Info("memory consolidation queued",
		"job_id", memJob.ID, "persons_since_last", personsSince)
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
		base := filepath.Base(c.NotePath)
		if strings.EqualFold(base, strings.TrimSpace(w.memCfg.File)) {
			continue // never link the managed memory file
		}
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

// memorySkeleton is the seeded structure of the LLM-maintained memory file.
// MemorySkeleton is exported for startup seeding.
const MemorySkeleton = `# Memory

## About the author

## People

## Ongoing threads

## Preferences
`

// filenameOrDefault resolves the managed memory filename.
func filenameOrDefault(f string) string {
	if f == "" {
		return "memory.md"
	}
	return f
}

// processMemory consolidates the LLM-maintained memory file: current file +
// recent facts go to the LLM for a merge-style rewrite. Failures fail the
// job (standard retry); the previous memory file survives untouched until a
// rewrite succeeds.
func (w *Worker) processMemory(ctx context.Context, job *queue.Job) error {
	current := w.vault.ReadManagedFile(w.memCfg.File)
	if strings.TrimSpace(current) == "" {
		_ = w.vault.WriteManagedFile(filenameOrDefault(w.memCfg.File), MemorySkeleton)
		current = w.vault.ReadManagedFile(w.memCfg.File)
	}

	recent, _ := w.recentNoteDigests(ctx, 10)
	personDelta, _ := w.queue.CountPersonsSince(ctx, time.Now().Add(-w.memCfg.ConsolidationInterval()))

	prompt := fmt.Sprintf("CURRENT MEMORY FILE:\n%s\n\nRECENT CAPTURED FACTS:\n%s\n\nNEW PEOPLE SINCE LAST RUN: %d",
		current, strings.Join(recent, "\n"), personDelta)

	systemPrompt := constants.DefaultSystemPrompts.ConsolidateMemory
	merged, err := w.llm.GenerateWithSystem(systemPrompt, prompt)
	if err != nil {
		return fmt.Errorf("memory consolidation failed: %w", err)
	}
	sanitized, err := memory.SanitizeConsolidatedOutput(merged)
	if err != nil {
		return fmt.Errorf("invalid memory consolidation output: %w", err)
	}
	merged = sanitized

	filename := filenameOrDefault(w.memCfg.File)
	if err := w.vault.WriteManagedFile(filename, merged); err != nil {
		return fmt.Errorf("write memory file: %w", err)
	}

	if err := w.queue.SetStat(ctx, "memory_last_consolidation", time.Now().UTC().Format(time.RFC3339)); err != nil {
		w.logger.Warn("failed to record consolidation marker", "error", err)
	}
	return nil
}

// recentNoteDigests returns short digests of the newest notes' content.
func (w *Worker) recentNoteDigests(ctx context.Context, limit int) ([]string, error) {
	jobs, _, err := w.queue.ListJobs(ctx, "done", limit, 0)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, j := range jobs {
		if j.NotePath == "" || j.Content == "" {
			continue
		}
		digest := j.Content
		if len(digest) > 200 {
			digest = digest[:200]
		}
		out = append(out, fmt.Sprintf("- %s", digest))
	}
	return out, nil
}
