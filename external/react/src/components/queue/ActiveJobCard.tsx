import type { QueueJob } from '@/lib/api'

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

function getModels(type: string): string[] {
  switch (type) {
    case 'text': return ['llama3.2:3b']
    case 'image': return ['moondream', 'qwen2.5:7b']
    case 'article': return ['llama3.2:3b', 'nomic-embed-text']
    default: return []
  }
}

function getSteps(type: string): string[] {
  switch (type) {
    case 'text': return ['saved', 'tagging', 'summarizing', 'writing']
    case 'image': return ['saved', 'describing', 'tagging', 'writing']
    case 'article': return ['saved', 'extracting', 'summarizing', 'writing']
    default: return ['saved', 'processing']
  }
}

export function ActiveJobCard({ job }: ActiveJobCardProps) {
  const steps = getSteps(job.type)
  const models = getModels(job.type)
  const progress = job.type === 'image' ? 65 : job.type === 'article' ? 40 : 50
  const activeStep = Math.floor((progress / 100) * steps.length)

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
            <span key={step} className={`prog-step ${i < activeStep ? 'done' : ''}`}>
              {step}
            </span>
          ))}
        </div>
        <div className="prog-bar">
          <div className="prog-fill" style={{ width: `${progress}%` }} />
        </div>
        <div className="model-row">
          {models.map((model: string) => (
            <span key={model} className="mc">{model}</span>
          ))}
        </div>
      </div>
    </>
  )
}
