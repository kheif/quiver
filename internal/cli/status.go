package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"slices"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/server"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the skills the MCP server serves in this folder",
		Long: `Show the project in this folder and the skills the MCP server serves here.

Like the server, it uses $` + server.ProjectEnv + ` instead of the current
folder when it is set.`,
		Example: `  quiver status
  QUIVER_PROJECT=~/code/api quiver status
  quiver status --json`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return status(cmd.OutOrStdout(), asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}

// statusJSON is the JSON output of status.
type statusJSON struct {
	Folder   string         `json:"folder"`
	Project  *statusProject `json:"project"` // null when the folder is not a project
	Skills   []servedSkill  `json:"skills"`
	Warnings []string       `json:"warnings"`
}

type statusProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Root string `json:"root"`
}

// servedSkill is one skill served in a folder.
type servedSkill struct {
	ID        string `json:"id"`
	Scope     string `json:"scope"` // global or project
	AlwaysOn  bool   `json:"always_on"`
	InLibrary bool   `json:"in_library"` // false: assigned but not served
}

func status(out io.Writer, asJSON bool) error {
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
	copies, err := servedCopies(r, l)
	if err != nil {
		return err
	}
	where := "every folder"
	var copied *registry.PossibleCopy
	if p := w.Project; p != nil {
		where = filepath.Base(p.Root)
		if rp, ok := r.Projects[p.ID]; ok {
			where = rp.Name
		}
		// EnsureProject changes only this copy; status writes nothing.
		copied = r.EnsureProject(p.ID, p.Root)
	}
	if asJSON {
		st := statusJSON{Folder: w.Dir, Skills: []servedSkill{}, Warnings: []string{}}
		if p := w.Project; p != nil {
			st.Project = &statusProject{ID: p.ID, Name: where, Root: p.Root}
		}
		if copied != nil {
			st.Warnings = append(st.Warnings, copied.Error())
		}
		global := map[string]bool{}
		for _, id := range r.Global {
			global[id] = true
		}
		always := r.AlwaysOnServed(w.ProjectID())
		for _, id := range r.Served(w.ProjectID()) {
			s := servedSkill{ID: id, Scope: "project", AlwaysOn: slices.Contains(always, id)}
			if global[id] {
				s.Scope = "global"
			}
			_, err := l.Get(id)
			s.InLibrary = err == nil
			st.Skills = append(st.Skills, s)
		}
		for _, c := range copies {
			st.Warnings = append(st.Warnings, fmt.Sprintf("%s is also installed in Claude Code, so it may be loaded twice: %s", c.ID, describeCopy(c)))
		}
		return printJSON(out, st)
	}
	if p := w.Project; p == nil {
		fmt.Fprintf(out, "Folder:  %s\n", w.Dir)
		fmt.Fprintln(out, "not registered; only global skills are served")
	} else {
		fmt.Fprintf(out, "Project: %s\nID:      %s\nRoot:    %s\n", where, p.ID, p.Root)
		if copied != nil {
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
	for _, c := range copies {
		fmt.Fprintf(out, "warning: %s is also installed in Claude Code, so it may be loaded twice: %s\n", c.ID, describeCopy(c))
	}
	if len(copies) > 0 {
		fmt.Fprintln(out, "Run quiver doctor --fix to delete the copies.")
	}
	return nil
}
