import { motion } from 'framer-motion'
import type { SearchResult } from '@/lib/api'

interface ResultCardProps {
  result: SearchResult
  index?: number
}

export function ResultCard({ result, index = 0 }: ResultCardProps) {
  const formatDate = (dateStr: string) => {
    try {
      const date = new Date(dateStr)
      return date.toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
      })
    } catch {
      return dateStr
    }
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: index * 0.03, duration: 0.2 }}
      className="py-4 border-b border-border/20 last:border-b-0"
    >
      {/* Title + Score */}
      <div className="flex justify-between items-baseline">
        <span className="font-semibold text-foreground text-sm leading-tight">
          {result.title || result.note_path}
        </span>
        <span className="text-xs text-muted-foreground ml-3 shrink-0 tabular-nums">
          {result.score.toFixed(2)}
        </span>
      </div>

      {/* Meta line */}
      <div className="flex items-center gap-1.5 mt-1.5 text-xs text-muted-foreground flex-wrap">
        <span>{formatDate(result.created_at)}</span>
        <span className="text-border">·</span>
        <span>{result.type}</span>
        {result.tags.slice(0, 3).map((tag) => (
          <span key={tag} className="text-primary/70">#{tag}</span>
        ))}
      </div>

      {/* Excerpt */}
      <p className="text-xs text-muted-foreground mt-2 leading-relaxed italic border-l-2 border-primary/20 pl-3">
        {result.excerpt}
      </p>
    </motion.div>
  )
}
