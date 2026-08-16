import { useState, useRef, forwardRef, useImperativeHandle } from "react";
import { Image } from "lucide-react";
import { formatFileSize } from "@/lib/time";

interface ImageCaptureProps {
  onUpload: (file: File, note?: string) => Promise<void>;
  loading: boolean;
}

export interface ImageCaptureRef {
  submit: () => void;
}

export const ImageCapture = forwardRef<ImageCaptureRef, ImageCaptureProps>(
  function ImageCapture(
    { onUpload }: ImageCaptureProps,
    ref: React.Ref<ImageCaptureRef>,
  ) {
    const [file, setFile] = useState<File | null>(null);
    const [note, setNote] = useState<string>("");
    const [preview, setPreview] = useState<string | null>(null);
    const fileRef = useRef<HTMLInputElement>(null);

    useImperativeHandle(ref, () => ({
      submit: async () => {
        if (!file) return;
        await onUpload(file, note);
        setFile(null);
        setNote("");
        setPreview(null);
      },
    }));

    const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const selected = e.target.files?.[0];
      if (selected) {
        setFile(selected);
        const reader = new FileReader();
        reader.onload = (ev) => setPreview(ev.target?.result as string);
        reader.readAsDataURL(selected);
      }
    };

    const handleRemove = () => {
      setFile(null);
      setPreview(null);
      if (fileRef.current) fileRef.current.value = "";
    };

    return (
      <div className="flex flex-col gap-3">
        <input
          ref={fileRef}
          type="file"
          accept="image/*"
          onChange={handleFileChange}
          className="hidden"
          aria-label="select an image"
        />

        {!preview ? (
          <div
            className="img-drop"
            onClick={() => fileRef.current?.click()}
            role="button"
            tabIndex={0}
            aria-label="tap to choose an image"
          >
            <div className="img-drop-icon">
              <Image className="w-5 h-5" style={{ color: "var(--gold)" }} />
            </div>
            <div className="img-drop-lbl">tap to choose</div>
            <div className="img-drop-sub">jpg · png · webp · heic</div>
          </div>
        ) : (
          <>
            <div className="img-filled">
              <img
                src={preview}
                alt={file?.name || "selected image"}
                className="w-full h-full object-cover"
                style={{ position: "absolute", inset: 0 }}
              />
              <div className="img-overlay">
                <span className="img-name">{file?.name}</span>
                <span className="img-size">
                  {file ? formatFileSize(file.size) : ""}
                </span>
                <button
                  className="img-rm"
                  onClick={handleRemove}
                  aria-label="remove image"
                >
                  <svg
                    width="10"
                    height="10"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                  >
                    <line x1="18" y1="6" x2="6" y2="18" />
                    <line x1="6" y1="6" x2="18" y2="18" />
                  </svg>
                </button>
              </div>
            </div>

            <div className="note-input">
              <label htmlFor="image-note" className="sr-only">
                add a note
              </label>
              <input
                id="image-note"
                type="text"
                placeholder="add a note..."
                className="w-full bg-transparent text-base outline-none focus-visible:ring-1 focus-visible:ring-[var(--gold)]"
                style={{
                  color: "rgba(245,245,245,0.5)",
                  fontFamily: "'IBM Plex Mono', monospace",
                  fontWeight: 400,
                }}
                value={note}
                onChange={(e) => setNote(e.target.value)}
              />
            </div>
          </>
        )}
      </div>
    );
  },
);
