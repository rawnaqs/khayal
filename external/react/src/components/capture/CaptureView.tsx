import { useState, useRef, useEffect, useCallback } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { SendHorizontal } from 'lucide-react'
import { TextCapture, type TextCaptureRef } from './TextCapture'
import { UrlCapture, type UrlCaptureRef } from './UrlCapture'
import { ImageCapture, type ImageCaptureRef } from './ImageCapture'
import { CaptureResult } from './CaptureResult'
import { CaptureStats } from './CaptureStats'
import { useCapture } from '@/hooks/useCapture'
import { useStats } from '@/hooks/useStats'
import { cn } from '@/lib/utils'

type CaptureMode = 'text' | 'url' | 'image'

function getGreeting(): string {
  const hour = new Date().getHours()
  if (hour < 5) return 'late night thoughts?'
  if (hour < 12) return 'good morning'
  if (hour < 17) return 'good afternoon'
  if (hour < 21) return 'good evening'
  return 'late night thoughts?'
}

interface CaptureViewProps {
  captureQuery?: string
  onCaptureQueryConsumed?: () => void
}

export function CaptureView({ captureQuery, onCaptureQueryConsumed }: CaptureViewProps) {
  const [mode, setMode] = useState<CaptureMode>('text')
  const [initialContent, setInitialContent] = useState<string | undefined>(undefined)
  const { loading, result, error, errorCode, isOffline, processingTime, capture, uploadImage, clear } = useCapture()
  const { stats, loading: statsLoading } = useStats()

  const textRef = useRef<TextCaptureRef>(null)
  const urlRef = useRef<UrlCaptureRef>(null)
  const imageRef = useRef<ImageCaptureRef>(null)

  const hasResult = !!(result || error || isOffline)

  // Handle captureQuery from search
  useEffect(() => {
    if (captureQuery) {
      setMode('text')
      setInitialContent(captureQuery)
      onCaptureQueryConsumed?.()
    }
  }, [captureQuery, onCaptureQueryConsumed])

  useEffect(() => {
    if (result || isOffline) {
      const timer = setTimeout(() => clear(), 3500)
      return () => clearTimeout(timer)
    }
  }, [result, isOffline, clear])

  const handleSubmit = async (content: string) => {
    await capture(mode, content)
    setInitialContent(undefined)
  }

  const handleImageUpload = async (file: File, note?: string) => {
    await uploadImage(file, note)
  }

  const handleSend = () => {
    switch (mode) {
      case 'text':
        textRef.current?.submit()
        break
      case 'url':
        urlRef.current?.submit()
        break
      case 'image':
        imageRef.current?.submit()
        break
    }
  }

  const handleRetry = useCallback(() => {
    clear()
    // Re-submit with the same mode — user will need to re-enter content
  }, [clear])

  const getHint = () => {
    switch (mode) {
      case 'text': return 'cmd+enter to capture'
      case 'url': return 'article · will extract content'
      case 'image': return 'image · will be describe'
    }
  }

  return (
    <div className="cap-body">
      <div className="cap-greeting">{getGreeting()}</div>
      <CaptureStats stats={stats} loading={statsLoading} />

      <div className="compose">
        {/* Type pills at TOP */}
        <div className="pills">
          <span className={cn('tp', mode === 'text' && 'on')} onClick={() => setMode('text')}>txt</span>
          <span className={cn('tp', mode === 'url' && 'on')} onClick={() => setMode('url')}>url</span>
          <span className={cn('tp', mode === 'image' && 'on')} onClick={() => setMode('image')}>img</span>
        </div>

        {/* Content area */}
        <div className="flex-1 min-h-0">
          <AnimatePresence mode="wait">
            <motion.div
              key={mode}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.15 }}
              className="h-full"
            >
              {mode === 'text' && (
                <TextCapture ref={textRef} onSubmit={handleSubmit} loading={loading} initialContent={initialContent} />
              )}
              {mode === 'url' && (
                <UrlCapture ref={urlRef} onSubmit={handleSubmit} loading={loading} />
              )}
              {mode === 'image' && (
                <ImageCapture ref={imageRef} onUpload={handleImageUpload} loading={loading} />
              )}
            </motion.div>
          </AnimatePresence>
        </div>

        {/* Footer with hint + send */}
        <div className="footer">
          <span className="hint">{getHint()}</span>
          <button
            className="send"
            onClick={handleSend}
            disabled={loading}
          >
            <SendHorizontal className="w-4 h-4" style={{ color: '#000' }} />
          </button>
        </div>
      </div>

      {/* Status tile below compose */}
      <AnimatePresence mode="wait">
        {hasResult && (
          <motion.div
            key="tile"
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 8 }}
            transition={{ duration: 0.2, ease: 'easeOut' }}
          >
            <CaptureResult
              result={result}
              error={error}
              errorCode={errorCode}
              isOffline={isOffline}
              processingTime={processingTime}
              onDismiss={clear}
              onRetry={handleRetry}
            />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
