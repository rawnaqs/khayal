package commands

import (
	"fmt"
	"os"

	"charm.land/huh/v2"
	"github.com/rawnaqs/khayal/cmd/kl/internal"
	klapi "github.com/rawnaqs/khayal/cmd/kl/internal/api"
	"github.com/rawnaqs/khayal/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var (
		host  string
		token string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Setup kl configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(host, token)
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "khayal server address")
	cmd.Flags().StringVar(&token, "token", "", "API token")

	return cmd
}

func runInit(host, token string) error {
	existingCfg, _ := internal.LoadConfig()
	if existingCfg != nil && existingCfg.Host != "" && existingCfg.Token != "" {
		internal.Warnf("kl is already configured")
		fmt.Println("  Run 'kl config view' to see current settings")
		fmt.Println("  Run 'kl config set <key> <value>' to update settings")
		return nil
	}

	// Interactive mode: fill in missing flags via prompts
	if token == "" {
		if host == "" {
			host = config.DefaultServerURL()
		}

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("khayal server address").
					Description(fmt.Sprintf("default: %s", config.DefaultServerURL())).
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

		if token == "" {
			internal.Warnf("token is required")
			return nil
		}
	}

	// Non-interactive: use default host if not provided
	if host == "" {
		host = config.DefaultServerURL()
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
}
