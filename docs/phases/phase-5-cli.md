# Phase 5: CLI Implementation Plan

> Command line interface implementation — khayal (server admin) + kl (client).
> Updated: 2026-03-20

## Architecture

### Directory Structure

```
cmd/
├── khayal/                          # Server admin CLI
│   ├── main.go                      # Cobra root
│   │
│   ├── internal/                    # khayal-only utilities
│   │   ├── config.go               # Config loading/writing
│   │   ├── pid.go                  # PID file management
│   │   ├── deps.go                 # Dependency checking
│   │   ├── output.go                # Styled output helpers
│   │   └── errors.go                # Error formatting + exit codes
│   │
│   └── commands/
│       ├── init.go                  # First-run setup
│       ├── start.go                 # Start server + deps check
│       ├── stop.go                  # Graceful shutdown
│       ├── restart.go               # Stop + Start
│       ├── status.go                # Bubble Tea TUI dashboard
│       ├── reindex.go               # Progress bar reindex
│       ├── version.go                # Version info
│       ├── logs.go                  # Log tail
│       └── config.go                # View config
│
└── kl/                              # Client CLI
    ├── main.go                      # Cobra root
    │
    ├── internal/                    # kl-only utilities
    │   └── api/
    │       └── client/
    │           └── client.go       # HTTP client
    │
    └── commands/
        ├── root.go                  # Default capture (text)
        ├── capture/
        │   ├── url.go               # Capture URL
        │   └── image.go              # Capture image
        ├── search.go                 # Search with glamour
        ├── recent.go                 # Recent captures
        ├── browse.go                 # Browse by tag/person/amount
        ├── stats.go                  # ASCII bar charts
        ├── status.go                 # Lightweight check
        ├── init.go                  # Huh wizard
        └── config/
            ├── set.go                # Set config value
            ├── get.go                # Get config value
            └── view.go               # View all config
```

### Design Principles

| Principle | Implementation |
|-----------|----------------|
| **Separation of concerns** | Thin commands delegate to internal packages |
| **No shared code** | khayal and kl are completely separate |
| **Scoped utilities** | Internal packages live inside cmd/ only |
| **goreleaser builds** | Builds from cmd/khayal and cmd/kl |

---

## Dependencies

### Required Packages

```go
require (
    github.com/spf13/cobra v1.9+
    charm.land/lipgloss/v2
    charm.land/huh/v2
    charm.land/bubbletea/v2
    charm.land/bubbles/v2
    github.com/charmbracelet/glamour
    github.com/rawnaqs/theme
)
```

### Import Paths (v2 Libraries)

```go
import (
    "github.com/spf13/cobra"
    "charm.land/lipgloss/v2"
    "charm.land/huh/v2"
    "charm.land/bubbletea/v2"
    "charm.land/bubbles/v2"
    "github.com/charmbracelet/glamour"
    "github.com/rawnaqs/theme"
)
```

---

## khayal Commands (v1)

### Command Reference

| Command | File | Description |
|---------|------|-------------|
| `khayal init` | `commands/init.go` | First-run setup |
| `khayal start` | `commands/start.go` | Start server + worker |
| `khayal stop` | `commands/stop.go` | Graceful shutdown |
| `khayal restart` | `commands/restart.go` | Stop + start |
| `khayal status` | `commands/status.go` | Bubble Tea TUI |
| `khayal reindex` | `commands/reindex.go` | Rebuild embeddings |
| `khayal version` | `commands/version.go` | Version info |
| `khayal logs` | `commands/logs.go` | Tail logs |
| `khayal config` | `commands/config.go` | View config |

### Internal Packages

| Package | Purpose |
|---------|---------|
| `internal/config.go` | Load/write config.yaml |
| `internal/pid.go` | PID file for stop/restart |
| `internal/deps.go` | Check ollama only |
| `internal/output.go` | Styled terminal output |
| `internal/errors.go` | Exit codes, error formatting |

### Sample Output

