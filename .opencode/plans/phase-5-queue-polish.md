# Phase 5 Polish: Queue UX Overhaul (+ WebSocket follow-up)

User-expanded polish scope. Decisions: flare = connections chip + enriched
marker; gorilla/websocket; land A–F now, WebSocket as separate follow-up.

## Batch 1: A–F (this round)

### A. Open note from queue
- App.tsx already owns selectedNote → pass `onNoteSelect` into QueueView
- DoneItem + FailedJobCard/Expanded clickable when job.note_path exists
- Reuse NoteView sheet; deletedPaths filtering already works

### B. Connection/memory flare on done items
- api.ts QueueJob += result?, connections_job_id?, content?
- Server queueListHandler: hydrate each done job's connections payload —
  fetch chained connections job result via existing queue lookup (batch:
  one query joining jobs ON connections_job_id, or N small lookups capped
  at page size)
- DoneItem flares: `🔗 N` gold chip (count from result.connections) +
  `✦ enriched` marker when result non-empty; chips tap → open note

### C. Status/loader overhaul
- First-load skeleton shimmer rows in QueueView (same pattern as AI answer)
- ActiveJobCard indeterminate progress bar; processing dot pulse animation
- framer-motion layout transitions between status groups

### D. Logout button
- Header: logout icon next to SecuritySheet trigger
- Two-step inline confirm; calls useVaultLock.lock() + clears stored
  token/session keys → LockScreen reappears

### E. Full history "show more"
- Replace dead label with button; useQueue gains loadMoreDone(offset)
  using GET /v1/queue?status=done&limit=50&offset=N until total reached;
  collapse restores the 5-item slice

### F. Refresh animations
- AnimatePresence on section lists; refresh icon spins during any
  in-flight queue request (loading state already exists)

Wrap-up batch 1: vitest for flare parse + show-more pagination + logout
confirm; tsc; rebuild embedded assets; docs (UI_SPEC queue section);
commit(s).

## Batch 2: WebSocket (follow-up round)

1. go get github.com/gorilla/websocket
2. internal/api/ws.go: GET /v1/queue/ws, token via query param, origin
   check, ping/pong keepalive
3. internal/queue pubsub: PublishJobUpdate(job) called from UpdateJob/
   status transition paths; hub with per-client send channels, slow-
   consumer drop policy
4. Client: useQueueWS hook — connects when queue tab active; patches jobs
   in place on job_updated; silent fallback to polling on error/close
5. Tests: hub fan-out unit test, handler upgrade/auth test
6. Docs: openapi (ws endpoint note), ARCHITECTURE realtime section,
   UI_SPEC live updates
