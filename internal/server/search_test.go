package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/registry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func searchCall(t *testing.T, cs *mcp.ClientSession, query string) (string, SearchResult, bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "search_skills", Arguments: map[string]any{"query": query}})
	if err != nil {
		t.Fatal(err)
	}
	var text string
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			text += tc.Text
		}
	}
	var out SearchResult
	if res.StructuredContent != nil {
		data, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatal(err)
		}
	}
	return text, out, res.IsError
}

// searchEnv is skillsEnv with draft Searchable in every folder and
// "pipeline-notes" Searchable in the project only. pdf is listed, db-migration
// is listed in the project, and "loose" is assigned nowhere.
func searchEnv(t *testing.T) Where {
	t.Helper()
	w, local := skillsEnv(t)
	writeFiles(t, local, map[string]string{
		"pipeline-notes/SKILL.md": "---\nname: pipeline-notes\ndescription: Notes on the release pipeline.\n---\nNotes.\n",
		"loose/SKILL.md":          "---\nname: loose\ndescription: Assigned nowhere.\n---\nLoose.\n",
	})
	err := registry.Update(func(r *registry.Registry) error {
		r.AssignSearchableGlobal("draft")
		return r.AssignSearchable(w.ProjectID(), "pipeline-notes")
	})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestSearchSkillsReturnsOnlySearchableSkills(t *testing.T) {
	w := searchEnv(t)
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	text, out, isErr := searchCall(t, cs, "draft")
	if isErr {
		t.Fatal(text)
	}
	if len(out.Skills) != 1 || out.Skills[0] != (FoundSkill{ID: "draft", Name: "draft", Description: "Not assigned anywhere.", Source: "local", Assigned: true, Scope: "global"}) {
		t.Errorf("draft: %+v", out.Skills)
	}
	if want := "- draft (global): Not assigned anywhere."; text != want {
		t.Errorf("text %q, want %q", text, want)
	}
	if _, out, _ := searchCall(t, cs, "pipeline-notes"); len(out.Skills) != 1 || out.Skills[0].Scope != "project" || !out.Skills[0].Assigned {
		t.Errorf("project scope: %+v", out.Skills)
	}
	// Listed skills are known to the assistant already, and a skill that is
	// assigned nowhere is not the user's choice: neither is returned.
	for _, q := range []string{"pdf", "db-migration", "loose"} {
		if text, out, _ := searchCall(t, cs, q); len(out.Skills) != 0 || text != fmt.Sprintf("No searchable skill matches %q.", q) {
			t.Errorf("%s: %+v %q", q, out.Skills, text)
		}
	}
}

// Outside the project only the global Searchable skills are searched.
func TestSearchSkillsOutsideTheProject(t *testing.T) {
	searchEnv(t)
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: Where{Dir: t.TempDir()}}))
	if _, out, _ := searchCall(t, cs, "pipeline-notes"); len(out.Skills) != 0 {
		t.Errorf("project skill outside the project: %+v", out.Skills)
	}
	if _, out, _ := searchCall(t, cs, "draft"); len(out.Skills) != 1 {
		t.Errorf("global skill: %+v", out.Skills)
	}
}

// A skill that is listed through the other scope is not Searchable here.
func TestSearchSkillsListedThroughTheOtherScope(t *testing.T) {
	w := searchEnv(t)
	err := registry.Update(func(r *registry.Registry) error {
		return r.Assign(w.ProjectID(), "draft")
	})
	if err != nil {
		t.Fatal(err)
	}
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	if _, out, _ := searchCall(t, cs, "draft"); len(out.Skills) != 0 {
		t.Errorf("a listed skill is searched: %+v", out.Skills)
	}
}

func TestSearchSkillsEmptyAndNoMatch(t *testing.T) {
	w := searchEnv(t)
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	if text, _, isErr := searchCall(t, cs, "  "); !isErr || !strings.Contains(text, "give the text") {
		t.Errorf("empty query: %v %s", isErr, text)
	}
	if text, out, isErr := searchCall(t, cs, "zzz"); isErr || len(out.Skills) != 0 || text != `No searchable skill matches "zzz".` {
		t.Errorf("no match: %v %+v %q", isErr, out, text)
	}
}

