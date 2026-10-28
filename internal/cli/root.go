// Package cli holds the quiver commands, one file per command.
package cli

import (
	"fmt"
	"os"

	"github.com/kheif/quiver/internal/setup"
	"github.com/spf13/cobra"
)

// version is set at build time with -ldflags "-X github.com/kheif/quiver/internal/cli.version=...".
var version = "dev"

// verbose is the global --verbose flag.
var verbose bool

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "quiver",
		Short:         "Central skill manager for AI assistants",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		// Every command needs the Quiver home, so create it on first use.
		PersistentPreRunE: func(*cobra.Command, []string) error {
			return setup.EnsureInit()
		},
	}
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "print more details")
	cmd.SetVersionTemplate("quiver {{.Version}}\n")
	cmd.AddCommand(newInitCmd(), newSourceCmd(), newListCmd(), newAddCmd(), newRemoveCmd(), newStatusCmd(), newUpdateCmd(), newMCPCmd())
	return cmd
}

// Execute runs the root command and returns the process exit code.
func Execute() int {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}
