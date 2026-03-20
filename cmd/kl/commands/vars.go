package commands

import "github.com/spf13/cobra"

var (
	searchLimit       int
	searchMode        string
	searchExcerptLen  int
	searchFrom        string
	searchTo          string
	searchConnections bool
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
	cmd.Flags().IntVar(&searchExcerptLen, "excerpt-length", 200, "max excerpt characters (max: 500)")
	cmd.Flags().StringVar(&searchFrom, "from", "", "filter: notes created after this date (ISO: 2024-01-01)")
	cmd.Flags().StringVar(&searchTo, "to", "", "filter: notes created before this date (ISO: 2024-12-31)")
	cmd.Flags().BoolVar(&searchConnections, "connections", false, "include related connections")

	return cmd
}
