package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

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

func contentWidth(title, path, date string, tags []string) int {
	titleWidth := len(title)
	pathWidth := len(path)
	dateWidth := len(date)
	tagWidth := 0
	for _, tag := range tags {
		tagWidth += len(tag) + 1
	}

	headerWidth := titleWidth
	if pathWidth > headerWidth {
		headerWidth = pathWidth
	}

	metaWidth := dateWidth + tagWidth + 2
	if metaWidth > headerWidth {
		headerWidth = metaWidth
	}

	return headerWidth
}

func dividerWidth(contentW, termW int) int {
	d := contentW + 8
	min := 50
	max := termW * 60 / 100

	if d < min {
		return min
	}
	if d > max {
		return max
	}
	return d
}

func printResult(r klapi.SearchResult, termW int) {
	displayPath := r.Title
	if displayPath == "" {
		displayPath = r.NotePath
	}

	dateStr := formatDate(r.CreatedAt)

	contentW := contentWidth(r.Title, r.NotePath, dateStr, r.Tags)
	divW := dividerWidth(contentW, termW)

	fmt.Println(theme.Divider(divW))

	path := theme.Bold.Render(displayPath)
	score := theme.Dim.Render(fmt.Sprintf("%.2f", r.Score))
	gap := divW - len(stripAnsi(path)) - len(stripAnsi(score))
	if gap < 1 {
		gap = 1
	}
	fmt.Println(path + strings.Repeat(" ", gap) + score)

	if dateStr != "" || len(r.Tags) > 0 {
		parts := []string{}
		if dateStr != "" {
			parts = append(parts, theme.Muted.Render(dateStr))
		}
		for _, tag := range r.Tags {
			parts = append(parts, theme.RenderTag(tag))
		}
		if len(parts) > 1 {
			dateLine := parts[0]
			for i := 1; i < len(parts); i++ {
				dateLine += " " + theme.Dim.Render("·") + " " + parts[i]
			}
			fmt.Println(dateLine)
		} else {
			fmt.Println(parts[0])
		}
	}

	fmt.Println()
	fmt.Println(theme.Primary.Render(r.Excerpt))
	fmt.Println()
}

func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for _, c := range s {
		if c == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if c == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(c)
	}
	return result.String()
}

func runSearch(query string) error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		internal.Fatal(internal.ExitServer, "%s", err.Error())
		return err
	}

	client := klapi.NewClient(cfg.Host, cfg.Token)
	result, err := client.Search(query, searchMode, searchLimit)
	if err != nil {
		internal.ServerUnreachable(cfg.Host)
		return err
	}

	total := len(result.Results)
	tookMs := int(result.Time)
	mode := result.Mode
	width := termWidth()

	if total == 0 {
		fmt.Println()
		fmt.Println(theme.Dim.Render(fmt.Sprintf("0 results · %s · %dms", mode, tookMs)))
		fmt.Println()
		fmt.Println(theme.Muted.Render("  nothing found for") + " " + theme.Primary.Render(`"`+query+`"`))
		fmt.Println(theme.Dim.Render("  → try: kl search \"" + query + "\" --mode keyword"))
		return nil
	}

	fmt.Println()
	fmt.Println(theme.Dim.Render(fmt.Sprintf("  %d results · %s · %dms", total, mode, tookMs)))
	fmt.Println()

	for _, r := range result.Results {
		printResult(r, width)
	}

	return nil
}
