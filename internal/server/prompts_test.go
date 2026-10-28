package server

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/registry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// promptText gets a prompt and returns the text of its one user message.
func promptText(t *testing.T, cs *mcp.ClientSession, name string, args map[string]string) string {
	t.Helper()
	res, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Messages) != 1 || res.Messages[0].Role != "user" {
		t.Fatalf("messages = %+v", res.Messages)
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("content = %T", res.Messages[0].Content)
	}
	return tc.Text
}

func TestListPrompts(t *testing.T) {
	w, _ := skillsEnv(t)
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	res, err := cs.ListPrompts(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, p := range res.Prompts {
		names = append(names, p.Name)
		if p.Name == "pdf" && p.Description != "Read, create and edit PDF files." {
			t.Errorf("pdf description = %q", p.Description)
		}
		if len(p.Arguments) != maxArgs || p.Arguments[0].Name != "args" || p.Arguments[0].Required || p.Arguments[maxArgs-1].Name != "args32" {
			t.Errorf("%s arguments = %d, first %+v", p.Name, len(p.Arguments), p.Arguments[0])
		}
	}
	slices.Sort(names)
	// draft is not assigned and gone is not in the library.
	if want := []string{"db-migration", "pdf"}; !slices.Equal(names, want) {
		t.Errorf("prompts = %v, want %v", names, want)
	}
}

func TestGetPrompt(t *testing.T) {
	w, local := skillsEnv(t)
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	got := promptText(t, cs, "db-migration", nil)
	want := "Read references/checklist.md first.\n\n---\n" +
		"Extra files (load with get_skill_file): references/checklist.md, scripts/run.sh\n" +
		"Scripts and files referenced by this skill are in " + filepath.Join(local, "db-migration") + ". Use absolute paths.\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	// Claude Code gives one word to each argument.
	got = promptText(t, cs, "db-migration", map[string]string{"args": "add", "args2": "a", "args3": "users", "args4": "table"})
	if want += "\nTask: add a users table\n"; got != want {
		t.Errorf("with args: got %q\nwant %q", got, want)
	}
	// An edit after startup is seen.
	writeFiles(t, local, map[string]string{"pdf/SKILL.md": "---\nname: pdf\ndescription: d\n---\nUse qpdf now.\n"})
	if got := promptText(t, cs, "pdf", nil); !strings.HasPrefix(got, "Use qpdf now.\n") {
		t.Errorf("after edit: %q", got)
	}
	// A skill deleted after startup is an error, not a crash.
	if err := os.RemoveAll(filepath.Join(local, "pdf")); err != nil {
		t.Fatal(err)
	}
	if _, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{Name: "pdf"}); err == nil || !strings.Contains(err.Error(), `skill "pdf" not found`) {
		t.Errorf("deleted skill: %v", err)
	}
}

// A source skill keeps the / of its ID in the prompt name.
func TestPromptSourceSkill(t *testing.T) {
	w, _ := skillsEnv(t)
	home := os.Getenv("QUIVER_HOME")
	cfg := config.Default()
	cfg.Sources["acme"] = config.Source{Repo: "https://example.com/acme.git"}
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, filepath.Join(home, "sources", "acme"), map[string]string{
		"skills/review/SKILL.md": "---\nname: review\ndescription: Review code.\n---\nCheck the tests.\n",
	})
	if err := registry.Update(func(r *registry.Registry) error { r.AssignGlobal("acme/review"); return nil }); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	if got := promptText(t, cs, "acme/review", map[string]string{"args": "main.go"}); !strings.HasPrefix(got, "Check the tests.") || !strings.HasSuffix(got, "\nTask: main.go\n") {
		t.Errorf("got %q", got)
	}
}

func TestJoinArgs(t *testing.T) {
	for _, c := range []struct {
		in   map[string]string
		want string
	}{
		{nil, ""},
		{map[string]string{"args": "  ship  "}, "ship"},
		{map[string]string{"args": "ship", "args2": "", "args3": "1.2"}, "ship 1.2"},
		{map[string]string{"args32": "last", "args": "first"}, "first last"},
		{map[string]string{"args33": "dropped"}, ""},
	} {
		if got := joinArgs(c.in); got != c.want {
			t.Errorf("joinArgs(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
