package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/kheif/quiver/internal/homesync"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync your skills and assignments with your other machines",
		Long: `Sync the Quiver home with a git remote, so every machine has the same local
skills, sources and assignments. Set it up once per machine with
quiver sync init <git-remote>; use a private repository.

quiver sync commits the changes on this machine, pulls the others with
rebase and pushes. When registry.json or config.json changed on both
machines, the changes are merged: a skill assigned on either machine stays
assigned. Any other conflict stops the sync with nothing pulled or pushed,
and says how to resolve it. Sources added on another machine are cloned
afterwards.

Only local/, config.json and registry.json are synced. sources/ is cloned
again from config.json, and the folders where each project was seen stay
on each machine.`,
		Example: `  quiver sync init git@github.com:me/quiver-home.git
  quiver sync`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := homesync.Sync()
			if err != nil {
				return err
			}
			printSync(cmd.OutOrStdout(), res)
			return nil
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "init <git-remote>",
		Short: "Start syncing the Quiver home with a git remote",
		Long: `Turn the Quiver home into a git repository with <git-remote> as its remote,
and push it. The history of local/ is kept; local/ stops being a repository
of its own.

Run it on every machine with the same remote. When the remote already holds
another machine's home, this machine's skills, sources and assignments are
merged with it. If the same local skill differs on both, it stops and
changes nothing.`,
		Example:           `  quiver sync init git@github.com:me/quiver-home.git`,
		ValidArgsFunction: noArgs,
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := homesync.Init(args[0])
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "The Quiver home now syncs with %s (branch %s).\n", res.Remote, res.Branch)
			printSync(out, res)
			fmt.Fprintln(out, "Run quiver sync to send and get changes; on another machine, start with quiver sync init.")
			return nil
		},
	})
	return cmd
}

func printSync(out io.Writer, res *homesync.Result) {
	if res.Pulled {
		fmt.Fprintf(out, "Got the changes from %s.\n", res.Remote)
	}
	if len(res.Merged) > 0 {
		fmt.Fprintf(out, "Merged the changes of both machines in %s.\n", strings.Join(res.Merged, " and "))
	}
	if res.Pushed {
		fmt.Fprintf(out, "Sent this machine's changes to %s.\n", res.Remote)
	}
	if !res.Pulled && !res.Pushed {
		fmt.Fprintln(out, "Already in sync.")
	}
	for _, name := range res.Cloned {
		fmt.Fprintf(out, "Downloaded source %s.\n", name)
	}
	for _, f := range res.Failed {
		fmt.Fprintf(out, "Could not restore %s\n", f)
	}
}
