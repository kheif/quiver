package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/source"
	"github.com/spf13/cobra"
)

func newSourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Manage the git repositories that skills come from",
		Long: `A source is a git repository with skills, such as a team's shared skills.
Its skills get IDs like <source>/<skill>. They are read-only: quiver update
pulls new versions.

Add only sources you trust. Their skills give instructions to your
assistant and can include scripts.`,
		Example: `  quiver source add anthropics/skills --name anthropic
  quiver source list
  quiver source remove anthropic`,
	}
	cmd.AddCommand(newSourceAddCmd(), newSourceListCmd(), newSourceRemoveCmd())
	return cmd
}

func newSourceAddCmd() *cobra.Command {
	var (
		name       string
		searchable bool
	)
	cmd := &cobra.Command{
		Use:   "add <repo>",
		Short: "Clone a repository and add its skills to the library",
		Long: `Clone a repository and add its skills to the library.

<repo> is owner/repo (GitHub), host/owner/repo, a git URL or a local path.
The source name defaults to the repository owner.

In a terminal it then asks whether to make the skills Searchable in every
folder: not listed or loaded at session start, found by the assistant when it
searches. The default is no. --searchable answers yes; without a terminal and
without the flag nothing is assigned.`,
		Example: `  # GitHub owner/repo
  quiver source add anthropics/skills --name anthropic

  # Any git URL, or a local folder
  quiver source add git@github.com:acme/skills.git
  quiver source add ~/code/team-skills

  # Make its skills Searchable right away
  quiver source add anthropics/skills --name anthropic --searchable`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := source.Parse(args[0])
			if err != nil {
				return err
			}
			if err := checkTransport(cmd.ErrOrStderr(), p.URL); err != nil {
				return err
			}
			if name == "" {
				name = p.Name
			}
			return addSource(cmd, name, p.URL, searchable)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "source name (default: the repository owner)")
	cmd.Flags().BoolVar(&searchable, "searchable", false, "make the skills Searchable in every folder, without asking")
	return cmd
}

// checkTransport refuses an http:// or git:// source unless config.json sets
// allowInsecureSources, and warns on w when it lets one through.
func checkTransport(w io.Writer, url string) error {
	if !source.Insecure(url) {
		return nil
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if !cfg.AllowInsecureSources {
		return fmt.Errorf("%s is not encrypted: anyone on the network can change the skills you get. Use https:// or ssh, or set \"allowInsecureSources\": true in config.json", url)
	}
	fmt.Fprintf(w, "warning: %s is not encrypted; anyone on the network can change the skills you get\n", url)
	return nil
}

func addSource(cmd *cobra.Command, name, url string, searchable bool) error {
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
	fmt.Fprintln(out, "Note: skills give instructions to your assistant and may contain scripts. Read them with quiver show <id> before you assign them.")
	ids := make([]string, len(s.Discovery.Skills))
	for i, f := range s.Discovery.Skills {
		ids[i] = name + "/" + f.ID
	}
	made, err := offerSearchable(cmd.InOrStdin(), out, name, ids, searchable)
	if err != nil || !made {
		return err
	}
	return offerEngineSetup(cmd.Context(), cmd.InOrStdin(), out, cmd.ErrOrStderr())
}

// offerSearchable makes the skills of a new source Searchable in every
// folder when the user says yes: with the flag, or at the question in a
// terminal. Without either nothing is assigned. A source that is new has no
// other skill assigned, so the printed command undoes exactly this.
func offerSearchable(in io.Reader, out io.Writer, name string, ids []string, flag bool) (made bool, err error) {
	if len(ids) == 0 {
		return false, nil
	}
	yes := flag
	if !yes && interactive(in) {
		yes = confirm(in, out, fmt.Sprintf("Make the %d skills searchable in every folder? The assistant will then load them without asking when a search finds them.", len(ids)))
	}
	if !yes {
		return false, nil
	}
	err = registry.Update(func(r *registry.Registry) error {
		r.AssignSearchableGlobal(ids...)
		return nil
	})
	if err != nil {
		return false, err
	}
	fmt.Fprintf(out, "Made %d skills Searchable in every folder.\n", len(ids))
	fmt.Fprintf(out, "To undo: quiver remove --source %s --global\n", name)
	return true, nil
}

func newSourceListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List sources with their skill count and current commit",
		Long: `List the sources: name, number of skills, the commit in use and when it was
made, and the repository.`,
		Example:           `  quiver source list`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
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
		Example: `  quiver source remove acme
  quiver source remove acme --force`,
		ValidArgsFunction: completeSources(false, 1),
		Args:              cobra.ExactArgs(1),
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

// confirmYes is confirm with yes as the default. No input at all, as at
// the end of a script's input, is no.
func confirmYes(in io.Reader, out io.Writer, question string) bool {
	fmt.Fprintf(out, "%s [Y/n] ", question)
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "" || answer == "y" || answer == "yes"
}
