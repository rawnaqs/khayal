import { motion, AnimatePresence } from 'framer-motion'
import { Sparkles, RefreshCw, X, ChevronDown } from 'lucide-react'
import type { SearchOverview } from '@/lib/api'
import type { AIAnswerState } from '@/hooks/useAIAnswer'
import { cn } from '@/lib/utils'

interface AIAnswerRowProps {
  state: AIAnswerState
  expanded: boolean
  overview: SearchOverview | null
  onAsk: () => void
  onToggle: () => void
  onRetry: () => void
  onClose: () => void
  onCitationClick: (index: number) => void
}

function CitationText({ text, onCitationClick }: { text: string; onCitationClick: (n: number) => void }) {
  const parts = text.split(/(\[\d+\])/g)
  return (
    <>
      {parts.map((part, i) => {
        const m = part.match(/^\[(\d+)\]$/)
        if (m) {
          const n = parseInt(m[1], 10) - 1
          return (
            <button
              key={i}
              className="ai-cite"
              onClick={(e) => {
                e.stopPropagation()
                onCitationClick(n)
              }}
              title={`jump to source ${n + 1}`}
            >
              [{n + 1}]
            </button>
          )
        }
        return <span key={i}>{part}</span>
      })}
    </>
  )
}

export function AIAnswerRow({
  state,
  expanded,
  overview,
  onAsk,
  onToggle,
  onRetry,
  onClose,
  onCitationClick,
}: AIAnswerRowProps) {
  const loading = state === 'loading'
  const ready = state === 'ready' && overview !== null

  const handleHeaderClick = () => {
    if (!expanded && state === 'idle') onAsk()
    else onToggle()
  }

  const label = loading ? 'Thinking' : ready ? 'AI answer' : 'AI Answer'

  return (
    <div className={cn('ai-row', expanded && 'open')} data-testid="ai-answer-row">
      {/* Header — collapsed trigger / expanded title bar */}
      <button className="ai-row-head" onClick={handleHeaderClick} data-testid="ai-answer-trigger">
        <Sparkles className={cn('w-3.5 h-3.5 ai-spark', loading && 'spin')} />
        <span className="ai-row-label">{label}</span>
        <ChevronDown className={cn('w-3 h-3 ai-chevron', expanded && 'up')} />
      </button>

      {/* Fluent expansion */}
      <AnimatePresence initial={false}>
        {expanded && (
          <motion.div
            key="body"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.32, ease: [0.4, 0, 0.2, 1] }}
            style={{ overflow: 'hidden' }}
          >
            <div
              className="ai-row-body"
              data-testid={loading ? 'ai-answer-skeleton' : state === 'error' ? 'ai-answer-error' : 'ai-answer'}
            >
              {/* Skeleton — mirrors the results shimmer */}
              {loading && (
                <div className="ai-skel-lines">
                  <div className="animate-shimmer ai-skel w-11/12" />
                  <div className="animate-shimmer ai-skel w-full" />
                  <div className="animate-shimmer ai-skel w-4/5" />
                  <div className="animate-shimmer ai-skel w-2/3" />
                </div>
              )}

              {/* Error */}
              {state === 'error' && (
                <div className="ai-error-line">
                  <span className="ai-text dim">The answer engine didn&rsquo;t respond. Your results are unaffected.</span>
                  <div className="ai-actions">
                    <button
                      className="ai-action"
                      onClick={(e) => {
                        e.stopPropagation()
                        onRetry()
                      }}
                      title="try again"
                    >
                      <RefreshCw className="w-3 h-3" />
                      retry
                    </button>
                    <button
                      className="ai-action"
                      onClick={(e) => {
                        e.stopPropagation()
                        onClose()
                      }}
                      title="dismiss"
                    >
                      <X className="w-3 h-3" />
                    </button>
                  </div>
                </div>
              )}

              {/* Ready */}
              {ready && overview && (
                <>
                  <p className="ai-text">
                    <CitationText text={overview.text} onCitationClick={onCitationClick} />
                  </p>
                  <div className="ai-actions ai-foot">
                    <button
                      className="ai-action"
                      onClick={(e) => {
                        e.stopPropagation()
                        onRetry()
                      }}
                      title="regenerate"
                    >
                      <RefreshCw className="w-3 h-3" />
                    </button>
                    <button
                      className="ai-action"
                      onClick={(e) => {
                        e.stopPropagation()
                        onClose()
                      }}
                      title="dismiss"
                    >
                      <X className="w-3 h-3" />
                    </button>
                  </div>
                </>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
