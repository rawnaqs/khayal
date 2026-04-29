import { HighlightedText } from '@/components/search/HighlightedText'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import type { NoteResponse } from '@/lib/api'

interface FullNoteViewProps {
  note: NoteResponse
}

export function FullNoteView({ note }: FullNoteViewProps) {
  const query = note.search_query

  return (
    <div className="note-content">
      {/* Summary */}
      {note.summary && (
        <section id="excerpt-Summary" className="note-section">
          <h3 className="note-section-heading">Summary</h3>
          <p className="text-sm leading-relaxed text-muted-foreground">
            <HighlightedText text={note.summary} query={query} />
          </p>
        </section>
      )}

      {/* Key Ideas */}
      {note.key_ideas && note.key_ideas.length > 0 && (
        <section id="excerpt-Key Ideas" className="note-section">
          <h3 className="note-section-heading">Key Ideas</h3>
          <ul className="note-list">
            {note.key_ideas.map((idea, i) => (
              <li key={i}>
                <HighlightedText text={idea} query={query} />
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Raw */}
      <section id="excerpt-Raw" className="note-section">
        <h3 className="note-section-heading">Raw</h3>
        <div className="text-sm text-muted-foreground whitespace-pre-wrap">
          {query ? (
            <HighlightedText text={note.raw} query={query} />
          ) : (
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{note.raw}</ReactMarkdown>
          )}
        </div>
      </section>

      {/* Description */}
      {note.description && (
        <section id="excerpt-Description" className="note-section">
          <h3 className="note-section-heading">Description</h3>
          <p className="text-sm leading-relaxed text-muted-foreground">
            <HighlightedText text={note.description} query={query} />
          </p>
        </section>
      )}

      {/* Source URL */}
      {note.source_url && (
        <section className="note-section">
          <h3 className="note-section-heading">Source</h3>
          <a
            href={note.source_url}
            target="_blank"
            rel="noopener noreferrer"
            style={{
              color: '#c9933a',
              fontSize: '0.875rem',
              textDecoration: 'underline',
            }}
          >
            {note.source_url}
          </a>
        </section>
      )}
    </div>
  )
}
