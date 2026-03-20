package config

import (
	"fmt"

	"github.com/rawnaqs/khayal/cmd/kl/internal"
	klapi "github.com/rawnaqs/khayal/cmd/kl/internal/api"
	"github.com/rawnaqs/theme"
	"github.com/spf13/cobra"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	cmd.AddCommand(newConfigViewCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigGetCmd())

	return cmd
}

func newConfigViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				internal.Warnf("no config found")
				fmt.Println("Run 'kl init' to set up configuration")
				return nil
			}

			fmt.Println()
			fmt.Println(theme.Primary.Render("khayal configuration"))
			fmt.Println()

			if cfg.Host != "" {
				fmt.Printf("  %-12s %s\n", "host", cfg.Host)
			} else {
				fmt.Printf("  %-12s %s\n", "host", theme.Muted.Render("(not set)"))
			}

			if cfg.Token != "" {
				fmt.Printf("  %-12s %s\n", "token", maskToken(cfg.Token))
			} else {
				fmt.Printf("  %-12s %s\n", "token", theme.Muted.Render("(not set)"))
			}

			fmt.Println()
			fmt.Println("config path:", theme.Muted.Render(internal.GetConfigPath()))

			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				internal.Fatal(internal.ExitUser, "%s", err.Error())
				return err
			}

			key := args[0]
			var value string

			switch key {
			case "host":
				value = cfg.Host
			case "token":
				value = cfg.Token
			default:
				internal.Fatal(internal.ExitUser, "unknown config key: %s", key)
				return fmt.Errorf("unknown config key: %s", key)
			}

			if value == "" {
				internal.Warnf("%s is not set", key)
				return nil
			}

			if key == "token" {
				value = maskToken(value)
			}

			fmt.Println(value)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				cfg = &internal.Config{}
			}

			key := args[0]
			value := args[1]

			switch key {
			case "host":
				cfg.Host = value
			case "token":
				cfg.Token = value
			default:
				internal.Fatal(internal.ExitUser, "unknown config key: %s", key)
				return fmt.Errorf("unknown config key: %s", key)
			}

			if err := internal.SaveConfig(cfg); err != nil {
				internal.Fatal(internal.ExitServer, "%s", err.Error())
				return err
			}

			internal.Successf("%s set to %s", key, value)

			if key == "host" || key == "token" {
				testConnection(cfg)
			}

			return nil
		},
	}
}

func testConnection(cfg *internal.Config) {
	client := klapi.NewClient(cfg.Host, cfg.Token)
	if err := client.CheckConnection(); err != nil {
		internal.Warnf("connection test failed: %s", err.Error())
	} else {
		internal.Successf("connection successful")
	}
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
