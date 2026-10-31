package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

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
it. Skills that cannot be loaded are listed as warnings at the end.

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
}

// assigned describes where the skill is assigned: "global", "3 projects",
// both, or "".
func (s listedSkill) assigned() string {
	var parts []string
	if s.Global {
		parts = append(parts, "global")
	}
	switch n := len(s.Projects); n {
	case 0:
	case 1:
		parts = append(parts, "1 project")
	default:
		parts = append(parts, fmt.Sprintf("%d projects", n))
	}
	return strings.Join(parts, ", ")
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
		}
		if ls.Projects == nil {
			ls.Projects = []string{}
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
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if n <= 0 || len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
