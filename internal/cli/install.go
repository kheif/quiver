package cli

import "github.com/spf13/cobra"

// newInstallCmd and newUninstallCmd are the top-level quiver install and
// quiver uninstall. The same commands are kept under mcp as hidden,
// deprecated aliases (newMCPInstallCmd, newMCPUninstallCmd) until the next
// major version.
func newInstallCmd() *cobra.Command   { return installCommand("quiver", false) }
func newUninstallCmd() *cobra.Command { return uninstallCommand("quiver", false) }

// installCommand builds the install command. prefix is the command path
// without the leaf ("quiver" or "quiver mcp"), used in the help and in the
// --for deprecation message. When deprecated is set, the command is the
// hidden alias under mcp that points to quiver install.
func installCommand(prefix string, deprecated bool) *cobra.Command {
	var printOnly bool
	var forID string
	cmd := &cobra.Command{
		Use:   "install [assistant...]",
		Short: "Register the MCP server with your assistants",
		Long: `Register the MCP server with every supported assistant that is installed,
or only with the assistants named. Some assistants use Quiver's skills only
when you ask for them in the prompt; quiver install says which. For any other
assistant, add the entry that quiver mcp config prints to its MCP config.

Supported assistants: ` + supportedList() + `

Run it again after moving the quiver binary: a registration that points to
another binary is replaced. In the assistants' JSON files, every other key
is kept, and the old file is saved next to it with the suffix
.quiver-backup.`,
		Example: `  ` + prefix + ` install

  # Only Cursor
  ` + prefix + ` install cursor

  # Claude Code and Claude Desktop
  ` + prefix + ` install claude-code claude-desktop`,
		ValidArgsFunction: assistantArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if printOnly {
				return mcpPrint(cmd.OutOrStdout())
			}
			return mcpInstall(cmd.OutOrStdout(), withFor(args, forID))
		},
	}
	cmd.Flags().BoolVar(&printOnly, "print", false, "only print the setup, change nothing")
	_ = cmd.Flags().MarkDeprecated("print", "use quiver mcp config")
	addForFlag(cmd, &forID, prefix)
	if deprecated {
		cmd.Hidden = true
		cmd.Deprecated = `use "quiver install"`
	}
	return cmd
}

// uninstallCommand builds the uninstall command; see installCommand.
func uninstallCommand(prefix string, deprecated bool) *cobra.Command {
	var forID string
	cmd := &cobra.Command{
		Use:   "uninstall [assistant...]",
		Short: "Remove the MCP server from your assistants",
		Long: `Remove the MCP server from every supported assistant that is installed, or
only from the assistants named. The Quiver home and its skills are not
touched.

Supported assistants: ` + supportedList(),
		Example: `  ` + prefix + ` uninstall

  # Only Claude Desktop
  ` + prefix + ` uninstall claude-desktop`,
		ValidArgsFunction: assistantArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mcpUninstall(cmd.OutOrStdout(), withFor(args, forID))
		},
	}
	addForFlag(cmd, &forID, prefix)
	if deprecated {
		cmd.Hidden = true
		cmd.Deprecated = `use "quiver uninstall"`
	}
	return cmd
}
