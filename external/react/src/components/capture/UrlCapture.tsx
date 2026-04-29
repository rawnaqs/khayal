import { useState, useRef, useEffect, forwardRef, useImperativeHandle } from 'react'
import { Link, Link2 } from 'lucide-react'

interface UrlCaptureProps {
  onSubmit: (content: string) => Promise<void>
  loading: boolean
}

export interface UrlCaptureRef {
  submit: () => void
}

function extractDomain(url: string): string {
  try {
    return new URL(url).hostname
  } catch {
    return ''
  }
}

export const UrlCapture = forwardRef<UrlCaptureRef, UrlCaptureProps>(
  function UrlCapture({ onSubmit, loading }, ref) {
    const [url, setUrl] = useState('')
    const inputRef = useRef<HTMLInputElement>(null)

    useEffect(() => {
      inputRef.current?.focus()
    }, [])

    useImperativeHandle(ref, () => ({
      submit: async () => {
        if (!url.trim()) return
        await onSubmit(url)
        setUrl('')
      },
    }))

    const handleKeyDown = (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.preventDefault()
        if (url.trim()) {
          onSubmit(url).then(() => setUrl(''))
        }
      }
    }

    const domain = extractDomain(url)

    return (
      <div className="flex flex-col gap-3">
        {/* URL input row */}
        <div className="url-row">
          <Link />
          <input
            ref={inputRef}
            type="url"
            placeholder="https://example.com/article"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={loading}
            className="url-val bg-transparent outline-none"
          />
        </div>

        {/* URL preview (shown when URL is entered) */}
        {url && domain && (
          <div className="url-preview">
            <div className="url-thumb">
              <Link2 className="w-5 h-5" style={{ color: 'rgba(201,147,58,0.4)' }} />
            </div>
            <div className="url-info">
              <div className="url-domain">{domain}</div>
              <div className="url-title">Extracting content...</div>
            </div>
          </div>
        )}

        {/* Optional note */}
        <div className="note-input">
          <input
            type="text"
            placeholder="add a note... (optional)"
            className="w-full bg-transparent text-base text-[rgba(245,245,245,0.3)] placeholder-[rgba(245,245,245,0.2)] outline-none"
            style={{ fontWeight: 300 }}
          />
        </div>
      </div>
    )
  }
)
