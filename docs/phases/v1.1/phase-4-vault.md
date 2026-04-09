# Phase 4: Vault Commands

> Vault maintenance subcommands. Updated: 2026-04-09

## Goals

- [ ] Create vault parent command
- [ ] vault health command
- [ ] vault fix-links command
- [ ] vault clean-media command
- [ ] vault show-duplicates command
- [ ] Unit tests

## Existing Code (Don't Re-Create)

- **RecomputeStats** — Already exists in `internal/queue/queue.go` (lines 1463+)
  - Returns: total_notes, today_delta, last_capture_at, last_7_days
- **CountByStatus** — Already exists in `internal/queue/queue.go`
  - Can count pending/done jobs for health percentage
- **theme package** — Already imported: `github.com/rawnaqs/theme`

## Specification

Per SPEC.md (lines 403-479):

## Step 4.1: Create Vault Parent Command

**File:** `cmd/khayal/commands/vault.go`

```go
package commands

import (
    "github.com/spf13/cobra"
)

var vaultCmd = &cobra.Command{
    Use:   "vault",
    Short: "Vault maintenance commands",
    Long: `Vault maintenance subcommands:
  health          Show vault health report
  fix-links      Remove broken wikilinks
  clean-media    Delete orphaned media files
  show-duplicates Show potential duplicate notes`,
}

func init() {
    rootCmd.AddCommand(vaultCmd)
}
```

## Step 4.2: vault health

**File:** `cmd/khayal/commands/vault_health.go`

Per SPEC.md (lines 412-427):

```
khayal vault health

  vault · ~/brain
  notes     2,847
  indexed   2,203  (77%)
  orphans   12 media files not referenced
  links     4 broken wikilinks found

  health    ⚠ needs attention
  → fix links:      khayal vault fix-links
  → clean media:   khayal vault clean-media
  → reindex:       khayal reindex
```

```go
package commands

import (
    "fmt"
    "path/filepath"
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/spf13/cobra"
    "github.com/rawnaqs/khayal/internal/config"
    "github.com/rawnaqs/khayal/internal/queue"
    "github.com/rawnaqs/khayal/pkg/theme"
)

var vaultHealthCmd = &cobra.Command{
    Use:   "health",
    Short: "Show vault health report",
    RunE:  runVaultHealth,
}

func init() {
    vaultCmd.AddCommand(vaultHealthCmd)
}

func runVaultHealth(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }

    q, err := queue.New(cfg.DB.Path)
    if err != nil {
        return err
    }
    defer q.Close()

    vaultPath := cfg.Vault.Path
    inboxPath := filepath.Join(vaultPath, cfg.Vault.InboxDir)

    // Get stats
    stats, err := q.GetStats()
    if err != nil {
        return err
    }

    // Count notes
    noteCount, _ := countNotes(inboxPath)

    // Count indexed
    indexedCount := stats.IndexedNotes

    // Count orphans
    orphanCount, _ := countOrphanedMedia(inboxPath, q)

    // Count broken links
    brokenLinkCount, _ := countBrokenLinks(inboxPath)

    indexedPct := 0
    if noteCount > 0 {
        indexedPct = (indexedCount * 100) / noteCount
    }

    healthStatus := "✓ healthy"
    if orphanCount > 0 || brokenLinkCount > 0 || indexedPct < 100 {
        healthStatus = "⚠ needs attention"
    }

    // Output
    fmt.Println(theme.Bold.Render("vault · ") + theme.Primary.Render(vaultPath))

    keyWidth := 12
    fmt.Printf("%-*s %s\n", keyWidth, "  notes", theme.Primary.Render(fmt.Sprintf("%d", noteCount)))
    fmt.Printf("%-*s %s (%d%%)\n", keyWidth, "  indexed", theme.Primary.Render(fmt.Sprintf("%d", indexedCount)), indexedPct)
    fmt.Printf("%-*s %s\n", keyWidth, "  orphans", theme.Primary.Render(fmt.Sprintf("%d", orphanCount)))
    fmt.Printf("%-*s %s\n", keyWidth, "  links", theme.Primary.Render(fmt.Sprintf("%d broken wikilinks found", brokenLinkCount)))

    fmt.Println()
    fmt.Printf("%-*s %s\n", keyWidth, "  health", theme.Primary.Render(healthStatus))

    if healthStatus != "✓ healthy" {
        fmt.Println()
        fmt.Printf("  → fix links:      %s\n", theme.Muted.Render("khayal vault fix-links"))
        fmt.Printf("  → clean media:   %s\n", theme.Muted.Render("khayal vault clean-media"))
        fmt.Printf("  → reindex:       %s\n", theme.Muted.Render("khayal reindex"))
    }

    return nil
}

func countNotes(inboxPath string) (int, error) {
    entries, err := os.ReadDir(inboxPath)
    if err != nil {
        return 0, err
    }

    count := 0
    for _, e := range entries {
        if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
            count++
        }
    }
    return count, nil
}

func countOrphanedMedia(inboxPath string, q *queue.Queue) (int, error) {
    mediaPath := filepath.Join(inboxPath, "media")

    entries, err := os.ReadDir(mediaPath)
    if err != nil {
        return 0, err
    }

    // Check which media files are referenced
    referenced, err := q.GetReferencedMedia()
    if err != nil {
        return 0, err
    }

    refSet := make(map[string]bool)
    for _, r := range referenced {
        refSet[r] = true
    }

    orphanCount := 0
    for _, e := range entries {
        if e.IsDir() {
            continue
        }
        if !refSet[e.Name()] {
            orphanCount++
        }
    }

    return orphanCount, nil
}

func countBrokenLinks(inboxPath string) (int, error) {
    entries, err := os.ReadDir(inboxPath)
    if err != nil {
        return 0, err
    }

    count := 0
    noteNames := make(map[string]bool)

    for _, e := range entries {
        if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
            continue
        }
        name := strings.TrimSuffix(e.Name(), ".md")
        noteNames[name] = true
    }

    for _, e := range entries {
        if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
            continue
        }

        path := filepath.Join(inboxPath, e.Name())
        content, _ := os.ReadFile(path)

        links := extractWikilinks(string(content))
        for _, link := range links {
            if !noteNames[link] {
                count++
            }
        }
    }

    return count, nil
}

func extractWikilinks(content string) []string {
    var links []string
    re := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
    matches := re.FindAllStringSubmatch(content, -1)
    for _, m := range matches {
        if len(m) > 1 {
            links = append(links, m[1])
        }
    }
    return links
}
```

