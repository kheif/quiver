package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

// appCommand returns the path the JSON-configured assistants should start
// Quiver with: always the binary itself. Desktop apps do not get the shell's
// PATH, so the npm launcher, a node script, fails there with
// "env: node: No such file or directory".
func appCommand() (string, error) {
	return executable()
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
			return server.Run(cmd.Context(), currentVersion())
		},
	}
	cmd.AddCommand(newMCPInstallCmd(), newMCPUninstallCmd(), newMCPConfigCmd())
	return cmd
}

// claudeCode is the --for value of Claude Code, which is set up with the
// claude command. The other assistants are configured by JSON.
const claudeCode = "claude-code"

// mcpTarget is an assistant mcp install or uninstall works on: Claude Code
// when json is nil.
type mcpTarget struct {
	json *assistant.JSONAssistant
}

func (t mcpTarget) name() string {
	if t.json == nil {
		return "Claude Code"
	}
	return t.json.Name
}

// assistantIDs returns the --for values.
func assistantIDs() ([]string, error) {
	all, err := assistant.JSONAssistants()
	if err != nil {
		return nil, err
	}
	ids := []string{claudeCode}
	for _, a := range all {
		ids = append(ids, a.ID)
	}
	return ids, nil
}

// mcpTargets returns the assistant named by id, or with id "" every assistant
// that is installed. Claude Code counts as installed when the claude command
// is on PATH. An assistant that drops the server's instructions is only
// included by name, or for uninstall when it has a quiver entry.
func mcpTargets(id string, uninstall bool) ([]mcpTarget, error) {
	all, err := assistant.JSONAssistants()
	if err != nil {
		return nil, err
	}
	if id == claudeCode {
		return []mcpTarget{{}}, nil
	}
	var ts []mcpTarget
	if id == "" && assistant.HasClaude() {
		ts = append(ts, mcpTarget{})
	}
	for _, a := range all {
		if a.ID == id || (id == "" && a.Installed() && (!a.NoInstructions || uninstall && hasEntry(a))) {
			ts = append(ts, mcpTarget{a})
		}
	}
	if id != "" && len(ts) == 0 {
		ids, _ := assistantIDs()
		return nil, fmt.Errorf("unknown assistant %q; use one of: %s", id, strings.Join(ids, ", "))
	}
	return ts, nil
}

// hasEntry reports whether a has a quiver entry. A config that cannot be
// read counts as registered, so the caller reports its error.
func hasEntry(a *assistant.JSONAssistant) bool {
	reg, err := a.Get()
	return reg != nil || err != nil
}

func addForFlag(cmd *cobra.Command, forID *string, verb string) {
	ids, _ := assistantIDs()
	cmd.Flags().StringVar(forID, "for", "", verb+" only this assistant: "+strings.Join(ids, ", "))
	_ = cmd.RegisterFlagCompletionFunc("for", cobra.FixedCompletions(ids, cobra.ShellCompDirectiveNoFileComp))
}

func newMCPInstallCmd() *cobra.Command {
	var printOnly bool
	var forID string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Register the MCP server with your assistants",
		Long: `Register the MCP server with every assistant that is installed: Claude Code
(for every project of this user) and Cursor. With --for, only with that
assistant. Claude Desktop is set up only with --for claude-desktop: it does
not pass Quiver's skill list to the model, so skills are used only when you
ask for them. For any other assistant, add the entry that quiver mcp config
prints to its MCP config.

Run it again after moving the quiver binary: a registration that points to
another binary is replaced. In Cursor's and Claude Desktop's JSON files,
every other key is kept, and the old file is saved next to it with the
suffix .quiver-backup.`,
		Example: `  quiver mcp install

  # Only Cursor
  quiver mcp install --for cursor`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if printOnly {
				return mcpPrint(cmd.OutOrStdout())
			}
			return mcpInstall(cmd.OutOrStdout(), forID)
		},
	}
	addForFlag(cmd, &forID, "set up")
	cmd.Flags().BoolVar(&printOnly, "print", false, "only print the setup, change nothing")
	_ = cmd.Flags().MarkDeprecated("print", "use quiver mcp config")
	return cmd
}

func mcpPrint(out io.Writer) error {
	bin, _, err := command()
	if err != nil {
		return err
	}
	appBin, err := appCommand()
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Claude Code:\n  %s\n\nOther assistants (MCP config JSON):\n%s\n", assistant.AddCommand(bin), assistant.ConfigSnippet(appBin))
	return nil
}

