# v1.1 Phase Index

| Phase | File | Description |
|-------|------|-------------|
| 1 | [phase-1-chunking.md](phase-1-chunking.md) | Split notes into chunks |
| 2 | [phase-2-entities.md](phase-2-entities.md) | Extract entities from notes |
| 3 | [phase-3-connections.md](phase-3-connections.md) | Proactive connections |
| 4 | [phase-4-vault.md](phase-4-vault.md) | Vault maintenance commands |
| 5 | [phase-5-backup.md](phase-5-backup.md) | Backup and restore |
| 6 | [phase-6-polish.md](phase-6-polish.md) | Final integration |

## Quick Reference

```bash
# Phase 1: Chunking
go build ./cmd/khayal
kl "note content"

# Phase 2: Entities
# Automatic after capture

# Phase 3: Connections
# Automatic after capture (polled via queue)

# Phase 4: Vault Commands
khayal vault health
khayal vault fix-links
khayal vault fix-links --fix
khayal vault clean-media
khayal vault clean-media --fix
khayal vault show-duplicates

# Phase 5: Backup/Restore
khayal backup --dest /path
khayal backup --dest /path --encrypt
khayal backup --init-key
khayal restore --from /path
khayal restore --from /path --overwrite

# Phase 6: Polish
go vet ./...
go test ./...
```