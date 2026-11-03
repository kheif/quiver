package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
do not need to run it yourself. Register it once with: quiver install

The server serves the skills of the project in its working directory, or in
$QUIVER_PROJECT when that is set. It logs to ~/.quiver/logs.`,
		Example: `  # What Claude Code runs after quiver install
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

// The install names of the assistants set up with their own command:
// Claude Code with claude, Codex with codex. The other assistants are
// configured by JSON.
const (
	claudeCode = "claude-code"
	codexID    = "codex"
)

// mcpTarget is an assistant install or uninstall works on: Claude Code
// or Codex by id when json is nil.
type mcpTarget struct {
	id   string
	json *assistant.JSONAssistant
}

func (t mcpTarget) name() string {
	switch {
	case t.json != nil:
		return t.json.Name
	case t.id == codexID:
		return "Codex"
	}
	return "Claude Code"
}

// assistantIDs returns the names install and uninstall take.
func assistantIDs() ([]string, error) {
	all, err := assistant.JSONAssistants()
	if err != nil {
		return nil, err
	}
	ids := []string{claudeCode, codexID}
	for _, a := range all {
		ids = append(ids, a.ID)
	}
	return ids, nil
}

// mcpTargets returns the assistants named by ids, or with no ids every
// assistant that is installed. Claude Code and Codex count as installed when
// their command is on PATH.
func mcpTargets(ids []string) ([]mcpTarget, error) {
	jsons, err := assistant.JSONAssistants()
	if err != nil {
		return nil, err
	}
	known, err := assistantIDs()
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if !slices.Contains(known, id) {
			return nil, fmt.Errorf("unknown assistant %q; use one of: %s", id, strings.Join(known, ", "))
		}
	}
	all := len(ids) == 0
	var ts []mcpTarget
	if slices.Contains(ids, claudeCode) || all && assistant.HasClaude() {
		ts = append(ts, mcpTarget{id: claudeCode})
	}
	if slices.Contains(ids, codexID) || all && assistant.HasCodex() {
		ts = append(ts, mcpTarget{id: codexID})
	}
	for _, a := range jsons {
		if slices.Contains(ids, a.ID) || all && a.Installed() {
			ts = append(ts, mcpTarget{id: a.ID, json: a})
		}
	}
	return ts, nil
}

// supportedList returns the names install takes, for the help.
func supportedList() string {
	ids, _ := assistantIDs()
	return strings.Join(ids, ", ")
}

// assistantArgs completes the assistant names not given yet.
func assistantArgs(_ *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	ids, _ := assistantIDs()
	return suggest(ids, args, toComplete), noFiles
}

// addForFlag adds --for, with which v1.2.0 named one assistant. It counts
// as one more argument. prefix is the command path without the leaf, named in
// the deprecation message.
func addForFlag(cmd *cobra.Command, forID *string, prefix string) {
	cmd.Flags().StringVar(forID, "for", "", "only this assistant")
	_ = cmd.Flags().MarkDeprecated("for", "name the assistant as an argument: "+prefix+" "+cmd.Name()+" <assistant>")
	ids, _ := assistantIDs()
	_ = cmd.RegisterFlagCompletionFunc("for", cobra.FixedCompletions(ids, cobra.ShellCompDirectiveNoFileComp))
}

// withFor adds the assistant --for names to args.
func withFor(args []string, forID string) []string {
	if forID == "" {
		return args
	}
	return append(args, forID)
}

// newMCPInstallCmd is the hidden, deprecated alias quiver mcp install; the
// command is now quiver install. It is removed in the next major version.
func newMCPInstallCmd() *cobra.Command { return installCommand("quiver mcp", true) }

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

func mcpInstall(out io.Writer, ids []string) error {
	bin, viaNpm, err := command()
	if err != nil {
		return err
	}
	ts, err := mcpTargets(ids)
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
		var err error
		switch t.id {
		case claudeCode:
			err = installClaude(out, bin)
		case codexID:
			err = installCodex(out, appBin)
		default:
			err = installJSON(out, t.json, appBin)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	if viaNpm {
		fmt.Fprintln(out, "Installed with npm: the path depends on your Node version. After switching Node versions (nvm, fnm, volta), run quiver install again.")
	}
	return errors.Join(errs...)
}

// onRequestNote says that name uses Quiver's skills only when asked.
func onRequestNote(out io.Writer, name string) {
	fmt.Fprintf(out, "%s uses Quiver's skills only when you ask for them in the prompt, such as \"check the quiver skills for this\".\n", name)
}

func installJSON(out io.Writer, a *assistant.JSONAssistant, bin string) error {
	res, err := a.Install(bin)
	if err != nil {
		fmt.Fprintf(out, "%s: not registered: %v\n", a.Name, err)
		return err
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
	if a.OnRequest {
		onRequestNote(out, a.Name)
	}
	return nil
}

// installCodex registers bin with Codex. The Codex app reads the same
// config, and desktop apps have no node on PATH, so bin is the binary itself.
func installCodex(out io.Writer, bin string) error {
	res, err := assistant.CodexInstall(bin)
	if errors.Is(err, assistant.ErrNoCodex) {
		fmt.Fprintf(out, "The codex command was not found. To register Quiver with Codex, run:\n  %s\n", assistant.CodexAddCommand(bin))
		return nil
	}
	if err != nil {
		fmt.Fprintf(out, "Codex: not registered: %v\n", err)
		return err
	}
	switch res {
	case assistant.Unchanged:
		fmt.Fprintf(out, "Already registered with Codex: %s mcp\n", bin)
	case assistant.Updated:
		fmt.Fprintf(out, "Updated the Codex registration to %s mcp\n", bin)
	default:
		fmt.Fprintf(out, "Registered with Codex: %s mcp\n", bin)
	}
	if res != assistant.Unchanged {
		fmt.Fprintln(out, "Start a new Codex session to use it.")
	}
	onRequestNote(out, "Codex")
	return nil
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

// newMCPUninstallCmd is the hidden, deprecated alias quiver mcp uninstall; the
// command is now quiver uninstall. It is removed in the next major version.
func newMCPUninstallCmd() *cobra.Command { return uninstallCommand("quiver mcp", true) }

func mcpUninstall(out io.Writer, ids []string) error {
	ts, err := mcpTargets(ids)
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
		switch t.id {
		case claudeCode:
			res, err = assistant.Uninstall()
			if errors.Is(err, assistant.ErrNoClaude) {
				fmt.Fprintf(out, "The claude command was not found. To remove Quiver from Claude Code, run:\n  %s\n", assistant.RemoveCommand())
				continue
			}
		case codexID:
			res, err = assistant.CodexUninstall()
			if errors.Is(err, assistant.ErrNoCodex) {
				fmt.Fprintf(out, "The codex command was not found. To remove Quiver from Codex, run:\n  %s\n", assistant.CodexRemoveCommand())
				continue
			}
		default:
			res, err = t.json.Uninstall()
		}
		switch {
		case err != nil:
			fmt.Fprintf(out, "%s: not removed: %v\n", t.name(), err)
			errs = append(errs, err)
		case res == assistant.Removed:
			fmt.Fprintf(out, "Removed from %s.\n", t.name())
		case t.id == claudeCode:
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
quiver install does not set up. Add it to that assistant's MCP config.
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
