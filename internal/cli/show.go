package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/skill"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show a skill: where it comes from, where it is assigned, its text",
		Long: `Show a skill of the library: its ID and name, where it comes from (local,
or the source and its commit), its folder, where it is assigned and whether
it is always-on there, its extra files, then the text of its SKILL.md.`,
		Example: `  quiver show pdf
  quiver show anthropic/docx --json`,
		ValidArgsFunction: skillIDs(false, 1),
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return show(cmd.OutOrStdout(), args[0], asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}

// shownSkill is the JSON output of show.
type shownSkill struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Source      string       `json:"source"`
	Commit      string       `json:"commit"` // HEAD of local/ or of the source, or the pinned commit; "" if unknown
	Pinned      bool         `json:"pinned"`
	Dir         string       `json:"dir"`
	Assigned    []assignment `json:"assigned"`
	Files       []string     `json:"files"`
	Text        string       `json:"text"` // the whole SKILL.md
}

// assignment is one place a skill is assigned: every folder, or a project.
type assignment struct {
	Project  string   `json:"project,omitempty"` // project ID; "" for every folder
	Name     string   `json:"name,omitempty"`
	Paths    []string `json:"paths,omitempty"`
	AlwaysOn bool     `json:"alwaysOn"`
}

func show(out io.Writer, id string, asJSON bool) error {
	l, err := library.Load()
	if err != nil {
		return err
	}
	s, err := l.Get(id)
	if err != nil {
		return err
	}
	r, err := registry.Load()
	if err != nil {
		return err
	}
	text, err := os.ReadFile(s.Path())
	if err != nil {
		return err
	}
	files, err := skill.Files(s)
	if err != nil {
		return err
	}
	sh := shownSkill{
		ID: s.ID, Name: s.Name, Description: s.Description, Source: s.Source, Dir: s.Dir,
		Assigned: assignments(r, s.ID), Files: files, Text: string(text),
	}
	repo := filepath.Dir(s.Dir)
	if s.Source != library.LocalSource {
		if dir, err := library.SourcesDir(); err == nil {
			repo = filepath.Join(dir, s.Source)
		}
	}
	if pin := l.PinnedAt(s.ID); pin != "" {
		sh.Commit, sh.Pinned = short(pin), true
	} else if head, err := git.Head(repo); err == nil {
		sh.Commit = short(head)
	}
	if sh.Files == nil {
		sh.Files = []string{}
	}
	if asJSON {
		return printJSON(out, sh)
	}
	printShown(out, sh)
	return nil
}

// assignments lists where id is assigned: every folder first, then the
// projects by name.
func assignments(r *registry.Registry, id string) []assignment {
	out := []assignment{}
	if slices.Contains(r.Global, id) {
		out = append(out, assignment{AlwaysOn: slices.Contains(r.AlwaysOn, id)})
	}
	using := r.ProjectsUsing(id)
	sort.Slice(using, func(i, j int) bool { return r.Projects[using[i]].Name < r.Projects[using[j]].Name })
	for _, pid := range using {
		p := r.Projects[pid]
		out = append(out, assignment{Project: pid, Name: p.Name, Paths: p.Paths, AlwaysOn: slices.Contains(p.AlwaysOn, id)})
	}
	return out
}

func printShown(out io.Writer, sh shownSkill) {
	source := sh.Source
	switch {
	case sh.Pinned:
		source += " (pinned at " + sh.Commit + ")"
	case sh.Commit != "":
		source += " (" + sh.Commit + ")"
	}
	fmt.Fprintf(out, "ID:       %s\n", sh.ID)
	if sh.Name != sh.ID {
		fmt.Fprintf(out, "Name:     %s\n", sh.Name)
	}
	fmt.Fprintf(out, "Source:   %s\n", source)
	fmt.Fprintf(out, "Folder:   %s\n", sh.Dir)
	var where []string
	for _, a := range sh.Assigned {
		w := "every folder"
		if a.Project != "" {
			w = "project " + registry.ProjectLabel(a.Name, a.Paths)
		}
		if a.AlwaysOn {
			w += ", always-on"
		}
		where = append(where, w)
	}
	if len(where) == 0 {
		where = []string{"nowhere"}
	}
	fmt.Fprintf(out, "Assigned: %s\n", strings.Join(where, "\n          "))
	if len(sh.Files) > 0 {
		fmt.Fprintf(out, "Files:    %s\n", strings.Join(sh.Files, "\n          "))
	}
	fmt.Fprintln(out)
	fmt.Fprint(out, sh.Text)
	if !strings.HasSuffix(sh.Text, "\n") {
		fmt.Fprintln(out)
	}
}
