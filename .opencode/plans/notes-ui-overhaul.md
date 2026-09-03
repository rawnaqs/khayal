# Notes UI Overhaul (scope: items 1–6 + 9 from ideas review)

State at pause: red test written — internal/api/media_test.go (5 cases,
build-fails on missing Server.mediaHandler). All research done.

## Batch 1 — server (TDD, red test exists)

1. internal/api/media.go — mediaHandler (test-ready):
   - GET /v1/media?path=media/pic.jpg
   - path.Clean("/"+rel) pin against s.vault.MediaPath(); prefix check -> 400
   - extension allowlist -> Content-Type (jpg/jpeg/png/gif/webp/heic/pdf),
     unsupported ext -> 400; missing -> 404 via http.ServeFile
   - Cache-Control: private, max-age=3600
   - register r.Get("/v1/media", s.mediaHandler) inside authed /v1 group

## Batch 2 — NoteView upgrades (UI + small server)

2. RelatedLink.types (internal/api/notes.go):
   - noteHandler: SELECT result FROM jobs WHERE type='connections' AND
     note_path=? ORDER BY created_at DESC LIMIT 1 (new queue method
     GetConnectionsResultByPath or reuse); parse
     {"connections":[{note_path,type}]} -> map target path -> []string types
   - RelatedLink gains Types []string `json:"types,omitempty"`
3. React: image preview (api.ts + NoteView):
   - client.mediaUrl(path): fetch('/v1/media?path=..', token header) -> blob
     -> objectURL; NoteView renders <img> at top when note.type==='image'
     && note.source_file; revoke on unmount
4. Entity chips (NoteView + App + SearchView):
   - entities from note response? NOTE: check NoteResponse lacks entities —
     reader NoteContent.Entities exists; add Entities map to NoteResponse
     (from note.Entities) with people/amounts/dates arrays
   - NoteView chips: people/amounts/dates rows (gold people, neutral others)
   - tap person -> App onSearchQuery flow: App gains searchQuery-driven
     initialQuery passed to SearchView (new prop) which fires search on
     mount; chip click = switch tab + prefill
5. Copy raw button: header icon (Copy) -> navigator.clipboard.writeText
   (note.raw + frontmatter-ish markdown) -> toast "copied"
6. Source URL: show domain only (new URL(url).hostname), title attr full
7. Linked notes: move ABOVE content (right after linked chips = before
   excerpt/full toggle); chip shows type badges from related_links[].types
   (icons: contradiction GitCompare/Zap, revisit Repeat2, follow_up Clock,
   person User, similar Sparkles); keep title + click behavior

## Batch 3 — typography pass (item 9)

8. SheetContent: sm:max-w-[560px] on md+ screens
9. note-section typography: heading scale (11px mono uppercase exists —
   keep), body line-height 1.7, max-width for raw prose, Raw section
   rendered via ReactMarkdown ALWAYS (fix query-time plain-text
   inconsistency: wrap markdown render then apply highlighting via
   HTML-escaped text-node walk — or simpler: render markdown, skip
   highlighting in full view when markdown; keep highlight in excerpt
   view) — DECIDE at impl: markdown always, highlight only ExcerptView
10. Footer: subtle divider + smaller mono

## Batch 4 — wrap

11. openapi: /v1/media endpoint + related_links.types; UI_SPEC note-view
    section rewrite; docs sync
12. vitest: media blob preview mock, entity chips render, type badges;
    Go: media handler tests green, notes.go types test
13. rebuild assets; live verify: image note preview, chips -> search,
    linked notes above content with badges; full suites; commit(s)
