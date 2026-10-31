package cli

import (
	"encoding/json"
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
	cmd := &cobra.Command{
		Use:   "search <text>",
		Short: "Search every skill of the library, also unassigned ones",
		Long: `Search the IDs, names and descriptions of every skill in the library,
ignoring case. Exact names come first, then names that start with the text,
then names that contain it, then descriptions. HERE says whether a skill is
served in this folder: global or project.`,
		Example: `  quiver search pdf
  quiver search "pull request" --json`,
		ValidArgsFunction: noArgs,
		Args:              cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return search(cmd.OutOrStdout(), strings.Join(args, " "), asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}

// foundSkill is one row of search, and one entry of its JSON output.
type foundSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	Description string `json:"description"`
	Here        string `json:"here"` // "global", "project" or ""
}

func search(out io.Writer, text string, asJSON bool) error {
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
	found := []foundSkill{}
	for _, m := range l.Search(text) {
		f := foundSkill{ID: m.ID, Name: m.Name, Source: m.Source, Description: m.Description}
		switch {
		case slices.Contains(r.Global, m.ID):
			f.Here = "global"
		case slices.Contains(own, m.ID):
			f.Here = "project"
		}
		found = append(found, f)
	}
	if asJSON {
		e := json.NewEncoder(out)
		e.SetIndent("", "  ")
		return e.Encode(struct {
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
		hereW = max(hereW, len(f.Here))
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
		row(f.ID, f.Here, f.Description)
	}
	return nil
}
