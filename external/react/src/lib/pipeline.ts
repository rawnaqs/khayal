import type { QueueJob } from '@/lib/api'

export interface PipelineStep {
  label: string
  detail?: string
  state: 'done' | 'active' | 'future'
}

export interface Pipeline {
  jobId: string
  type: string
  title: string
  createdAt: string
  steps: PipelineStep[]
}

const INGEST_TYPES = new Set(['text', 'image', 'article', 'pdf'])
const ACTIVE_STATUSES = new Set(['pending', 'queued', 'processing'])

// Per-type stage labels. "queued" is the wait; the middle stage is the
// server's single processing phase (LLM enrichment dominates it); the
// final stage is the chained connections pass that used to be invisible.
export // Short labels for the stepper; long explanations live in `detail`
// (tooltip) — cramped progress rows must never wrap or truncate.
const STAGE_LABELS: Record<string, { label: string; detail?: string }[]> = {
  text: [
    { label: 'queued' },
    { label: 'enriching', detail: 'tags · summary · key ideas via local LLM' },
    { label: 'embedding', detail: 'chunk-level semantic index' },
    { label: 'connecting', detail: 'matching against older notes' },
  ],
  image: [
    { label: 'queued' },
    { label: 'describing', detail: 'vision model + enrichment' },
    { label: 'embedding', detail: 'chunk-level semantic index' },
    { label: 'connecting', detail: 'matching against older notes' },
  ],
  article: [
    { label: 'queued' },
    { label: 'fetching', detail: 'article extraction + enrichment' },
    { label: 'embedding', detail: 'chunk-level semantic index' },
    { label: 'connecting', detail: 'matching against older notes' },
  ],
  pdf: [
    { label: 'queued' },
    { label: 'enriching', detail: 'tags · summary · key ideas via local LLM' },
    { label: 'embedding', detail: 'chunk-level semantic index' },
    { label: 'connecting', detail: 'matching against older notes' },
  ],
}

function titleOf(job: QueueJob): string {
  if (job.note_path) return job.note_path
  const content = (job as { content?: string }).content
  if (content) return content.slice(0, 60)
  return job.type
}

/**
 * buildPipeline computes the end-to-end capture pipeline from the job
 * list (WS-patched): ingest job phase + the chained connections pass.
 * Returns null when nothing is actively processing — including the
 * previously-invisible "connecting" stage after the note is saved.
 */
export function buildPipeline(jobs: QueueJob[]): Pipeline | null {
  const byId = new Map(jobs.map((j) => [j.id, j]))

  // most recent ingest job that is still in flight, or just finished with
  // its connections pass still running
  const candidates = jobs
    .filter((j) => INGEST_TYPES.has(j.type))
    .filter((j) => ACTIVE_STATUSES.has(j.status) || j.status === 'done')
    .sort((a, b) => (a.created_at < b.created_at ? 1 : -1))

  for (const ingest of candidates) {
    const steps = STAGE_LABELS[ingest.type] || STAGE_LABELS.text
    const total = steps.length

    if (ACTIVE_STATUSES.has(ingest.status)) {
      // in-flight ingest: queued -> step 0, processing -> the work step
      const activeIndex = ingest.status === 'processing' ? 1 : 0
      return {
        jobId: ingest.id,
        type: ingest.type,
        title: titleOf(ingest),
        createdAt: ingest.created_at,
        steps: steps.map((st, i) => ({
          label: st.label,
          detail: st.detail,
          state: i < activeIndex ? 'done' : i === activeIndex ? 'active' : 'future',
        })),
      }
    }

    // ingest done: pipeline continues only if its connections job is active
    if (!ingest.connections_job_id) continue
    const conn = byId.get(ingest.connections_job_id)
    if (!conn || !ACTIVE_STATUSES.has(conn.status)) continue

    return {
      jobId: ingest.id,
      type: ingest.type,
      title: titleOf(ingest),
      createdAt: ingest.created_at,
      steps: steps.map((st, i) => ({
        label: st.label,
        detail: st.detail,
        state: i === total - 1 ? 'active' : 'done',
      })),
    }
  }

  return null
}
