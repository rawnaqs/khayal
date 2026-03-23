import { AlertTriangle, RotateCcw, Trash2 } from 'lucide-react'
import type { QueueJob } from '@/lib/api'

interface FailedJobCardProps {
  job: QueueJob
  onRetry: (id: string) => void
  onDiscard: (id: string) => void
}

function timeAgo(dateStr: string) {
  try {
    const date = new Date(dateStr)
    const now = new Date()
    const diff = Math.floor((now.getTime() - date.getTime()) / 1000)
    if (diff < 60) return `${diff}s ago`
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
    return `${Math.floor(diff / 3600)}h ago`
  } catch {
    return ''
  }
}

function parseError(error?: string): { code: string; message: string } {
  if (!error) return { code: 'UNKNOWN', message: 'unknown error' }
  const parts = error.split('·').map(s => s.trim())
  if (parts.length >= 2) return { code: parts[0], message: parts.slice(1).join(' · ') }
  return { code: 'ERR', message: error }
}

export function FailedJobCard({ job, onRetry, onDiscard }: FailedJobCardProps) {
  const { code, message } = parseError(job.error)
  const title = job.note_path || job.type

  return (
    <div className="fail-card">
      <div className="fail-main">
        <div className="fail-icon">
          <AlertTriangle className="w-4 h-4" style={{ color: '#ff4d4d' }} />
        </div>
        <div className="fail-body">
          <div className="fail-title">{title}</div>
          <div className="fail-reason">{code} · {message}</div>
          <div className="fail-time">failed {timeAgo(job.created_at)}</div>
        </div>
      </div>
      <div className="fail-actions">
        <div className="fa retry" onClick={() => onRetry(job.id)}>
          <RotateCcw className="fa-icon" />
          retry
        </div>
        <div className="fa discard" onClick={() => onDiscard(job.id)}>
          <Trash2 className="fa-icon" />
          discard
        </div>
      </div>
    </div>
  )
}
