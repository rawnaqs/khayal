import { useRef, useEffect, useImperativeHandle, forwardRef } from 'react'

interface TextCaptureProps {
  onSubmit: (content: string) => Promise<void>
  loading: boolean
  initialContent?: string
}

export interface TextCaptureRef {
  submit: () => void
  getContent: () => string
}

export const TextCapture = forwardRef<TextCaptureRef, TextCaptureProps>(
  function TextCapture({ onSubmit, loading, initialContent }, ref) {
    const textareaRef = useRef<HTMLTextAreaElement>(null)

    useEffect(() => {
      if (initialContent && textareaRef.current) {
        textareaRef.current.value = initialContent
        textareaRef.current.selectionStart = textareaRef.current.value.length
        textareaRef.current.selectionEnd = textareaRef.current.value.length
      }
    }, [initialContent])

    useImperativeHandle(ref, () => ({
      submit: async () => {
        const content = textareaRef.current?.value || ''
        if (!content.trim()) return
        await onSubmit(content)
        if (textareaRef.current) {
          textareaRef.current.value = ''
        }
      },
      getContent: () => textareaRef.current?.value || '',
    }))

    const handleKeyDown = (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        const content = textareaRef.current?.value || ''
        if (content.trim()) {
          onSubmit(content).then(() => {
            if (textareaRef.current) {
              textareaRef.current.value = ''
            }
          })
        }
      }
    }

    return (
      <textarea
        ref={textareaRef}
        placeholder="what's on your mind..."
        onKeyDown={handleKeyDown}
        disabled={loading}
        className="w-full h-full resize-none bg-transparent text-[17px] font-light text-[#f5f5f5] placeholder-[rgba(245,245,245,0.2)] outline-none leading-relaxed"
        style={{ fontFamily: "'Bricolage Grotesque', sans-serif" }}
      />
    )
  }
)
