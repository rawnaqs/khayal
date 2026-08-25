package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"

	"github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/rawnaqs/khayal/internal/config"
)

var fixLinksFix bool

func newVaultFixLinksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix-links",
		Short: "Remove broken wikilinks",
		RunE:  runVaultFixLinks,
	}
	cmd.Flags().BoolVar(&fixLinksFix, "fix", false, "Apply fixes (default: dry run)")
	return cmd
}

func runVaultFixLinks(cmd *cobra.Command, args []string) error {
	cfg, configPath, err := cli.LoadConfig()
	if err != nil {
		cli.Fatal(cli.ExitUser, "failed to load config: %v", err)
		return err
	}

	vaultPath := config.MakeAbsolute(cfg.Vault.Path, configPath)
	inboxPath := filepath.Join(vaultPath, cfg.Vault.InboxDir)

	fmt.Println(theme.Bold.Render("scanning for broken wikilinks..."))

	noteNames, err := listInboxNotes(inboxPath)
	if err != nil {
		cli.Fatal(cli.ExitVault, "failed to scan inbox: %v", err)
		return err
	}
	nameSet := make(map[string]bool, len(noteNames))
	for _, n := range noteNames {
		nameSet[strings.TrimSuffix(n, ".md")] = true
	}

	var fixes []brokenLink

	for _, n := range noteNames {
		content, err := readNoteFile(inboxPath, n)
		if err != nil {
			continue
		}
		for _, link := range extractWikilinks(content) {
			if !nameSet[link] {
				fixes = append(fixes, brokenLink{file: n, broken: link})
			}
		}
	}

	if len(fixes) == 0 {
		fmt.Println(theme.SuccessStyle.Render("✓ no broken links found"))
		return nil
	}

	fmt.Printf("%s in %d files\n\n", theme.Primary.Render(fmt.Sprintf("%d broken links found", len(fixes))), countDistinctFiles(fixes))

	byFile := make(map[string][]brokenLink)
	var fileOrder []string
	for _, f := range fixes {
		if _, ok := byFile[f.file]; !ok {
			fileOrder = append(fileOrder, f.file)
		}
		byFile[f.file] = append(byFile[f.file], f)
	}
	for _, file := range fileOrder {
		fmt.Println(theme.Bold.Render(file))
		for _, f := range byFile[file] {
			fmt.Printf("  → %s (missing)\n", theme.Muted.Render(f.broken))
		}
	}

	fmt.Println()
	if !fixLinksFix {
		fmt.Println(theme.Muted.Render("[dry run — use --fix to apply]"))
		return nil
	}

	applied := 0
	for _, f := range fixes {
		path := filepath.Join(inboxPath, f.file)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		newContent := strings.ReplaceAll(string(content), "[["+f.broken+"]]", "")
		if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
			fmt.Printf("  %s\n", theme.ErrorStyle.Render("✗ "+f.file))
			continue
		}
		applied++
	}
	fmt.Println(theme.SuccessStyle.Render(fmt.Sprintf("✓ removed %d links", applied)))
	return nil
}

type brokenLink struct {
	file   string
	broken string
}

func countDistinctFiles(fixes []brokenLink) int {
	set := make(map[string]bool, len(fixes))
	for _, f := range fixes {
		set[f.file] = true
	}
	return len(set)
}
