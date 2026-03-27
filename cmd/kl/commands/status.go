package commands

import (
	"fmt"

	"github.com/rawnaqs/khayal/cmd/kl/internal"
	klapi "github.com/rawnaqs/khayal/cmd/kl/internal/api"
	"github.com/rawnaqs/khayal/internal/version"
	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Server status + update check",
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

			// Update messaging
			if health.Update != nil {
				msg := getUpdateMessage(version.Get(), health.Update)
				if msg != "" {
					fmt.Println(theme.Dim.Render(fmt.Sprintf("  %s", msg)))
					fmt.Println()
				}
			}

			return nil
		},
	}
}

func getUpdateMessage(klVersion string, update *klapi.UpdateInfo) string {
	if !update.Available {
		return ""
	}

	klIsCurrent := klVersion == update.Latest
	serverIsCurrent := update.ServerVersion == update.Latest

	switch {
	case klIsCurrent:
		// kl is current, server is behind
		return fmt.Sprintf("↑ update khayal server to v%s", update.Latest)
	case serverIsCurrent:
		// server is current, kl is behind
		return fmt.Sprintf("↑ update kl to v%s", update.Latest)
	default:
		// both are behind
		return fmt.Sprintf("↑ update khayal server + kl to v%s", update.Latest)
	}
}
