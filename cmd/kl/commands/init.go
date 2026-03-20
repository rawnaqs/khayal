package commands

import (
	"fmt"
	"os"

	"charm.land/huh/v2"
	"github.com/rawnaqs/khayal/cmd/kl/internal"
	klapi "github.com/rawnaqs/khayal/cmd/kl/internal/api"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Setup kl configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			existingCfg, _ := internal.LoadConfig()
			if existingCfg != nil && existingCfg.Host != "" && existingCfg.Token != "" {
				internal.Warnf("kl is already configured")
				fmt.Println("Run 'kl config view' to see current settings")
				fmt.Println("Run 'kl config set <key> <value>' to update settings")
				return nil
			}

			var host string
			var token string

			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("khayal server address").
						Description("e.g., http://localhost:1133").
						Prompt(" host: ").
						Value(&host),
					huh.NewInput().
						Title("API token").
						Description("Found in ~/.config/khayal/config.yaml on server").
						Prompt(" token: ").
						Value(&token),
				),
			)

			if err := form.Run(); err != nil {
				internal.Fatal(internal.ExitUser, "%s", err.Error())
				return err
			}

			if host == "" || token == "" {
				internal.Warnf("host and token are required")
				return nil
			}

			cfg := &internal.Config{
				Host:  host,
				Token: token,
			}

			if err := internal.SaveConfig(cfg); err != nil {
				internal.Fatal(internal.ExitServer, "%s", err.Error())
				return err
			}

			internal.Successf("configuration saved")

			client := klapi.NewClient(host, token)
			if err := client.CheckConnection(); err != nil {
				internal.Warnf("connection test failed")
				internal.ActionableError("could not reach khayal server", []string{
					"check that khayal is running on server",
					"verify the host address is correct",
					"confirm the token is valid",
				})
			} else {
				internal.Successf("connection successful · kl is ready")
			}

			os.Exit(0)
			return nil
		},
	}
}
