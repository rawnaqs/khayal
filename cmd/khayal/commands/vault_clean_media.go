package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"

	"github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/rawnaqs/khayal/internal/queue"
)

var cleanMediaFix bool

func newVaultCleanMediaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean-media",
		Short: "Move orphaned media files to trash",
		RunE:  runVaultCleanMedia,
	}
	cmd.Flags().BoolVar(&cleanMediaFix, "fix", false, "Move to trash (default: dry run)")
	return cmd
}

func runVaultCleanMedia(cmd *cobra.Command, args []string) error {
	cfg, configPath, err := cli.LoadConfig()
	if err != nil {
		cli.Fatal(cli.ExitUser, "failed to load config: %v", err)
		return err
	}

	vaultPath := config.MakeAbsolute(cfg.Vault.Path, configPath)
	inboxPath := filepath.Join(vaultPath, cfg.Vault.InboxDir)
	mediaPath := filepath.Join(inboxPath, cfg.Vault.Media.DefaultDir)
	trashPath := filepath.Join(inboxPath, ".khayal-trash")

	dbPath := config.MakeAbsolute(cfg.DB.Path, configPath)
	q, err := queue.NewQueue(dbPath)
	if err != nil {
		cli.Fatal(cli.ExitServer, "failed to open database: %v", err)
		return err
	}
	defer q.Close()

	fmt.Println(theme.Bold.Render("scanning for orphaned media..."))

	referenced, _ := q.GetReferencedMedia(context.Background())
	refSet := make(map[string]bool, len(referenced))
	for _, r := range referenced {
		refSet[filepath.Base(r)] = true
	}

	type orphan struct {
		name string
		size int64
	}
	var orphans []orphan

	entries, err := os.ReadDir(mediaPath)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if !refSet[e.Name()] {
				info, err := e.Info()
				if err != nil {
					continue
				}
				orphans = append(orphans, orphan{e.Name(), info.Size()})
			}
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
	if !cleanMediaFix {
		fmt.Println(theme.Muted.Render("[dry run — use --fix to apply]"))
		return nil
	}

	if err := os.MkdirAll(trashPath, 0755); err != nil {
		cli.Fatal(cli.ExitVault, "failed to create trash: %v", err)
		return err
	}

	moved := 0
	for _, o := range orphans {
		src := filepath.Join(mediaPath, o.name)
		dst := filepath.Join(trashPath, o.name)
		if _, err := os.Stat(dst); err == nil {
			dst = filepath.Join(trashPath, fmt.Sprintf("%d-%s", time.Now().Unix(), o.name))
		}
		if err := os.Rename(src, dst); err != nil {
			fmt.Printf("  %s\n", theme.ErrorStyle.Render("✗ "+o.name))
			continue
		}
		moved++
	}
	fmt.Println(theme.SuccessStyle.Render(fmt.Sprintf("✓ moved %d files to trash", moved)))
	return nil
}
