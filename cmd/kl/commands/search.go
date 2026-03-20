package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/rawnaqs/khayal/cmd/kl/internal"
	klapi "github.com/rawnaqs/khayal/cmd/kl/internal/api"
	"github.com/rawnaqs/theme"
	"golang.org/x/term"
)

func termWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

func formatDate(iso string) string {
	if iso == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	return t.Format("January 2, 2006")
}

func resultWidth(termW int) int {
	if termW > 100 {
		return 100
	}
	if termW < 60 {
		return 60
	}
	return termW
}

func truncateTitle(title string, width int) string {
	const scoreW = 6
	maxTitle := width - scoreW
	if lipgloss.Width(title) <= maxTitle {
		return title
	}
	runes := []rune(title)
	for lipgloss.Width(string(runes)) > maxTitle-1 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func buildMetaLine(dateStr, noteType string, tags []string, width int) string {
	parts := []string{}
	if dateStr != "" {
		parts = append(parts, theme.SearchDate.Render(dateStr))
	}
	if noteType != "" {
		parts = append(parts, theme.RenderTypeBadge(noteType))
	}

	sep := theme.Dim.Render(" · ")
	line := strings.Join(parts, sep)

	for _, tag := range tags {
		candidate := line + sep + theme.RenderTag("#"+tag, noteType)
		if lipgloss.Width(candidate) > width {
			break
		}
		line = candidate
	}
	return line
}

func printResult(r klapi.SearchResult, termW int) {
	displayPath := r.Title
	if displayPath == "" {
		displayPath = r.NotePath
	}
	dateStr := formatDate(r.CreatedAt)
	width := resultWidth(termW)
	displayPath = truncateTitle(displayPath, width)

	fmt.Println(theme.Divider(width))

	// ── Row 1: title + score — use table for correct alignment ────
	t := table.New().
		Width(width).
		Border(lipgloss.HiddenBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch col {
			case 0:
				return theme.SearchTitle
			case 1:
				return theme.SearchScore.Align(lipgloss.Right)
			}
			return lipgloss.NewStyle()
		}).Row(displayPath, fmt.Sprintf("%.2f", r.Score))

	fmt.Println(t.Render())

	// ── Row 2: plain string, no table ─────────────────────────────
	metaLine := buildMetaLine(dateStr, r.Type, r.Tags, width)
	if metaLine != "" {
		fmt.Println(" " + metaLine)
	}

	// ── Excerpt ───────────────────────────────────────────────────
	fmt.Println()
	fmt.Println(theme.SearchExcerpt.Width(width - 2).Render(r.Excerpt))
	fmt.Println()
}

func tookMsStyle(ms int) lipgloss.Style {
	switch {
	case ms < 100:
		return theme.SuccessStyle
	case ms < 500:
		return theme.WarningStyle
	default:
		return theme.ErrorStyle
	}
}

func runSearch(query string) error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		internal.Fatal(internal.ExitServer, "%s", err.Error())
		return err
	}

	client := klapi.NewClient(cfg.Host, cfg.Token)
	result, err := client.Search(query, searchMode, searchLimit, searchExcerptLen, searchFrom, searchTo, searchConnections)
	if err != nil {
		internal.ServerUnreachable(cfg.Host)
		return err
	}

	total := len(result.Results)
	tookMs := int(result.TookMs)
	mode := result.Mode
	width := termWidth()

	fmt.Println()

	if total == 0 {
		fmt.Println(theme.Dim.Render(fmt.Sprintf("0 results · %s · %dms", mode, tookMs)))
		fmt.Println()
		fmt.Println(theme.Muted.Render("nothing found for") + " " + theme.Primary.Render(`"`+query+`"`))
		fmt.Println(theme.Dim.Render(`  → try: kl search "` + query + `" --mode keyword`))
		return nil
	}

	fmt.Println(
		theme.Bold.Render(fmt.Sprintf("%d", total)) +
			theme.Dim.Render(" results for ") +
			theme.Primary.Render(`"`+query+`"`) +
			theme.Dim.Render(" · "+mode+" · ") +
			tookMsStyle(tookMs).Render(fmt.Sprintf("%dms", tookMs)))
	fmt.Println()

	for _, r := range result.Results {
		printResult(r, width)
	}

	fmt.Println(theme.Divider(resultWidth(width)))
	fmt.Println()

	if total == 0 {
		fmt.Println(theme.Dim.Render(fmt.Sprintf("0 results · %s · %dms", mode, tookMs)))
		fmt.Println()
		fmt.Println(theme.Muted.Render("nothing found for") + " " + theme.Primary.Render(`"`+query+`"`))
		fmt.Println(theme.Dim.Render(`  → try: kl search "` + query + `" --mode keyword`))
		return nil
	}

	fmt.Println(
		theme.Bold.Render(fmt.Sprintf("%d", total)) +
			theme.Dim.Render(" results for ") +
			theme.Primary.Render(`"`+query+`"`) +
			theme.Dim.Render(" · "+mode+" · ") +
			tookMsStyle(tookMs).Render(fmt.Sprintf("%dms", tookMs)))
	fmt.Println()

	return nil
}
