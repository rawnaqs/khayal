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

        {/* Stepper: dots + short labels, connectors carry the progress */}
        <div className="pipe-steps" data-testid="pipeline-steps">
          {pipeline.steps.map((step, i) => (
            <div key={step.label + i} className="pipe-step-row">
              {i > 0 && (
                <div
                  className={`pipe-conn ${pipeline.steps[i - 1].state === 'done' ? 'done' : ''}`}
                />
              )}
              <div className={`pipe-step ${step.state}`} title={step.detail || step.label}>
                <div className="pipe-dot">
                  {step.state === 'done' && <Check className="w-2.5 h-2.5" />}
                </div>
                <span className="pipe-label">{step.label}</span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </>
  )
}
