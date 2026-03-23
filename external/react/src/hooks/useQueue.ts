import { useState, useEffect, useCallback } from 'react'
import { createClient, type QueueJob } from '@/lib/api'

export function useQueue() {
  const [loading, setLoading] = useState(false)
  const [jobs, setJobs] = useState<QueueJob[]>([])
  const [total, setTotal] = useState(0)
  const [error, setError] = useState<string | null>(null)

  const fetchQueue = useCallback(async (status?: string) => {
    setLoading(true)
    setError(null)

    try {
      const client = createClient()
      const response = await client.queue({ status, limit: 50 })
      setJobs(response.jobs || [])
      setTotal(response.total)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch queue')
    } finally {
      setLoading(false)
    }
  }, [])

  const retryJob = useCallback(async (id: string) => {
    try {
      const client = createClient()
      await client.retryJob(id)
      await fetchQueue()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to retry job')
    }
  }, [fetchQueue])

  const discardJob = useCallback(async (id: string) => {
    try {
      const client = createClient()
      await client.discardJob(id)
      await fetchQueue()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to discard job')
    }
  }, [fetchQueue])

  useEffect(() => {
    fetchQueue()
  }, [fetchQueue])

  return {
    loading,
    jobs,
    total,
    error,
    fetchQueue,
    retryJob,
    discardJob,
  }
}
