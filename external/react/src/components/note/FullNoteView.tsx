import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import type { NoteResponse } from '@/lib/api'

interface FullNoteViewProps {
  note: NoteResponse
}

export function FullNoteView({ note }: FullNoteViewProps) {
  return (
    <div className="note-content">
      {/* Summary */}
      {note.summary && (
        <section id="excerpt-Summary" className="note-section">
          <h3 className="note-section-heading">Summary</h3>
          <p className="text-sm leading-relaxed text-muted-foreground">
            {note.summary}
          </p>
        </section>
      )}

      {/* Key Ideas */}
      {note.key_ideas && note.key_ideas.length > 0 && (
        <section id="excerpt-Key Ideas" className="note-section">
          <h3 className="note-section-heading">Key Ideas</h3>
          <ul className="note-list">
            {note.key_ideas.map((idea, i) => (
              <li key={i}>{idea}</li>
            ))}
          </ul>
        </section>
      )}

      {/* Raw */}
      <section id="excerpt-Raw" className="note-section">
        <h3 className="note-section-heading">Raw</h3>
        <div className="note-raw-prose text-sm text-muted-foreground">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{note.raw}</ReactMarkdown>
        </div>
      </section>

      {/* Description */}
      {note.description && (
        <section id="excerpt-Description" className="note-section">
          <h3 className="note-section-heading">Description</h3>
          <p className="text-sm leading-relaxed text-muted-foreground">
            {note.description}
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
            {(() => { try { return new URL(note.source_url).hostname } catch { return note.source_url } })()}
          </a>
        </section>
      )}
    </div>
  )
}
