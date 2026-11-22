package cli

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/server"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var asJSON bool
	var engine string
	cmd := &cobra.Command{
		Use:   "search <text>",
		Short: "Search every skill of the library, also unassigned ones",
		Long: `Search the names and descriptions of every skill in the library,
ignoring case. Exact names come first, then names that start with the text,
then names that contain it, then skills whose descriptions fit the words,
best first. HERE says whether a skill is served in this folder: global or
project, or "searchable" when the assistant only finds it by searching. The
JSON gains a level field for this folder.

The engine is the one that "quiver engine" shows; --engine uses another for
this search only.`,
		Example: `  quiver search pdf
  quiver search "pull request" --json
  quiver search --engine words "review a contract"`,
		ValidArgsFunction: noArgs,
		Args:              cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return search(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), strings.Join(args, " "), engine, asJSON)
		},
	}
	cmd.Flags().StringVar(&engine, "engine", "", "search engine for this search only (default: the one of quiver engine)")
	_ = cmd.RegisterFlagCompletionFunc("engine", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return library.Engines(), cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}

// foundSkill is one row of search, and one entry of its JSON output.
type foundSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	Description string `json:"description"`
	Here        string `json:"here"`  // served here: "global", "project" or ""
	Level       string `json:"level"` // in this folder: "always-on", "on-demand", "searchable" or ""
}

// levelName is the name of a level in the JSON output.
func levelName(l registry.Level) string {
	switch l {
	case registry.AlwaysOn:
		return "always-on"
	case registry.OnDemand:
		return "on-demand"
	case registry.Searchable:
		return "searchable"
	}
	return ""
}

func search(ctx context.Context, out, errOut io.Writer, text, engine string, asJSON bool) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("give the text to search for")
	}
	l, err := library.Load()
	if err != nil {
		return err
	}
	r, err := registry.Load()
	if err != nil {
		return err
	}
	w, err := server.Resolve()
	if err != nil {
		return err
	}
	var own []string
	if p, ok := r.Projects[w.ProjectID()]; ok {
		own = p.Skills
	}
	se, err := l.NewSearch(engine)
	if err != nil {
		return err
	}
	if se.Problem != "" {
		fmt.Fprintln(errOut, "quiver: "+se.Problem)
	}
	matches, note := se.Find(ctx, text, l.All())
	if note != "" {
		fmt.Fprintln(errOut, "quiver: "+note)
	}
	found := []foundSkill{}
	for _, m := range matches {
		f := foundSkill{ID: m.ID, Name: m.Name, Source: m.Source, Description: m.Description}
		f.Level = levelName(r.LevelIn(w.ProjectID(), m.ID))
		switch {
		case slices.Contains(r.Global, m.ID):
			f.Here = "global"
		case slices.Contains(own, m.ID):
			f.Here = "project"
		}
		found = append(found, f)
	}
	if asJSON {
		return printJSON(out, struct {
			Skills []foundSkill `json:"skills"`
		}{found})
	}
	if len(found) == 0 {
		fmt.Fprintf(out, "No skill matches %q.\n", text)
		return nil
	}
	idW, hereW := len("ID"), len("HERE")
	for _, f := range found {
		idW = max(idW, len(f.ID))
		hereW = max(hereW, len(hereText(f)))
	}
	descW := 0
	if width, _ := terminal(out); width > 0 {
		descW = max(width-idW-hereW-4, 20)
	}
	row := func(id, here, desc string) {
		fmt.Fprintln(out, strings.TrimRight(fmt.Sprintf("%-*s  %-*s  %s", idW, id, hereW, here, cut(desc, descW)), " "))
	}
	row("ID", "HERE", "DESCRIPTION")
	for _, f := range found {
		row(f.ID, hereText(f), f.Description)
	}
	return nil
}

// hereText is the HERE column: where the skill is served here, or
// "searchable" when the assistant only finds it by searching.
func hereText(f foundSkill) string {
	if f.Here == "" && f.Level == "searchable" {
		return "searchable"
	}
	return f.Here
}
