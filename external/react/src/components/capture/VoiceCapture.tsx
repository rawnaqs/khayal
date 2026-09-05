import { forwardRef, useCallback, useImperativeHandle, useRef, useState } from 'react'
import { Mic, RotateCcw, Play, Pause } from 'lucide-react'

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

function formatClock(total: number): string {
  const mm = String(Math.floor(total / 60)).padStart(2, '0')
  const ss = String(Math.floor(total % 60)).padStart(2, '0')
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
    const secondsRef = useRef(0)
    const previewUrlRef = useRef<string | null>(null)

    const stop = useCallback(() => {
      recorderRef.current?.stop()
      setRecording(false)
      if (timerRef.current) clearInterval(timerRef.current)
    }, [])

    const start = useCallback(async () => {
      setError(null)
      if (previewUrlRef.current) {
        URL.revokeObjectURL(previewUrlRef.current)
        previewUrlRef.current = null
      }
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
          // read the live value — the closure's `seconds` is stale (0)
          setDuration(secondsRef.current)
          const url = URL.createObjectURL(blob)
          previewUrlRef.current = url
          setPreviewUrl(url)
        }
        recorder.start()
        recorderRef.current = recorder
        setRecording(true)
        secondsRef.current = 0
        setSeconds(0)
        timerRef.current = setInterval(() => {
          secondsRef.current += 1
          setSeconds(secondsRef.current)
        }, 1000)
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
            <span className="voice-rec-time">{formatClock(seconds)}</span>
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
              <span className="voice-review-label">voice note · {formatClock(duration)}</span>
              <div className="img-rm" onClick={start} title="re-record">
                <RotateCcw className="w-3 h-3" />
              </div>
            </div>
            <VoicePlayer src={previewUrl} recordedDuration={duration} />
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

// VoicePlayer: custom playback bar — play/pause, seekable gold progress,
// current/total time. Replaces the raw <audio controls>.
function VoicePlayer({ src, recordedDuration }: { src: string; recordedDuration: number }) {
  const audioRef = useRef<HTMLAudioElement>(null)
  const [playing, setPlaying] = useState(false)
  const [current, setCurrent] = useState(0)
  const [total, setTotal] = useState(recordedDuration)

  const toggle = () => {
    const el = audioRef.current
    if (!el) return
    if (playing) {
      el.pause()
    } else {
      el.play().catch(() => {})
    }
  }

  const seek = (e: React.MouseEvent<HTMLDivElement>) => {
    const el = audioRef.current
    if (!el || !Number.isFinite(el.duration)) return
    const rect = e.currentTarget.getBoundingClientRect()
    const ratio = Math.min(Math.max((e.clientX - rect.left) / rect.width, 0), 1)
    el.currentTime = ratio * el.duration
    setCurrent(el.currentTime)
  }

  const pct = total > 0 ? Math.min((current / total) * 100, 100) : 0

  return (
    <div className="vplayer" data-testid="voice-player">
      <button
        className="vplayer-btn"
        onClick={toggle}
        data-testid="voice-player-toggle"
        title={playing ? 'pause' : 'play'}
      >
        {playing ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" style={{ marginLeft: 2 }} />}
      </button>
      <div className="vplayer-body">
        <div className="vplayer-track" onClick={seek} data-testid="voice-player-track">
          <div className="vplayer-fill" style={{ width: `${pct}%` }} />
        </div>
        <div className="vplayer-times">
          <span>{formatClock(current)}</span>
          <span>{formatClock(total)}</span>
        </div>
      </div>
      <audio
        ref={audioRef}
        src={src}
        onPlay={() => setPlaying(true)}
        onPause={() => setPlaying(false)}
        onEnded={() => {
          setPlaying(false)
          setCurrent(0)
        }}
        onTimeUpdate={(e) => setCurrent(e.currentTarget.currentTime)}
        onLoadedMetadata={(e) => {
          const d = e.currentTarget.duration
          if (Number.isFinite(d) && d > 0) setTotal(d)
        }}
      />
    </div>
  )
}
