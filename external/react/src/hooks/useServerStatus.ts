import { useState, useEffect, useCallback } from 'react'
import { createClient, type HealthResponse } from '@/lib/api'
import { TIMEOUTS } from '@/lib/constants'
import { useVaultLock } from '@/hooks/useVaultLock'

export type ServerStatus = 'ok' | 'degraded' | 'offline'

export function useServerStatus(pollInterval = TIMEOUTS.SERVER_STATUS_POLL) {
  const { token } = useVaultLock()
  const [status, setStatus] = useState<ServerStatus>('offline')
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [lastChecked, setLastChecked] = useState<Date | null>(null)

  const checkStatus = useCallback(async () => {
    try {
      const client = createClient(token)
      const response = await client.health()
      setHealth(response)

      const deps = response.dependencies
      if (deps.db.status !== 'ok' || deps.vault.status !== 'ok') {
        setStatus('degraded')
      } else {
        setStatus('ok')
      }
    } catch {
      setStatus('offline')
      setHealth(null)
    }
    setLastChecked(new Date())
  }, [token])

  useEffect(() => {
    checkStatus()
    const interval = setInterval(checkStatus, pollInterval)
    return () => clearInterval(interval)
  }, [checkStatus, pollInterval])

  return { status, health, lastChecked, checkStatus }
}
