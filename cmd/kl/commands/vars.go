package commands

import "github.com/spf13/cobra"

var (
	searchLimit int
	searchMode  string
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search your vault",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(args[0])
		},
	}

	cmd.Flags().IntVarP(&searchLimit, "limit", "l", 5, "max results (max: 50)")
	cmd.Flags().StringVar(&searchMode, "mode", "hybrid", "hybrid|keyword|semantic")

	return cmd
}