func mcpInstall(out io.Writer, forID string) error {
	bin, viaNpm, err := command()
	if err != nil {
		return err
	}
	ts, err := mcpTargets(forID, false)
	if err != nil {
		return err
	}
	if len(ts) == 0 {
		fmt.Fprintf(out, "No assistant was found. To register Quiver with Claude Code, run:\n  %s\nFor another assistant, add this to its MCP config:\n%s\n", assistant.AddCommand(bin), assistant.ConfigSnippet(bin))
		return nil
	}
	appBin, err := appCommand()
	if err != nil {
		return err
	}
	var errs []error
	for _, t := range ts {
		if t.json == nil {
			if err := installClaude(out, bin); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		a, bin := t.json, appBin
		res, err := a.Install(bin)
		if err != nil {
			fmt.Fprintf(out, "%s: not registered: %v\n", a.Name, err)
			errs = append(errs, err)
			continue
		}
		switch res {
		case assistant.Unchanged:
			fmt.Fprintf(out, "Already registered with %s: %s mcp\n", a.Name, bin)
		case assistant.Updated:
			fmt.Fprintf(out, "Updated the %s registration to %s mcp (%s)\n", a.Name, bin, a.Path)
		default:
			fmt.Fprintf(out, "Registered with %s: %s mcp (%s)\n", a.Name, bin, a.Path)
		}
		if res != assistant.Unchanged {
			fmt.Fprintf(out, "Restart %s to use it.\n", a.Name)
		}
		if a.Global {
			fmt.Fprintf(out, "%s does not start Quiver in a project folder, so it gets only the global skills.\n", a.Name)
		}
		if a.NoInstructions {
			fmt.Fprintf(out, "%s does not pass Quiver's skill list to the model: a skill is used only when you ask for Quiver's skills in the prompt, and always-on skills do not apply.\n", a.Name)
		}
	}
	if viaNpm {
		fmt.Fprintln(out, "Installed with npm: the path depends on your Node version. After switching Node versions (nvm, fnm, volta), run quiver mcp install again.")
	}
	return errors.Join(errs...)
}

func installClaude(out io.Writer, bin string) error {
	res, hidden, err := assistant.Install(bin)
	if errors.Is(err, assistant.ErrNoClaude) {
		fmt.Fprintf(out, "The claude command was not found. To register Quiver with Claude Code, run:\n  %s\n", assistant.AddCommand(bin))
		return nil
	}
	if err != nil {
		fmt.Fprintf(out, "Claude Code: not registered: %v\n", err)
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
	var forID string
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the MCP server from your assistants",
		Long: `Remove the MCP server from every assistant that has it, or with --for from
that assistant only. The Quiver home and its skills are not touched.`,
		Example: `  quiver mcp uninstall

  # Only Claude Desktop
  quiver mcp uninstall --for claude-desktop`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mcpUninstall(cmd.OutOrStdout(), forID)
		},
	}
	addForFlag(cmd, &forID, "remove from")
	return cmd
}

func mcpUninstall(out io.Writer, forID string) error {
	ts, err := mcpTargets(forID, true)
	if err != nil {
		return err
	}
	if len(ts) == 0 {
		fmt.Fprintf(out, "No assistant was found. To remove Quiver from Claude Code, run:\n  %s\n", assistant.RemoveCommand())
		return nil
	}
	var errs []error
	for _, t := range ts {
		var res assistant.Result
		var err error
		if t.json == nil {
			res, err = assistant.Uninstall()
			if errors.Is(err, assistant.ErrNoClaude) {
				fmt.Fprintf(out, "The claude command was not found. To remove Quiver from Claude Code, run:\n  %s\n", assistant.RemoveCommand())
				continue
			}
		} else {
			res, err = t.json.Uninstall()
		}
		switch {
		case err != nil:
			fmt.Fprintf(out, "%s: not removed: %v\n", t.name(), err)
			errs = append(errs, err)
		case res == assistant.Removed:
			fmt.Fprintf(out, "Removed from %s.\n", t.name())
		case t.json == nil:
			fmt.Fprintln(out, "Quiver was not registered with Claude Code for this user.")
		default:
			fmt.Fprintf(out, "Quiver was not registered with %s.\n", t.name())
		}
	}
	return errors.Join(errs...)
}

func newMCPConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Print the MCP config entry for any assistant",
		Long: `Print the mcpServers entry that starts this quiver, for an assistant that
quiver mcp install does not set up. Add it to that assistant's MCP config.
Nothing is changed.`,
		Example:           `  quiver mcp config`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			bin, err := appCommand()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), assistant.ConfigSnippet(bin))
			return nil
		},
	}
}
