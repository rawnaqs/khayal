package commands

import (
	"fmt"
	"runtime"

	"github.com/rawnaqs/khayal/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print kl version info",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("kl %s\n", version.Get())
			if version.Commit != "" {
				fmt.Printf("commit  %s\n", version.Commit)
			}
			if version.BuildDate != "" {
				fmt.Printf("built   %s\n", version.BuildDate)
			}
			fmt.Printf("go      %s\n", runtime.Version())
		},
	}
}
