import { forwardRef, useCallback, useImperativeHandle, useRef, useState } from 'react'
import { Mic, RotateCcw } from 'lucide-react'

export interface VoiceCaptureRef {
  submit: () => void
}

interface VoiceCaptureProps {
  onUpload: (file: File, note?: string) => Promise<void>
  loading: boolean
}

const MIME_CANDIDATES = ['audio/webm;codecs=opus', 'audio/webm', 'audio/mp4']

function pickMime(): string | undefined {
  if (typeof MediaRecorder === 'undefined') return undefined
  return MIME_CANDIDATES.find((m) => MediaRecorder.isTypeSupported(m))
}

function formatSeconds(total: number): string {
  const mm = String(Math.floor(total / 60)).padStart(2, '0')
  const ss = String(total % 60).padStart(2, '0')
  return `${mm}:${ss}`
}

export const VoiceCapture = forwardRef<VoiceCaptureRef, VoiceCaptureProps>(
  function VoiceCapture({ onUpload }, ref) {
    const [recording, setRecording] = useState(false)
    const [seconds, setSeconds] = useState(0)
    const [error, setError] = useState<string | null>(null)
    const [previewUrl, setPreviewUrl] = useState<string | null>(null)
    const [duration, setDuration] = useState(0)
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
      setDuration(0)
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
          setDuration(seconds)
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
    }, [seconds])

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

    return (
      <div className="flex flex-col gap-3">
        {!recording && !previewUrl && (
          <div className="voice-idle" onClick={start} data-testid="voice-start">
            <div className="voice-idle-icon">
              <Mic className="w-5 h-5" style={{ color: '#C9933A' }} />
            </div>
            <div className="img-drop-lbl">tap to record</div>
            <div className="img-drop-sub">transcribed by your stt service</div>
          </div>
        )}

        {recording && (
          <div className="voice-rec" onClick={stop} data-testid="voice-stop">
            <span className="voice-rec-dot" />
            <span className="voice-rec-time">{formatSeconds(seconds)}</span>
            <div className="voice-rec-bars">
              <span />
              <span />
              <span />
              <span />
              <span />
            </div>
            <span className="voice-rec-hint">tap to stop</span>
          </div>
        )}

        {!recording && previewUrl && (
          <div className="voice-review" data-testid="voice-review">
            <div className="voice-review-head">
              <span className="voice-review-label">voice note · {formatSeconds(duration)}</span>
              <div className="img-rm" onClick={start} title="re-record">
                <RotateCcw className="w-3 h-3" />
              </div>
            </div>
            <audio controls src={previewUrl} className="voice-preview" />
            <div className="img-drop-sub" style={{ textAlign: 'center' }}>
              ready — tap send to transcribe
            </div>
          </div>
        )}

        {error && <div className="voice-error">{error}</div>}
      </div>
    )
  },
)
