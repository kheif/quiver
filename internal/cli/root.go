// Package cli holds the quiver commands, one file per command.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/server"
	"github.com/kheif/quiver/internal/setup"
	"github.com/spf13/cobra"
)

// version is set at build time with -ldflags "-X github.com/kheif/quiver/internal/cli.version=...".
var version = "dev"

// currentVersion is version, or for a build without ldflags, such as
// go install ...@v1.1.0, the module version Go recorded in the binary.
func currentVersion() string {
	if version != "dev" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return strings.TrimPrefix(bi.Main.Version, "v")
	}
	return version
}

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
  quiver init               set up the library and the MCP server, import your skills
  quiver new <name>         write a skill for the project in this folder
  quiver status             see what the assistant gets here`,
		Example: `  quiver init
  quiver new release-notes
  quiver source add anthropics/skills --name anthropic
  quiver add anthropic/pdf --global
  quiver status`,
		Version: currentVersion(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			if err := overview(out); err != nil {
				return err
			}
			fmt.Fprintln(out)
			fmt.Fprint(out, cmd.UsageString())
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		// Every command needs the Quiver home, so create it on first use.
		// Shell completion only reads, so it never creates it; quiver alone
		// does not either, so it can suggest quiver init.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if !cmd.HasParent() || cmd.Name() == cobra.ShellCompRequestCmd || cmd.Name() == cobra.ShellCompNoDescRequestCmd {
				return nil
			}
			return setup.EnsureInit()
		},
	}
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "print more details")
	cmd.SetVersionTemplate("quiver {{.Version}}\n")
	cmd.AddCommand(newInitCmd(), newSourceCmd(), newNewCmd(), newEditCmd(), newDeleteCmd(), newImportCmd(), newForkCmd(), newListCmd(), newShowCmd(), newSearchCmd(), newAddCmd(), newRemoveCmd(), newStatusCmd(), newWhereCmd(), newProjectsCmd(), newUpdateCmd(), newDoctorCmd(), newMCPCmd())
	return cmd
}

// overview prints three lines for quiver without arguments: the project
// here, the skills served here and a next step.
func overview(out io.Writer) error {
	w, err := server.Resolve()
	if err != nil {
		return err
	}
	here := fmt.Sprintf("Folder:  %s (not a project; only global skills are served)", w.Dir)
	home, err := fsutil.QuiverHome()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(home, "registry.json")); errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(out, "%s\nServed:  nothing yet; the Quiver home %s does not exist\nNext:    quiver init\n", here, home)
		return nil
	}
	r, err := registry.Load()
	if err != nil {
		return err
	}
	l, err := library.Load()
	if err != nil {
		return err
	}
	if p := w.Project; p != nil {
		name := filepath.Base(p.Root)
		if rp, ok := r.Projects[p.ID]; ok {
			name = rp.Name
		}
		here = fmt.Sprintf("Project: %s (%s)", name, p.Root)
	}
	served := r.Served(w.ProjectID())
	global := 0
	for _, id := range served {
		if slices.Contains(r.Global, id) {
			global++
		}
	}
	count := plural(len(served), "skill")
	if len(served) > 0 {
		count += fmt.Sprintf(" (%d global, %d project", global, len(served)-global)
		if n := len(r.AlwaysOnServed(w.ProjectID())); n > 0 {
			count += fmt.Sprintf(", %d always-on", n)
		}
		count += ")"
	}
	next := "quiver status"
	switch {
	case len(l.All()) == 0:
		next = "quiver new <name> to write a skill, or quiver source add <repo>"
	case len(served) == 0:
		next = "quiver add <id> (see quiver list)"
	}
	fmt.Fprintf(out, "%s\nServed:  %s\nNext:    %s\n", here, count, next)
	return nil
}

// Execute runs the root command and returns the process exit code.
func Execute() int {
	return execute(newRootCmd(), os.Args[1:], os.Stderr)
}

// execute runs root with args. While it runs, the check for a new Quiver
// version runs too; its line comes last.
func execute(root *cobra.Command, args []string, stderr io.Writer) int {
	check := startVersionCheck(root, args, stderr)
	root.SetArgs(args)
	ran, err := root.ExecuteC()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
	}
	check.finish(ran, stderr)
	if err != nil {
		return 1
	}
	return 0
}
