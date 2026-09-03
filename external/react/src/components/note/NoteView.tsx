import { useState, useEffect } from "react";
import { useNote } from "@/hooks/useNote";
import { useVaultLock } from "@/hooks/useVaultLock";
import { useToast } from "@/hooks/use-toast";
import { createClient } from "@/lib/api";
import { Sheet, SheetContent, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { Trash2, X, Link2, Copy, Zap, Repeat2, Clock, User, Sparkles } from "lucide-react";
import { ExcerptView } from "./ExcerptView";
import { FullNoteView } from "./FullNoteView";

const LINK_TYPE_ICONS: Record<string, React.ReactNode> = {
  contradiction: <Zap className="w-3 h-3 shrink-0" style={{ color: "#ff8a5c" }} />,
  revisit: <Repeat2 className="w-3 h-3 shrink-0" style={{ color: "#8ab4ff" }} />,
  follow_up: <Clock className="w-3 h-3 shrink-0" style={{ color: "#ffd166" }} />,
  person: <User className="w-3 h-3 shrink-0" style={{ color: "#c9933a" }} />,
  similar: <Sparkles className="w-3 h-3 shrink-0" style={{ color: "#3ddc84" }} />,
};

const LINK_TYPE_LABELS: Record<string, string> = {
  contradiction: "contradicts",
  revisit: "revisited",
  follow_up: "follow-up",
  person: "person",
  similar: "similar",
};

interface NoteViewProps {
  notePath: string | null;
  query?: string;
  onClose: () => void;
  onDeleted?: (notePath: string) => void;
  onOpenNote?: (notePath: string) => void;
  onSearch?: (query: string) => void;
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

export function NoteView({ notePath, query, onClose, onDeleted, onOpenNote, onSearch }: NoteViewProps) {
  const { note, loading, error } = useNote(notePath, query);
  const [view, setView] = useState<"excerpt" | "full">("excerpt");
  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [mediaUrl, setMediaUrl] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const { token } = useVaultLock();
  const { toast } = useToast();

  useEffect(() => {
    setView("excerpt");
    setConfirming(false);
    setDeleting(false);
    setCopied(false);
  }, [notePath]);

  // Image preview: token-authed blob fetch (no token in URL)
  useEffect(() => {
    setMediaUrl(null);
    if (!notePath || !note?.source_file || note.type !== "image") return;
    let revoke: string | null = null;
    let alive = true;
    createClient(token)
      .mediaBlob(note.source_file)
      .then((blob) => {
        if (!alive) return;
        revoke = URL.createObjectURL(blob);
        setMediaUrl(revoke);
      })
      .catch(() => {
        // preview is best-effort
      });
    return () => {
      alive = false;
      if (revoke) URL.revokeObjectURL(revoke);
    };
  }, [notePath, note?.source_file, note?.type, token, note]);

  const handleCopy = async () => {
    if (!note) return;
    const md = [
      `# ${note.title || "Note"}`,
      note.summary ? `\n${note.summary}` : "",
      note.key_ideas?.length ? `\n${note.key_ideas.map((k) => `- ${k}`).join("\n")}` : "",
      `\n${note.raw}`,
      note.source_url ? `\nSource: ${note.source_url}` : "",
    ]
      .filter(Boolean)
      .join("\n");
    try {
      await navigator.clipboard.writeText(md);
      setCopied(true);
      setTimeout(() => setCopied(false), 1600);
    } catch {
      toast({ title: "Copy failed", variant: "destructive" });
    }
  };

  const handleDelete = async () => {
    if (!notePath || deleting) return;
    setDeleting(true);
    try {
      const client = createClient(token);
      await client.deleteNote(notePath);
      toast({ title: 'Note deleted', description: 'Moved to trash — recoverable from .khayal-trash/' });
      onDeleted?.(notePath);
      onClose();
    } catch (err) {
      toast({
        title: 'Delete failed',
        description: err instanceof Error ? err.message : 'Unknown error',
        variant: 'destructive',
      });
      setDeleting(false);
      setConfirming(false);
    }
  };

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
        aria-describedby={undefined}
        className="w-[90vw] sm:max-w-[500px] md:max-w-[580px] p-0 flex flex-col [&>button:first-of-type]:hidden focus:outline-none"
        style={{
          background: "#0d0d0d",
          borderLeft: "1px solid rgba(255,255,255,0.08)",
          paddingBottom: 'max(env(safe-area-inset-bottom), 0px)',
        }}
      >
        {/* Screen-reader title: the visible h2 is decorative styling */}
        <SheetTitle className="sr-only">{note?.title || "Note"}</SheetTitle>
        <SheetDescription className="sr-only">
          Note details, connections, and actions
        </SheetDescription>

        {/* Header */}
        <div
          className="flex items-center gap-3 px-5 py-4 border-b border-white/5 shrink-0"
          style={{ paddingTop: 'calc(1rem + env(safe-area-inset-top))' }}
        >
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
          {!loading && note && (
            <>
              <button
                className="p-2 rounded-lg shrink-0 transition-colors"
                style={{ color: copied ? "#3ddc84" : "rgba(245,245,245,0.25)" }}
                onClick={handleCopy}
                title="copy note as markdown"
                data-testid="note-copy"
              >
                {copied ? <Copy className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
              </button>
              {confirming ? (
              <div className="flex items-center gap-1.5 shrink-0" data-testid="note-delete-confirm">
                <span className="text-[10px] font-mono whitespace-nowrap" style={{ color: "rgba(245,169,169,0.8)" }}>
                  move to trash?
                </span>
                <button
                  className="px-2 py-1 rounded-md text-[10px] font-mono font-bold uppercase tracking-wide"
                  style={{ color: "#f5a9a9", background: "rgba(255,99,99,0.08)", border: "1px solid rgba(255,99,99,0.25)" }}
                  onClick={handleDelete}
                  disabled={deleting}
                  data-testid="note-delete-go"
                >
                  {deleting ? "..." : "delete"}
                </button>
                <button
                  className="p-1.5 rounded-md"
                  style={{ color: "rgba(245,245,245,0.3)" }}
                  onClick={() => setConfirming(false)}
                  data-testid="note-delete-cancel"
                >
                  <X className="w-3.5 h-3.5" />
                </button>
              </div>
            ) : (
              <button
                className="p-2 rounded-lg shrink-0 transition-colors"
                style={{ color: "rgba(245,245,245,0.25)" }}
                onClick={() => setConfirming(true)}
                title="delete note"
                data-testid="note-delete-trigger"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            )}
            </>
          )}
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

              {/* Image preview for image captures */}
              {note.type === "image" && note.source_file && (
                mediaUrl ? (
                  <img
                    src={mediaUrl}
                    alt={note.title || "captured image"}
                    className="note-media"
                    data-testid="note-media"
                  />
                ) : (
                  <div className="note-media note-media-loading">
                    <Skeleton className="h-40 w-full rounded-xl" style={{ background: "#1a1a1a" }} />
                  </div>
                )
              )}

              {/* Entity chips — tap to search */}
              {(() => {
                // YAML frontmatter stores amounts (and sometimes dates) as
                // numbers — coerce everything to strings before use.
                const people = (note.entities?.people || []).map(String);
                const amounts = (note.entities?.amounts || []).map(String);
                const dates = (note.entities?.dates || []).map(String);
                if (people.length === 0 && amounts.length === 0 && dates.length === 0) return null;
                return (
                  <div className="entity-rows" data-testid="entity-chips">
                    {people.map((p, i) => (
                      <button
                        key={`p-${i}`}
                        className="entity-chip person"
                        onClick={() => onSearch?.(p)}
                        title={`search notes about ${p}`}
                      >
                        <User className="w-3 h-3" />
                        {p}
                      </button>
                    ))}
                    {amounts.map((a, i) => (
                      <button key={`a-${i}`} className="entity-chip" onClick={() => onSearch?.(a)} title={`search ${a}`}>
                        {a}
                      </button>
                    ))}
                    {dates.map((d, i) => (
                      <button key={`d-${i}`} className="entity-chip" onClick={() => onSearch?.(d)} title={`search ${d}`}>
                        {d}
                      </button>
                    ))}
                  </div>
                );
              })()}

              {/* Linked notes — above content, with reason badges.
                  Keyed by the note path: switching notes remounts the
                  list so hover/focus state never lingers on a chip. */}
              {note.related_links && note.related_links.length > 0 && (
                <div className="note-links" key={note.note_path} data-testid="note-links">
                  <div className="note-links-label">linked notes</div>
                  {note.related_links.map((link, i) => (
                    <button
                      key={note.note_path + "-" + i}
                      className="note-link-chip"
                      onClick={() => onOpenNote?.(link.note_path)}
                      onMouseDown={(e) => e.preventDefault()}
                      title={link.note_path}
                      data-testid="note-link-chip"
                    >
                      {link.types?.map((t) => (
                        <span key={t} className="note-link-type" title={LINK_TYPE_LABELS[t] || t}>
                          {LINK_TYPE_ICONS[t] || <Link2 className="w-3 h-3 shrink-0" />}
                        </span>
                      ))}
                      {!link.types?.length && <Link2 className="w-3 h-3 shrink-0" />}
                      <span className="note-link-title">{link.title}</span>
                      {link.types?.length ? (
                        <span className="note-link-types-label">
                          {link.types.map((t) => LINK_TYPE_LABELS[t] || t).join(" · ")}
                        </span>
                      ) : null}
                    </button>
                  ))}
                </div>
              )}

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
