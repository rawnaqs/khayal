import { useEffect, useRef } from 'react'
import type { QueueJob } from '@/lib/api'

export interface QueueWSEvent {
  event: 'job_updated'
  job: QueueJob
}

/**
 * useQueueWS subscribes to live queue job updates over WebSocket.
 *
 * - Token travels as a query param (browsers cannot set custom headers
 *   on the handshake).
 * - onJob fires for every broadcast; callers merge state themselves.
 * - Any failure closes silently — the caller keeps its polling fallback.
 */
export function useQueueWS(onJob: (job: QueueJob) => void, enabled = true) {
  const handlerRef = useRef(onJob)
  handlerRef.current = onJob

  useEffect(() => {
    if (!enabled) return

    let ws: WebSocket | null = null
    let closedByUs = false
    let retryTimer: ReturnType<typeof setTimeout> | null = null
    let attempts = 0

    const connect = () => {
      const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
      const token = localStorage.getItem('khayal_token') || ''
      try {
        ws = new WebSocket(`${proto}://${window.location.host}/v1/queue/ws?token=${encodeURIComponent(token)}`)
      } catch {
        return // malformed host etc. — stay on polling
      }

      ws.onmessage = (msg) => {
        try {
          const evt = JSON.parse(msg.data as string) as QueueWSEvent
          if (evt.event === 'job_updated' && evt.job) {
            handlerRef.current(evt.job)
          }
        } catch {
          // malformed frame — ignore
        }
      }

      ws.onopen = () => {
        attempts = 0
      }

      ws.onclose = () => {
        if (closedByUs) return
        // capped backoff reconnect; polling keeps flowing regardless
        attempts++
        const delay = Math.min(1000 * Math.pow(2, attempts), 15000)
        retryTimer = setTimeout(connect, delay)
      }

      ws.onerror = () => {
        ws?.close()
      }
    }

    connect()

    return () => {
      closedByUs = true
      if (retryTimer) clearTimeout(retryTimer)
      ws?.close()
    }
  }, [enabled])
}
