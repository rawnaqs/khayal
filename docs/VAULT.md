# Khayal Vault Documentation

> Vault structure, path handling, and safety guarantees. Updated: 2026-03-19

---

## Vault Structure

```
<vault_path>/
├── <inbox_dir>/              # Default: khayal/
│   ├── <media_dir>/          # Default: media/ (relative to inbox)
│   │   └── *.png, *.jpg, ... # Images from capture
│   └── *.md                   # Notes
└── .khayal-trash/            # Soft-deleted notes
```

**Example with defaults:**
```
~/Documents/brain/
├── khayal/
│   ├── media/
│   │   └── 1234567890.png
│   └── 2026-03-19-test-note.md
└── .khayal-trash/             # Soft-deleted files
```

---

## Configuration

### Vault Settings

```yaml
vault:
  path: ~/Documents/brain        # Vault root directory
  inbox_dir: khayal               # Relative to vault path
  media:
    default_dir: media           # Relative to inbox_dir
```

### Path Resolution Rules

| Path Type | Example | Resolution |
|-----------|---------|------------|
| `~` home | `~/brain` | Expanded to `$HOME/brain` |
| `$VAR` env | `$HOME/brain` | Expanded to `/Users/name/brain` |
| Absolute | `/var/data/brain` | Used as-is |
| Relative | `./brain` | Relative to config file location |

### Path Validation

All vault sub-paths (`inbox_dir`, `media.default_dir`) must be:

- ✅ Relative (not absolute)
- ✅ Not contain `~` or `$VAR`
- ✅ Not contain `..` (path traversal)
- ✅ Not be hidden files (starting with `.`)

**Invalid examples:**
```yaml
# Bad - absolute paths
inbox_dir: /tmp/inbox

# Bad - path traversal
inbox_dir: ../escape

# Bad - hidden files
inbox_dir: .hidden

# Bad - environment variables
media:
  default_dir: $HOME/data
```

---

## Safety Guarantees

### 1. Path Containment

khayal **never** writes or reads files outside the vault structure:

| Operation | Boundary | Behavior |
|-----------|----------|----------|
| Write note | `<inbox_dir>/*` | Constrained |
| Write media | `<inbox_dir>/<media.default_dir>/*` | Constrained |
| Read media | Must be within `<inbox_dir>/*` | Validated |
| Update note | Must be within `<inbox_dir>/*` | Validated |
| Delete note | Must be within `<inbox_dir>/*` | Validated |

### 2. Soft Delete

Notes are **never** hard-deleted:

```
DeleteNote() → moves to <inbox_dir>/.khayal-trash/
```

Files remain in trash until manually cleaned.

### 3. Atomic Writes

All file writes use atomic operations:

```
1. Write to temp file in same directory
2. Flush and sync
3. Rename temp → target
```

This ensures no partial writes or corruption.

### 4. UTF-8 Validation

All content is validated before writing:
- Invalid UTF-8 sequences are replaced with `U+FFFD`
- Notes with invalid content are rejected

### 5. Filename Sanitization

Illegal characters are replaced:
```
< > : " / \ | ? * \x00-\x1f → _
```

---

## Error Codes

### Vault Errors

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VAULT_PATH_NOT_ABSOLUTE` | 400 | Source path must be absolute |
| `VAULT_PATH_OUTSIDE_VAULT` | 400 | Path must be within vault |
| `VAULT_PATH_OUTSIDE_INBOX` | 400 | Path must be within inbox |
| `VAULT_NOTE_NOT_FOUND` | 404 | Note not found in inbox |

### Usage in API

When vault operations fail, the API returns structured errors:

```json
{
  "error": "path must be within inbox",
  "code": "VAULT_PATH_OUTSIDE_INBOX",
  "status": 400
}
```

---

## Media Storage

### Storage Strategy

Media files can be stored in different locations based on config:

```yaml
vault:
  media:
    strategy:
      image: vault    # Store in vault (default)
      pdf: vault     # Store in vault
      audio: config  # Store in ~/.config/khayal/media/
      video: config  # Store in ~/.config/khayal/media/
