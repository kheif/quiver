package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/kheif/quiver/internal/assistant"
	"github.com/kheif/quiver/internal/server"
	"github.com/spf13/cobra"
)

// executable returns the path assistants start Quiver with. Symlinks are
// kept: Homebrew's stable path links to a versioned folder that an upgrade
// deletes.
var executable = os.Executable

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run the MCP server on stdio (assistants start this)",
		Long: `Run the MCP server on stdin and stdout. Assistants start this command; you
do not need to run it yourself. Register it once with: quiver mcp install

The server serves the skills of the project in its working directory, or in
$QUIVER_PROJECT when that is set. It logs to ~/.quiver/logs.`,
		Example: `  # What Claude Code runs after quiver mcp install
  quiver mcp`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return server.Run(cmd.Context(), version)
		},
	}
	cmd.AddCommand(newMCPInstallCmd(), newMCPUninstallCmd())
	return cmd
}

func newMCPInstallCmd() *cobra.Command {
	var printOnly bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Register the MCP server with Claude Code",
		Long: `Register the MCP server with Claude Code, for every project of this user.

Run it again after moving the quiver binary: a registration that points to
another binary is replaced. With --print nothing is changed; the command and
a JSON snippet for manual setup are printed.`,
		Example: `  quiver mcp install

  # Print the setup for Claude Code and for JSON-configured assistants
  quiver mcp install --print`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mcpInstall(cmd.OutOrStdout(), printOnly)
		},
	}
	cmd.Flags().BoolVar(&printOnly, "print", false, "only print the setup, change nothing")
	return cmd
}

func mcpInstall(out io.Writer, printOnly bool) error {
	bin, err := executable()
	if err != nil {
		return err
	}
	if printOnly {
		fmt.Fprintf(out, "Claude Code:\n  %s\n\nOther assistants (MCP config JSON):\n%s\n", assistant.AddCommand(bin), assistant.ConfigSnippet(bin))
		return nil
	}
	res, hidden, err := assistant.Install(bin)
	if errors.Is(err, assistant.ErrNoClaude) {
		fmt.Fprintf(out, "The claude command was not found. To register Quiver with Claude Code, run:\n  %s\n", assistant.AddCommand(bin))
		return nil
	}
	if err != nil {
		return err
	}
	switch res {
	case assistant.Unchanged:
		fmt.Fprintf(out, "Already registered with Claude Code: %s mcp\n", bin)
	case assistant.Updated:
		fmt.Fprintf(out, "Updated the Claude Code registration to %s mcp\n", bin)
	default:
		fmt.Fprintf(out, "Registered with Claude Code: %s mcp\n", bin)
	}
	if hidden != nil {
		fmt.Fprintf(out, "warning: in this folder, a registration from %s runs %s %s instead\n", hidden.Scope, hidden.Command, hidden.Args)
	}
	if res != assistant.Unchanged {
		fmt.Fprintln(out, "Start a new Claude Code session to use it.")
	}
	return nil
}

func newMCPUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the MCP server from Claude Code",
		Long: `Remove the MCP server from Claude Code's user configuration. The Quiver
home and its skills are not touched.`,
		Example:           `  quiver mcp uninstall`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mcpUninstall(cmd.OutOrStdout())
		},
	}
}

func mcpUninstall(out io.Writer) error {
	res, err := assistant.Uninstall()
	if errors.Is(err, assistant.ErrNoClaude) {
		fmt.Fprintf(out, "The claude command was not found. To remove Quiver from Claude Code, run:\n  %s\n", assistant.RemoveCommand())
		return nil
	}
	if err != nil {
		return err
	}
	if res == assistant.Removed {
		fmt.Fprintln(out, "Removed from Claude Code.")
	} else {
		fmt.Fprintln(out, "Quiver was not registered with Claude Code for this user.")
	}
	return nil
}
