import { Check } from 'lucide-react'
import type { QueueJob } from '@/lib/api'

interface DoneItemProps {
  job: QueueJob
}

function timeAgo(dateStr: string) {
  try {
    const date = new Date(dateStr)
    const now = new Date()
    const diff = Math.floor((now.getTime() - date.getTime()) / 1000)
    if (diff < 60) return `${diff}s`
    if (diff < 3600) return `${Math.floor(diff / 60)}m`
    return `${Math.floor(diff / 3600)}h`
  } catch {
    return ''
  }
}

function truncateTitle(title: string, maxLen = 40) {
  if (!title) return ''
  if (title.length <= maxLen) return title
  return title.slice(0, maxLen - 3) + '...'
}

export function DoneItem({ job }: DoneItemProps) {
  const title = truncateTitle(job.note_path || job.type)

  return (
    <div className="done-item">
      <div className="done-check">
        <Check className="w-3 h-3" style={{ color: '#3ddc84' }} />
      </div>
      <div className="done-body">
        <div className="done-title">{title}</div>
        <div className="done-meta">{job.type}</div>
      </div>
      <span className="done-ago">{timeAgo(job.processed_at || job.created_at)}</span>
    </div>
  )
}