## Step 4.3: vault fix-links

**File:** `cmd/khayal/commands/vault_fix_links.go`

Per SPEC.md (lines 429-444):

```
khayal vault fix-links

  scanning for broken wikilinks...
  4 broken links found in 3 files

  khayal/2024-03-10-project.md
    → khayal/old-note.md (deleted)
    → khayal/renamed-note.md (exists, will update)

  [dry run — use --fix to apply]
```

```go
package commands

import (
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"

    "github.com/spf13/cobra"
    "github.com/rawnaqs/khayal/internal/config"
    "github.com/rawnaqs/khayal/pkg/theme"
)

var fixLinksDryRun = true

var vaultFixLinksCmd = &cobra.Command{
    Use:   "fix-links",
    Short: "Remove broken wikilinks",
    RunE:  runVaultFixLinks,
}

func init() {
    vaultFixLinksCmd.Flags().BoolVar(&fixLinksDryRun, "fix", false, "Apply fixes (default: dry run)")
    vaultCmd.AddCommand(vaultFixLinksCmd)
}

func runVaultFixLinks(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }

    vaultPath := cfg.Vault.Path
    inboxPath := filepath.Join(vaultPath, cfg.Vault.InboxDir)

    fmt.Println(theme.Bold.Render("scanning for broken wikilinks..."))

    // Get all note names
    entries, _ := os.ReadDir(inboxPath)
    noteNames := make(map[string]string)
    for _, e := range entries {
        if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
            name := strings.TrimSuffix(e.Name(), ".md")
            noteNames[name] = e.Name()
        }
    }

    // Find broken links
    type fix struct {
        file    string
        broken string
        action string
    }
    var fixes []fix

    for _, e := range entries {
        if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
            continue
        }

        path := filepath.Join(inboxPath, e.Name())
        content, _ := os.ReadFile(path)

        links := extractWikilinks(string(content))
        for _, link := range links {
            target := filepath.Join(inboxPath, link+".md")
            if _, err := os.Stat(target); os.IsNotExist(err) {
                fixes = append(fixes, fix{
                    file:    e.Name(),
                    broken: link,
                    action: "remove",
                })
            } else if !noteNames[link] {
                fixes = append(fixes, fix{
                    file:    e.Name(),
                    broken: link,
                    action: "rename",
                })
            }
        }
    }

    if len(fixes) == 0 {
        fmt.Println(theme.SuccessStyle.Render("✓ no broken links found"))
        return nil
    }

    fmt.Printf("%s\n\n", theme.Primary.Render(fmt.Sprintf("%d broken links found", len(fixesixes))))

    // Group by file
    byFile := make(map[string][]fix)
    for _, f := range fixes {
        byFile[f.file] = append(byFile[f.file], f)
    }

    for file, flist := range byFile {
        fmt.Println(theme.Bold.Render(file))
        for _, f := range flist {
            fmt.Printf("  → %s (%s)\n", theme.Muted.Render(f.broken), theme.Primary.Render(f.action))
        }
    }

    fmt.Println()
    if fixLinksDryRun {
        fmt.Println(theme.Muted.Render("[dry run — use --fix to apply]"))
    } else {
        // Apply fixes
        for _, f := range fixes {
            // Read file, replace/remove link, write back
            path := filepath.Join(inboxPath, f.file)
            content, _ := os.ReadFile(path)

            newContent := strings.ReplaceAll(string(content), "[["+f.broken+"]]", "")
            os.WriteFile(path, []byte(newContent), 0644)
        }
        fmt.Println(theme.SuccessStyle.Render("✓ fixed"))
    }

    return nil
}
```

