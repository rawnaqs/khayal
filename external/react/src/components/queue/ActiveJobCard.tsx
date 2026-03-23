import type { QueueJob } from '@/lib/api'
import { PROCESSING_STEPS } from '@/lib/constants'

interface ActiveJobCardProps {
  job: QueueJob
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

function getSteps(type: string): string[] {
  return PROCESSING_STEPS[type] || ['saved', 'processing']
}

export function ActiveJobCard({ job }: ActiveJobCardProps) {
  const steps = getSteps(job.type)

  return (
    <>
      <div className="sec">now processing</div>
      <div className="hero-card">
        <div className="hero-top">
          <div>
            <div className="hero-filename">{job.note_path || job.type}</div>
            <div className="hero-meta">
              {job.type} · {timeAgo(job.created_at)}
            </div>
          </div>
          <div className="hero-badge">
            <div className="badge-dot" />
            live
          </div>
        </div>
        <div className="prog-labels">
          {steps.map((step, i) => (
            <span key={step} className={`prog-step ${i === 0 ? 'done' : ''}`}>
              {step}
            </span>
          ))}
        </div>
        <div className="prog-bar">
          <div className="prog-fill" style={{ animation: 'indeterminate 2s linear infinite' }} />
        </div>
      </div>
    </>
  )
}
