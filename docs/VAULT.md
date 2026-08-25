# Khayal Vault Documentation

> Vault structure, path handling, and safety guarantees. Updated: 2026-04-29

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
DELETE /v1/note?path=...    → moves to <inbox_dir>/.khayal-trash/<name>.<unix-ts>
                              and purges FTS/chunks/entities rows
kl delete <path-or-id>      → same, with confirmation prompt (--yes skips)
PWA note view → trash icon  → two-step confirm
```

Deleting a note removes it from search immediately; the file remains in
trash until manually cleaned. Path-safety violations (outside inbox) are
rejected with 400; missing files return 404.

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

## Frontmatter Entities Block (v1.1)

Every processed note carries an `entities:` block between `tags:` and
`history:` — always present, even when empty, so all notes share the same
frontmatter shape. Values are LLM-extracted and normalized by khayal
(amounts become plain integer strings).

```yaml
entities:
  people:
    - John Doe
  amounts:
    - 2000        # "$2,000" and "2k" both normalize to 2000
  dates:
    - March 2024
  places:  []
  urls:  []
  orgs:
    - Acme Corp
```

Field order is fixed: people, amounts, dates, places, urls, orgs.
At most 10 values per field. The same entities live in the SQLite
`entities` table for fast lookup.

## Frontmatter Connections Block (v1.1)

When proactive connections find related older notes, khayal writes them
into a `connections:` block as verified Obsidian wikilinks — only targets
that exist on disk become links, and the body text is never touched.

```yaml
connections:
  - "[[2026-08-21-meeting-with-bob]]"
```

Idempotent: reprocessing replaces the block; no stale links survive.
The same data is available via the queue API (`jobs.result`).

## Managed File: memory.md (v1.1)

`<inbox_dir>/memory.md` is khayal-maintained long-term memory: the LLM
periodically consolidates recent captures into headed sections (# Memory /
## About the author / ## People / ## Ongoing threads / ## Preferences) and
its contents are injected back into enrichment prompts. It is excluded from
indexing, connections, and reindex scans. You may edit it by hand — manual
changes survive until the next merge-style consolidation. Filename is
configurable via `memory.file`. Consolidation is not a cron — it runs on
the background worker after captures, throttled by
`memory.consolidation_interval_hours` / `memory.new_persons_threshold`.

Consolidation hardening: a dedicated `llm.consolidation_model` may be
configured (defaults to `text_model`); generation runs at temperature 0.2
and passes a structural sanitizer that truncates echoed prompt labels,
validates the canonical headings, and collapses duplicated headings with
their content — invalid output retries instead of overwriting the file.

## Reading Notes (Reader)

The `vault.Reader` reads and parses markdown notes from the vault inbox. It parses YAML frontmatter and extracts markdown sections.

### NoteContent

```go
type NoteContent struct {
    // Frontmatter fields
    Created     string   `yaml:"created"`
    Updated     string   `yaml:"updated"`
    Type        string   `yaml:"type"`
    Status      string   `yaml:"status"`
    Tags        []string `yaml:"tags"`
    SourceURL   string   `yaml:"source_url,omitempty"`
    SourceFile  string   `yaml:"source_file,omitempty"`
    UserContext string   `yaml:"user_context,omitempty"`
    Entities    map[string]interface{} `yaml:"entities,omitempty"`
    Related     []string `yaml:"related,omitempty"`

    // Sections (parsed from markdown body)
    Title       string
    Summary     string
    KeyIdeas    []string
    Raw         string
    Description string
    Source      string
}
```

### Usage

```go
reader := vault.NewReader(vaultPath, "inbox")

// Read a note (path is relative to vault, e.g., "inbox/test.md")
note, err := reader.ReadNote("inbox/test.md")
if err != nil {
    // handles path traversal, missing files, parse errors
}

fmt.Println(note.Title)     // Extracted from first # heading or frontmatter
fmt.Println(note.Summary)   // Content under ## Summary
fmt.Println(note.KeyIdeas)  // List items under ## Key Ideas
fmt.Println(note.Raw)       // Content under ## Raw
fmt.Println(note.Tags)      // From frontmatter
```

### Path Safety

The Reader enforces the same path containment guarantees as the Writer:

- Notes can only be read from within `<vault>/<inbox_dir>/`
- Path traversal (`../../../etc/passwd`) is rejected
- Only relative paths under the inbox are accepted

### Markdown Section Extraction

The Reader recognizes these section headers in the markdown body:

| Header | Maps to |
|--------|---------|
| `# Title` | `NoteContent.Title` |
| `## Summary` | `NoteContent.Summary` |
| `## Key Ideas` | `NoteContent.KeyIdeas` (list items) |
| `## Raw` | `NoteContent.Raw` |
| `## Description` | `NoteContent.Description` |
| `## Source` | `NoteContent.Source` |

Notes without frontmatter are still parsed — the title is extracted from the first `#` heading and the entire content is set as `Raw`.

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
| `writer.ResolveMediaPath(relative)` | Resolve media file path to absolute |
| `writer.NoteExists(path)` | Check if note exists (validates path) |
| `writer.IsPathInVault(path)` | Check if path is in vault |
| `writer.IsPathInInbox(path)` | Check if path is in inbox |
| `reader.ReadNote(path)` | Read and parse a note by relative path |

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
