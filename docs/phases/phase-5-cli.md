# Phase 5: CLI

> Command line interface implementation — khayal (server) + kl (client). Updated: 2026-03-18

## Two CLI Tools

This phase implements TWO separate CLI tools:

| Tool | Binary | Purpose |
|------|--------|---------|
| `khayal` | Server admin | init, start, stop, status, reindex, logs |
| `kl` | Client | capture, search, recent, browse, stats, init, config |

## Goals

### khayal (server admin)
- [ ] `khayal init` — first-run setup
- [ ] `khayal start` — start server + worker with dependency checks
- [ ] `khayal stop` — graceful shutdown (wait for jobs)
- [ ] `khayal restart` — stop + start
- [ ] `khayal status` — Bubble Tea admin dashboard
- [ ] `khayal reindex` — rebuild chunk embeddings with progress
- [ ] `khayal version` — version + build info
- [ ] `khayal logs` — tail logs
- [ ] `khayal config` — view config (token redacted)

### kl (client)
- [ ] `kl "text"` — capture text (default command)
- [ ] `kl --url` — capture URL
- [ ] `kl --image` — capture image
- [ ] `kl search` — search vault
- [ ] `kl recent` — recent captures
- [ ] `kl browse` — browse by tag/person/amount
- [ ] `kl stats` — vault statistics
- [ ] `kl status` — lightweight server check
- [ ] `kl init` — Huh wizard setup
- [ ] `kl config` — config management

## Directory Structure

```
cmd/
├── khayal/                      # Server admin CLI
│   └── main.go
└── kl/                          # Client CLI
    └── main.go

cli/                            # Shared code (used by both)
├── main.go
├── root.go
├── capture.go
├── search.go
├── recent.go
├── browse.go
├── stats.go
├── status.go
├── init.go
└── config.go

internal/
├── api/client/                  # Shared HTTP client
│   └── client.go
└── config/
    └── config.go
```

## Dependencies

```go
require (
    github.com/spf13/cobra v1.8.0
    github.com/charmbracelet/lipgloss v0.9.1
    github.com/charmbracelet/glamour v0.6.0
    github.com/charmbracelet/huh v0.3.0
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/charmbracelet/bubbles v0.16.1
    github.com/rawnaqs/theme v0.0.0
)
```

---

## Part A: khayal CLI (Server Admin)

### Directory Structure

```
cmd/khayal/
├── main.go
└── commands/
    ├── init.go
    ├── start.go
    ├── stop.go
    ├── restart.go
    ├── status.go
    ├── reindex.go
    ├── version.go
    ├── logs.go
    └── config.go
```

### Implementation Details

#### khayal init

**File:** `cmd/khayal/commands/init.go`

```go
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "First-run setup — generates config.yaml + token",
    RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
    // Create ~/.config/khayal/
    // Generate 32-byte hex token
    // Write config.yaml with 600 permissions
    // Create log directory
    // Print token ONCE (never again)
    
    fmt.Println("creating config directory...  ~/.config/khayal/")
    fmt.Println("generating token...           " + token[:16] + "... (save this — shown once)")
    fmt.Println("writing config...             ~/.config/khayal/config.yaml (600)")
    fmt.Println("creating log directory...     ~/.config/khayal/logs/")
    // Print next steps...
}
```

Rules:
- Token: 32-byte hex, shown once only
- Config: 600 permissions
- End with clear next steps

#### khayal start

**File:** `cmd/khayal/commands/start.go`

