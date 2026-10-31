package cli

import (
	"fmt"
	"io"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/spf13/cobra"
)

func newForkCmd() *cobra.Command {
	var as string
	cmd := &cobra.Command{
		Use:   "fork <source/skill>",
		Short: "Copy a skill from a source into your own skills, to change it",
		Long: `Copy a skill from a source into your own skills, so you can change it.
Skills from sources are read-only, and quiver update replaces them with
the upstream version.

The copy is named after the skill, or --as. Its frontmatter records where
it came from (forkedFrom: <id>@<commit>). Wherever the original is
assigned, globally or in a project, the copy takes its place, always-on
marks included. From then on quiver edit changes what those projects get,
and quiver update no longer does.`,
		Example: `  quiver fork anthropic/pdf
  quiver fork acme/review --as team-review`,
		ValidArgsFunction: sourceSkillIDs,
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fork(cmd.OutOrStdout(), args[0], as)
		},
	}
	cmd.Flags().StringVar(&as, "as", "", "name of the local copy (default: the skill's name)")
	return cmd
}

func fork(out io.Writer, id, as string) error {
	f, err := library.ForkRemote(id, as)
	if err != nil {
		return err
	}
	for _, w := range f.Warnings {
		fmt.Fprintf(out, "warning: %s\n", w)
	}
	fmt.Fprintf(out, "Forked %s into %s (%s).\n", id, f.Name, f.Dir)
	if len(f.Replaced.Global) > 0 {
		fmt.Fprintf(out, "Replaced %s with %s globally.\n", id, f.Name)
	}
	for _, p := range f.Replaced.Projects {
		fmt.Fprintf(out, "Replaced %s with %s in project %s.\n", id, f.Name, registry.ProjectLabel(p.Name, p.Paths))
	}
	if f.Replaced.Empty() {
		fmt.Fprintf(out, "%s was not assigned anywhere; assign the copy with quiver add %s.\n", id, f.Name)
	}
	fmt.Fprintf(out, "Change it with quiver edit %s.\n", f.Name)
	return nil
}
