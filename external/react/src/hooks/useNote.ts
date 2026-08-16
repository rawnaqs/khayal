import { useState, useEffect } from 'react'
import { createClient, type NoteResponse } from '@/lib/api'
import { useVaultLock } from '@/hooks/useVaultLock'

export function useNote(notePath: string | null, query?: string) {
  const { token } = useVaultLock()
  const [note, setNote] = useState<NoteResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!notePath) {
      setNote(null)
      setError(null)
      return
    }

    const fetchNote = async () => {
      setLoading(true)
      setError(null)

      try {
        const client = createClient(token)
        const response = await client.getNote(notePath, query)
        setNote(response)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load note')
      } finally {
        setLoading(false)
      }
    }

    fetchNote()
  }, [notePath, query, token])

  return { note, loading, error }
}
