import { useState, useEffect, useCallback } from 'react'
import { createClient, type QueueFlare, type QueueJob } from '@/lib/api'
import { LIMITS } from '@/lib/constants'
import { useVaultLock } from '@/hooks/useVaultLock'

const DONE_PAGE_SIZE = 50

export function useQueue() {
  const { token } = useVaultLock()
  const [loading, setLoading] = useState(false)
  const [jobs, setJobs] = useState<QueueJob[]>([])
  const [total, setTotal] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [flares, setFlares] = useState<Record<string, QueueFlare>>({})
  const [doneExpanded, setDoneExpanded] = useState(false)
  const [doneLoadingMore, setDoneLoadingMore] = useState(false)

  const applyResponse = useCallback((response: { jobs?: QueueJob[]; total: number; flares?: Record<string, QueueFlare> }) => {
    setJobs(response.jobs || [])
    setTotal(response.total)
    if (response.flares) {
      setFlares(prev => ({ ...prev, ...response.flares }))
    }
  }, [])

  const fetchQueue = useCallback(async (status?: string) => {
    setLoading(true)
    setError(null)

    try {
      const client = createClient(token)
      const response = await client.queue({ status, limit: LIMITS.QUEUE_JOBS })
      // a fresh poll resets the expanded history view
      setDoneExpanded(false)
      applyResponse(response)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch queue')
    } finally {
      setLoading(false)
    }
  }, [token, applyResponse])

  // applyLiveJob merges a WebSocket broadcast into the local list
  const applyLiveJob = useCallback((job: QueueJob) => {
    setJobs(prev => {
      const idx = prev.findIndex(j => j.id === job.id)
      if (idx === -1) return [job, ...prev]
      const next = [...prev]
      next[idx] = { ...next[idx], ...job }
      return next
    })
  }, [])

  // loadMoreDone pages through the full done history
  const loadMoreDone = useCallback(async () => {
    if (doneLoadingMore) return
    setDoneLoadingMore(true)
    setError(null)
    try {
      const client = createClient(token)
      let offset = jobs.filter(j => j.status === 'done').length
      for (;;) {
        const response = await client.queue({ status: 'done', limit: DONE_PAGE_SIZE, offset })
        const batch = response.jobs || []
        if (batch.length === 0) break
        // merge without duplicating jobs already in the list
        setJobs(prev => {
          const seen = new Set(prev.map(j => j.id))
          return [...prev, ...batch.filter(j => !seen.has(j.id))]
        })
        if (response.flares) {
          setFlares(prev => ({ ...prev, ...response.flares }))
        }
        if (batch.length < DONE_PAGE_SIZE) break
        offset += batch.length
      }
      setDoneExpanded(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load history')
    } finally {
      setDoneLoadingMore(false)
    }
  }, [doneLoadingMore, jobs, token])

  const retryJob = useCallback(async (id: string) => {
    try {
      const client = createClient(token)
      await client.retryJob(id)
      await fetchQueue()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to retry job')
    }
  }, [fetchQueue, token])

  const discardJob = useCallback(async (id: string) => {
    try {
      const client = createClient(token)
      await client.discardJob(id)
      await fetchQueue()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to discard job')
    }
  }, [fetchQueue, token])

  useEffect(() => {
    fetchQueue()
  }, [fetchQueue])

  return {
    loading,
    jobs,
    total,
    error,
    flares,
    doneExpanded,
    doneLoadingMore,
    loadMoreDone,
    applyLiveJob,
    setDoneExpanded,
    fetchQueue,
    retryJob,
    discardJob,
  }
}
