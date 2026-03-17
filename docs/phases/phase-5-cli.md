# Phase 5: CLI

> User-facing command line interface (`kl`). Updated: 2026-03-17

## Goals

- [ ] Cobra root
- [ ] Capture command (`kl "text"`)
- [ ] URL capture (`kl --url`)
- [ ] Image capture (`kl --image`)
- [ ] Search command (Glamour output)
- [ ] Status command (Bubble Tea)
- [ ] Init wizard (Huh)
- [ ] Config commands

## Directory Structure

```
cli/
├── main.go
├── root.go
├── capture.go
├── search.go
├── status.go
├── init.go
└── config.go
```

## Dependencies

Add to `go.mod`:

```go
require (
    github.com/spf13/cobra v1.8.0
    github.com/charmbracelet/lipgloss v0.9.1
    github.com/charmbracelet/glamour v0.6.0
    github.com/charmbracelet/huh v0.3.0
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/charmbl/bubbles v0.16.1
)
```

## Step 5.1: Root Command

**File:** `cli/root.go`

### Structure

```go
var (
    configPath string
    host       string
    token      string
)

var rootCmd = &cobra.Command{
    Use:   "kl",
    Short: "Khayal CLI - Your private second brain",
    Long: `Capture thoughts, images, and articles.
Search your knowledge base semantically.

Examples:
  kl "my thought"
  kl --url https://example.com
  kl --image screenshot.png
  kl search "distributed systems"
  kl status`,
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file (default: ~/.config/khayal/kl.yaml)")
    rootCmd.PersistentFlags().StringVar(&host, "host", "", "server host (default: from config)")
    rootCmd.PersistentFlags().StringVar(&token, "token", "", "auth token (default: from config)")
}

func getConfig() (*ClientConfig, error) {
    // Load from kl.yaml or flags
}

type ClientConfig struct {
    Host  string `yaml:"host"`
    Token string `yaml:"token"`
}
```

## Step 5.2: Capture Command

**File:** `cli/capture.go`

### Text Capture

```bash
kl "my thought"
kl "useEffect cleanup runs after every render"
kl "meeting notes #work"  # with implicit tags
```

### Output Styles

Using `rawnaqs/theme`:
- Success: `✓ saved · #tag · 3ms`
- Queued: `⏳ queued · image · id: abc123`

### Implementation

```go
var (
    captureText  string
    captureURL   string
    captureImage string
)

var captureCmd = &cobra.Command{
    Use:   "capture [text]",
    Short: "Capture a thought, URL, or image",
    Args:  cobra.MaximumNArgs(1),
    RunE:  runCapture,
}

func init() {
    rootCmd.AddCommand(captureCmd)
    captureCmd.Flags().StringVarP(&captureURL, "url", "u", "", "capture URL")
    captureCmd.Flags().StringVarP(&captureImage, "image", "i", "", "capture image file")
}

func runCapture(cmd *cobra.Command, args []string) error {
    cfg, err := getConfig()
    if err != nil {
        return err
    }
    
    // Text from args
    if len(args) > 0 {
        captureText = args[0]
    }
    
    // Determine capture type
    var reqType, content string
    var file *os.File
    
    switch {
    case captureURL != "":
        reqType = "url"
        content = captureURL
    case captureImage != "":
        reqType = "image"
        f, err := os.Open(captureImage)
        if err != nil {
            return err
        }
        defer f.Close()
        file = f
    case captureText != "":
        reqType = "text"
        content = captureText
    default:
        return fmt.Errorf("must provide text, --url, or --image")
    }
    
    // Send request
    client := &http.Client{Timeout: 30 * time.Second}
    
    var resp *http.Response
    
    if file != nil {
        // Multipart form
        var b bytes.Buffer
        w := multipart.NewWriter(&b)
        w.WriteField("type", reqType)
        
        part, _ := w.CreateFormFile("file", filepath.Base(captureImage))
        io.Copy(part, file)
        w.Close()
        
        req, _ := http.NewRequest("POST", cfg.Host+"/v1/capture", &b)
        req.Header.Set("Content-Type", w.FormDataContentType())
        req.Header.Set("X-Khayal-Token", cfg.Token)
        
        resp, err = client.Do(req)
    } else {
        // JSON
        reqBody, _ := json.Marshal(map[string]string{
            "type":    reqType,
            "content": content,
        })
        
        req, _ := http.NewRequest("POST", cfg.Host+"/v1/capture", bytes.NewReader(reqBody))
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("X-Khayal-Token", cfg.Token)
        
        resp, err = client.Do(req)
    }
    
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    var result CaptureResponse
    json.NewDecoder(resp.Body).Decode(&result)
    
    // Output with style
    if result.Status == "done" {
        fmt.Println(styles.SuccessStyle.Render("✓ saved"))
    } else {
        fmt.Println(styles.QueuedStyle.Render("⏳ queued · "+result.Type+" · id: "+result.ID))
    }
    
    return nil
}

type CaptureResponse struct {
    ID        string `json:"id"`
    Type      string `json:"type"`
    Status    string `json:"status"`
    NotePath  string `json:"note_path"`
    CreatedAt string `json:"created_at"`
}
```

