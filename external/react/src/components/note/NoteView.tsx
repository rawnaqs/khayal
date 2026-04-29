import { useState, useEffect } from "react";
import { useNote } from "@/hooks/useNote";
import { Sheet, SheetContent } from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { ExcerptView } from "./ExcerptView";
import { FullNoteView } from "./FullNoteView";

interface NoteViewProps {
  notePath: string | null;
  query?: string;
  onClose: () => void;
}

function getTypeBadgeClass(type: string) {
  switch (type) {
    case "text": return "rb-t";
    case "article": return "rb-a";
    case "image": return "rb-t";
    default: return "rb-t";
  }
}

function formatDate(dateStr: string) {
  try {
    const date = new Date(dateStr);
    return date.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  } catch {
    return dateStr;
  }
}

export function NoteView({ notePath, query, onClose }: NoteViewProps) {
  const { note, loading, error } = useNote(notePath, query);
  const [view, setView] = useState<"excerpt" | "full">("excerpt");

  useEffect(() => {
    setView("excerpt");
  }, [notePath]);

  return (
    <Sheet
      open={!!notePath}
      modal
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <SheetContent
        side="right"
        className="w-[90vw] sm:max-w-[500px] p-0 flex flex-col [&>button:first-of-type]:hidden focus:outline-none"
        style={{
          background: "#0d0d0d",
          borderLeft: "1px solid rgba(255,255,255,0.08)",
        }}
      >
        {/* Header */}
        <div className="flex items-center gap-3 px-5 py-4 border-b border-white/5 shrink-0">
          <h2
            className="flex-1 text-base font-semibold truncate"
            style={{ fontFamily: "'Bricolage Grotesque', sans-serif" }}
          >
            {loading ? (
              <Skeleton
                className="h-5 w-48"
                style={{ background: "#1a1a1a" }}
              />
            ) : (
              note?.title || "Note"
            )}
          </h2>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-5 py-4 space-y-4">
          {loading && (
            <div className="space-y-4">
              <Skeleton
                className="h-4 w-32"
                style={{ background: "#1a1a1a" }}
              />
              <div className="flex gap-2">
                <Skeleton
                  className="h-5 w-16 rounded-full"
                  style={{ background: "#1a1a1a" }}
                />
                <Skeleton
                  className="h-5 w-20 rounded-full"
                  style={{ background: "#1a1a1a" }}
                />
              </div>
              <Skeleton
                className="h-24 w-full rounded-xl"
                style={{ background: "#1a1a1a" }}
              />
              <Skeleton
                className="h-40 w-full rounded-xl"
                style={{ background: "#1a1a1a" }}
              />
            </div>
          )}

          {error && (
            <div className="note-detail-error">
              <div className="error-text">Failed to load note: {error}</div>
            </div>
          )}

          {note && (
            <>
              {/* Metadata */}
              <div className="r1-meta">
                {note.created_at && (
                  <span className="rdate">{formatDate(note.created_at)}</span>
                )}
                {note.type && (
                  <span className={`rb ${getTypeBadgeClass(note.type)}`}>{note.type}</span>
                )}
                {note.tags?.map((tag, i) => (
                  <span key={i} className="rb rb-tag">#{tag}</span>
                ))}
              </div>

              {/* Excerpt box */}
              {note.excerpt && (
                <div className="excerpt-box">
                  <p className="excerpt-text">
                    <span className="excerpt-label">matched excerpt</span>
                    <br />
                    {note.excerpt}
                  </p>
                </div>
              )}

              {/* Toggle */}
              {note.excerpt && (
                <div className="view-toggle">
                  <button
                    className={`toggle-btn ${view === "excerpt" ? "active" : ""}`}
                    onClick={() => setView("excerpt")}
                  >
                    Excerpt
                  </button>
                  <button
                    className={`toggle-btn ${view === "full" ? "active" : ""}`}
                    onClick={() => setView("full")}
                  >
                    Full Note
                  </button>
                </div>
              )}

              {/* Content */}
              {view === "excerpt" && note.excerpt ? (
                <ExcerptView note={note} />
              ) : (
                <FullNoteView note={note} />
              )}

              {/* Footer */}
              <div
                className="text-xs pt-4 mt-4 border-t border-white/5"
                style={{ color: "rgba(245,245,245,0.3)" }}
              >
                <div className="font-mono truncate">{note.note_path}</div>
                {note.created_at && (
                  <div className="mt-1">
                    Created {formatDate(note.created_at)}
                    {note.updated_at && note.updated_at !== note.created_at && (
                      <> · Updated {formatDate(note.updated_at)}</>
                    )}
                  </div>
                )}
              </div>
            </>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
