package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var reset, diff, yes bool
	cmd := &cobra.Command{
		Use:   "update [<name>...]",
		Short: "Pull the latest skills of the given sources, or of all sources",
		Long: `Pull the latest skills of the given sources, or of every source. For each
source it first downloads the new commits without applying them, and lists
the skills that would be added, changed or removed. For each assigned skill
that changes, it lists the changed files, and marks scripts (such as .sh,
.py and .js files, and executable files) as "needs review": the assistant
may run them. --diff shows the whole change of those skills.

When an assigned skill would change or go away, it asks before it applies
the changes; --yes applies them without asking. Changes to skills that are
not assigned are applied right away. Pinned skills are left alone.

A source whose history was rewritten (force-pushed) is not updated; run
again with --reset to take the remote history.`,
		Example: `  quiver update
  quiver update anthropic --diff
  quiver update --yes
  quiver update acme --reset`,
		ValidArgsFunction: completeSources(false, 0),
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := updateNames(args)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(names) == 0 {
				fmt.Fprintln(out, "No sources. Add one with: quiver source add <repo>")
				return nil
			}
			r, err := registry.Load()
			if err != nil {
				return err
			}
			// One reader for every question, so no answer is lost.
			in := bufio.NewReader(cmd.InOrStdin())
			// One failed source does not stop the others.
			var failed []string
			for _, name := range names {
				if err := updateSource(in, out, r, name, reset, diff, yes); err != nil {
					fmt.Fprintf(out, "%s: failed: %v\n", name, err)
					if errors.Is(err, library.ErrDiverged) {
						fmt.Fprintf(out, "  run quiver update %s --reset to take the remote history\n", name)
					}
					failed = append(failed, name)
				}
			}
			if len(failed) > 0 {
				return fmt.Errorf("could not update %s", strings.Join(failed, ", "))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&reset, "reset", false, "replace a rewritten history (after a force-push) with the remote one")
	cmd.Flags().BoolVar(&diff, "diff", false, "show the whole change of each assigned skill that changes")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "apply the changes without asking")
	return cmd
}

// updateSource previews the update of one source, asks when assigned
// skills change, and applies it.
func updateSource(in *bufio.Reader, out io.Writer, r *registry.Registry, name string, reset, diff, yes bool) error {
	u, err := library.FetchSource(name, reset)
	if err != nil {
		return err
	}
	ask := printUpdate(out, u, r)
	if diff {
		for _, id := range slices.Concat(u.Added, u.Changed) {
			if assignedWhere(r, id) == nil {
				continue
			}
			patch, err := u.Diff(id)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "\n%s\n", patch)
		}
	}
	if ask && !yes && !confirmLine(in, out, fmt.Sprintf("Apply the changes to %s?", name)) {
		fmt.Fprintf(out, "%s: not updated; run quiver update %s --yes to apply the changes without asking\n", name, name)
		return nil
	}
	if err := u.Apply(); err != nil {
		return err
	}
	if ask {
		fmt.Fprintf(out, "%s: updated to %s\n", name, short(u.To))
	}
	for _, e := range u.PinErrors {
		fmt.Fprintf(out, "  warning: pin of %s\n", e)
	}
	return nil
}

// updateNames returns the sources to update: the given names, which must
// exist, or every source, sorted.
func updateNames(args []string) ([]string, error) {
	c, err := config.Load()
	if err != nil {
		return nil, err
	}
	for _, name := range args {
		if _, ok := c.Sources[name]; !ok {
			return nil, fmt.Errorf("source %q not found; see quiver source list", name)
		}
	}
	if len(args) > 0 {
		return args, nil
	}
	names := make([]string, 0, len(c.Sources))
	for name := range c.Sources {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// confirmLine is confirm for a shared reader. When the input has ended it
// ends the question's line, so the next output starts on its own.
func confirmLine(in *bufio.Reader, out io.Writer, question string) bool {
	fmt.Fprintf(out, "%s [y/N] ", question)
	line, err := in.ReadString('\n')
	if err != nil {
		fmt.Fprintln(out)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

// printUpdate lists what the update changes, with the changed files of
// assigned skills. It reports whether an assigned skill changes or goes
// away, so the update needs a yes.
func printUpdate(out io.Writer, u *library.Update, r *registry.Registry) (ask bool) {
	if u.From == u.To {
		fmt.Fprintf(out, "%s: up to date (%s)\n", u.Source, short(u.To))
		return false
	}
	reset := ""
	if u.Reset {
		reset = " (reset to the rewritten remote history)"
	}
	fmt.Fprintf(out, "%s: %s..%s%s\n", u.Source, short(u.From), short(u.To), reset)
	for _, group := range []struct {
		label string
		ids   []string
	}{{"added", u.Added}, {"changed", u.Changed}, {"removed", u.Removed}} {
		for _, id := range group.ids {
			where := assignedWhere(r, id)
			if where == nil {
				fmt.Fprintf(out, "  %-8s %s\n", group.label, id)
				continue
			}
			ask = true
			fmt.Fprintf(out, "  %-8s %s (assigned: %s)\n", group.label, id, strings.Join(where, ", "))
			printFiles(out, u.Files[id])
		}
	}
	for _, id := range u.Pinned {
		fmt.Fprintf(out, "  %-8s %s (changed upstream; kept at its pin until quiver unpin %s)\n", "pinned", id, id)
	}
	if len(u.Added)+len(u.Changed)+len(u.Removed)+len(u.Pinned) == 0 {
		fmt.Fprintln(out, "  no skill changes")
	}
	// Assignments of removed skills stay, so the user decides what replaces
	// them. doctor keeps reporting them.
	for _, id := range u.Removed {
		if where := assignedWhere(r, id); where != nil {
			fmt.Fprintf(out, "  warning: %s will be removed upstream but stays assigned (%s); unassign it with quiver remove\n", id, strings.Join(where, ", "))
		}
	}
	return ask
}

// printFiles lists the changed files of a skill: status, path and lines
// added and deleted, with scripts marked for review.
func printFiles(out io.Writer, files []git.FileChange) {
	width := 0
	for _, f := range files {
		width = max(width, len(f.Path))
	}
	for _, f := range files {
		lines := "binary"
		if !f.Binary {
			lines = fmt.Sprintf("+%d -%d", f.Added, f.Deleted)
		}
		review := ""
		if library.NeedsReview(f) {
			review = "  needs review"
		}
		fmt.Fprintf(out, "           %s %-*s  %s%s\n", f.Status, width, f.Path, lines, review)
	}
}

// assignedWhere lists where the skill is assigned: "global", "always-on"
// and each project by name. It is nil when the skill is not assigned.
func assignedWhere(r *registry.Registry, id string) []string {
	var where []string
	if slices.Contains(r.Global, id) {
		where = append(where, "global")
	}
	for _, pid := range r.ProjectsUsing(id) {
		where = append(where, "project "+r.Projects[pid].Name)
	}
	if slices.Contains(r.AlwaysOn, id) || slices.ContainsFunc(r.ProjectsUsing(id), func(pid string) bool { return slices.Contains(r.Projects[pid].AlwaysOn, id) }) {
		where = append(where, "always-on")
	}
	return where
}

func short(hash string) string { return hash[:min(7, len(hash))] }
