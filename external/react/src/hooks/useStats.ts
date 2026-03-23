import { useState, useEffect, useCallback } from 'react'
import { createClient, type StatsResponse } from '@/lib/api'

export function useStats(pollInterval = 60000) {
  const [stats, setStats] = useState<StatsResponse | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchStats = useCallback(async () => {
    try {
      const client = createClient()
      const response = await client.stats()
      setStats(response)
    } catch {
      // Silently fail - stats are not critical
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchStats()
    const interval = setInterval(fetchStats, pollInterval)
    return () => clearInterval(interval)
  }, [fetchStats, pollInterval])

  return { stats, loading, refresh: fetchStats }
}