#### khayal start
```
khayal v0.1.0

loading config...
checking dependencies...
  ✓ ollama        http://localhost:11434

  ✓ vault         /absolute/path/to/vault
  ✓ db            /absolute/path/to/khayal.db
  ✓ log           /absolute/path/to/logs/khayal.log
  ✓ queue         ready
  ✓ llm           ollama
  ✓ worker        started
  ✓ server        127.0.0.1:1133
  ✓ pid           12345

khayal is running.
press ctrl+c to stop
```

#### khayal status (TUI)
```
┌─ Server ─────────────────────────────────────┐
│  host      http://127.0.0.1:1133            │
│  uptime    3h 24m                           │
│  pid       12847                            │
└──────────────────────────────────────────────┘
┌─ Queue ─────────────────────────────────────┐
│  pending      2    ██░░░░░░░░░░░░           │
│  processing   1    ██░░░░░░░░░░░░  image    │
│  done       147                               │
└──────────────────────────────────────────────┘
```

---

## kl Commands (v1)

### Command Reference

| Command | File | Description |
|---------|------|-------------|
| `kl "text"` | `commands/root.go` | Default capture (text) |
| `kl capture url` | `commands/capture/url.go` | Capture URL |
| `kl capture image` | `commands/capture/image.go` | Capture image |
| `kl search` | `commands/search.go` | Search vault |
| `kl recent` | `commands/recent.go` | Recent captures |
| `kl browse` | `commands/browse.go` | Browse by tag/person |
| `kl stats` | `commands/stats.go` | Vault statistics |
| `kl status` | `commands/status.go` | Lightweight check |
| `kl init` | `commands/init.go` | Huh wizard setup |
| `kl config set` | `commands/config/set.go` | Set value |
| `kl config get` | `commands/config/get.go` | Get value |
| `kl config view` | `commands/config/view.go` | View all |

### Internal Packages

| Package | Purpose |
|---------|---------|
| `internal/api/client/client.go` | HTTP client for API calls |

### Sample Output

#### kl search
```
3 results · hybrid · 42ms

──────────────────────────────────────────────────────────
khayal/2019-03-03-designer.md                          0.94
March 3, 2019 · #finance #design

...paid John Doe $2,000 for logo design work...

──────────────────────────────────────────────────────────
```

#### kl stats
```
vault · ~/brain

total         2,847   notes
this week        23   ████░░░░░░░░░░░░░░░
this month       94   ████████░░░░░░░░░░░

top tags
#react           142  ████████████████████
#go               98  ████████████████░░░░
```

#### kl init (huh wizard)
```
? Server address
  http://127.0.0.1:1133

? Token
  ••••••••••••••••••

  ✓ connected!

  saved to ~/.config/khayal/kl.yaml

  you're ready. try: kl "your first thought"
```

---

## UX/UI Guidelines

### Typography System

Using `github.com/rawnaqs/theme` for all styling:

```go
import "github.com/rawnaqs/theme"

// Pre-built styles from theme
theme.Primary           // Gold light (#E8B86D)
theme.Muted             // Gold dark (#8B6020)
theme.Dim               // Gold dim (#3A2E18)
theme.SuccessStyle      // Success green (#4A7C59), bold
theme.ErrorStyle        // Error red (#8B3A3A), bold
theme.WarningStyle      // Warning gold (#C9933A), bold
theme.ProcessingStyle   // Italic gold dark
theme.Tag               // Gold background, dark text
theme.TagMuted          // Muted tag variant
theme.Panel             // Panel with rounded border
theme.PanelGold         // Panel with gold border
```

### Error Message Format

Never show raw Go errors. Every error tells the user what to do next.

```
✗ short description of what failed

  → action to try first
  → action to try second
  → where to get more info
```

### Exit Codes

```go
const (
    ExitSuccess = 0  // Success
    ExitUser    = 1  // Wrong args, not found
    ExitServer  = 2  // Unreachable, auth failed
    ExitVault   = 3  // Write failed, permission
    ExitDep     = 4  // Ollama missing
)
```

