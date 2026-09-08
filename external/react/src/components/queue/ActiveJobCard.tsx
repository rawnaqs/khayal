import { Check } from 'lucide-react'
import type { Pipeline } from '@/lib/pipeline'

interface ActiveJobCardProps {
  pipeline: Pipeline
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

export function ActiveJobCard({ pipeline }: ActiveJobCardProps) {
  const doneCount = pipeline.steps.filter((s) => s.state === 'done').length
  const pct = (doneCount / pipeline.steps.length) * 100

  return (
    <>
      <div className="sec">now processing</div>
      <div className="hero-card" data-testid="pipeline-card">
        <div className="hero-top">
          <div>
            <div className="hero-filename">{pipeline.title}</div>
            <div className="hero-meta">
              {pipeline.type} · {timeAgo(pipeline.createdAt)}
            </div>
          </div>
          <div className="hero-badge">
            <div className="badge-dot" />
            live
          </div>
        </div>
        <div className="prog-labels">
          {pipeline.steps.map((step, i) => (
            <span key={step.label + i} className={`prog-step ${step.state}`} title={step.label}>
              {step.state === 'done' && <Check className="w-2.5 h-2.5 inline" style={{ marginRight: 2, verticalAlign: -1 }} />}
              {step.label}
            </span>
          ))}
        </div>
        <div className="prog-bar">
          <div className="prog-fill" style={{ width: `${pct}%` }} />
          {pct < 100 && <div className="prog-fill-live" />}
        </div>
      </div>
    </>
  )
}