## Step 4.4: vault clean-media

**File:** `cmd/khayal/commands/vault_clean_media.go`

Per SPEC.md (lines 446-461):

```
khayal vault clean-media

  scanning for orphaned media...
  12 orphaned files found · 34 MB

  khayal/media/unused-1.png    2.1 MB
  khayal/media/unused-2.jpg    1.8 MB
  ...

  [dry run — use --fix to apply]
```

```go
package commands

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"

    "github.com/spf13/cobra"
    "github.com/rawnaqs/khayal/internal/config"
    "github.com/rawnaqs/khayal/internal/queue"
    "github.com/rawnaqs/khayal/pkg/theme"
)

var cleanMediaDryRun = true

var vaultCleanMediaCmd = &cobra.Command{
    Use:   "clean-media",
    Short: "Delete orphaned media files",
    RunE:  runVaultCleanMedia,
}

func init() {
    vaultCleanMediaCmd.Flags().BoolVar(&cleanMediaDryRun, "fix", false, "Move to trash (default: dry run)")
    vaultCmd.AddCommand(vaultCleanMediaCmd)
}

func runVaultCleanMedia(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    0
    if err != nil {
        return err
    }

    q, err := queue.New(cfg.DB.Path)
    if err != nil {
        return err
    }
    defer q.Close()

    vaultPath := cfg.Vault.Path
    inboxPath := filepath.Join(vaultPath, cfg.Vault.InboxDir)
    mediaPath := filepath.Join(inboxPath, "media")
    trashPath := filepath.Join(inboxPath, ".khayal-trash")

    fmt.Println(theme.Bold.Render("scanning for orphaned media..."))

    // Get referenced media
    referenced, _ := q.GetReferencedMedia()
    refSet := make(map[string]bool)
    for _, r := range referenced {
        refSet[r] = true
    }
    if refSet[""] == false { // always check
    }

    // Find orphaned files
    entries, _ := os.ReadDir(mediaPath)
    var orphans []struct {
        name string
        size int64
    }

    for _, e := range entries {
        if e.IsDir() {
            continue
        }
        if !refSet[e.Name()] {
            info, _ := e.Info()
            orphans = append(orphans, struct {
                name string
                size int64
            }{e.Name(), info.Size()})
        }
    }

    if len(orphans) == 0 {
        fmt.Println(theme.SuccessStyle.Render("✓ no orphaned media found"))
        return nil
    }

    totalSize := int64(0)
    for _, o := range orphans {
        totalSize += o.size
    }

    fmt.Println(theme.Primary.Render(fmt.Sprintf("%d orphaned files found · %s\n", len(orphans), formatSize(totalSize))))

    for _, o := range orphans {
        fmt.Printf("  %s    %s\n", theme.Muted.Render(o.name), theme.Primary.Render(formatSize(o.size)))
    }

    fmt.Println()
    if cleanMediaDryRun {
        fmt.Println(theme.Muted.Render("[dry run — use --fix to apply]"))
    } else {
        // Move to trash
        os.MkdirAll(trashPath, 0755)

        for _, o := range orphans {
            src := filepath.Join(mediaPath, o.name)
            dst := filepath.Join(trashPath, o.name)

            if _, err := os.Stat(dst); err == nil {
                timestamp := fmt.Sprintf("%d", os.Now().Unix())
                dst = filepath.Join(trashPath, timestamp+"-"+o.name)
            }

            if err := os.Rename(src, dst); err != nil {
                fmt.Printf("  %s\n", theme.ErrorStyle.Render("✗ "+o.name))
            } else {
                fmt.Printf("  %s\n", theme.SuccessStyle.Render("✓ "+o.name))
            }
        }
    }

    return nil
}
```

