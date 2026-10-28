package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/spf13/cobra"
)

func newWhereCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "where <id>",
		Short: "Show where a skill is assigned",
		Long: `Show where a skill is assigned: "global" if it is served in every folder,
then each project that has it, with its folders.`,
		Example: `  quiver where pdf
  quiver where anthropic/docx`,
		ValidArgsFunction: skillIDs(false, 1),
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return where(cmd.OutOrStdout(), args[0])
		},
	}
}

func where(out io.Writer, id string) error {
	r, err := registry.Load()
	if err != nil {
		return err
	}
	global := slices.Contains(r.Global, id)
	using := r.ProjectsUsing(id)
	l, err := library.Load()
	if err != nil {
		return err
	}
	_, missing := l.Get(id)
	if missing != nil && !global && len(using) == 0 {
		return fmt.Errorf("skill %q not found; see quiver list", id)
	}
	if !global && len(using) == 0 {
		fmt.Fprintf(out, "%s is not assigned anywhere.\n", id)
		return nil
	}
	if global {
		fmt.Fprintln(out, "global")
	}
	sort.Slice(using, func(i, j int) bool { return r.Projects[using[i]].Name < r.Projects[using[j]].Name })
	for _, pid := range using {
		p := r.Projects[pid]
		fmt.Fprintf(out, "project %s\n", registry.ProjectLabel(p.Name, p.Paths))
	}
	if missing != nil {
		fmt.Fprintf(out, "warning: %s is assigned but not in the library, so it is not served\n", id)
	}
	return nil
}

func newProjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List registered projects",
		Long:  "List registered projects: name, ID, number of skills and folders. Folders that no longer exist are marked (missing).",
		Example: `  quiver projects
  quiver projects rename app-k3f9q2 web
  quiver projects forget old-site`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return listProjects(cmd.OutOrStdout())
		},
	}
	cmd.AddCommand(newProjectsRenameCmd(), newProjectsForgetCmd(), newProjectsSplitCmd())
	return cmd
}

func listProjects(out io.Writer) error {
	r, err := registry.Load()
	if err != nil {
		return err
	}
	if len(r.Projects) == 0 {
		fmt.Fprintln(out, "No projects. Run quiver add inside a folder to register it.")
		return nil
	}
	ids := make([]string, 0, len(r.Projects))
	for id := range r.Projects {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := r.Projects[ids[i]], r.Projects[ids[j]]
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return ids[i] < ids[j]
	})
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tID\tSKILLS\tPATHS")
	for _, id := range ids {
		p := r.Projects[id]
		paths := make([]string, len(p.Paths))
		for i, path := range p.Paths {
			paths[i] = path
			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
				paths[i] += " (missing)"
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", p.Name, id, len(p.Skills), strings.Join(paths, ", "))
	}
	return w.Flush()
}

func newProjectsRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <id|name> <new-name>",
		Short: "Change a project's name",
		Long: `Change the name Quiver shows for a project. The name is only a label; the
project keeps its ID and skills.`,
		Example: `  quiver projects rename app-k3f9q2 web
  quiver projects rename web web-frontend`,
		ValidArgsFunction: projectRefs,
		Args:              cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var id string
			err := registry.Update(func(r *registry.Registry) error {
				var err error
				if id, err = r.Lookup(args[0]); err != nil {
					return err
				}
				return r.RenameProject(id, args[1])
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Renamed project %s to %s.\n", id, args[1])
			return nil
		},
	}
}

func newProjectsForgetCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "forget <id|name>",
		Short: "Remove a project and its assignments",
		Long: `Remove a project and its skill assignments from the registry. The
project's .quiver.json is left alone, so using the folder again registers
it again with the same ID and no skills.`,
		Example: `  quiver projects forget old-site
  quiver projects forget app-k3f9q2 --yes`,
		ValidArgsFunction: projectRefs,
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return forget(cmd.InOrStdin(), cmd.OutOrStdout(), args[0], yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "do not ask for confirmation")
	return cmd
}

func forget(in io.Reader, out io.Writer, ref string, yes bool) error {
	r, err := registry.Load()
	if err != nil {
		return err
	}
	id, err := r.Lookup(ref)
	if err != nil {
		return err
	}
	p := r.Projects[id]
	question := fmt.Sprintf("Forget project %s [%s] and its %s?", registry.ProjectLabel(p.Name, p.Paths), id, plural(len(p.Skills), "skill assignment"))
	if !yes && !confirm(in, out, question) {
		fmt.Fprintln(out, "Project not forgotten.")
		return nil
	}
	err = registry.Update(func(r *registry.Registry) error { return r.ForgetProject(id) })
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Forgot project %s. Its .quiver.json was not changed.\n", id)
	return nil
}

func newProjectsSplitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "split",
		Short: "Give a copied project folder its own ID",
		Long: `Run inside a project folder that was copied with its .quiver.json. The
folder gets a new ID and starts with the skills of the original; after that,
changing one copy's skills does not change the other's.`,
		Example: `  cp -R ~/code/app ~/code/app-v2
  cd ~/code/app-v2
  quiver projects split`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			oldID, p, err := registry.SplitProject(dir)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s is now project %s, with the skills of %s.\nIts .quiver.json was rewritten.\n", p.Root, p.ID, oldID)
			return nil
		},
	}
}
