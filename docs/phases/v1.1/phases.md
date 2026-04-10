# v1.1 Phase Index

| Phase | File | Description |
|-------|------|-------------|
| 1 | [phase-1-entities.md](phase-1-entities.md) | Extract entities from notes |
| 2 | [phase-2-connections.md](phase-2-connections.md) | Proactive connections |
| 3 | [phase-3-vault.md](phase-3-vault.md) | Vault maintenance commands |
| 4 | [phase-4-backup.md](phase-4-backup.md) | Backup and restore |
| 5 | [phase-5-polish.md](phase-5-polish.md) | Final integration |

## Quick Reference

```bash
# Phase 1: Entity Extraction
# Automatic after capture

# Phase 2: Connections
# Automatic after capture (polled via queue)

# Phase 3: Vault Commands
khayal vault health
khayal vault fix-links
khayal vault fix-links --fix
khayal vault clean-media
khayal vault clean-media --fix
khayal vault show-duplicates

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