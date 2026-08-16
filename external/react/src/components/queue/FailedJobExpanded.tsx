import { AlertTriangle, RotateCcw, Trash2 } from 'lucide-react'
import type { QueueJob } from '@/lib/api'
import { timeAgo } from '@/lib/time'

interface FailedJobExpandedProps {
  job: QueueJob
  onRetry: (id: string) => void
  onDiscard: (id: string) => void
}

function parseError(error?: string): { code: string; message: string } {
  if (!error) return { code: 'UNKNOWN', message: 'unknown error' }
  const parts = error.split('·').map(s => s.trim())
  if (parts.length >= 2) return { code: parts[0], message: parts.slice(1).join(' · ') }
  return { code: 'ERR', message: error }
}

export function FailedJobExpanded({ job, onRetry, onDiscard }: FailedJobExpandedProps) {
  const { code, message } = parseError(job.error)
  const title = job.note_path || job.type

  return (
    <div className="fail-expanded">
      <div className="fe-main">
        <div className="fail-icon">
          <AlertTriangle className="w-4 h-4" style={{ color: "var(--bad)" }} />
        </div>
        <div className="fe-body">
          <div className="fe-title">{title}</div>
          <div className="fe-error-box">
            <div className="fe-code">{code}</div>
            <div className="fe-msg">{message}</div>
          </div>
          <div className="fe-attempts">failed {timeAgo(job.created_at)}</div>
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
