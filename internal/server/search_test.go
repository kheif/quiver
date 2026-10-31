package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

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

// skillsEnv serves pdf globally and db-migration in the project; draft is
// in the library but not assigned.
func TestSearchSkillsMarksAssigned(t *testing.T) {
	w, _ := skillsEnv(t)
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	text, out, isErr := searchCall(t, cs, "a")
	if isErr {
		t.Fatal(text)
	}
	got := map[string]FoundSkill{}
	for _, f := range out.Skills {
		got[f.ID] = f
	}
	for id, want := range map[string]struct {
		assigned bool
		scope    string
	}{"pdf": {true, "global"}, "db-migration": {true, "project"}, "draft": {false, ""}} {
		if f, ok := got[id]; !ok || f.Assigned != want.assigned || f.Scope != want.scope || f.Source != "local" {
			t.Errorf("%s: got %+v", id, f)
		}
	}
	for _, want := range []string{"- draft, not assigned here: Not assigned anywhere.", "- pdf, assigned here (global):", "- db-migration, assigned here (project):"} {
		if !strings.Contains(text, want) {
			t.Errorf("text missing %q:\n%s", want, text)
		}
	}

	// Outside the project only global skills are assigned.
	cs = connect(t, New(Options{Version: "dev", Log: discard, Where: Where{Dir: t.TempDir()}}))
	if _, out, _ := searchCall(t, cs, "migration"); len(out.Skills) != 1 || out.Skills[0].Assigned {
		t.Errorf("outside the project: %+v", out.Skills)
	}
}

func TestSearchSkillsEmptyAndNoMatch(t *testing.T) {
	w, _ := skillsEnv(t)
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	if text, _, isErr := searchCall(t, cs, "  "); !isErr || !strings.Contains(text, "give the text") {
		t.Errorf("empty query: %v %s", isErr, text)
	}
	if text, out, isErr := searchCall(t, cs, "zzz"); isErr || len(out.Skills) != 0 || text != `No skill matches "zzz".` {
		t.Errorf("no match: %v %+v %q", isErr, out, text)
	}
}

func TestSearchSkillsLimit(t *testing.T) {
	w, local := skillsEnv(t)
	files := map[string]string{}
	for i := range maxFound + 3 {
		name := fmt.Sprintf("bulk-%02d", i)
		files[name+"/SKILL.md"] = "---\nname: " + name + "\ndescription: Bulk.\n---\n"
	}
	writeFiles(t, local, files)
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	text, out, _ := searchCall(t, cs, "bulk")
	if len(out.Skills) != maxFound || out.More != 3 || !strings.Contains(text, "3 more matches") {
		t.Errorf("got %d skills, more %d:\n%s", len(out.Skills), out.More, text)
	}
}
