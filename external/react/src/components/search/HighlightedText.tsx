interface HighlightedTextProps {
  text: string
  query?: string
  className?: string
}

export function HighlightedText({ text, query, className = '' }: HighlightedTextProps) {
  if (!query || !text) {
    return <span className={className}>{text}</span>
  }

  const keywords = query
    .split(/\s+/)
    .filter(k => k.length > 1)
    .map(k => k.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))

  if (keywords.length === 0) {
    return <span className={className}>{text}</span>
  }

  const pattern = keywords.join('|')
  const regex = new RegExp(`(${pattern})`, 'gi')
  const parts = text.split(regex)

  return (
    <span className={className}>
      {parts.map((part, i) => {
        const isMatch = keywords.some(k =>
          part.toLowerCase() === k.replace(/\\/g, '').toLowerCase()
        )
        return isMatch ? (
          <mark key={i} className="hl">
            {part}
          </mark>
        ) : (
          <span key={i}>{part}</span>
        )
      })}
    </span>
  )
}
