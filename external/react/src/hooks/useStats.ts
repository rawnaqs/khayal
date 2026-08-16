import { useState, useEffect, useCallback } from 'react'
import { createClient, type StatsResponse } from '@/lib/api'
import { TIMEOUTS } from '@/lib/constants'
import { useVaultLock } from '@/hooks/useVaultLock'

export function useStats(pollInterval = TIMEOUTS.STATS_POLL) {
  const { token } = useVaultLock()
  const [stats, setStats] = useState<StatsResponse | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchStats = useCallback(async () => {
    try {
      const client = createClient(token)
      const response = await client.stats()
      setStats(response)
    } catch {
      // Silently fail - stats are not critical
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => {
    fetchStats()
    const interval = setInterval(fetchStats, pollInterval)
    return () => clearInterval(interval)
  }, [fetchStats, pollInterval])

  return { stats, loading, refresh: fetchStats }
}
