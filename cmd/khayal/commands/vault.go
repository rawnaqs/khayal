package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Vault maintenance commands",
	Long: `Vault maintenance subcommands:
  health           Show vault health report
  fix-links        Remove broken wikilinks
  clean-media      Move orphaned media files to trash
  show-duplicates  Show potential duplicate notes`,
}

func newVaultCmd() *cobra.Command {
	vaultCmd.AddCommand(newVaultHealthCmd())
	vaultCmd.AddCommand(newVaultFixLinksCmd())
	vaultCmd.AddCommand(newVaultCleanMediaCmd())
	vaultCmd.AddCommand(newVaultShowDuplicatesCmd())
	return vaultCmd
}

// listInboxNotes returns the .md filenames directly inside the inbox,
// excluding the trash directory and khayal-managed files.
func listInboxNotes(inboxPath string) ([]string, error) {
	entries, err := os.ReadDir(inboxPath)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		names = append(names, e.Name())
	}
	return names, nil
}

func readNoteFile(inboxPath, name string) (string, error) {
	b, err := os.ReadFile(filepath.Join(inboxPath, name))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

var wikilinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

func extractWikilinks(content string) []string {
	var links []string
	for _, m := range wikilinkRe.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 && m[1] != "" {
			links = append(links, m[1])
		}
	}
	return links
}

func formatSize(b int64) string {
	const kb, mb = 1024, 1024 * 1024
	switch {
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/mb)
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/kb)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// wordSimilarity reports Jaccard-style overlap between two texts plus the
// count of shared unique words.
func wordSimilarity(a, b string) (float64, int) {
	setA := make(map[string]bool)
	for _, w := range strings.Fields(strings.ToLower(a)) {
		setA[w] = true
	}
	setB := make(map[string]bool)
	for _, w := range strings.Fields(strings.ToLower(b)) {
		setB[w] = true
	}
	shared := 0
	for w := range setB {
		if setA[w] {
			shared++
		}
	}
	total := len(setA) + len(setB)
	if total == 0 {
		return 0, 0
	}
	return float64(shared*2) / float64(total), shared
}
