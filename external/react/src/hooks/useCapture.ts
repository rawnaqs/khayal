import { useState } from 'react'
import { createClient, type CaptureResponse } from '@/lib/api'
import { saveOffline } from '@/lib/offline'

export function useCapture() {
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<CaptureResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [errorCode, setErrorCode] = useState<string | undefined>(undefined)
  const [isOffline, setIsOffline] = useState(false)
  const [processingTime, setProcessingTime] = useState<number | undefined>(undefined)

  const capture = async (type: 'text' | 'url' | 'image', content: string) => {
    setLoading(true)
    setError(null)
    setErrorCode(undefined)
    setResult(null)
    setIsOffline(false)
    setProcessingTime(undefined)

    const startTime = performance.now()

    try {
      if (!navigator.onLine) {
        await saveOffline({ type, content })
        setIsOffline(true)
        setProcessingTime(Math.round(performance.now() - startTime))
        return
      }

      const client = createClient()
      const response = await client.capture({ type, content })
      setProcessingTime(Math.round(performance.now() - startTime))
      setResult(response)
    } catch (err) {
      setProcessingTime(Math.round(performance.now() - startTime))
      if (err instanceof Error && err.message.includes('fetch')) {
        await saveOffline({ type, content })
        setIsOffline(true)
      } else {
        setError(err instanceof Error ? err.message : 'Capture failed')
      }
    } finally {
      setLoading(false)
    }
  }

  const uploadImage = async (file: File, note?: string) => {
    setLoading(true)
    setError(null)
    setErrorCode(undefined)
    setResult(null)
    setIsOffline(false)
    setProcessingTime(undefined)

    const startTime = performance.now()

    try {
      if (!navigator.onLine) {
        setError('Image upload requires connection')
        setProcessingTime(Math.round(performance.now() - startTime))
        return
      }

      const client = createClient()
      const response = await client.uploadImage(file, note)
      setProcessingTime(Math.round(performance.now() - startTime))
      setResult(response)
    } catch (err) {
      setProcessingTime(Math.round(performance.now() - startTime))
      setError(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setLoading(false)
    }
  }

  const clear = () => {
    setResult(null)
    setError(null)
    setErrorCode(undefined)
    setIsOffline(false)
    setProcessingTime(undefined)
  }

  return {
    loading,
    result,
    error,
    errorCode,
    isOffline,
    processingTime,
    capture,
    uploadImage,
    clear,
  }
}
