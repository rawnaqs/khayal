import { motion, AnimatePresence } from 'framer-motion'
import { Sparkles, RefreshCw, X } from 'lucide-react'
import type { SearchOverview } from '@/lib/api'
import type { AIAnswerState } from '@/hooks/useAIAnswer'
import { cn } from '@/lib/utils'

interface AIAnswerProps {
  state: AIAnswerState
  overview: SearchOverview | null
  onCitationClick: (index: number) => void
  onRetry: () => void
  onClose: () => void
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
              onClick={() => onCitationClick(n)}
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

export function AIAnswer({ state, overview, onCitationClick, onRetry, onClose }: AIAnswerProps) {
  return (
    <AnimatePresence mode="wait">
      {/* Skeleton — mirrors the results shimmer */}
      {state === 'loading' && (
        <motion.div
          key="skeleton"
          initial={{ opacity: 0, y: -6 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0 }}
          className="ai-answer"
          data-testid="ai-answer-skeleton"
        >
          <div className="ai-head">
            <span className="ai-badge loading">
              <Sparkles className="w-3 h-3" />
              thinking
            </span>
          </div>
          <div className="ai-skel-lines">
            <div className="animate-shimmer ai-skel w-11/12" />
            <div className="animate-shimmer ai-skel w-full" />
            <div className="animate-shimmer ai-skel w-4/5" />
            <div className="animate-shimmer ai-skel w-2/3" />
          </div>
        </motion.div>
      )}

      {/* Error */}
      {state === 'error' && (
        <motion.div
          key="error"
          initial={{ opacity: 0, y: -6 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0 }}
          className="ai-answer"
          data-testid="ai-answer-error"
        >
          <div className="ai-head">
            <span className="ai-badge error">couldn&rsquo;t generate an answer</span>
            <div className="ai-actions">
              <button className="ai-action" onClick={onRetry} title="try again">
                <RefreshCw className="w-3 h-3" />
                retry
              </button>
              <button className="ai-action" onClick={onClose} title="dismiss">
                <X className="w-3 h-3" />
              </button>
            </div>
          </div>
          <p className="ai-text dim">The answer engine didn&rsquo;t respond. Your results are unaffected.</p>
        </motion.div>
      )}

      {/* Ready */}
      {state === 'ready' && overview && (
        <motion.div
          key="ready"
          initial={{ opacity: 0, y: -6 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.25, ease: 'easeOut' }}
          className="ai-answer"
          data-testid="ai-answer"
        >
          <div className="ai-head">
            <span className="ai-badge">
              <Sparkles className="w-3 h-3" />
              AI answer
            </span>
            <div className="ai-actions">
              <button className="ai-action" onClick={onRetry} title="regenerate">
                <RefreshCw className="w-3 h-3" />
              </button>
              <button className="ai-action" onClick={onClose} title="dismiss">
                <X className="w-3 h-3" />
              </button>
            </div>
          </div>
          <p className="ai-text">
            <CitationText text={overview.text} onCitationClick={onCitationClick} />
          </p>
        </motion.div>
      )}
    </AnimatePresence>
  )
}

export function AIAnswerCTA({
  visible,
  disabled,
  active,
  onClick,
}: {
  visible: boolean
  disabled?: boolean
  active?: boolean
  onClick: () => void
}) {
  if (!visible) return null
  return (
    <motion.button
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
      whileTap={{ scale: 0.96 }}
      className={cn('ai-cta', active && 'on')}
      onClick={onClick}
      disabled={disabled}
      data-testid="ai-answer-cta"
    >
      <Sparkles className="w-3.5 h-3.5" />
      AI Answer
    </motion.button>
  )
}
