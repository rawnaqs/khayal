import { useState, useRef, forwardRef, useImperativeHandle } from 'react'
import { Image, Camera, X, FileText } from 'lucide-react'

interface ImageCaptureProps {
  onUpload: (file: File, note?: string) => Promise<void>
  loading: boolean
}

export interface ImageCaptureRef {
  submit: () => void
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export const ImageCapture = forwardRef<ImageCaptureRef, ImageCaptureProps>(
  function ImageCapture({ onUpload }: ImageCaptureProps, ref: React.Ref<ImageCaptureRef>) {
    const [file, setFile] = useState<File | null>(null)
    const [note, setNote] = useState<string>("")
    const [preview, setPreview] = useState<string | null>(null)
    const fileRef = useRef<HTMLInputElement>(null)

    const isPdf = !!file && file.name.toLowerCase().endsWith('.pdf')

    useImperativeHandle(ref, () => ({
      submit: async () => {
        if (!file) return
        await onUpload(file, note)
        setFile(null)
        setNote("")
        setPreview(null)
      },
    }))

    const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const selected = e.target.files?.[0]
      if (!selected) return
      setFile(selected)
      setPreview(null)
      // PDFs have no <img> preview — show a document card instead.
      if (!selected.name.toLowerCase().endsWith('.pdf')) {
        const reader = new FileReader()
        reader.onload = (ev) => setPreview(ev.target?.result as string)
        reader.readAsDataURL(selected)
      }
    }

    const handleRemove = () => {
      setFile(null)
      setPreview(null)
      if (fileRef.current) fileRef.current.value = ''
    }

    return (
      <div className="flex flex-col gap-3">
        <input
          ref={fileRef}
          type="file"
          accept="image/*,.pdf"
          onChange={handleFileChange}
          className="hidden"
        />

        {!file ? (
          <>
            {/* Empty state - drop zone */}
            <div className="img-drop" onClick={() => fileRef.current?.click()}>
              <div className="img-drop-icon">
                <Image className="w-5 h-5" style={{ color: '#C9933A' }} />
              </div>
              <div className="img-drop-lbl">tap to choose</div>
              <div className="img-drop-sub">jpg · png · webp · heic · pdf</div>
            </div>

            {/* OR divider */}
            <div className="img-or">
              <div className="img-or-line" />
              <span className="img-or-txt">OR</span>
              <div className="img-or-line" />
            </div>

            {/* Camera button */}
            <div className="cam-btn" onClick={() => fileRef.current?.click()}>
              <Camera className="w-4 h-4" style={{ color: 'rgba(245,245,245,0.5)' }} />
              <span className="cam-txt">open camera</span>
            </div>
          </>
        ) : isPdf ? (
          <>
            {/* PDF preview — document card, no broken <img> */}
            <div className="pdf-filled" data-testid="pdf-preview">
              <div className="pdf-icon">
                <FileText className="w-7 h-7" style={{ color: '#C9933A' }} />
              </div>
              <div className="pdf-meta">
                <div className="pdf-name">{file.name}</div>
                <div className="pdf-sub">{formatFileSize(file.size)} · pdf · text will be extracted</div>
              </div>
              <div className="img-rm" onClick={handleRemove}>
                <X className="w-3 h-3" />
              </div>
            </div>

            {/* Optional note */}
            <div className="note-input">
              <input
                type="text"
                placeholder="add a note..."
                className="w-full bg-transparent text-base text-[rgba(245,245,245,0.3)] placeholder-[rgba(245,245,245,0.2)] outline-none"
                style={{ fontWeight: 300 }}
                value={note}
                onChange={(e) => setNote(e.target.value)}
              />
            </div>
          </>
        ) : (
          <>
            {/* Image preview */}
            <div className="img-filled">
              <img
                src={preview || ''}
                alt="preview"
                className="w-full h-full object-cover"
                style={{ position: 'absolute', inset: 0 }}
              />
              <div className="img-overlay">
                <span className="img-name">{file.name}</span>
                <span className="img-size">{formatFileSize(file.size)}</span>
                <div className="img-rm" onClick={handleRemove}>
                  <X className="w-3 h-3" />
                </div>
              </div>
            </div>

            {/* Optional note */}
            <div className="note-input">
              <input
                type="text"
                placeholder="add a note..."
                className="w-full bg-transparent text-base text-[rgba(245,245,245,0.3)] placeholder-[rgba(245,245,245,0.2)] outline-none"
                style={{ fontWeight: 300 }}
                value={note}
                onChange={(e) => setNote(e.target.value)}
              />
            </div>
          </>
        )}
      </div>
    )
  }
)
