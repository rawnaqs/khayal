# Phase 3: Delete Note + Vault Commands

## State at pause
- Red test written: `internal/queue/queue_test.go::TestRemoveNote` (build-fails on missing `Queue.RemoveNote`)
- `vault.Writer.DeleteNote` exists+tested (writer.go:226) — moves to `<inbox>/.khayal-trash/<name>.<unix-ts>`, unused
- openapi.yaml already documents `DELETE /v1/note` — no handler yet

## Workstream A: delete note
1. `internal/queue/queue.go`: add `RemoveNote(ctx, notePath)` after `DeleteChunksByNote`
   - single tx: DELETE FROM notes_fts / chunks / entities WHERE note_path = ?
   - retry loop pattern copied from DeleteFromIndex (`constants.SQLiteMaxRetries`, isFTSErr/isLockError)
   - idempotent; make TestRemoveNote green (fts/chunks/entities purged, search clean, idempotent)
2. API handler `DELETE /v1/note?path=...` in `internal/api/notes.go`
   - path safety: must resolve inside inbox dir; reject traversal, trash dir itself
   - flow: vault.DeleteNote → queue.RemoveNote → RecomputeStats → 200 {deleted:true}; missing → 404
   - tests: happy (file trashed + index purged), traversal rejected, 404
3. `kl delete <path-or-id>` cmd/kl/commands/delete.go
   - id→path resolution via queue lookup endpoint or search; `--yes` skips confirm
   - themed output per CLI_RULES (✓ SuccessStyle / ✗ ErrorStyle)
4. PWA delete in `NoteView.tsx`: trash icon → confirm sheet → DELETE call → toast + navigate back
   - api.ts client method; component test for confirm flow; rebuild assets

## Workstream B: vault maintenance commands (phase doc steps 4.1–4.5)
5. `cmd/khayal/commands/vault.go`: parent + health (RecomputeStats/CountByStatus based)
6. fix-links (broken wikilinks, --dry-run default/--fix), clean-media (orphans → .khayal-trash), show-duplicates
7. Unit tests per phase-3 checklist

## Wrap-up
8. Docs: VAULT.md (trash/delete), REPO_STRUCTURE (new files), UI_SPEC (delete affordance), CLI docs
9. Full suites: go test ./... + vitest + vet + golangci-lint; live verify: delete a testdata capture → gone from search, recoverable in .khayal-trash/; commits per workstream

## Known limitation (document only)
Old connection results JSON may reference deleted paths — cosmetic staleness, no retrieval impact.
