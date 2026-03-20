package commands

import (
	"fmt"

	ver "github.com/rawnaqs/khayal/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("  khayal %s\n", VersionCmd())
			fmt.Printf("  commit  %s\n", ver.Commit)
			fmt.Printf("  built   %s\n", ver.BuildDate)
			return nil
		},
	}
}
