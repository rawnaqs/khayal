package commands

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"

	"github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/rawnaqs/khayal/internal/config"
)

var duplicateThreshold = 0.85
var duplicateMinShared = 20

func newVaultShowDuplicatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-duplicates",
		Short: "Show potential duplicate notes",
		RunE:  runVaultShowDuplicates,
	}
	cmd.Flags().Float64Var(&duplicateThreshold, "threshold", 0.85, "similarity threshold (0-1)")
	cmd.Flags().IntVar(&duplicateMinShared, "min-shared", 20, "minimum shared words")
	return cmd
}

type noteDoc struct {
	name    string
	content string
}

func runVaultShowDuplicates(cmd *cobra.Command, args []string) error {
	cfg, configPath, err := cli.LoadConfig()
	if err != nil {
		cli.Fatal(cli.ExitUser, "failed to load config: %v", err)
		return err
	}

	vaultPath := config.MakeAbsolute(cfg.Vault.Path, configPath)
	inboxPath := filepath.Join(vaultPath, cfg.Vault.InboxDir)

	fmt.Println(theme.Bold.Render("checking for duplicates..."))

	names, err := listInboxNotes(inboxPath)
	if err != nil {
		cli.Fatal(cli.ExitVault, "failed to scan inbox: %v", err)
		return err
	}

	var notes []noteDoc
	for _, n := range names {
		content, err := readNoteFile(inboxPath, n)
		if err != nil {
			continue
		}
		notes = append(notes, noteDoc{n, content})
	}

	type dupPair struct {
		file1 string
		file2 string
		sim   float64
		words int
	}
	var duplicates []dupPair

	for i := 0; i < len(notes); i++ {
		for j := i + 1; j < len(notes); j++ {
			sim, shared := wordSimilarity(notes[i].content, notes[j].content)
			if sim >= duplicateThreshold && shared >= duplicateMinShared {
				duplicates = append(duplicates, dupPair{notes[i].name, notes[j].name, sim, shared})
			}
		}
	}

	sort.Slice(duplicates, func(a, b int) bool { return duplicates[a].sim > duplicates[b].sim })

	if len(duplicates) == 0 {
		fmt.Println(theme.SuccessStyle.Render("✓ no duplicates found"))
		return nil
	}

	fmt.Println()
	for _, d := range duplicates {
		fmt.Println(theme.Primary.Render(d.file1))
		fmt.Println(theme.Primary.Render(d.file2))
		meta := theme.Muted.Render(fmt.Sprintf("%.2f · %d shared words", d.sim, d.words))
		fmt.Println("  similarity: " + meta)
		fmt.Println()
	}

	fmt.Println(theme.Muted.Render(fmt.Sprintf("%d pairs total", len(duplicates))))
	return nil
}
