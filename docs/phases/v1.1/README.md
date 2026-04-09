# v1.1 Implementation Guide

> Chunking, Entity Extraction, Proactive Connections, Vault Commands, Backup/Restore. Updated: 2026-04-09

## Overview

v1.1 adds:
1. **Chunking** — Split notes into 150-200 word chunks for better semantic search
2. **Entity Extraction** — Extract people, amounts, dates, places, orgs, URLs
3. **Proactive Connections** — Auto-discover related notes after capture
4. **Vault Commands** — health, fix-links, clean-media, show-duplicates
5. **Backup/Restore** — Backup vault, db, config with optional age encryption

## Phases

| Phase | Name | Description |
|-------|------|-------------|
| 1 | Chunking | Split notes into chunks |
| 2 | Entity Extraction | Extract entities from notes |
| 3 | Proactive Connections | Find related notes |
| 4 | Vault Commands | Maintenance subcommands |
| 5 | Backup/Restore | Backup and restore |
| 6 | Polish | Final integration |

## Dependencies

```bash
go get github.com/jonathanhecl/chunker@v0.0.1
go get filippo.io/age@v1.2.0
go mod tidy
```

**Notes:**
- Pin chunker to v0.0.1
- Pin age to v1.2.0 (not @latest)

## Files Created

```
internal/
├── chunker/
│   └── chunker.go          # Wrapper (uses github.com/jonathanhecl/chunker)
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
| `github.com/jonathanhecl/chunker` | Text chunking |
| `filippo.io/age` | Backup encryption |

- **Chunking**: SPEC.md lines 1321-1356
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
- **v1.1**: Chunking, entity extraction, proactive connections, vault commands, backup