import { describe, it, expect } from 'vitest'
import { buildPipeline, STAGE_LABELS } from '../pipeline'
import type { QueueJob } from '@/lib/api'

function job(over: Partial<QueueJob>): QueueJob {
  return {
    id: 'j1',
    type: 'text',
    status: 'pending',
    created_at: '2026-09-09T10:00:00Z',
    ...over,
  } as QueueJob
}

describe('buildPipeline', () => {
  it('queued ingest: stage 0 active', () => {
    const p = buildPipeline([job({ id: 'a', status: 'queued' })])
    expect(p).not.toBeNull()
    expect(p!.steps[0].state).toBe('active')
    expect(p!.steps[1].state).toBe('future')
    expect(p!.type).toBe('text')
  })

  it('processing ingest: work stage active, queued done', () => {
    const p = buildPipeline([job({ id: 'a', status: 'processing' })])
    expect(p!.steps[0].state).toBe('done')
    expect(p!.steps[1].state).toBe('active')
  })

  it('ingest done + connections processing: connecting active, rest done', () => {
    const ingest = job({ id: 'a', status: 'done', connections_job_id: 'c1' })
    const conn = job({ id: 'c1', type: 'connections', status: 'processing' })
    const p = buildPipeline([ingest, conn])
    expect(p!.steps[0].state).toBe('done')
    expect(p!.steps[1].state).toBe('done')
    expect(p!.steps[p!.steps.length - 1].state).toBe('active')
    expect(p!.steps[p!.steps.length - 1].label).toBe('connecting')
  })

  it('all done: no pipeline (falls to the done list with flares)', () => {
    const ingest = job({ id: 'a', status: 'done', connections_job_id: 'c1' })
    const conn = job({ id: 'c1', type: 'connections', status: 'done' })
    expect(buildPipeline([ingest, conn])).toBeNull()
  })

  it('failed ingest: no pipeline (goes to failed section)', () => {
    expect(buildPipeline([job({ id: 'a', status: 'failed' })])).toBeNull()
  })

  it('latest capture wins', () => {
    const older = job({ id: 'a', created_at: '2026-09-09T09:00:00Z', status: 'processing' })
    const newer = job({ id: 'b', created_at: '2026-09-09T10:00:00Z', status: 'queued' })
    const p = buildPipeline([older, newer])
    expect(p!.jobId).toBe('b')
  })

  it('stage labels make sense per type', () => {
    expect(STAGE_LABELS.text).toContain('connecting')
    expect(STAGE_LABELS.image).toContain('describing · enriching')
    expect(STAGE_LABELS.pdf).toContain('enriching · tags · summary')
  })
})
