import { useState, useEffect, useCallback } from 'react'
import { createClient, type QueueJob } from '@/lib/api'
import { LIMITS } from '@/lib/constants'
import { useVaultLock } from '@/hooks/useVaultLock'

export function useQueue() {
  const { token } = useVaultLock()
  const [loading, setLoading] = useState(false)
  const [jobs, setJobs] = useState<QueueJob[]>([])
  const [total, setTotal] = useState(0)
  const [error, setError] = useState<string | null>(null)

  const fetchQueue = useCallback(async (status?: string) => {
    setLoading(true)
    setError(null)

    try {
      const client = createClient(token)
      const response = await client.queue({ status, limit: LIMITS.QUEUE_JOBS })
      setJobs(response.jobs || [])
      setTotal(response.total)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch queue')
    } finally {
      setLoading(false)
    }
  }, [token])

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
    fetchQueue,
    retryJob,
    discardJob,
  }
}
