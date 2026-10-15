package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/source"
	"github.com/spf13/cobra"
)

func newSourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Manage the git repositories that skills come from",
	}
	cmd.AddCommand(newSourceAddCmd(), newSourceListCmd(), newSourceRemoveCmd())
	return cmd
}

func newSourceAddCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "add <repo>",
		Short: "Clone a repository and add its skills to the library",
		Long: `Clone a repository and add its skills to the library.

<repo> is owner/repo (GitHub), host/owner/repo, a git URL or a local path.
The source name defaults to the repository owner.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := source.Parse(args[0])
			if err != nil {
				return err
			}
			if name == "" {
				name = p.Name
			}
			return addSource(cmd, name, p.URL)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "source name (default: the repository owner)")
	return cmd
}

func addSource(cmd *cobra.Command, name, url string) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Cloning %s as source %q...\n", url, name)
	s, err := library.StageSource(name, url)
	if err != nil {
		return err
	}
	for _, w := range s.Discovery.Warnings {
		fmt.Fprintln(out, "warning:", w)
	}
	if len(s.Discovery.Skills) == 0 {
		if !confirm(cmd.InOrStdin(), out, "No skills found. Keep the source anyway?") {
			fmt.Fprintln(out, "Source not added.")
			return s.Discard()
		}
	}
	if err := s.Keep(); err != nil {
		_ = s.Discard()
		return err
	}
	fmt.Fprintf(out, "Added source %q with %d skills:\n", name, len(s.Discovery.Skills))
	for _, f := range s.Discovery.Skills {
		fmt.Fprintf(out, "  %s/%s\n", name, f.ID)
	}
	fmt.Fprintln(out, "Note: skills give instructions to your assistant and may contain scripts. Review them before you assign them.")
	return nil
}

func newSourceListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List sources with their skill count and current commit",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			l, err := library.Load()
			if err != nil {
				return err
			}
			sources, err := l.Sources()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(sources) == 0 {
				fmt.Fprintln(out, "No sources. Add one with: quiver source add <repo>")
				return nil
			}
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSKILLS\tCOMMIT\tUPDATED\tREPO")
			for _, s := range sources {
				commit, updated := s.Commit, s.Updated.Format("2006-01-02")
				if commit == "" {
					commit, updated = "missing", "-"
				}
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n", s.Name, s.Skills, commit, updated, s.Repo)
			}
			return w.Flush()
		},
	}
}

func newSourceRemoveCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Delete a source and its skills",
		Long:  "Delete a source and its skills. It refuses while skills from the source are assigned, unless --force is given.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u, err := library.RemoveSource(args[0], force)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(u.Global) > 0 {
				fmt.Fprintf(out, "Unassigned globally: %s\n", strings.Join(u.Global, ", "))
			}
			for _, p := range u.Projects {
				fmt.Fprintf(out, "Unassigned from %s: %s\n", p.Name, strings.Join(p.Skills, ", "))
			}
			fmt.Fprintf(out, "Removed source %q.\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "unassign the source's skills first")
	return cmd
}

// confirm asks a yes/no question. Anything but y or yes, and no input at
// all, means no.
func confirm(in io.Reader, out io.Writer, question string) bool {
	fmt.Fprintf(out, "%s [y/N] ", question)
	line, _ := bufio.NewReader(in).ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
