package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/kheif/quiver/internal/homesync"
	"github.com/spf13/cobra"
)

// syncConfirm lists the incoming changes that alter what sessions start with
// and asks before they come in. yes takes them without asking.
func syncConfirm(in io.Reader, out io.Writer, yes bool) homesync.Confirm {
	return func(inc homesync.Incoming) bool {
		fmt.Fprintln(out, "The other machine's changes alter what your sessions start with:")
		for _, id := range inc.AlwaysOn {
			fmt.Fprintf(out, "  becomes always-on: %s\n", id)
		}
		for _, id := range inc.Changed {
			fmt.Fprintf(out, "  always-on skill with changed files: %s (see quiver show %s)\n", id, id)
		}
		for _, s := range inc.Sources {
			fmt.Fprintf(out, "  source added or changed: %s\n", s)
		}
		if yes {
			return true
		}
		return confirm(in, out, "Take them?")
	}
}

func newSyncCmd() *cobra.Command {
	var yes bool
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

Whatever another machine pushed becomes live here, so when it makes a skill
always-on, changes the files of an always-on skill or adds a source, sync
lists those changes and asks before it pulls. --yes takes them without
asking; without a terminal and without --yes, sync pulls nothing.

Only local/, config.json and registry.json are synced. sources/ is cloned
again from config.json, and the folders where each project was seen stay
on each machine.`,
		Example: `  quiver sync init git@github.com:me/quiver-home.git
  quiver sync`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := homesync.Sync(syncConfirm(cmd.InOrStdin(), cmd.OutOrStdout(), yes))
			if err != nil {
				return err
			}
			printSync(cmd.OutOrStdout(), res)
			return nil
		},
	}
	initCmd := &cobra.Command{
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
			res, err := homesync.Init(args[0], syncConfirm(cmd.InOrStdin(), cmd.OutOrStdout(), yes))
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "The Quiver home now syncs with %s (branch %s).\n", res.Remote, res.Branch)
			printSync(out, res)
			fmt.Fprintln(out, "Run quiver sync to send and get changes; on another machine, start with quiver sync init.")
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "take changes to always-on skills and sources without asking")
	initCmd.Flags().BoolVarP(&yes, "yes", "y", false, "take changes to always-on skills and sources without asking")
	cmd.AddCommand(initCmd)
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