### Spinner Rules

| Operation | Duration | Show Spinner? |
|-----------|----------|---------------|
| Text capture | <100ms | No |
| URL/image capture | varies | Yes (>200ms) |
| Search | varies | Only if >200ms |
| Status check | <200ms | No |
| Init wizard | varies | Yes (network) |

---

## Implementation Order

### Phase 5A: khayal CLI

1. [x] Set up Cobra root with command groups
2. [x] Implement `khayal version`
3. [x] Implement `khayal init` (config generation)
4. [x] Implement `khayal start` (deps check, step output)
5. [x] Implement `khayal stop` (PID file)
6. [x] Implement `khayal config` (view config)
7. [x] Implement `khayal logs` (log tail)
8. [x] Implement `khayal restart`
9. [x] Implement `khayal status` (Bubble Tea TUI)
10. [x] Implement `khayal reindex` (progress bar)

### Phase 5B: kl CLI

1. [x] Set up Cobra root with command groups
2. [x] Create API client (`internal/api/`)
3. [x] Implement default capture (`kl "text"`)
4. [x] Implement `kl capture url`
5. [x] Implement `kl capture image`
6. [x] Implement `kl search` (glamour)
7. [x] Implement `kl status`
8. [x] Implement `kl recent`
9. [x] Implement `kl browse`
10. [x] Implement `kl stats` (ASCII charts)
11. [x] Implement `kl init` (huh wizard)
12. [x] Implement `kl config set/get/view`

---

## Config Files

### khayal config (~/.config/khayal/config.yaml)

```yaml
vault:
  path: ~/Documents/brain
  inbox_dir: khayal

server:
  host: 127.0.0.1
  port: 1133
  token: ""
  log_file: ~/.config/khayal/logs/khayal.log

llm:
  provider: ollama
  ollama_host: http://localhost:11434
  embed_model: nomic-embed-text
  text_model: qwen2.5:3b
  vision_model: moondream

worker:
  max_workers: 1
  max_retries: 3

db:
  path: ~/.config/khayal/khayal.db

log:
  level: info
  worker_level: info
  file: ~/.config/khayal/logs/khayal.log
```

### kl config (~/.config/khayal/kl.yaml)

```yaml
host: http://127.0.0.1:1133
token: your-token-here
```

---

## goreleaser Configuration

```yaml
# .goreleaser.yml
builds:
  - id: khayal
    dir: ./cmd/khayal
    main: ./main.go
    binary: khayal
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64

  - id: kl
    dir: ./cmd/kl
    main: ./main.go
    binary: kl
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
```

---

## Checklist

### khayal (server admin)
- [x] khayal init
- [x] khayal start (dep checks, verbose output)
- [x] khayal stop (graceful)
- [x] khayal restart
- [x] khayal status (Bubble Tea TUI)
- [x] khayal reindex (progress bar, mtime check)
- [x] khayal version
- [x] khayal logs
- [x] khayal config (token redacted)

### kl (client)
- [x] kl "text" (default capture)
- [x] kl capture url
- [x] kl capture image
- [x] kl search (Glamour, proper format)
- [x] kl recent (grouped by day)
- [x] kl browse (tag/person/amount)
- [x] kl stats (ASCII charts)
- [x] kl status (lightweight)
- [x] kl init (Huh wizard, validate before save)
- [x] kl config set/get/view

### Shared
- [x] Typography (rawnaqs/theme)
- [x] Error messages (actionable)
- [x] Spinner rules
- [x] Exit codes
- [x] Help format (examples first)

---

## Next Phase

[Phase 6: PWA](phase-6-pwa.md)

---

## Notes

- **Two binaries**: `khayal` for server admin, `kl` for client
- **UX spec**: See SPEC.md "CLI UX" section for detailed output formats
- **khayal**: Sysadmin feel — dense, verbose, operational
- **kl**: Personal feel — minimal, fast, one-line
- **Build**: Both binaries built via goreleaser from cmd/ directories
