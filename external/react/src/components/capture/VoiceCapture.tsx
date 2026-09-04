import { forwardRef, useCallback, useImperativeHandle, useRef, useState } from 'react'
import { Mic, Square } from 'lucide-react'

export interface VoiceCaptureRef {
  submit: () => void
}

interface VoiceCaptureProps {
  onUpload: (file: File, note?: string) => void
  loading: boolean
  noteRef?: React.RefObject<HTMLTextAreaElement | null>
}

const MIME_CANDIDATES = ['audio/webm;codecs=opus', 'audio/webm', 'audio/mp4']

function pickMime(): string | undefined {
  if (typeof MediaRecorder === 'undefined') return undefined
  return MIME_CANDIDATES.find((m) => MediaRecorder.isTypeSupported(m))
}

export const VoiceCapture = forwardRef<VoiceCaptureRef, VoiceCaptureProps>(
  function VoiceCapture({ onUpload, loading }, ref) {
    const [recording, setRecording] = useState(false)
    const [seconds, setSeconds] = useState(0)
    const [error, setError] = useState<string | null>(null)
    const [previewUrl, setPreviewUrl] = useState<string | null>(null)
    const recorderRef = useRef<MediaRecorder | null>(null)
    const chunksRef = useRef<Blob[]>([])
    const blobRef = useRef<Blob | null>(null)
    const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

    const stop = useCallback(() => {
      recorderRef.current?.stop()
      setRecording(false)
      if (timerRef.current) clearInterval(timerRef.current)
    }, [])

    const start = useCallback(async () => {
      setError(null)
      setPreviewUrl(null)
      blobRef.current = null
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
        const mimeType = pickMime()
        const recorder = new MediaRecorder(stream, mimeType ? { mimeType } : undefined)
        chunksRef.current = []
        recorder.ondataavailable = (e) => {
          if (e.data.size > 0) chunksRef.current.push(e.data)
        }
        recorder.onstop = () => {
          stream.getTracks().forEach((t) => t.stop())
          const type = mimeType?.split(';')[0] || 'audio/webm'
          const blob = new Blob(chunksRef.current, { type })
          blobRef.current = blob
          setPreviewUrl(URL.createObjectURL(blob))
        }
        recorder.start()
        recorderRef.current = recorder
        setRecording(true)
        setSeconds(0)
        timerRef.current = setInterval(() => setSeconds((s) => s + 1), 1000)
      } catch {
        setError('Microphone access denied — allow the mic and try again.')
      }
    }, [])

    useImperativeHandle(ref, () => ({
      submit: () => {
        if (recording) stop()
        const blob = blobRef.current
        if (!blob) {
          setError('Record something first.')
          return
        }
        const ext = blob.type.includes('mp4') ? 'm4a' : 'webm'
        const file = new File([blob], `voice-note.${ext}`, { type: blob.type })
        onUpload(file)
      },
    }))

    const mm = String(Math.floor(seconds / 60)).padStart(2, '0')
    const ss = String(seconds % 60).padStart(2, '0')

    return (
      <div className="voice-capture" data-testid="voice-capture">
        {!recording && !previewUrl && (
          <button className="voice-start" onClick={start} disabled={loading} data-testid="voice-start">
            <Mic className="w-5 h-5" />
            start recording
          </button>
        )}

        {recording && (
          <button className="voice-stop" onClick={stop} data-testid="voice-stop">
            <Square className="w-4 h-4" />
            stop · {mm}:{ss}
          </button>
        )}

        {!recording && previewUrl && (
          <div className="voice-review" data-testid="voice-review">
            <audio controls src={previewUrl} className="voice-preview" />
            <div className="voice-review-actions">
              <button className="voice-restart" onClick={start} disabled={loading}>
                <Mic className="w-3.5 h-3.5" />
                re-record
              </button>
            </div>
          </div>
        )}

        {error && <div className="voice-error">{error}</div>}
      </div>
    )
  },
)
