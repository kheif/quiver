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
		Use:   "quiver",
		Short: "Central skill manager for AI assistants",
		Long: `Quiver keeps your AI assistant skills in one library (~/.quiver) and serves
them over MCP: the global skills in every folder, and each project's own
skills in that project. Skills are never copied into projects.

Get started:
  quiver mcp install        register the MCP server with Claude Code
  quiver new <name>         write a skill for the project in this folder
  quiver status             see what the assistant gets here`,
		Example: `  quiver mcp install
  quiver new release-notes
  quiver source add anthropics/skills --name anthropic
  quiver add anthropic/pdf --global
  quiver status`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		// Every command needs the Quiver home, so create it on first use.
		// Shell completion only reads, so it never creates it.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Name() == cobra.ShellCompRequestCmd || cmd.Name() == cobra.ShellCompNoDescRequestCmd {
				return nil
			}
			return setup.EnsureInit()
		},
	}
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "print more details")
	cmd.SetVersionTemplate("quiver {{.Version}}\n")
	cmd.AddCommand(newInitCmd(), newSourceCmd(), newNewCmd(), newEditCmd(), newDeleteCmd(), newImportCmd(), newListCmd(), newAddCmd(), newRemoveCmd(), newStatusCmd(), newWhereCmd(), newProjectsCmd(), newUpdateCmd(), newDoctorCmd(), newMCPCmd())
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