// The cap is 10, and a small Searchable set is not crowded out by the
// skills that are not Searchable.
func TestSearchSkillsLimit(t *testing.T) {
	w, local := skillsEnv(t)
	files := map[string]string{}
	var ids []string
	for i := range maxFound + 3 {
		name := fmt.Sprintf("bulk-%02d", i)
		ids = append(ids, name)
		files[name+"/SKILL.md"] = "---\nname: " + name + "\ndescription: Bulk.\n---\n"
	}
	// Skills that fit the query as well, but are not Searchable.
	for i := range 20 {
		name := fmt.Sprintf("bulk-extra-%02d", i)
		files[name+"/SKILL.md"] = "---\nname: " + name + "\ndescription: Bulk.\n---\n"
	}
	writeFiles(t, local, files)
	if err := registry.Update(func(r *registry.Registry) error {
		r.AssignSearchableGlobal(ids...)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	text, out, _ := searchCall(t, cs, "bulk")
	if maxFound != 10 || len(out.Skills) != 10 || out.More != 3 || !strings.Contains(text, "3 more matches") {
		t.Errorf("got %d skills, more %d:\n%s", len(out.Skills), out.More, text)
	}
	for _, f := range out.Skills {
		if strings.Contains(f.ID, "extra") {
			t.Errorf("a skill that is not Searchable: %s", f.ID)
		}
	}
}

// The Searchable skills of a small set are scored like in the whole library:
// a description that fits is found, and a query that fits nothing is not.
func TestSearchSkillsScoresLikeTheWholeLibrary(t *testing.T) {
	w, local := skillsEnv(t)
	files := map[string]string{
		"db-tuning/SKILL.md": "---\nname: db-tuning\ndescription: Finds slow SQL queries and suggests indexes. Use when a database report takes too long.\n---\nTune.\n",
	}
	for i := range 80 {
		name := fmt.Sprintf("filler-%02d", i)
		files[name+"/SKILL.md"] = fmt.Sprintf("---\nname: %s\ndescription: Handles filler task number %d for the team.\n---\n", name, i)
	}
	writeFiles(t, local, files)
	if err := registry.Update(func(r *registry.Registry) error {
		r.AssignSearchableGlobal("db-tuning")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	if _, out, _ := searchCall(t, cs, "my database report takes too long because of slow sql queries"); len(out.Skills) != 1 || out.Skills[0].ID != "db-tuning" {
		t.Errorf("a description that fits: %+v", out.Skills)
	}
	if _, out, _ := searchCall(t, cs, "plan a birthday party for twenty guests"); len(out.Skills) != 0 {
		t.Errorf("a query that fits nothing: %+v", out.Skills)
	}
}

// A settings file that cannot be used is logged once and never printed to
// stdout, which is the protocol channel; search runs as words.
func TestSearchSkillsSettingsProblem(t *testing.T) {
	w := searchEnv(t)
	writeFiles(t, filepath.Join(os.Getenv("QUIVER_HOME"), "state"), map[string]string{"search.json": `{"version": 2, "engine": "local"}`})
	var logged bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logged, nil))
	cs := connect(t, New(Options{Version: "dev", Log: log, Where: w}))

	stdout := os.Stdout
	r, wr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = wr
	for range 2 {
		if _, out, isErr := searchCall(t, cs, "draft"); isErr || len(out.Skills) != 1 {
			t.Errorf("search did not run as words: %v %+v", isErr, out.Skills)
		}
	}
	os.Stdout = stdout
	_ = wr.Close()
	if data, _ := io.ReadAll(r); len(data) != 0 {
		t.Errorf("wrote to stdout: %q", data)
	}
	if n := strings.Count(logged.String(), "search.json has version 2, but this quiver supports only up to 1: update quiver"); n != 1 {
		t.Errorf("logged %d times:\n%s", n, logged.String())
	}
}
