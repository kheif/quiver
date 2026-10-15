package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/source"
	"github.com/spf13/cobra"
)

func newSourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Manage the git repositories that skills come from",
	}
	cmd.AddCommand(newSourceAddCmd())
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

// confirm asks a yes/no question. Anything but y or yes, and no input at
// all, means no.
func confirm(in io.Reader, out io.Writer, question string) bool {
	fmt.Fprintf(out, "%s [y/N] ", question)
	line, _ := bufio.NewReader(in).ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
