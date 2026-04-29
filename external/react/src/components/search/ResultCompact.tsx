import type { SearchResult } from '@/lib/api'
import { LIMITS } from '@/lib/constants'

interface ResultCompactProps {
  result: SearchResult
  rank: number
  query?: string
  onSelect?: (notePath: string) => void
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

export function ResultCompact({ result, rank, query, onSelect }: ResultCompactProps) {
  return (
    <div className="rc" onClick={() => onSelect?.(result.note_path)}>
      <span className="rc-n">{rank}</span>
      <div className="rc-body">
        <div className="rc-title">{highlightText(result.title || result.note_path, query)}</div>
        <div className="rc-meta">
          <span className="rdate">{formatDate(result.created_at)}</span>
          <span className={`rb ${getTypeBadgeClass(result.type)}`}>{result.type}</span>
          {result.tags?.slice(0, LIMITS.TAGS_COMPACT)?.map((tag) => (
            <span key={tag} className="rb rb-tag">#{tag}</span>
          ))}
        </div>
      </div>
      <span className="rc-score">{result.score.toFixed(2)}</span>
    </div>
  )
}
