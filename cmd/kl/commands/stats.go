package commands

import (
	"fmt"
	"strings"

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

			// Vault
			fmt.Printf("  %-12s %s\n", theme.Muted.Render("notes"), theme.Primary.Render(fmt.Sprintf("%d", stats.Vault.TotalNotes)))
			if stats.Vault.TodayDelta > 0 {
				fmt.Printf("  %-12s %s\n", theme.Muted.Render("today"), theme.Primary.Render(fmt.Sprintf("+%d", stats.Vault.TodayDelta)))
			}
			fmt.Printf("  %-12s %s\n", theme.Muted.Render("last 7d"), theme.Primary.Render(fmt.Sprintf("%d", sum7(stats.Vault.Last7Days))))
			fmt.Println()

			// Today
			fmt.Println(theme.Dim.Render("TODAY"))
			fmt.Printf("  %-12s %s\n", theme.Muted.Render("captures"), theme.Primary.Render(fmt.Sprintf("%d", stats.Today.Count)))
			fmt.Printf("  %-12s %s\n", theme.Muted.Render("avg/day"), theme.Primary.Render(fmt.Sprintf("%.1f", stats.Today.AvgPerDay)))
			fmt.Println()

			// Streak
			if stats.Streak.Current > 0 || stats.Streak.Best > 0 {
				fmt.Println(theme.Dim.Render("STREAK"))
				fmt.Printf("  %-12s %s\n", theme.Muted.Render("current"), theme.Primary.Render(fmt.Sprintf("%d days", stats.Streak.Current)))
				fmt.Printf("  %-12s %s\n", theme.Muted.Render("best"), theme.Primary.Render(fmt.Sprintf("%d days", stats.Streak.Best)))
				if stats.Streak.DaysToMilestone > 0 {
					fmt.Printf("  %-12s %s\n", theme.Muted.Render("next goal"), theme.Primary.Render(fmt.Sprintf("%d days to %d", stats.Streak.DaysToMilestone, stats.Streak.NextMilestone)))
				}
				// Week dots
				dots := renderWeekDots(stats.Streak.ThisWeek)
				fmt.Printf("  %-12s %s\n", theme.Muted.Render("this week"), dots)
			}

			return nil
		},
	}
}

func sum7(arr [7]int) int {
	total := 0
	for _, v := range arr {
		total += v
	}
	return total
}

func renderWeekDots(week [7]bool) string {
	var parts []string
	labels := []string{"M", "T", "W", "T", "F", "S", "S"}
	for i, on := range week {
		if on {
			parts = append(parts, theme.Primary.Render(labels[i]))
		} else {
			parts = append(parts, theme.Muted.Render("·"))
		}
	}
	return strings.Join(parts, " ")
}
