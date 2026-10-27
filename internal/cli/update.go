package cli

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update [<name>...]",
		Short: "Pull the latest skills of the given sources, or of all sources",
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
			// One failed source does not stop the others.
			var failed []string
			for _, name := range names {
				u, err := library.UpdateSource(name)
				if err != nil {
					fmt.Fprintf(out, "%s: failed: %v\n", name, err)
					failed = append(failed, name)
					continue
				}
				printUpdate(out, u, r)
			}
			if len(failed) > 0 {
				return fmt.Errorf("could not update %s", strings.Join(failed, ", "))
			}
			return nil
		},
	}
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

func printUpdate(out io.Writer, u *library.Update, r *registry.Registry) {
	if u.From == u.To {
		fmt.Fprintf(out, "%s: up to date (%s)\n", u.Source, short(u.To))
		return
	}
	fmt.Fprintf(out, "%s: %s..%s\n", u.Source, short(u.From), short(u.To))
	for _, group := range []struct {
		label string
		ids   []string
	}{{"added", u.Added}, {"changed", u.Changed}, {"removed", u.Removed}} {
		for _, id := range group.ids {
			fmt.Fprintf(out, "  %-8s %s\n", group.label, id)
		}
	}
	if len(u.Added)+len(u.Changed)+len(u.Removed) == 0 {
		fmt.Fprintln(out, "  no skill changes")
	}
	// Assignments of removed skills stay, so the user decides what replaces
	// them. doctor keeps reporting them.
	for _, id := range u.Removed {
		var where []string
		if slices.Contains(r.Global, id) {
			where = append(where, "global")
		}
		for _, pid := range r.ProjectsUsing(id) {
			where = append(where, "project "+r.Projects[pid].Name)
		}
		if len(where) > 0 {
			fmt.Fprintf(out, "  warning: %s was removed upstream but is still assigned (%s); unassign it with quiver remove\n", id, strings.Join(where, ", "))
		}
	}
}

func short(hash string) string { return hash[:min(7, len(hash))] }
