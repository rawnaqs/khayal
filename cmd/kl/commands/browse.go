package commands

import (
	"fmt"

	"github.com/rawnaqs/khayal/cmd/kl/internal"
	klapi "github.com/rawnaqs/khayal/cmd/kl/internal/api"
	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"
)

var browseFilter string
var browseLimit int

func newBrowseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse notes by tag, type, or date",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				internal.Fatal(internal.ExitServer, "%s", err.Error())
				return err
			}

			client := klapi.NewClient(cfg.Host, cfg.Token)

			var filter, value string
			if browseFilter != "" {
				filter = browseFilter
				value = ""
				if len(args) > 0 {
					value = args[0]
				}
			} else if len(args) > 0 {
				filter = "tag"
				value = args[0]
			} else {
				filter = "all"
				value = ""
			}

			result, err := client.Browse(filter, value, browseLimit)
			if err != nil {
				internal.ServerUnreachable(cfg.Host)
				return err
			}

			fmt.Println()
			if filter == "all" {
				fmt.Println(theme.Dim.Render("ALL NOTES"))
			} else {
				fmt.Println(theme.Dim.Render(filter + ": " + value))
			}
			fmt.Println()

			if len(result.Notes) == 0 {
				fmt.Println(theme.Dim.Render("no notes found"))
				return nil
			}

			fmt.Printf("%d notes\n\n", result.Total)

			for _, n := range result.Notes {
				tags := ""
				for _, t := range n.Tags {
					tags += " " + theme.Tag.Render(t)
				}
				fmt.Printf("  %s  %s\n", theme.Dim.Render(n.Date), theme.Primary.Render(n.Path))
				if tags != "" {
					fmt.Printf("    %s\n", tags)
				}
				if n.Excerpt != "" {
					fmt.Printf("    %s\n", theme.Primary.Render(n.Excerpt))
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&browseFilter, "filter", "f", "", "Filter type (tag, type, date)")
	cmd.Flags().IntVarP(&browseLimit, "limit", "l", 20, "Number of results to show")

	return cmd
}