### Register as Root Alias

```go
// Make "kl thought" work without "capture" subcommand
rootCmd.SetArgs(append([]string{"capture"}, rootCmd.Flags().Args()...))
```

## Step 5.3: Search Command

**File:** `cli/search.go`

```bash
kl search "distributed systems"
kl search "react hooks" --limit 5
kl search "vim tricks" --mode semantic
```

### Implementation

```go
var (
    searchLimit int
    searchMode  string
)

var searchCmd = &cobra.Command{
    Use:   "search [query]",
    Short: "Search your knowledge base",
    Args:  cobra.ExactArgs(1),
    RunE:  runSearch,
}

func init() {
    rootCmd.AddCommand(searchCmd)
    searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 10, "max results")
    searchCmd.Flags().StringVar(&searchMode, "mode", "hybrid", "search mode: hybrid, keyword, semantic")
}

func runSearch(cmd *cobra.Command, args []string) error {
    cfg, err := getConfig()
    if err != nil {
        return err
    }
    
    query := args[0]
    
    url := fmt.Sprintf("%s/v1/search?q=%s&limit=%d&mode=%s",
        cfg.Host, url.QueryEscape(query), searchLimit, searchMode)
    
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("X-Khayal-Token", cfg.Token)
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    var result SearchResponse
    json.NewDecoder(resp.Body).Decode(&result)
    
    // Render with Glamour
    output := renderSearchResults(query, result)
    fmt.Println(output)
    
    return nil
}

func renderSearchResults(query string, resp SearchResponse) string {
    var b strings.Builder
    
    b.WriteString(fmt.Sprintf("## Search: %s\n\n", query))
    b.WriteString(fmt.Sprintf("*%d results in %dms*\n\n", resp.Total, resp.TookMs))
    
    for i, r := range resp.Results {
        b.WriteString(fmt.Sprintf("### %d. %s\n", i+1, r.Title))
        b.WriteString(fmt.Sprintf("`%s` · score: %.2f\n\n", r.NotePath, r.Score))
        b.WriteString(r.Excerpt + "\n\n")
        b.WriteString("---\n\n")
    }
    
    // Render markdown
    out, _ := glamour.Render(b.String(), "dark")
    return out
}

type SearchResponse struct {
    Query   string        `json:"query"`
    Mode    string        `json:"mode"`
    Results []SearchResult `json:"results"`
    Total   int           `json:"total"`
    TookMs  int64         `json:"took_ms"`
}

type SearchResult struct {
    ID        string  `json:"id"`
    NotePath  string  `json:"note_path"`
    Title     string  `json:"title"`
    Excerpt   string  `json:"excerpt"`
    Score     float64 `json:"score"`
    Type      string  `json:"type"`
    CreatedAt string  `json:"created_at"`
}
```

## Step 5.4: Status Command

**File:** `cli/status.go`

```bash
kl status
```

### Bubble Tea TUI

