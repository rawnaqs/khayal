# v1.1 Phase Index

| Phase | File | Description |
|-------|------|-------------|
| 0 | [GitHub issue #7](https://github.com/rawnaqs/khayal/issues/7) | Chunk-level indexing (shipped) |
| 1 | [phase-1-entities.md](phase-1-entities.md) | Extract entities from notes (**shipped**) |
| 2 | [phase-2-connections.md](phase-2-connections.md) | Proactive connections |
| 2.5 | [phase-2.5-capture-intelligence.md](phase-2.5-capture-intelligence.md) | Relative-date resolution + LLM context memory |
| 2.6 | [phase-2.6-search-overview.md](phase-2.6-search-overview.md) | On-demand AI answer above search results |
| 3 | [phase-3-vault.md](phase-3-vault.md) | Vault maintenance commands + user-facing delete note |
| 4 | [phase-4-backup.md](phase-4-backup.md) | Backup and restore |
| 5 | [phase-5-polish.md](phase-5-polish.md) | Final integration |

## Quick Reference

```bash
# Phase 1: Entity Extraction — SHIPPED
# Automatic after capture; entities in frontmatter + entities table

# Phase 2: Connections
# Automatic after capture (polled via queue)

# Phase 2.5: Capture Intelligence
# Date resolution automatic after capture (dates: `tomorrow → YYYY-MM-DD`)
# Memory injection automatic when memory.enabled (default true)

# Phase 2.6: Search Overview
kl search "query" --answer          # explicit CTA, no config option
# PWA: "AI Answer" button on search results view

# Phase 3: Vault Commands
khayal vault health
khayal vault fix-links
khayal vault fix-links --fix
khayal vault clean-media
khayal vault clean-media --fix
khayal vault show-duplicates
DELETE /v1/note?path=...            # soft-delete to .khayal-trash/
kl delete <note-path-or-id>         # same via CLI

# Phase 4: Backup/Restore
khayal backup --dest /path
khayal backup --dest /path --encrypt
khayal backup --init-key
khayal restore --from /path
khayal restore --from /path --overwrite

# Phase 5: Polish
go vet ./...
go test ./...
```
