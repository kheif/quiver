package cli

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/server"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the skills the MCP server serves in this folder",
		Long: `Show the project in this folder and the skills the MCP server serves here.

Like the server, it uses $` + server.ProjectEnv + ` instead of the current
folder when it is set.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return status(cmd.OutOrStdout())
		},
	}
}

func status(out io.Writer) error {
	w, err := server.Resolve()
	if err != nil {
		return err
	}
	r, err := registry.Load()
	if err != nil {
		return err
	}
	l, err := library.Load()
	if err != nil {
		return err
	}
	where := "every folder"
	if p := w.Project; p == nil {
		fmt.Fprintf(out, "Folder:  %s\n", w.Dir)
		fmt.Fprintln(out, "not registered; only global skills are served")
	} else {
		where = filepath.Base(p.Root)
		if rp, ok := r.Projects[p.ID]; ok {
			where = rp.Name
		}
		fmt.Fprintf(out, "Project: %s\nID:      %s\nRoot:    %s\n", where, p.ID, p.Root)
		// EnsureProject changes only this copy; status writes nothing.
		if copied := r.EnsureProject(p.ID, p.Root); copied != nil {
			fmt.Fprintln(out, "warning:", copied)
		}
	}
	fmt.Fprintln(out)
	if err := printServed(out, r, w.ProjectID(), where); err != nil {
		return err
	}
	for _, id := range r.Served(w.ProjectID()) {
		if _, err := l.Get(id); err != nil {
			fmt.Fprintf(out, "warning: %s is assigned but not in the library, so it is not served\n", id)
		}
	}
	return nil
}
