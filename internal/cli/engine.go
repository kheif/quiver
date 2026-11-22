package cli

import (
	"fmt"
	"io"

	"github.com/kheif/quiver/internal/library"
	"github.com/spf13/cobra"
)

func newEngineCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "engine",
		Short: "Show or choose the engine that searches skills",
		Long: `Show the engine that quiver search, the search_skills tool and the panel
use, and any problem found in its settings file, state/search.json in the
Quiver home. The file belongs to this machine and is never synced.

The only engine now is "words": names and descriptions, ranked by words. A
missing file or an engine this quiver does not know also means words.`,
		Example: `  quiver engine
  quiver engine use words`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return showEngine(cmd.OutOrStdout(), asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	cmd.AddCommand(newEngineUseCmd())
	return cmd
}

func newEngineUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <engine>",
		Short: "Choose the search engine of this machine",
		Long: `Write the engine to state/search.json in the Quiver home. An engine this
quiver does not know is an error, and so is a file written by a newer quiver,
which is left as it is.`,
		Example: `  quiver engine use words`,
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return library.Engines(), cobra.ShellCompDirectiveNoFileComp
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := library.UseEngine(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Search engine: %s\n", args[0])
			return nil
		},
	}
}

func showEngine(out io.Writer, asJSON bool) error {
	cur := library.LoadEngine()
	if asJSON {
		return printJSON(out, statusSearch{cur.Engine, cur.Problem})
	}
	fmt.Fprintf(out, "Search engine: %s\n", cur.Engine)
	if cur.Problem != "" {
		fmt.Fprintln(out, "warning:", cur.Problem)
	}
	return nil
}
