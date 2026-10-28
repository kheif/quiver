package cli

import (
	"fmt"
	"io"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var yes, force bool
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a local skill from the library",
		Long: `Delete one of your own skills from the library and commit the removal in
the Quiver home, so git history still has it. It asks first.

An assigned skill is not deleted unless --force is given; then it is also
unassigned everywhere. To take a skill away from one project only, use
quiver remove. Skills from sources cannot be deleted one by one: quiver
source remove deletes the whole source.`,
		Example: `  quiver delete old-draft
  quiver delete release-notes --force --yes`,
		ValidArgsFunction: skillIDs(true, 1),
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteSkill(cmd.InOrStdin(), cmd.OutOrStdout(), args[0], yes, force)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "do not ask for confirmation")
	cmd.Flags().BoolVar(&force, "force", false, "also unassign the skill where it is assigned")
	return cmd
}

func deleteSkill(in io.Reader, out io.Writer, id string, yes, force bool) error {
	if !yes && !confirm(in, out, fmt.Sprintf("Delete skill %s from the library?", id)) {
		fmt.Fprintln(out, "Skill not deleted.")
		return nil
	}
	u, err := library.DeleteLocal(id, force)
	if err != nil {
		return err
	}
	if len(u.Global) > 0 {
		fmt.Fprintln(out, "Unassigned globally.")
	}
	for _, p := range u.Projects {
		fmt.Fprintf(out, "Unassigned from project %s.\n", registry.ProjectLabel(p.Name, p.Paths))
	}
	fmt.Fprintf(out, "Deleted skill %s.\n", id)
	return nil
}
