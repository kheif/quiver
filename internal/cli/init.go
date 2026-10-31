package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/setup"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create the Quiver home (~/.quiver) and set up Quiver",
		Long: `Create the Quiver home and anything missing in it. Existing files are never
overwritten.

In a terminal it then offers the rest of the setup, one step at a time; each
can be skipped:

  1. register the MCP server with your assistants (quiver mcp install)
  2. import the skills in ~/.claude/skills, for every folder
  3. import the skills in the .claude/skills of the project in this folder

With --yes, or when not run in a terminal, it only creates the home.`,
		Example: `  quiver init

  # Only create the home
  quiver init --yes`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		// Skip the root's auto-init, so this command reports what it creates.
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			if err := createHome(out); err != nil {
				return err
			}
			if yes || !interactive(cmd.InOrStdin()) {
				return nil
			}
			return guidedSetup(bufio.NewReader(cmd.InOrStdin()), out)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "ask nothing; only create the home")
	return cmd
}

// createHome creates the Quiver home and prints what it created.
func createHome(out io.Writer) error {
	created, err := setup.Init()
	if err != nil {
		return err
	}
	home, _ := fsutil.QuiverHome()
	if len(created) == 0 {
		fmt.Fprintf(out, "Quiver home is ready: %s\n", home)
		return nil
	}
	for _, p := range created {
		fmt.Fprintf(out, "created %s\n", p)
	}
	return nil
}

// guidedSetup offers the steps after the home is created. Every question
// reads from in, so no answer is read ahead and lost.
func guidedSetup(in *bufio.Reader, out io.Writer) error {
	var errs []error
	fmt.Fprintln(out)
	if confirmYes(in, out, "Register the MCP server with your assistants?") {
		if err := mcpInstall(out, ""); err != nil {
			errs = append(errs, err)
		}
	} else {
		fmt.Fprintln(out, "Skipped. Run it later with: quiver mcp install")
	}

	claude, err := library.ClaudeDir()
	if err != nil {
		return errors.Join(append(errs, err)...)
	}
	userSkills := filepath.Join(claude, "skills")
	if err := offerImport(in, out, userSkills, true, importOptions{global: true}, userSkills); err != nil {
		errs = append(errs, err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return errors.Join(append(errs, err)...)
	}
	root := wd
	if p, err := project.Find(wd); err == nil {
		root = p.Root
	}
	// In the home folder, the project's .claude/skills is ~/.claude/skills.
	if filepath.Join(root, ".claude", "skills") != userSkills {
		if err := offerImport(in, out, root, false, importOptions{}, ""); err != nil {
			errs = append(errs, err)
		}
	}
	fmt.Fprintln(out, "\nSee what the assistant gets in this folder with: quiver status")
	return errors.Join(errs...)
}

// offerImport asks whether to import the skills FindSkills finds in root,
// and imports them like quiver import <path>. It says nothing when there
// are none.
func offerImport(in *bufio.Reader, out io.Writer, root string, deep bool, o importOptions, path string) error {
	dir := root
	if !deep {
		dir = filepath.Join(root, ".claude", "skills")
	}
	if _, err := os.Stat(dir); err != nil {
		return nil
	}
	home, err := fsutil.QuiverHome()
	if err != nil {
		return err
	}
	found, err := library.FindSkills(root, deep, home)
	if err != nil {
		return err
	}
	n := 0
	for _, f := range found {
		if f.Err == nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	fmt.Fprintln(out)
	if !confirmYes(in, out, fmt.Sprintf("Found %s in %s. Import them into the library?", plural(n, "skill"), dir)) {
		fmt.Fprintln(out, "Skipped. Run it later with:", strings.TrimSpace("quiver import "+path))
		return nil
	}
	o.terminal = true
	return importSkills(in, out, o, path)
}
