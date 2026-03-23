import type { SearchResult } from '@/lib/api'

interface ResultHeroProps {
  result: SearchResult
  query?: string
}

function formatDate(dateStr: string) {
  try {
    const date = new Date(dateStr)
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
  } catch {
    return dateStr
  }
}

function getTypeBadgeClass(type: string) {
  switch (type) {
    case 'text': return 'rb-t'
    case 'article': return 'rb-a'
    case 'image': return 'rb-t'
    default: return 'rb-t'
  }
}

function highlightText(text: string, query?: string): React.ReactNode {
  if (!query || !text) return text
  const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const regex = new RegExp(`(${escaped})`, 'gi')
  const parts = text.split(regex)
  return parts.map((part, i) =>
    regex.test(part) ? <span key={i} className="hl">{part}</span> : part
  )
}

export function ResultHero({ result, query }: ResultHeroProps) {
  return (
    <div className="r1">
      <div className="r1-ghost">1</div>
      <div className="r1-title">{highlightText(result.title || result.note_path, query)}</div>
      <div className="r1-meta">
        <span className="rdate">{formatDate(result.created_at)}</span>
        <span className={`rb ${getTypeBadgeClass(result.type)}`}>{result.type}</span>
        {result.tags.slice(0, 3).map((tag) => (
          <span key={tag} className="rb rb-tag">#{tag}</span>
        ))}
      </div>
      <div className="r1-ex">{highlightText(result.excerpt, query)}</div>
    </div>
  )
}
