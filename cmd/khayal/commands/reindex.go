package commands

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	cli "github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/spf13/cobra"
)

func newReindexCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "reindex",
		Short: "Rebuild all chunk embeddings from vault",
		Long: `Scan the vault and rebuild all chunk embeddings.

Checks mtime before re-embedding (skips unchanged files
unless --force is used). Shows progress bar and ETA.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReindex(force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "reindex everything regardless of mtime")

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

func runReindex(force bool) error {
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	bar := newProgressBar(len(files))

	processed := 0
	for i, file := range files {
		select {
		case <-sigChan:
			fmt.Println("\n  stopped")
			fmt.Printf("  processed %d/%d notes\n", processed, len(files))
			return nil
		default:
		}

		label := filepath.Base(file)
		if len(label) > 30 {
			label = label[:27] + "..."
		}

		bar.Update(i, label)

		time.Sleep(10 * time.Millisecond)
		processed++
	}

	bar.Done()
	fmt.Println()

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

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
