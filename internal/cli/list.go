package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/spf13/cobra"
)

type listOptions struct {
	source     string
	unassigned bool
	json       bool
}

func newListCmd() *cobra.Command {
	var o listOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List library skills and where each one is assigned",
		Long: `List every skill in the library: your own (source "local") and those from
sources. ASSIGNED says whether a skill is global and how many projects have
it, and where it is Searchable (found by searching, not listed at session
start), as stored for each scope. Skills that cannot be loaded are listed as warnings at the end.

To see what one folder gets, use quiver status; for one skill, quiver where.`,
		Example: `  quiver list
  quiver list --source local
  quiver list --unassigned
  quiver list --json`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return list(cmd.OutOrStdout(), o)
		},
	}
	cmd.Flags().StringVar(&o.source, "source", "", `only skills from this source ("local" for your own skills)`)
	_ = cmd.RegisterFlagCompletionFunc("source", completeSources(true, 0))
	cmd.Flags().BoolVar(&o.unassigned, "unassigned", false, "only skills that are not assigned anywhere")
	cmd.Flags().BoolVar(&o.json, "json", false, "print JSON")
	return cmd
}

// listedSkill is one row of list, and one entry of its JSON output.
type listedSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Source      string   `json:"source"`
	Description string   `json:"description"`
	Global      bool     `json:"global"`
	Projects    []string `json:"projects"` // project IDs
	// Searchable is where the skill is Searchable. Global and Projects keep
	// describing the served assignments.
	Searchable searchableAt `json:"searchable"`
}

// searchableAt is where a skill is Searchable: in every folder, or in
// these projects (IDs).
type searchableAt struct {
	Global   bool     `json:"global"`
	Projects []string `json:"projects"`
}

// assigned describes where the skill is assigned: "global", "3 projects",
// both, and "searchable (global, 2 projects)" for the Searchable level, or
// "" when it is assigned nowhere.
func (s listedSkill) assigned() string {
	parts := scopes(s.Global, len(s.Projects))
	if search := scopes(s.Searchable.Global, len(s.Searchable.Projects)); len(search) > 0 {
		parts = append(parts, "searchable ("+strings.Join(search, ", ")+")")
	}
	return strings.Join(parts, ", ")
}

// scopes names the scopes of one level: "global" and "N projects".
func scopes(global bool, projects int) []string {
	var parts []string
	if global {
		parts = append(parts, "global")
	}
	switch projects {
	case 0:
	case 1:
		parts = append(parts, "1 project")
	default:
		parts = append(parts, fmt.Sprintf("%d projects", projects))
	}
	return parts
}

func list(out io.Writer, o listOptions) error {
	if o.source != "" && o.source != library.LocalSource {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if _, ok := cfg.Sources[o.source]; !ok {
			return fmt.Errorf("unknown source %q; see quiver source list", o.source)
		}
	}
	l, err := library.Load()
	if err != nil {
		return err
	}
	r, err := registry.Load()
	if err != nil {
		return err
	}
	all := l.All()
	skills := []listedSkill{}
	for _, s := range all {
		ls := listedSkill{
			ID:          s.ID,
			Name:        s.Name,
			Source:      s.Source,
			Description: s.Description,
			Global:      slices.Contains(r.Global, s.ID),
			Projects:    r.ProjectsUsing(s.ID),
			Searchable: searchableAt{
				Global:   slices.Contains(r.Searchable, s.ID),
				Projects: r.SearchableProjects(s.ID),
			},
		}
		if ls.Projects == nil {
			ls.Projects = []string{}
		}
		if ls.Searchable.Projects == nil {
			ls.Searchable.Projects = []string{}
		}
		if o.source != "" && ls.Source != o.source || o.unassigned && ls.assigned() != "" {
			continue
		}
		skills = append(skills, ls)
	}
	if o.json {
		warnings := l.Warnings
		if warnings == nil {
			warnings = []string{}
		}
		return printJSON(out, struct {
			Skills   []listedSkill `json:"skills"`
			Warnings []string      `json:"warnings"`
		}{skills, warnings})
	}
	switch {
	case len(all) == 0:
		fmt.Fprintln(out, "No skills. Add a source with: quiver source add <repo>")
	case len(skills) == 0:
		fmt.Fprintln(out, "No skills match.")
	default:
		width, _ := terminal(out)
		printSkills(out, skills, width)
	}
	if len(l.Warnings) > 0 {
		fmt.Fprintln(out)
	}
	for _, w := range l.Warnings {
		fmt.Fprintln(out, dim(out, "warning: "+w))
	}
	return nil
}

// printJSON prints v as indented JSON.
func printJSON(out io.Writer, v any) error {
	e := json.NewEncoder(out)
	e.SetIndent("", "  ")
	return e.Encode(v)
}

// printSkills prints the table. Descriptions are cut to fit width; 0 means
// no limit.
func printSkills(out io.Writer, skills []listedSkill, width int) {
	idW, asgW := len("ID"), len("ASSIGNED")
	for _, s := range skills {
		idW = max(idW, len(s.ID))
		asgW = max(asgW, len(s.assigned()))
	}
	descW := 0
	if width > 0 {
		// Keep some description even in a narrow terminal.
		descW = max(width-idW-asgW-4, 20)
	}
	row := func(id, assigned, desc string) {
		line := fmt.Sprintf("%-*s  %-*s  %s", idW, id, asgW, assigned, cut(desc, descW))
		fmt.Fprintln(out, strings.TrimRight(line, " "))
	}
	row("ID", "ASSIGNED", "DESCRIPTION")
	for _, s := range skills {
		row(s.ID, s.assigned(), s.Description)
	}
}

// cut puts s on one line and shortens it to n characters, ending in "…".
// n of 0 means no limit.
func cut(s string, n int) string {
	// A skill's text must not move the cursor or recolor the terminal.
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if n <= 0 || len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