## Step 4.5: vault show-duplicates

**File:** `cmd/khayal/commands/vault_show_duplicates.go`

Per SPEC.md (lines 463-479):

```
khayal vault show-duplicates

  checking for duplicates...
  potential duplicates found:
  
  khayal/2024-03-15-rust-thoughts.md
  khayal/2024-03-10-rust-notes.md
    similarity: 0.87 · 234 shared words
  
  khayal/2024-02-20-meeting-notes.md
  khayal/2024-02-19-meeting.md
    similarity: 0.91 · 189 shared words
  
  3 pairs total
```

```go
package commands

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/spf13/cobra"
    "github.com/rawnaqs/khayal/internal/config"
    "github.com/rawnaqs/khayal/internal/queue"
    "github.com/rawnaqs/khayal/pkg/theme"
)

var duplicateThreshold = 0.85

var vaultShowDuplicatesCmd = &cobra.Command{
    Use:   "show-duplicates",
    Short: "Show potential duplicate notes",
    RunE:  runVaultShowDuplicates,
}

func init() {
    vaultCmd.AddCommand(vaultShowDuplicatesCmd)
}

func runVaultShowDuplicates(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }

    q, err := queue.New(cfg.DB.Path)
    if err != nil {
        return err
    }
    defer q.Close()

    vaultPath := cfg.Vault.Path
    inboxPath := filepath.Join(vaultPath, cfg.Vault.InboxDir)

    fmt.Println(theme.Bold.Render("checking for duplicates..."))

    // Get all notes
    entries, _ := os.ReadDir(inboxPath)
    var notes []struct {
        path    string
        name   string
        content string
    }

    for _, e := range entries {
        if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
            continue
        }

        path := filepath.Join(inboxPath, e.Name())
        content, _ := os.ReadFile(path)

        notes = append(notes, struct {
            path    string
            name   string
            content string
        }{path, e.Name(), string(content)})
    }

    // Compare each pair
    var duplicates []struct {
        file1   string
        file2   string
        sim     float32
        words   int
    }

    for i := 0; i < len(notes); i++ {
        for j := i + 1; j < len(notes); j++ {
            sim, shared := similarity(notes[i].content, notes[j].content)
            if sim > duplicateThreshold && shared > 50 {
                duplicates = append(duplicates, struct {
                    file1   string
                    file2   string
                    sim     float32
                    words   int
                }{notes[i].name, notes[j].name, sim, shared})
            }
        }
    }

    if len(duplicates) == 0 {
        fmt.Println(theme.SuccessStyle.Render("✓ no duplicates found"))
        return nil
    }

    fmt.Println()
    for _, d := range duplicates {
        fmt.Println(theme.Primary.Render(d.file1))
        fmt.Println(theme.Primary.Render(d.file2))
        fmt.Printf("  similarity: %s · %d shared words\n\n", 
            theme.Muted.Render(fmt.Sprintf("%.2f", d.sim)), 
            theme.Muted.Render(fmt.Sprintf("%d", d.words)))
    }

    fmt.Println(theme.Muted.Render(fmt.Sprintf("%d pairs total", len(duplicates))))

    return nil
}

func similarity(a, b string) (float32, int) {
    wordsA := strings.Fields(strings.ToLower(a))
    wordsB := strings.Fields(strings.ToLower(b))

    setA := make(map[string]bool)
    for _, w := range wordsA {
        setA[w] = true
    }

    shared := 0
    for _, w := range wordsB {
        if setA[w] {
            shared++
        }
    }

    total := len(wordsA) + len(wordsB)
    if total == 0 {
        return 0, shared
    }

    return float32(shared*2) / float32(total), shared
}
```

