package cli

import (
	"fmt"
	"io"

	"github.com/kheif/quiver/internal/library"
	"github.com/spf13/cobra"
)

func newPinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pin <source/skill> [<commit>]",
		Short: "Keep a source skill at one commit",
		Long: `Keep a skill from a source at one commit of the source: its current commit,
or the one given (a hash, tag or branch). quiver update then leaves the
skill alone, so its text changes only when you decide. The pin applies
everywhere the skill is assigned, and its ID stays the same.

The commit is checked out into ~/.quiver/sources/.pins. Older commits are
downloaded first when the source does not have them yet. quiver unpin goes
back to the source's current version. To change the skill itself, use
quiver fork instead.`,
		Example: `  quiver pin anthropic/pdf
  quiver pin acme/review 3f2a9c1`,
		ValidArgsFunction: sourceSkillIDs,
		Args:              cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			rev := ""
			if len(args) == 2 {
				rev = args[1]
			}
			return pin(cmd.OutOrStdout(), args[0], rev)
		},
	}
}

func pin(out io.Writer, id, rev string) error {
	p, err := library.PinSkill(id, rev)
	if err != nil {
		return err
	}
	if p.Previous != "" {
		fmt.Fprintf(out, "Pinned %s to %s (was %s).\n", id, short(p.Commit), short(p.Previous))
	} else {
		fmt.Fprintf(out, "Pinned %s to %s.\n", id, short(p.Commit))
	}
	fmt.Fprintf(out, "quiver update leaves it alone; quiver unpin %s goes back to the source's current version.\n", id)
	return nil
}

func newUnpinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unpin <source/skill>",
		Short: "Serve a pinned skill from its source's current version again",
		Long: `Remove the pin of a source skill, so it is served from the source's current
version again and quiver update changes it as usual.`,
		Example:           `  quiver unpin anthropic/pdf`,
		ValidArgsFunction: pinnedIDs,
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			commit, err := library.UnpinSkill(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Unpinned %s (was at %s); it follows its source again.\n", args[0], short(commit))
			return nil
		},
	}
}
