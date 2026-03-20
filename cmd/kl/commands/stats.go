package commands

import (
	"fmt"

	"github.com/rawnaqs/khayal/cmd/kl/internal"
	klapi "github.com/rawnaqs/khayal/cmd/kl/internal/api"
	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"
)

func newStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show vault statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				internal.Fatal(internal.ExitServer, "%s", err.Error())
				return err
			}

			client := klapi.NewClient(cfg.Host, cfg.Token)
			stats, err := client.Stats()
			if err != nil {
				internal.ServerUnreachable(cfg.Host)
				return err
			}

			fmt.Println()
			fmt.Println(theme.Dim.Render("VAULT STATISTICS"))
			fmt.Println()

			keyStyle := theme.Muted.Width(16)
			fmt.Printf("  %s %s\n", keyStyle.Render("total notes"), theme.Primary.Render(fmt.Sprintf("%d", stats.Total)))
			fmt.Printf("  %s %s\n", keyStyle.Render("queue size"), theme.Primary.Render(fmt.Sprintf("%d", stats.QueueSize)))
			fmt.Println()

			if len(stats.ByType) > 0 {
				fmt.Println(theme.Dim.Render("BY TYPE"))
				for t, count := range stats.ByType {
					fmt.Printf("  %s %s\n", theme.Primary.Render(t), theme.Muted.Render(fmt.Sprintf("%d", count)))
				}
				fmt.Println()
			}

			if len(stats.ByTag) > 0 {
				fmt.Println(theme.Dim.Render("TOP TAGS"))
				for i, tag := range getTopTags(stats.ByTag, 5) {
					if i >= 5 {
						break
					}
					fmt.Printf("  %s  %s\n", theme.Tag.Render(tag.name), theme.Muted.Render(fmt.Sprintf("%d", tag.count)))
				}
			}

			return nil
		},
	}
}

type tagCount struct {
	name  string
	count int
}

func getTopTags(byTag map[string]int, n int) []tagCount {
	result := make([]tagCount, 0, len(byTag))
	for name, count := range byTag {
		result = append(result, tagCount{name, count})
	}

	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].count > result[i].count {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if len(result) > n {
		return result[:n]
	}
	return result
}