```

### Supported Media Types

| Type | Strategy Options | Notes |
|------|-----------------|-------|
| Image | `vault`, `config` | PNG, JPG, GIF, WEBP |
| PDF | `vault`, `config` | Future |
| Audio | `vault`, `config` | Future |
| Video | `vault`, `config` | Future |

---

## Programmatic Access

### From Go Code

```go
import "github.com/rawnaqs/khayal/internal/vault"

// Create writer with config
writer, err := vault.NewWriter(cfg, configPath)

// Write a note
note := &vault.Note{
    Metadata: vault.NoteMetadata{
        Type:   "text",
        Status: "done",
        Tags:   []string{"golang", "notes"},
    },
    Title: "My Note",
    Raw:   "Note content...",
}
notePath, err := writer.WriteNote(note, jobID)

// Resolve paths safely
absolutePath := writer.ResolvePath("khayal/2026-03-19-note.md")

// Check if path is in vault
if writer.IsPathInVault(absolutePath) {
    // Safe to access
}
```

### Path Helper Methods

| Method | Description |
|--------|-------------|
| `writer.BasePath()` | Vault root directory |
| `writer.InboxPath()` | Inbox directory |
| `writer.MediaPath()` | Media subdirectory |
| `writer.ResolvePath(relative)` | Resolve relative path to absolute |
| `writer.NoteExists(path)` | Check if note exists (validates path) |
| `writer.IsPathInVault(path)` | Check if path is in vault |
| `writer.IsPathInInbox(path)` | Check if path is in inbox |

---

## Security Considerations

### Path Traversal Protection

All file operations validate paths:

```go
func (w *Writer) ensurePathInInbox(path string) error {
    if !strings.HasPrefix(path, w.inboxPath) {
        slog.Warn("vault path validation failed",
            "reason", "outside_inbox",
            "path", path)
        return fmt.Errorf("%w: %s", ErrVaultPathOutsideInbox, path)
    }
    return nil
}
```

### Logging

Validation failures are logged for security monitoring:
```
WARN vault path validation failed reason=outside_inbox path=/etc/passwd
```

### What Gets Logged

| Event | Log Level | Details |
|-------|-----------|---------|
| Path not absolute | WARN | Path value |
| Path outside vault | WARN | Path value |
| Path outside inbox | WARN | Path value |
| Note not found | WARN | Note path |

---

## Trash Management

### Automatic Trash Location

Trash is always within the inbox:

```
<vault>/
└── <inbox_dir>/
    └── .khayal-trash/     # Never outside inbox
```

### Manual Cleanup

Trash can be cleaned via:

```bash
khayal vault clean-trash    # Future command
```

Or by manually deleting files in `.khayal-trash/`.

---

## Upgrading from Previous Versions

### v1 → v1.1

**Changes:**
- `media.default_dir` now relative to `inbox_dir` (was relative to vault)
- Default changed from `khayal/media` to `media`
- Trash moved from `<vault>/.khayal-trash` to `<inbox_dir>/.khayal-trash`

**Migration:**
```bash
# Move existing trash to new location
mv ~/.config/khayal/vault/.khayal-trash \
   ~/.config/khayal/vault/khayal/.khayal-trash
```

---

## FAQ

### Q: Can I use a vault outside my home directory?
Yes. Absolute paths work:
```yaml
vault:
  path: /mnt/data/brain
```

### Q: Can I change the inbox directory name?
Yes. The name is configurable:
```yaml
vault:
  inbox_dir: captures  # Notes go to /vault/captures/
```

### Q: What happens if I delete a note externally?
Notes deleted outside khayal are not tracked. The search index may reference missing files.

### Q: Can I move the vault?
Yes, but:
1. Stop khayal server
2. Move files
3. Update `vault.path` in config.yaml
4. Restart khayal

### Q: Are media files backed up with the vault?
Yes, if they're stored in the vault (default). Files in `~/.config/khayal/media/` are separate.