```go
var startCmd = &cobra.Command{
    Use:   "start",
    Short: "Start server + worker, run dependency checker",
    RunE:  runStart,
}

func runStart(cmd *cobra.Command, args []string) error {
    // 1. Check dependencies (ollama, ffmpeg, yt-dlp, easyocr)
    // 2. Load config
    // 3. Start server
    // 4. Start worker
    // 5. Print "khayal is running"
    
    // Dependency check output:
    fmt.Println("checking dependencies...")
    fmt.Println("  ✓ ollama        localhost:11434")
    fmt.Println("      models: qwen2.5:3b · moondream · nomic-embed-text")
    fmt.Println("  ✓ ffmpeg        /usr/local/bin/ffmpeg")
    fmt.Println("  ✗ yt-dlp        not found")
    fmt.Println("      → brew install yt-dlp")
    
    // End with success signal
    fmt.Println("\nkhayal is running.")
    fmt.Println("press ctrl+c to stop")
}
```

Rules:
- Show every dep check (pass and fail)
- On dep failure: show install commands
- Redact token: show first 8 chars + "..."
- End with "khayal is running"

#### khayal stop

```go
var stopCmd = &cobra.Command{
    Use:   "stop",
    Short: "Graceful shutdown",
    RunE:  runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
    // Wait for current job to complete
    // Stop worker
    // Stop server
    fmt.Println("stopping worker...    ✓ (waited for current job to finish)")
    fmt.Println("stopping server...    ✓")
    fmt.Println("khayal stopped.")
}
```

Rules:
- Wait for current job to finish
- Never kill mid-processing

#### khayal status — Bubble Tea TUI

**File:** `cmd/khayal/commands/status.go`

```go
func initialModel() (model, tea.Cmd) {
    return model{
        server: fetchServerStatus(),
        vault:  fetchVaultStatus(),
        deps:   fetchDeps(),
        queue:  fetchQueueStats(),
        system: fetchSystemStats(),
    }, nil
}

func (m model) View() string {
    // Render full admin dashboard
    // Server: host, uptime, pid
    // Vault: path, notes, indexed %
    // Dependencies: ollama, ffmpeg, yt-dlp
    // Queue: pending, processing, done, failed
    // System: memory, db size, log size
}
```

Rules:
- Refresh every 3 seconds
- Show progress bar for indexed %
- Show current job (type + ID) when processing
- Keyboard hints at bottom: q, r, l, c, ?

#### khayal reindex

**File:** `cmd/khayal/commands/reindex.go`

```go
var reindexCmd = &cobra.Command{
    Use:   "reindex",
    Short: "Rebuild all chunk embeddings from vault",
    RunE:  runReindex,
}

var force bool

func init() {
    reindexCmd.Flags().BoolVar(&force, "force", false, "reindex everything regardless of mtime")
}

func runReindex(cmd *cobra.Command, args []string) error {
    // Scan vault
    // Show notes found, already indexed, to reindex
    // Progress bar with percentage + ETA
    // Show each note as it completes (last 4 visible)
    // Check mtime before re-embedding (skip if unchanged unless --force)
}
```

Rules:
- Progress bar: percentage + ETA
- Show last 4 notes as they complete
- Ctrl+c: graceful stop, show progress, resumable
- Check mtime (skip unless --force)

#### khayal logs

```go
var logsCmd = &cobra.Command{
    Use:   "logs",
    Short: "Tail ~/.config/khayal/logs/khayal.log",
    RunE:  runLogs,
}

func runLogs(cmd *cobra.Command, args []string) error {
    // Simple log tail
    // No flags
    // Ctrl+c to exit
}
```

#### khayal version

```
khayal version

  khayal v0.1.0
  commit  a3f9c2e
  built   2024-03-16T10:00:00Z
  go      1.22.1
```

#### khayal config

```go
var configCmd = &cobra.Command{
    Use:   "config",
    Short: "View current config (token redacted)",
    RunE:  runConfig,
}

func runConfig(cmd *cobra.Command, args []string) error {
    // Load config
    // Print all config with token redacted
    // Read-only (edit the file directly)
}
```

---

## Part B: kl CLI (Client)

### Directory Structure

```
cmd/kl/
└── main.go

cli/
├── root.go          # Cobra root + default capture
├── capture.go       # Text, URL, image capture
├── search.go        # Search with Glamour
├── recent.go        # Recent captures
├── browse.go        # Browse by tag/person/amount
├── stats.go         # Vault statistics
├── status.go        # Lightweight status
├── init.go          # Huh wizard
└── config.go        # Config management
```

