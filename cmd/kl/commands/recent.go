package commands

import (
	"fmt"
	"time"

	"github.com/rawnaqs/khayal/cmd/kl/internal"
	klapi "github.com/rawnaqs/khayal/cmd/kl/internal/api"
	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"
)

func formatDateTime(iso string) string {
	if iso == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	return t.Format("January 2, 2006")
}

func newRecentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "recent",
		Short: "Show recent captures",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				internal.Fatal(internal.ExitServer, "%s", err.Error())
				return err
			}

			client := klapi.NewClient(cfg.Host, cfg.Token)
			result, err := client.Search("", "recent", 10, 200, "", "", false)
			if err != nil {
				internal.ServerUnreachable(cfg.Host)
				return err
			}

			fmt.Println()
			fmt.Println(theme.Dim.Render("RECENT"))
			fmt.Println()

			for _, r := range result.Results {
				date := formatDateTime(r.CreatedAt)
				tags := ""
				if len(r.Tags) > 0 {
					for _, t := range r.Tags {
						tags += " " + theme.RenderTag(t, r.Type)
					}
				}
				if date != "" {
					fmt.Printf("  %s", theme.Muted.Render(date))
					if tags != "" {
						fmt.Printf("  %s", tags)
					}
					fmt.Println()
				}
				fmt.Println("  " + theme.Primary.Render(r.NotePath))
				fmt.Println()
			}

			return nil
		},
	}
}
