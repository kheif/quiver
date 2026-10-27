package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/registry"
	"github.com/spf13/cobra"
)

// target is where add and remove change assignments: the global list or one
// project.
type target struct {
	global  bool
	project string // --project path; empty means the current folder
}

func (t *target) flags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&t.global, "global", false, "change the skills served in every folder")
	cmd.Flags().StringVar(&t.project, "project", "", "project folder (default: the current folder)")
	cmd.MarkFlagsMutuallyExclusive("global", "project")
}

// dir returns the folder to look for the project in. A --project path must
// exist, or Find would walk up to a parent project.
func (t target) dir() (string, error) {
	if t.project == "" {
		return os.Getwd()
	}
	info, err := os.Stat(t.project)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a folder", t.project)
	}
	return t.project, nil
}

func newAddCmd() *cobra.Command {
	var t target
	cmd := &cobra.Command{
		Use:   "add <id>...",
		Short: "Assign skills to this project, or to every folder with --global",
		Long: `Assign skills to this project, or to every folder with --global.

A folder without .quiver.json is registered first. Commit the new file, so
the project keeps its ID.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return add(cmd.OutOrStdout(), t, args)
		},
	}
	t.flags(cmd)
	return cmd
}

func add(out io.Writer, t target, ids []string) error {
	l, err := library.Load()
	if err != nil {
		return err
	}
	// Check every ID first, so one typo changes nothing.
	var errs []error
	for _, id := range ids {
		if _, err := l.Get(id); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	var dir string
	if !t.global {
		if dir, err = t.dir(); err != nil {
			return err
		}
	}
	var (
		reg     *registry.Registry
		p       *project.Project
		created bool
		copied  *registry.PossibleCopy
	)
	// Register runs under the registry lock too, so two processes cannot
	// write different IDs into the same new folder.
	err = registry.Update(func(r *registry.Registry) error {
		reg = r
		if t.global {
			r.AssignGlobal(ids...)
			return nil
		}
		var err error
		p, err = project.Find(dir)
		if errors.Is(err, project.ErrNotRegistered) {
			created = true
			p, err = project.Register(dir)
		}
		if err != nil {
			return err
		}
		copied = r.EnsureProject(p.ID, p.Root)
		return r.Assign(p.ID, ids...)
	})
	if err != nil {
		return err
	}
	if t.global {
		fmt.Fprintf(out, "Assigned globally: %s\n", strings.Join(ids, ", "))
		return printServed(out, reg, "")
	}
	if created {
		fmt.Fprintf(out, "Created %s. Commit it, so the project keeps its ID.\n", filepath.Join(p.Root, project.FileName))
	}
	if copied != nil {
		fmt.Fprintln(out, "warning:", copied)
	}
	fmt.Fprintf(out, "Assigned to %s: %s\n", reg.Projects[p.ID].Name, strings.Join(ids, ", "))
	return printServed(out, reg, p.ID)
}

// printServed lists the skills served in the project, each marked global or
// project. An empty projectID lists only the global skills.
func printServed(out io.Writer, r *registry.Registry, projectID string) error {
	if p, ok := r.Projects[projectID]; ok {
		fmt.Fprintf(out, "Served in %s:\n", p.Name)
	} else {
		fmt.Fprintln(out, "Served in every folder:")
	}
	served := r.Served(projectID)
	if len(served) == 0 {
		fmt.Fprintln(out, "  (none)")
		return nil
	}
	global := map[string]bool{}
	for _, id := range r.Global {
		global[id] = true
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, id := range served {
		scope := "project"
		if global[id] {
			scope = "global"
		}
		fmt.Fprintf(w, "  %s\t%s\n", id, scope)
	}
	return w.Flush()
}