### Implementation

#### kl Root — Default Capture

**File:** `cli/root.go`

```go
var rootCmd = &cobra.Command{
    Use:   "kl",
    Short: "Your private second brain",
    Long: `Capture thoughts, images, and articles.
Search your knowledge base semantically.

Examples:
  kl "my thought"
  kl --url https://example.com
  kl --image screenshot.png
  kl search "distributed systems"
  kl status`,
    // Default command is capture
    RunE: runCapture,
}

func init() {
    // Register all subcommands
    // Make "kl thought" work without "capture" subcommand
    rootCmd.SetArgs(append([]string{"capture"}, rootCmd.Flags().Args()...))
}
```

#### kl capture — Output Styles

**File:** `cli/capture.go`

Output must be minimal and instant:

```
✓ saved · #react #performance · 3ms          # text done
⏳ queued · article · id: abc123              # URL queued
⏳ queued · image · id: def456                 # image queued
✓ saved · unprocessed                          # ollama down
✗ cannot reach khayal at http://...            # server unreachable
```

Using `rawnaqs/theme`:

```go
import styles "github.com/rawnaqs/theme/custom/go"

// Success
fmt.Println(styles.SuccessStyle.Render("✓ saved"))

// Processing
fmt.Println(styles.ProcessingStyle.Render("⏳ queued · " + result.Type))

// Error
fmt.Println(styles.ErrorStyle.Render("✗ cannot reach khayal..."))
```

Rules (from UX spec):
- No spinner for text capture (<100ms)
- Spinner only for image/URL if >200ms
- Error messages: actionable, never raw Go errors

#### kl search

**File:** `cli/search.go`

```go
var (
    searchLimit int
    searchMode  string
    searchFrom  string
    searchTo    string
    verbose     bool
)

var searchCmd = &cobra.Command{
    Use:   "search <query>",
    Short: "Search your vault",
    Args:  cobra.ExactArgs(1),
    RunE:  runSearch,
}

func init() {
    searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 5, "max results (max: 50)")
    searchCmd.Flags().StringVar(&searchMode, "mode", "hybrid", "hybrid|keyword|semantic")
    searchCmd.Flags().StringVar(&searchFrom, "from", "", "start date YYYY-MM-DD")
    searchCmd.Flags().StringVar(&searchTo, "to", "", "end date YYYY-MM-DD")
    searchCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show scores and debug info")
}
```

Output format (from UX spec):

```
$ kl search "paid john money"

  3 results · hybrid · 42ms

  ──────────────────────────────────────────────────────────
  khayal/2019-03-03-designer.md                          0.94
  March 3, 2019 · #finance #design

  ...paid John Doe $2,000 for logo design work...

  ──────────────────────────────────────────────────────────
  ...
```

Rules:
- Score right-aligned
- Date prominent
- Tags on second line
- Dividers between results
- No color on content

#### kl recent

**File:** `cli/recent.go`

```go
var (
    recentDays int
    recentType string
)

var recentCmd = &cobra.Command{
    Use:   "recent",
    Short: "Show recent captures",
    RunE:  runRecent,
}

func init() {
    recentCmd.Flags().IntVar(&recentDays, "days", 1, "days to show")
    recentCmd.Flags().StringVar(&recentType, "type", "", "filter by type: text|image|article")
}
```

Output:
```
$ kl recent

  today

  14:23  text     useEffect cleanup runs after every render    #react
  14:18  image    whiteboard-arch.png                         #system-design
  13:55  article  CAP theorem explained                       #distributed

  yesterday

  18:44  text     meeting with Sarah re contract              #work
  ...
```

Rules:
- Group by day
- Type badge: text/image/article (consistent width)
- Truncate at 7 per day, show "N more →"

#### kl browse

