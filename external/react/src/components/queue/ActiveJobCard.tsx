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
  const active = pipeline.steps.find((s) => s.state === 'active')

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

        {/* Compact dot stepper: width-proof at any card size. The active
            stage's label + detail render below instead of beside dots. */}
        <div className="pipe-dots" data-testid="pipeline-steps">
          {pipeline.steps.map((step, i) => (
            <div key={step.label + i} className="pipe-step-row">
              {i > 0 && (
                <div className={`pipe-conn ${pipeline.steps[i - 1].state === 'done' ? 'done' : ''}`} />
              )}
              <div className={`pipe-dot ${step.state}`} title={step.detail || step.label}>
                {step.state === 'done' && <Check className="w-2 h-2" />}
              </div>
            </div>
          ))}
        </div>

        {active && (
          <div className="pipe-now" data-testid="pipeline-active">
            <span className="pipe-now-label">{active.label}</span>
            {active.detail && <span className="pipe-now-detail">{active.detail}</span>}
          </div>
        )}
      </div>
    </>
  )
}
