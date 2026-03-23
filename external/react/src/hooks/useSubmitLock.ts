import { useRef, useCallback } from 'react'

export function useSubmitLock() {
  const locked = useRef(false)

  const withLock = useCallback(async (fn: () => Promise<void>) => {
    if (locked.current) return
    locked.current = true
    try {
      await fn()
    } finally {
      locked.current = false
    }
  }, [])

  return { locked: locked.current, withLock }
}