**File:** `cli/browse.go`

```go
var (
    browseTag    string
    browsePerson string
    browseAmount int
    browseAll    bool
)

var browseCmd = &cobra.Command{
    Use:   "browse",
    Short: "Browse notes by tag, person, or amount",
    RunE:  runBrowse,
}

func init() {
    browseCmd.Flags().StringVar(&browseTag, "tag", "", "browse by tag")
    browseCmd.Flags().StringVar(&browsePerson, "person", "", "browse by person")
    browseCmd.Flags().IntVar(&browseAmount, "amount", 0, "browse by amount")
    browseCmd.Flags().BoolVar(&browseAll, "all", false, "show all results")
}
```

Output:
```
$ kl browse --tag react

  #react · 23 notes

  2024-03-16  useEffect cleanup runs after every render
  2024-03-10  React Server Components mental model
  ...
```

#### kl stats

**File:** `cli/stats.go`

```go
var statsCmd = &cobra.Command{
    Use:   "stats",
    Short: "Show vault statistics",
    RunE:  runStats,
}
```

Output with ASCII bar charts:
```
$ kl stats

  vault · ~/brain

  total         2,847   notes
  this week        23   ████░░░░░░░░░░░░░░░
  this month       94   ████████░░░░░░░░░░░

  top tags
  #react           142  ████████████████████
  #go               98  ████████████████░░░░
  ...
```

Rules:
- ASCII bar charts (normalized)
- Top 5 tags, top 3 people
- Pure SQL, no LLM

#### kl status — Lightweight

**File:** `cli/status.go`

```go
var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "Quick server + queue check",
    RunE:  runStatus,
}
```

Output (6 lines max):
```
$ kl status

  ✓ khayal v0.1.0 · http://100.x.x.x:1133

  queue
    processing   1   image
    pending      2
    failed       0

  last capture  14:23 · useEffect cleanup runs after...
```

#### kl init — Huh Wizard

**File:** `cli/init.go`

```go
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Setup kl configuration",
    RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
    form := huh.NewForm(
        huh.NewGroup(
            huh.NewInput().Title("Server address").Value(&cfg.Host).Placeholder("http://127.0.0.1:1133"),
        ),
        huh.NewGroup(
            huh.NewInput().Title("Token").Value(&cfg.Token).Placeholder("Enter your token").Mask('•'),
        ),
    )
    
    // Validate connection BEFORE writing config
    // If fails: show error, stay in wizard
    
    // Write to ~/.config/khayal/kl.yaml
    // End with "you're ready. try: kl \"your first thought\""
}
```

Rules:
- Validate connection BEFORE writing
- Token masked with bullets
- End with first command to try

#### kl config

**File:** `cli/config.go`

```go
var configCmd = &cobra.Command{
    Use:   "config",
    Short: "Manage configuration",
}

var configSetCmd = &cobra.Command{
    Use:   "set <key> <value>",
    Short: "Set a config value",
    Args:  cobra.ExactArgs(2),
    RunE:  runConfigSet,
}

var configGetCmd = &cobra.Command{
    Use:   "get <key>",
    Short: "Get a config value",
    Args:  cobra.ExactArgs(1),
    RunE:  runConfigGet,
}

var configViewCmd = &cobra.Command{
    Use:   "view",
    Short: "View all config",
    RunE:  runConfigView,
}
```

Silent success:
```
$ kl config set host http://100.x.x.x:1133
  ✓ host updated
    ~/.config/khayal/kl.yaml
```

---

## Shared: Typography System

Both tools use `rawnaqs/theme`:

```go
import styles "github.com/rawnaqs/theme/custom/go"

// Usage
styles.Primary.Render("text")        // GoldLight #E8B86D
styles.Muted.Render("text")          // GoldDark #8B6020
styles.SuccessStyle.Render("✓ saved") // Green #4A7C59
styles.ErrorStyle.Render("✗ error")   // Red #8B3A3A
styles.ProcessingStyle.Render("⏳")   // GoldDark italic
styles.Tag.Render("#tag")             // Gold bg, dark text
```

