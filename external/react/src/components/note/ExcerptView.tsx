import { HighlightedText } from '@/components/search/HighlightedText'
import type { NoteResponse } from '@/lib/api'

interface ExcerptViewProps {
  note: NoteResponse
}

export function ExcerptView({ note }: ExcerptViewProps) {
  const section = note.excerpt_section || 'Raw'
  const query = note.search_query

  return (
    <div className="note-content">
      <section className="note-section">
        <h3 className="note-section-heading">{section}</h3>
        <div className="text-sm leading-relaxed text-muted-foreground whitespace-pre-wrap">
          <HighlightedText text={note.raw} query={query} />
        </div>
      </section>
    </div>
  )
}
