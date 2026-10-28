package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kheif/quiver/internal/assistant"
	"github.com/kheif/quiver/internal/server"
	"github.com/spf13/cobra"
)

// executable returns the path of this binary. Symlinks are kept: a stable
// link may point to a versioned folder that an upgrade deletes.
var executable = os.Executable

// npmEnv is set by the npm package's launcher (npm/quiver/bin/quiver.js).
const npmEnv = "QUIVER_NPM"

// command returns the path assistants should start Quiver with. Installed
// with npm, the binary sits inside node_modules, in a folder that changes
// with the package version, so the quiver command on PATH is used instead.
// viaNpm reports that case.
func command() (path string, viaNpm bool, err error) {
	if os.Getenv(npmEnv) == "1" {
		if p, err := exec.LookPath("quiver"); err == nil {
			if abs, err := filepath.Abs(p); err == nil {
				return abs, true, nil
			}
		}
	}
	bin, err := executable()
	return bin, false, err
}

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
	bin, viaNpm, err := command()
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
	if viaNpm {
		fmt.Fprintln(out, "Installed with npm: the path depends on your Node version. After switching Node versions (nvm, fnm, volta), run quiver mcp install again.")
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
