package cli

import (
	"fmt"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/setup"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "init",
		Short:             "Create the Quiver home (~/.quiver)",
		Long:              "Create the Quiver home and anything missing in it. Existing files are never overwritten.",
		Example:           `  quiver init`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		// Skip the root's auto-init, so this command reports what it creates.
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
		RunE: func(cmd *cobra.Command, _ []string) error {
			created, err := setup.Init()
			if err != nil {
				return err
			}
			home, _ := fsutil.QuiverHome()
			if len(created) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Quiver home is ready: %s\n", home)
				return nil
			}
			for _, p := range created {
				fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", p)
			}
			return nil
		},
	}
}
