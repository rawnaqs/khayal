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

			fmt.Printf("  %-12s %s\n", theme.Muted.Render("total"), theme.Primary.Render(fmt.Sprintf("%d", stats.Total)))
			fmt.Printf("  %-12s %s\n", theme.Muted.Render("this week"), theme.Primary.Render(fmt.Sprintf("%d", stats.ThisWeek)))
			fmt.Printf("  %-12s %s\n", theme.Muted.Render("this month"), theme.Primary.Render(fmt.Sprintf("%d", stats.ThisMonth)))
			fmt.Println()

			if len(stats.TopTags) > 0 {
				fmt.Println(theme.Dim.Render("TOP TAGS"))
				for _, tag := range stats.TopTags {
					fmt.Printf("  %s  %s\n", theme.RenderTag(tag.Name, ""), theme.Muted.Render(fmt.Sprintf("%d", tag.Count)))
				}
				fmt.Println()
			}

			if len(stats.TopPeople) > 0 {
				fmt.Println(theme.Dim.Render("TOP PEOPLE"))
				for _, person := range stats.TopPeople {
					fmt.Printf("  %-12s %s\n", theme.Primary.Render(person.Name), theme.Muted.Render(fmt.Sprintf("%d mentions", person.Count)))
				}
				fmt.Println()
			}

			if stats.CaptureStreak > 0 || stats.LongestStreak > 0 {
				fmt.Printf("  %-12s %s\n", theme.Muted.Render("capture streak"), theme.Primary.Render(fmt.Sprintf("%d days", stats.CaptureStreak)))
				fmt.Printf("  %-12s %s\n", theme.Muted.Render("longest streak"), theme.Primary.Render(fmt.Sprintf("%d days", stats.LongestStreak)))
			}

			return nil
		},
	}
}
