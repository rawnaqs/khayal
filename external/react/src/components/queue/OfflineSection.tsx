import { Zap, Upload } from 'lucide-react'

interface OfflineItem {
  id: string
  content: string
  timestamp: number
}

interface OfflineSectionProps {
  items: OfflineItem[]
  onSync: () => void
}

function timeAgo(timestamp: number) {
  const now = Date.now()
  const diff = Math.floor((now - timestamp) / 1000)
  if (diff < 60) return `${diff}s`
  if (diff < 3600) return `${Math.floor(diff / 60)}m`
  return `${Math.floor(diff / 3600)}h`
}

function truncateContent(content: string, maxLen = 40) {
  if (content.length <= maxLen) return content
  // Check if it's a URL
  if (content.startsWith('http')) {
    try {
      const url = new URL(content)
      return url.hostname + url.pathname.slice(0, maxLen - url.hostname.length - 3) + '...'
    } catch {
      return content.slice(0, maxLen - 3) + '...'
    }
  }
  // Quote text content
  const quoted = `"${content}"`
  if (quoted.length <= maxLen) return quoted
  return quoted.slice(0, maxLen - 3) + '..."'
}

export function OfflineSection({ items, onSync }: OfflineSectionProps) {
  if (items.length === 0) return null

  return (
    <div className="off-card">
      <div className="off-hdr">
        <div className="off-title-row">
          <Zap className="w-4 h-4 text-[#C9933A]" />
          <span className="off-title">offline</span>
        </div>
        <span className="off-ct">{items.length} waiting</span>
      </div>
      <div className="off-list">
        {items.slice(0, 3).map((item) => (
          <div key={item.id} className="oi">
            <div className="oi-bar" />
            <span className="oi-txt">{truncateContent(item.content)}</span>
            <span className="oi-t">{timeAgo(item.timestamp)}</span>
          </div>
        ))}
      </div>
      <div className="sync-btn" onClick={onSync}>
        <span className="sync-txt">sync {items.length} captures</span>
        <Upload className="w-4 h-4 text-[#C9933A]" />
      </div>
    </div>
  )
}