Rules:
- One color family: gold on dark
- Color on metadata only — never on content
- Error color only for actual errors

---

## Shared: Error Message Format

Never show raw Go errors:

```
✗ short description of what failed

  → action to try first
  → action to try second
  → where to get more info
```

```go
func handleServerUnreachable(err error) error {
    return fmt.Errorf("✗ cannot reach khayal at %s\n\n  → is khayal running?    ssh mac-air khayal start\n  → wrong address?        kl config set host <address>\n  → check logs            ssh mac-air khayal logs", cfg.Host)
}
```

---

## Shared: Spinner Rules

Operations under 200ms: NO spinner (jarring)
Operations over 200ms: SHOW spinner

| Command | Show Spinner? |
|---------|---------------|
| `kl "thought"` | No (<100ms) |
| `kl --url` | Yes (upload) |
| `kl --image` | Yes (upload) |
| `kl search` | Only with reranking |
| `kl status` | No (<200ms) |
| `kl recent` | No (<50ms) |
| `kl stats` | No (<50ms) |
| `khayal start` | No (step-by-step) |
| `khayal reindex` | No (progress bar) |
| `khayal stop` | No (text) |
| `kl init` | Yes (network) |

---

## Shared: Exit Codes

```go
import "os"

// 0 - success
// 1 - user error (wrong args, not found)
// 2 - server error (unreachable, auth failed)
// 3 - vault error (write failed, permission)
// 4 - dep error (ollama, ffmpeg missing)
```

---

## Shared: Help Format

Examples first, then flags:

```go
searchCmd.SetHelpTemplate(`Search your vault using keyword and semantic search.

Usage:
  kl search <query> [flags]

Examples:
  kl search "paid john money"
  kl search "react hooks" --mode keyword
  kl search "distributed systems" --from 2024-01-01

Flags:
{{.Flags}}
`)
```

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
  fallback_provider: ""
  fallback_api_key: ""

worker:
  max_workers: 1
  max_retries: 3
  retry_backoff: exponential

db:
  path: ~/.config/khayal/khayal.db
```

### kl config (~/.config/kahyyal/kl.yaml)

```yaml
host: http://127.0.0.1:1133
token: your-token-here
```

---

## Testing

```bash
go test ./cmd/khayal/... -v
go test ./cmd/kl/... -v
go test ./cli/... -v
```

## Checklist

### khayal (server admin)
- [ ] khayal init
- [ ] khayal start (dep checks, verbose output)
- [ ] khayal stop (graceful)
- [ ] khayal restart
- [ ] khayal status (Bubble Tea TUI)
- [ ] khayal reindex (progress bar, mtime check)
- [ ] khayal version
- [ ] khayal logs
- [ ] khayal config (token redacted)

### kl (client)
- [ ] kl "text" (default capture)
- [ ] kl --url
- [ ] kl --image
- [ ] kl search (Glamour, proper format)
- [ ] kl recent (grouped by day)
- [ ] kl browse (tag/person/amount)
- [ ] kl stats (ASCII charts)
- [ ] kl status (lightweight)
- [ ] kl init (Huh wizard, validate before save)
- [ ] kl config set/get/view

### Shared
- [ ] Typography (rawnaqs/theme)
- [ ] Error messages (actionable)
- [ ] Spinner rules
- [ ] Exit codes
- [ ] Help format

## Next Phase

[Phase 6: PWA](phase-6-pwa.md)

## Notes

- **Two binaries**: `khayal` for server admin, `kl` for client
- **UX spec**: See SPEC.md "CLI UX" section for detailed output formats
- **khayal**: Sysadmin feel — dense, verbose, operational
- **kl**: Personal feel — minimal, fast, warm
- Config: khayal uses `~/.config/khayal/config.yaml`, kl uses `~/.config/khayal/kl.yaml`