## Checklist

- [ ] vault parent command (root + subcommands)
- [ ] vault health showing notes/indexed/orphans/links
- [ ] vault fix-links --dry-run (shows broken)
- [ ] vault fix-links --fix (applies)
- [ ] vault clean-media --dry-run (shows orphans)
- [ ] vault clean-media --fix (moves to trash)
- [ ] vault show-duplicates
- [ ] Using RecomputeStats for health stats
- [ ] Using CountByStatus for indexed count
- [ ] Unit tests

## Next Phase

[Phase 5: Backup and Restore](phase-5-backup.md)

## Notes

- All vault commands require config load
- All commands show themed output per CLI_RULES.md
- fix-links: dry-run default, --fix to apply
- clean-media: moves to .khayal-trash/, not delete permanently
- show-duplicates: threshold 0.85 per SPEC.md
- Reuse RecomputeStats() for health stats
- Reuse CountByStatus("done") for indexed count

## Step 4.X: Unit Tests

**File:** `cmd/khayal/commands/vault_health_test.go`

```go
package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rawnaqs/khayal/internal/config"
)

func TestVaultHealth_Output(t *testing.T) {
	// Create temp config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &config.Config{
		Vault: config.VaultConfig{
			Path:     filepath.Join(tmpDir, "vault"),
			InboxDir: "inbox",
		},
		DB: config.DBConfig{
			Path: filepath.Join(tmpDir, "khayal.db"),
		},
	}

	if err := config.Save(cfg, configPath); err != nil {
		t.Fatal(err)
	}

	// Create vault directory
	os.MkdirAll(filepath.Join(tmpDir, "vault", "inbox"), 0755)

	// Run health command (mock)
	// ... test output format
}
```

**File:** `cmd/khayal/commands/vault_fix_links_test.go`

```go
package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractWikilinks(t *testing.T) {
	content := `This is a note with [[link-one]] and [[link-two]].

Also [[another-link]] here.`

	links := extractWikilinks(content)

	if len(links) != 3 {
		t.Errorf("expected 3 links, got %d", len(links))
	}

	expected := []string{"link-one", "link-two", "another-link"}
	for i, link := range links {
		if link != expected[i] {
			t.Errorf("link %d = %s, want %s", i, link, expected[i])
		}
	}
}

func TestExtractWikilinks_Empty(t *testing.T) {
	links := extractWikilinks("no links here")

	if len(links) != 0 {
		t.Errorf("expected 0 links, got %d", len(links))
	}
}
```

**File:** `cmd/khayal/commands/vault_show_duplicates_test.go`

```go
package commands

import (
	"testing"
)

func TestSimilarity(t *testing.T) {
	tests := []struct {
		a       string
		b       string
		want    float32
		shared int
	}{
		{"hello world", "hello world", 1.0, 2},
		{"hello world", "world hello", 1.0, 2},
		{"hello world", "foo bar", 0.0, 0},
		{"the quick brown fox", "quick brown fox jumps", 0.75, 4},
	}

	for _, tt := range tests {
		sim, shared := similarity(tt.a, tt.b)
		if sim != tt.want {
			t.Errorf("similarity(%q, %q) = %v, want %v", tt.a, tt.b, sim, tt.want)
		}
		if shared != tt.shared {
			t.Errorf("shared words = %d, want %d", shared, tt.shared)
		}
	}
}

func TestSimilarity_Empty(t *testing.T) {
	sim, shared := similarity("", "")
	if sim != 0 || shared != 0 {
		t.Errorf("empty strings: got sim=%v, shared=%d", sim, shared)
	}
}
```