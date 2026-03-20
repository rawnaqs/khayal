package commands

import (
	"fmt"

	"github.com/rawnaqs/khayal/cmd/kl/internal"
	klapi "github.com/rawnaqs/khayal/cmd/kl/internal/api"
	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Quick server + queue check",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				internal.Fatal(internal.ExitServer, "%s", err.Error())
				return err
			}

			client := klapi.NewClient(cfg.Host, cfg.Token)
			health, err := client.Status()
			if err != nil {
				internal.ServerUnreachable(cfg.Host)
				return err
			}

			fmt.Println()
			internal.Successf("khayal %s · %s", health.Version, cfg.Host)
			fmt.Println()
			fmt.Println(theme.Dim.Render("QUEUE"))
			keyStyle := theme.Muted.Width(12)
			fmt.Printf("  %s %s\n", keyStyle.Render("processing"), theme.Primary.Render(fmt.Sprintf("%d", health.Queue.Processing)))
			fmt.Printf("  %s %s\n", keyStyle.Render("pending"), theme.Primary.Render(fmt.Sprintf("%d", health.Queue.Pending)))
			fmt.Printf("  %s %s\n", keyStyle.Render("done"), theme.Primary.Render(fmt.Sprintf("%d", health.Queue.Done)))
			fmt.Printf("  %s %s\n", keyStyle.Render("failed"), theme.Primary.Render(fmt.Sprintf("%d", health.Queue.Failed)))
			fmt.Println()

			return nil
		},
	}
}
