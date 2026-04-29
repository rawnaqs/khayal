import { useState, useEffect } from 'react'
import { createClient, type NoteResponse } from '@/lib/api'

export function useNote(notePath: string | null, query?: string) {
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
        const client = createClient()
        const response = await client.getNote(notePath, query)
        setNote(response)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load note')
      } finally {
        setLoading(false)
      }
    }

    fetchNote()
  }, [notePath, query])

  return { note, loading, error }
}
