package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"

	cli "github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/queue"
	"github.com/spf13/cobra"
)

func newReindexCmd() *cobra.Command {
	var force bool
	var ftsOnly bool

	cmd := &cobra.Command{
		Use:   "reindex",
		Short: "Rebuild search index from vault",
		Long: `Scan the vault and rebuild the search index.

Updates FTS5 index for all notes. Use --force to reindex
everything even if unchanged. Shows progress bar.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReindex(force, ftsOnly)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "reindex everything regardless of mtime")
	cmd.Flags().BoolVar(&ftsOnly, "fts-only", false, "only rebuild FTS index (skip embeddings)")

	return cmd
}

type progressBar struct {
	width    int
	current  int
	total    int
	lastLine string
}

func newProgressBar(total int) *progressBar {
	return &progressBar{
		width:   40,
		current: 0,
		total:   total,
	}
}

func (p *progressBar) Update(current int, label string) {
	p.current = current
	percent := float64(current) / float64(p.total)
	filled := int(percent * float64(p.width))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)
	percentStr := fmt.Sprintf("%.0f%%", percent*100)

	p.lastLine = fmt.Sprintf("\r  [%s] %s  %d/%d  %s", bar, percentStr, current, p.total, label)
}

func (p *progressBar) Done() {
	p.Update(p.total, "done")
	fmt.Println(p.lastLine)
}

func runReindex(force bool, ftsOnly bool) error {
	fmt.Println("scanning vault...")

	cfg, configPath, err := cli.LoadConfig()
	if err != nil {
		cli.Fatal(cli.ExitUser, "failed to load config: %v", err)
		return err
	}

	vaultPath := config.MakeAbsolute(cfg.Vault.Path, configPath)
	inboxPath := filepath.Join(vaultPath, cfg.Vault.InboxDir)

	files, err := scanForMarkdown(inboxPath)
	if err != nil {
		cli.Fatal(cli.ExitVault, "failed to scan vault: %v", err)
		return err
	}

	if len(files) == 0 {
		fmt.Println("  no notes found")
		return nil
	}

	fmt.Printf("  found %d notes\n", len(files))

	// Initialize queue
	dbPath := config.MakeAbsolute(cfg.DB.Path, configPath)
	logger := slog.Default()
	q, err := queue.NewQueueWithLogger(dbPath, logger)
	if err != nil {
		cli.Fatal(cli.ExitUser, "failed to open database: %v", err)
		return err
	}
	defer q.Close()

	ctx := context.Background()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	bar := newProgressBar(len(files))

	processed := 0
	skipped := 0
	errors := 0

	for i, file := range files {
		select {
		case <-sigChan:
			fmt.Println("\n  stopped")
			fmt.Printf("  processed: %d  skipped: %d  errors: %d\n", processed, skipped, errors)
			return nil
		default:
		}

		label := filepath.Base(file)
		if len(label) > 30 {
			label = label[:27] + "..."
		}

		bar.Update(i, label)

		// Get relative path from inbox
		relPath, err := filepath.Rel(inboxPath, file)
		if err != nil {
			errors++
			continue
		}

		// Read file content
		content, err := readFile(file)
		if err != nil {
			logger.Error("failed to read file", "file", file, "error", err)
			errors++
			continue
		}

		// Extract title
		title := extractTitle(content, file)

		// Get tags from frontmatter
		tags := extractTags(content)

		// Index the note for FTS
		if err := q.IndexNote(ctx, relPath, title, content, tags); err != nil {
			logger.Error("failed to index note", "file", file, "error", err)
			errors++
			continue
		}

		processed++
	}

	bar.Done()
	fmt.Printf("  processed: %d  skipped: %d  errors: %d\n", processed, skipped, errors)

	return nil
}

func scanForMarkdown(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return files, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

func readFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	content, err := readAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func readAll(f *os.File) ([]byte, error) {
	var buf []byte
	for {
		if len(buf) == cap(buf) {
			buf = append(buf, 0)[:len(buf)]
		}
		n, err := f.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+n]
		if err != nil {
			if err.Error() == "EOF" {
				return buf, nil
			}
			return buf, err
		}
	}
}

// extractTitle extracts title from content
func extractTitle(content, filePath string) string {
	// Try YAML frontmatter title
	re := regexp.MustCompile(`(?m)^title:\s*(.+)$`)
	if match := re.FindStringSubmatch(content); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	// Try first heading
	re = regexp.MustCompile(`(?m)^#\s+(.+)$`)
	if match := re.FindStringSubmatch(content); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	// Use filename without extension
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// extractTags extracts tags from YAML frontmatter
func extractTags(content string) string {
	// Try YAML frontmatter tags
	re := regexp.MustCompile(`(?m)^tags:\s*\[(.+)\]`)
	if match := re.FindStringSubmatch(content); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	// Try multi-line tags
	re = regexp.MustCompile(`(?m)^tags:\s*\n((?:\s+-\s+.+\n)*)`)
	if match := re.FindStringSubmatch(content); len(match) > 1 {
		tagLines := strings.Split(match[1], "\n")
		var tags []string
		for _, line := range tagLines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "- ") {
				tags = append(tags, strings.TrimPrefix(line, "- "))
			}
		}
		return strings.Join(tags, ",")
	}

	return ""
}
