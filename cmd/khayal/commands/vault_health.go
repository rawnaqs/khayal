package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"

	"github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/queue"
)

func newVaultHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Show vault health report",
		RunE:  runVaultHealth,
	}
}

func runVaultHealth(cmd *cobra.Command, args []string) error {
	cfg, configPath, err := cli.LoadConfig()
	if err != nil {
		cli.Fatal(cli.ExitUser, "failed to load config: %v", err)
		return err
	}

	vaultPath := config.MakeAbsolute(cfg.Vault.Path, configPath)
	inboxPath := filepath.Join(vaultPath, cfg.Vault.InboxDir)
	mediaPath := filepath.Join(inboxPath, cfg.Vault.Media.DefaultDir)

	dbPath := config.MakeAbsolute(cfg.DB.Path, configPath)
	q, err := queue.NewQueue(dbPath)
	if err != nil {
		cli.Fatal(cli.ExitServer, "failed to open database: %v", err)
		return err
	}
	defer q.Close()
	ctx := context.Background()

	stats, _ := q.RecomputeStats(ctx)
	noteNames, _ := listInboxNotes(inboxPath)
	noteCount := len(noteNames)

	indexedCount := 0
	if stats != nil {
		indexedCount = stats.Vault.TotalNotes
	}

	orphanCount, _ := countOrphanedMedia(ctx, mediaPath, q)
	brokenLinkCount, _ := countBrokenLinks(inboxPath, noteNames)

	indexedPct := 100
	if noteCount > 0 && indexedCount < noteCount {
		indexedPct = indexedCount * 100 / noteCount
	}

	healthy := orphanCount == 0 && brokenLinkCount == 0 && indexedPct == 100
	status := theme.SuccessStyle.Render("✓ healthy")
	if !healthy {
		status = theme.WarningStyle.Render("⚠ needs attention")
	}

	fmt.Println(theme.Bold.Render("vault · ") + theme.Primary.Render(vaultPath))
	fmt.Printf("  %-9s %s\n", "notes", theme.Primary.Render(fmt.Sprintf("%d", noteCount)))
	fmt.Printf("  %-9s %s (%d%%)\n", "indexed", theme.Primary.Render(fmt.Sprintf("%d", indexedCount)), indexedPct)
	fmt.Printf("  %-9s %s\n", "orphans", theme.Primary.Render(fmt.Sprintf("%d", orphanCount)))
	fmt.Printf("  %-9s %s\n", "links", theme.Primary.Render(fmt.Sprintf("%d broken", brokenLinkCount)))
	fmt.Println()
	fmt.Printf("  %-9s %s\n", "health", status)

	if !healthy {
		fmt.Println()
		fmt.Printf("  → fix links:   %s\n", theme.Muted.Render("khayal vault fix-links"))
		fmt.Printf("  → clean media: %s\n", theme.Muted.Render("khayal vault clean-media"))
		fmt.Printf("  → reindex:     %s\n", theme.Muted.Render("khayal reindex"))
	}
	return nil
}

func countOrphanedMedia(ctx context.Context, mediaPath string, q *queue.Queue) (int, error) {
	entries, err := os.ReadDir(mediaPath)
	if err != nil {
		return 0, nil // no media dir is not an error
	}
	referenced, err := q.GetReferencedMedia(ctx)
	if err != nil {
		referenced = nil // treat as nothing referenced; scan still lists files
	}
	refSet := make(map[string]bool, len(referenced))
	for _, r := range referenced {
		refSet[filepath.Base(r)] = true
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

func countBrokenLinks(inboxPath string, noteNames []string) (int, error) {
	existing := make(map[string]bool, len(noteNames))
	for _, n := range noteNames {
		existing[strings.TrimSuffix(n, ".md")] = true
	}
	count := 0
	for _, n := range noteNames {
		content, err := readNoteFile(inboxPath, n)
		if err != nil {
			continue
		}
		for _, link := range extractWikilinks(content) {
			if !existing[link] {
				count++
			}
		}
	}
	return count, nil
}
