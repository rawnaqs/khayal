package commands

import (
	"github.com/rawnaqs/khayal/internal/version"
	"github.com/spf13/cobra"
)

var (
	appVersion string
)

func InitVersion(v string) {
	appVersion = v
}

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "khayal",
		Short: "khayal server admin CLI",
		Long: `khayal - Your private second brain

Server management:
  khayal init      First-run setup
  khayal start     Start server + worker
  khayal stop      Graceful shutdown
  khayal restart   Stop + start

Management:
  khayal status    Interactive dashboard
  khayal reindex   Rebuild embeddings

Configuration:
  khayal config    View configuration
  khayal version   Version info
`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(
		newInitCmd(),
		newStartCmd(),
		newStopCmd(),
		newRestartCmd(),
		newStatusCmd(),
		newReindexCmd(),
		newVersionCmd(),
		newConfigCmd(),
	)

	return rootCmd
}

func VersionCmd() string {
	if appVersion != "" {
		return appVersion
	}
	return version.Get()
}
