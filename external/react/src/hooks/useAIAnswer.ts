import { useState, useCallback } from 'react'
import { createClient, type SearchOverview } from '@/lib/api'
import { useVaultLock } from '@/hooks/useVaultLock'

export type AIAnswerState = 'idle' | 'loading' | 'ready' | 'error'

export function useAIAnswer() {
  const { token } = useVaultLock()
  const [state, setState] = useState<AIAnswerState>('idle')
  const [overview, setOverview] = useState<SearchOverview | null>(null)

  const ask = useCallback(async (query: string, mode: 'keyword' | 'semantic' | 'hybrid') => {
    if (!query.trim()) return
    setState('loading')
    setOverview(null)
    try {
      const client = createClient(token)
      const response = await client.search(query, { mode, overview: true })
      if (response.overview) {
        setOverview(response.overview)
        setState('ready')
      } else {
        setState('error')
      }
    } catch {
      setState('error')
    }
  }, [token])

  const reset = useCallback(() => {
    setState('idle')
    setOverview(null)
  }, [])

  return { state, overview, ask, reset }
}
