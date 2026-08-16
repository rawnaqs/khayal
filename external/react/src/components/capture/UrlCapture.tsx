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
    const [note, setNote] = useState('')
    const inputRef = useRef<HTMLInputElement>(null)

    useEffect(() => {
      inputRef.current?.focus()
    }, [])

    useImperativeHandle(ref, () => ({
      submit: async () => {
        if (!url.trim()) return
        const content = note.trim() ? `${url}\n\n${note.trim()}` : url
        await onSubmit(content)
        setUrl('')
        setNote('')
      },
    }))

    const handleKeyDown = (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.preventDefault()
        if (url.trim()) {
          const content = note.trim() ? `${url}\n\n${note.trim()}` : url
          onSubmit(content).then(() => {
            setUrl('')
            setNote('')
          })
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
          <label htmlFor="url-note" className="sr-only">add a note</label>
          <input
            id="url-note"
            type="text"
            placeholder="add a note... (optional)"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            className="w-full bg-transparent text-base outline-none"
            style={{ color: "rgba(245,245,245,0.5)", fontFamily: "'IBM Plex Mono', monospace", fontWeight: 400 }}
          />
        </div>
      </div>
    )
  }
)
