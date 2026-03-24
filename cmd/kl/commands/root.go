package commands

import (
	"github.com/rawnaqs/khayal/cmd/kl/commands/config"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "kl",
		Short:         "Your private second brain",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runCapture(cmd, args)
		},
	}

	rootCmd.AddCommand(
		newCaptureUrlCmd(),
		newCaptureImageCmd(),
		newSearchCmd(),
		newRecentCmd(),
		newStatsCmd(),
		newStatusCmd(),
		newInitCmd(),
		config.NewConfigCmd(),
	)

	return rootCmd
}
