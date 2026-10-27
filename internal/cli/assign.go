package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
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
	projectID := ""
	if t.global {
		fmt.Fprintf(out, "Assigned globally: %s\n", strings.Join(ids, ", "))
	} else {
		projectID = p.ID
		if created {
			fmt.Fprintf(out, "Created %s. Commit it, so the project keeps its ID.\n", filepath.Join(p.Root, project.FileName))
		}
		if copied != nil {
			fmt.Fprintln(out, "warning:", copied)
		}
		fmt.Fprintf(out, "Assigned to %s: %s\n", reg.Projects[p.ID].Name, strings.Join(ids, ", "))
	}
	for _, c := range clashes(l, reg.Served(projectID), ids) {
		fmt.Fprintln(out, "warning:", c)
	}
	where := "every folder"
	if projectID != "" {
		where = reg.Projects[projectID].Name
	}
	return printServed(out, reg, projectID, where)
}

func newRemoveCmd() *cobra.Command {
	var t target
	cmd := &cobra.Command{
		Use:   "remove <id>...",
		Short: "Unassign skills from this project, or from every folder with --global",
		Long:  "Unassign skills from this project, or from every folder with --global. The skills stay in the library.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return remove(cmd.OutOrStdout(), t, args)
		},
	}
	t.flags(cmd)
	return cmd
}

// remove does not look IDs up in the library, so a skill that was deleted or
// renamed in its source can still be unassigned.
func remove(out io.Writer, t target, ids []string) error {
	var p *project.Project
	if !t.global {
		dir, err := t.dir()
		if err != nil {
			return err
		}
		p, err = project.Find(dir)
		if errors.Is(err, project.ErrNotRegistered) {
			return fmt.Errorf("%s is not a Quiver project, so it has no skills of its own; use --global for global skills", dir)
		}
		if err != nil {
			return err
		}
	}
	var (
		reg         *registry.Registry
		notAssigned []string
	)
	err := registry.Update(func(r *registry.Registry) error {
		reg = r
		if t.global {
			notAssigned = r.UnassignGlobal(ids...)
			return nil
		}
		if _, ok := r.Projects[p.ID]; !ok {
			notAssigned = ids
			return nil
		}
		var err error
		notAssigned, err = r.Unassign(p.ID, ids...)
		return err
	})
	if err != nil {
		return err
	}
	projectID, where := "", "every folder"
	if !t.global {
		projectID, where = p.ID, filepath.Base(p.Root)
		if rp, ok := reg.Projects[p.ID]; ok {
			where = rp.Name
		}
	}
	skip := map[string]bool{}
	for _, id := range notAssigned {
		skip[id] = true
		switch {
		case t.global:
			fmt.Fprintf(out, "note: %s was not global\n", id)
		case slices.Contains(reg.Global, id):
			fmt.Fprintf(out, "note: %s is global; remove it with --global\n", id)
		default:
			fmt.Fprintf(out, "note: %s was not assigned to %s\n", id, where)
		}
	}
	var removed []string
	for _, id := range ids {
		if !skip[id] {
			removed = append(removed, id)
		}
	}
	if len(removed) > 0 {
		fmt.Fprintf(out, "Removed from %s: %s\n", where, strings.Join(removed, ", "))
	}
	return printServed(out, reg, projectID, where)
}

// clashes describes added skills that share their name with another served
// skill. Assistants pick skills by name, so only one of them would be used.
func clashes(l *library.Library, served, added []string) []string {
	byName := map[string][]string{}
	for _, id := range served {
		if s, err := l.Get(id); err == nil {
			byName[s.Name] = append(byName[s.Name], id)
		}
	}
	var out []string
	seen := map[string]bool{}
	for _, id := range added {
		s, err := l.Get(id)
		if err != nil || seen[s.Name] || len(byName[s.Name]) < 2 {
			continue
		}
		seen[s.Name] = true
		out = append(out, fmt.Sprintf("skills %s share the name %q; keep only one", strings.Join(byName[s.Name], ", "), s.Name))
	}
	return out
}

// printServed lists the skills served in the project, each marked global or
// project. An empty projectID lists only the global skills. where names the
// place in the heading.
func printServed(out io.Writer, r *registry.Registry, projectID, where string) error {
	fmt.Fprintf(out, "Served in %s:\n", where)
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
