import { useState, useCallback } from 'react'
import { createClient, type SearchResponse, type SearchOptions } from '@/lib/api'

export function useSearch() {
  const [loading, setLoading] = useState(false)
  const [results, setResults] = useState<SearchResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  const search = useCallback(async (query: string, opts: SearchOptions = {}) => {
    if (!query.trim()) {
      setResults(null)
      return
    }

    setLoading(true)
    setError(null)

    try {
      const client = createClient()
      const response = await client.search(query, {
        mode: 'hybrid',
        limit: 20,
        ...opts,
      })
      setResults(response)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed')
    } finally {
      setLoading(false)
    }
  }, [])

  const clear = () => {
    setResults(null)
    setError(null)
  }

  return {
    loading,
    results,
    error,
    search,
    clear,
  }
}