```go
type model struct {
    jobs     []Job
    selected int
}

type Job struct {
    ID        string
    Type      string
    Status    string
    NotePath  string
    CreatedAt string
}

var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "Show job queue status",
    RunE:  runStatus,
}

func init() {
    rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
    p := tea.NewProgram(initialModel)
    if err := p.Start(); err != nil {
        return err
    }
    return nil
}

func initialModel() (model, tea.Cmd) {
    jobs := fetchJobs() // Fetch from /v1/queue
    return model{jobs: jobs, selected: 0}, nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up":
            m.selected--
        case "down":
            m.selected++
        case "q", "ctrl+c":
            return m, tea.Quit
        }
    }
    return m, nil
}

func (m model) View() string {
    var b strings.Builder
    b.WriteString("Queue Status\n\n")
    
    for i, job := range m.jobs {
        prefix := "  "
        if i == m.selected {
            prefix = "> "
        }
        b.WriteString(fmt.Sprintf("%s%s %s (%s)\n", prefix, job.Type, job.Status, job.ID[:8]))
    }
    
    return b.String()
}
```

## Step 5.5: Init Command

**File:** `cli/init.go`

### Huh Wizard

```bash
kl init
```

```go
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize kl configuration",
    RunE:  runInit,
}

func init() {
    rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
    var cfg ClientConfig
    
    form := huh.NewForm(
        huh.NewGroup(
            huh.NewInput().Title("Server host").Value(&cfg.Host).Placeholder("http://127.0.0.1:7766"),
        ),
        huh.NewGroup(
            huh.NewInput().Title("Auth token").Value(&cfg.Token).Placeholder("Enter your token"),
        ),
    )
    
    if err := form.Run(); err != nil {
        return err
    }
    
    // Save to ~/.config/khayal/kl.yaml
    return saveConfig(cfg)
}
```

## Step 5.6: Config Commands

**File:** `cli/config.go`

```bash
kl config set token abc123
kl config set host http://100.x.x.x:7766
kl config view
```

```go
var configCmd = &cobra.Command{
    Use:   "config",
    Short: "Manage configuration",
}

var configSetCmd = &cobra.Command{
    Use:   "set [key] [value]",
    Short: "Set a config value",
    Args:  cobra.ExactArgs(2),
    RunE:  runConfigSet,
}

var configGetCmd = &cobra.Command{
    Use:   "get [key]",
    Short: "Get a config value",
    Args:  cobra.ExactArgs(1),
    RunE:  runConfigGet,
}

var configViewCmd = &cobra.Command{
    Use:   "view",
    Short: "View all config",
    RunE:  runConfigView,
}

func init() {
    rootCmd.AddCommand(configCmd)
    configCmd.AddCommand(configSetCmd)
    configCmd.AddCommand(configGetCmd)
    configCmd.AddCommand(configViewCmd)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
    key, value := args[0], args[1]
    // Update config file
}

func runConfigGet(cmd *cobra.Command, args []string) error {
    key := args[0]
    // Print value
}

func runConfigView(cmd *cobra.Command, args []string) error {
    // Print full config
}
```

## Klient Config File

```yaml
# ~/.config/khayal/kl.yaml
host: http://127.0.0.1:7766
token: your-token-here
```

## Testing

Write tests for:

- [ ] CLI flag parsing
- [ ] API requests (mock server)
- [ ] Output formatting

```bash
go test ./cli/... -v
```

## Checklist

- [ ] Cobra root command
- [ ] Text capture (`kl "text"`)
- [ ] URL capture (`kl --url`)
- [ ] Image capture (`kl --image`)
- [ ] Search command with Glamour
- [ ] Status command with Bubble Tea
- [ ] Init wizard with Huh
- [ ] Config set/get/view
- [ ] Theme integration
- [ ] Tests passing

## Next Phase

[Phase 6: PWA](phase-6-pwa.md)

## Notes

- CLI uses Lip Gloss from `rawnaqs/theme`
- Glamour for markdown rendering
- Huh for interactive prompts
- Bubble Tea for live dashboard
- Config file: `~/.config/khayal/kl.yaml`
