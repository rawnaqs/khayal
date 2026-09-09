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
        ) : (
          <>
            {/* Unified attachment card — same shell for image and pdf */}
            <div className="att-card" data-testid={isPdf ? 'pdf-preview' : 'image-preview'}>
              {isPdf ? (
                <div className="att-doc">
                  <div className="att-doc-icon">
                    <FileText className="w-7 h-7" style={{ color: '#C9933A' }} />
                  </div>
                  <span className="att-type">pdf</span>
                </div>
              ) : (
                <img src={preview || ''} alt="preview" className="att-img" />
              )}

              <div className="att-meta">
                <div className="att-meta-icon">
                  {isPdf
                    ? <FileText className="w-3.5 h-3.5" style={{ color: '#C9933A' }} />
                    : <Image className="w-3.5 h-3.5" style={{ color: '#C9933A' }} />}
                </div>
                <div className="att-meta-text">
                  <div className="att-name">{file.name}</div>
                  <div className="att-sub">
                    <span className="att-type-chip">{isPdf ? 'pdf' : 'image'}</span>
                    {formatFileSize(file.size)}
                    {isPdf && ' · text will be extracted'}
                  </div>
                </div>
                <button className="att-rm" onClick={handleRemove} title="remove" data-testid="att-remove">
                  <X className="w-3.5 h-3.5" />
                </button>
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
