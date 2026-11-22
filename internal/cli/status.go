package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

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
folder when it is set.

One line says what the assignments cost at the start of a session: how
many skills are listed, the size of the listing in characters and in
estimated tokens (characters divided by four), the always-on skills counted
apart, and how many skills are Searchable, which add nothing. With -v it
lists the ten listed skills with the longest descriptions and the command
that makes each one Searchable. The numbers inform; nothing is refused or
trimmed.`,
		Example: `  quiver status
  quiver status -v
  QUIVER_PROJECT=~/code/api quiver status
  quiver status --json`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return status(cmd.OutOrStdout(), asJSON, verbose)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}

// statusJSON is the JSON output of status.
type statusJSON struct {
	Folder     string            `json:"folder"`
	Project    *statusProject    `json:"project"` // null when the folder is not a project
	Skills     []servedSkill     `json:"skills"`
	Searchable []searchableSkill `json:"searchable"` // not in skills: found by searching
	Catalog    statusCatalog     `json:"catalog"`
	Search     statusSearch      `json:"search"`
	Assistants []statusAssistant `json:"assistants"`
	Warnings   []string          `json:"warnings"`
}

// statusCatalog is what the folder's assignments cost at the start of a
// session. The token figures estimate the characters divided by four.
type statusCatalog struct {
	Listed         int `json:"listed"`
	ListedChars    int `json:"listed_chars"`
	ListedTokens   int `json:"listed_tokens_estimate"`
	AlwaysOn       int `json:"always_on"`
	AlwaysOnChars  int `json:"always_on_chars"`
	AlwaysOnTokens int `json:"always_on_tokens_estimate"`
	Searchable     int `json:"searchable"` // add nothing to the listing
}

func catalogOf(c server.CatalogCost) statusCatalog {
	return statusCatalog{c.Listed, c.ListedChars, c.ListedTokens, c.AlwaysOn, c.AlwaysOnChars, c.AlwaysOnTokens, c.Searchable}
}

// statusSearch is the search engine, and what is wrong with its settings.
type statusSearch struct {
	Engine  string `json:"engine"`
	Problem string `json:"problem,omitempty"`
}

// statusAssistant is one installed assistant and how always-on skills reach it.
type statusAssistant struct {
	Name     string `json:"name"`
	Delivery string `json:"delivery"` // hook, instructions or on request
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

// searchableSkill is one skill that is Searchable in a folder.
type searchableSkill struct {
	ID        string `json:"id"`
	Scope     string `json:"scope"` // global or project
	InLibrary bool   `json:"in_library"`
}

func status(out io.Writer, asJSON, verbose bool) error {
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
	engine := library.LoadEngine()
	cost := server.Cost(r, l, w.ProjectID())
	if asJSON {
		st := statusJSON{Folder: w.Dir, Skills: []servedSkill{}, Searchable: []searchableSkill{}, Catalog: catalogOf(cost), Search: statusSearch{engine.Engine, engine.Problem}, Assistants: installedAssistants(), Warnings: []string{}}
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
		for _, id := range r.SearchableServed(w.ProjectID()) {
			s := searchableSkill{ID: id, Scope: "project"}
			if slices.Contains(r.Searchable, id) {
				s.Scope = "global"
			}
			_, err := l.Get(id)
			s.InLibrary = err == nil
			st.Searchable = append(st.Searchable, s)
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
	for _, id := range append(r.Served(w.ProjectID()), r.SearchableServed(w.ProjectID())...) {
		if _, err := l.Get(id); err != nil {
			fmt.Fprintf(out, "warning: %s is assigned but not in the library, so it is not served\n", id)
		}
	}
	for _, c := range copies {
		fmt.Fprintf(out, "warning: %s is also installed in Claude Code, so it may be loaded twice: %s\n", c.ID, describeCopy(c))
	}
	printCost(out, cost, verbose)
	fmt.Fprintf(out, "\nSearch engine: %s\n", engine.Engine)
	if engine.Problem != "" {
		fmt.Fprintln(out, "warning:", engine.Problem)
	}
	if hint := unfoundHint(r, l, w.ProjectID()); hint != "" {
		fmt.Fprintln(out, hint)
	}
	if engine.Engine == library.EngineWords && anySearchable(r) {
		fmt.Fprintln(out, "search by meaning:", setupCommand)
	}
	if len(copies) > 0 {
		fmt.Fprintln(out, "Run quiver doctor --fix to delete the copies.")
	}
	if as := installedAssistants(); len(as) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Assistants (how always-on skills reach them):")
		for _, a := range as {
			fmt.Fprintf(out, "  %s: %s\n", a.Name, a.Delivery)
		}
	}
	return nil
}

// installedAssistants lists the installed assistants and how always-on skills
// reach each one. It is empty on an error or when none is installed.
func installedAssistants() []statusAssistant {
	ts, err := mcpTargets(nil)
	if err != nil {
		return nil
	}
	as := make([]statusAssistant, 0, len(ts))
	for _, t := range ts {
		d, missing := t.actualDelivery()
		level := d.Level()
		if missing {
			level += " (the session-start hook is missing; run quiver doctor)"
		}
		as = append(as, statusAssistant{Name: t.name(), Delivery: level})
	}
	return as
}

// anySearchable reports whether a skill is Searchable in any scope.
func anySearchable(r *registry.Registry) bool {
	if len(r.Searchable) > 0 {
		return true
	}
	for _, p := range r.Projects {
		if len(p.Searchable) > 0 {
			return true
		}
	}
	return false
}

// printCost says what the assignments of the folder cost at session start.
// The numbers inform; nothing refuses an assignment or trims the listing.
func printCost(out io.Writer, c server.CatalogCost, verbose bool) {
	fmt.Fprintf(out, "\nSession start: %s listed, %d characters (about %d tokens, an estimate)", plural(c.Listed, "skill"), c.ListedChars, c.ListedTokens)
	if c.AlwaysOn > 0 {
		fmt.Fprintf(out, "; %d always-on, loaded in full: %d characters (about %d tokens)", c.AlwaysOn, c.AlwaysOnChars, c.AlwaysOnTokens)
	}
	fmt.Fprintf(out, "; %d Searchable, which add nothing to the listing.\n", c.Searchable)
	if !verbose || len(c.Largest) == 0 {
		return
	}
	fmt.Fprintln(out, "Longest listed entries; move a skill to Searchable to take its line out of the listing:")
	for _, e := range c.Largest {
		cmd := "quiver add " + e.ID + " --searchable"
		if e.Scope == "global" {
			cmd += " --global"
		}
		fmt.Fprintf(out, "  %-30s %5d characters  %s\n", e.ID, e.Chars, cmd)
	}
}

// unfoundHint is the line for a library whose skills the assistant's search
// does not find: skills at no level in the folder, and no Searchable skill at
// all. v1.5 searched the whole library; search_skills now finds only
// Searchable skills. It names the source with the most such skills. It is
// empty when the folder has a Searchable skill or nothing is unassigned.
func unfoundHint(r *registry.Registry, l *library.Library, projectID string) string {
	if len(r.SearchableServed(projectID)) > 0 {
		return ""
	}
	n, bySource := 0, map[string]int{}
	for _, s := range l.All() {
		if r.LevelIn(projectID, s.ID) != registry.NotAssigned {
			continue
		}
		n++
		if src, _, ok := strings.Cut(s.ID, "/"); ok {
			bySource[src]++
		}
	}
	if n == 0 {
		return ""
	}
	line := fmt.Sprintf("Search finds only Searchable skills: %s of the library are at no level here, so the assistant's search does not find them", plural(n, "skill"))
	best := ""
	for src, c := range bySource {
		if c > bySource[best] || c == bySource[best] && src < best {
			best = src
		}
	}
	if best == "" {
		return line + ". Make one findable with: quiver add <id> --searchable"
	}
	return fmt.Sprintf("%s (%d of them in source %s). Make them findable with: quiver add --source %s --searchable --global (or quiver add <id> --searchable)", line, bySource[best], best, best)
}
