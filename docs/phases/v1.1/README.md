# v1.1 Implementation Guide

> Entity Extraction, Proactive Connections, Vault Commands, Backup/Restore. Updated: 2026-04-11

## Overview

v1.1 adds:
1. **Entity Extraction** — Extract people, amounts, dates, places, orgs, URLs
2. **Proactive Connections** — Auto-discover related notes after capture
3. **Vault Commands** — health, fix-links, clean-media, show-duplicates
4. **Backup/Restore** — Backup vault, db, config with optional age encryption

## Phases

| Phase | Name | Description |
|-------|------|-------------|
| 1 | Entity Extraction | Extract entities from notes |
| 2 | Proactive Connections | Find related notes |
| 3 | Vault Commands | Maintenance subcommands |
| 4 | Backup/Restore | Backup and restore |
| 5 | Polish | Final integration |

## Dependencies

```bash
go get filippo.io/age@v1.2.0
go mod tidy
```

**Notes:**
- Pin age to v1.2.0 (not @latest)
- No chunker dependency (chunking dropped from v1.1)

## Files Created

```
internal/
├── connections/
│   └── connections.go    # Proactive connections
├── backup/
│   └── backup.go         # Backup/restore logic
cmd/khayal/commands/
├── vault.go              # Parent command
├── vault_health.go       # health check
├── vault_fix_links.go    # fix-links
├── vault_clean_media.go # clean-media
├── vault_show_duplicates.go
├── backup.go            # backup command
└── restore.go           # restore command
```

## Specification Reference

| Package | Purpose |
|---------|---------|
| `filippo.io/age` | Backup encryption |

- **Entity Extraction**: SPEC.md lines 1357-1385
- **Proactive Connections**: SPEC.md lines 1123-1296
- **Vault Commands**: SPEC.md lines 403-479
- **Backup/Restore**: SPEC.md lines 483-591

## Rules to Follow

Per RULES.md:
1. Never `io.ReadAll` — always stream decode
2. Never accumulate all in memory — use heap for search
3. Store path only in jobs — not content
4. HTTP body: MaxBytesReader + defer Close
5. strings.Builder for concatenation
6. Never defer in loops

Per CLI_RULES.md:
- All output via theme package
- Keys: theme.Muted + width-padded
- Values: theme.Primary
- ✓ = theme.SuccessStyle
- ✗ = theme.ErrorStyle

## Version

This guide covers **Khayal v1.1**.

- **v1.0**: Core capture, search (FTS5 + semantic), CLI, PWA
- **v1.1**: Entity extraction, proactive connections, vault commands, backup