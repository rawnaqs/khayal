package commands

import (
	cli "github.com/rawnaqs/khayal/cmd/khayal/internal"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "View current config (token redacted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, configPath, err := cli.LoadConfig()
			if err != nil {
				cli.Fatal(cli.ExitUser, "failed to load config: %v", err)
				return err
			}

			cli.ViewConfig(cfg, configPath)
			return nil
		},
	}
}
