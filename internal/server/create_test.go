package server

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/setup"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// createEnv is skillsEnv with local/ as a git repository, so skills can be
// committed.
func createEnv(t *testing.T) (Where, string) {
	t.Helper()
	w, local := skillsEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	if _, err := setup.Init(); err != nil {
		t.Fatal(err)
	}
	return w, local
}

// createSkill calls create_skill and decodes its structured result.
func createSkillCall(t *testing.T, cs *mcp.ClientSession, args map[string]any) (string, CreateResult, bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "create_skill", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	var text string
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			text += tc.Text
		}
	}
	var out CreateResult
	if res.StructuredContent != nil {
		data, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatal(err)
		}
	}
	return text, out, res.IsError
}

func lastCommit(t *testing.T, local string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", local, "log", "-1", "--format=%s").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestCreateSkillProject(t *testing.T) {
	w, local := createEnv(t)
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	text, out, isErr := createSkillCall(t, cs, map[string]any{
		"name":        "release-notes",
		"description": "Write release notes from the git log.",
		"body":        "# Release notes\n\nGroup changes by type.",
		"files":       map[string]string{"references/style.md": "Short lines."},
		"scope":       "project",
	})
	if isErr {
		t.Fatal(text)
	}
	want := CreateResult{ID: "release-notes", Path: filepath.Join(local, "release-notes"), Scope: "project"}
	if out.ID != want.ID || out.Path != want.Path || out.Scope != want.Scope || out.ProjectFile != "" {
		t.Errorf("result = %+v", out)
	}
	if !strings.Contains(text, "It is served in app.") || strings.Contains(text, "commit it") {
		t.Errorf("text:\n%s", text)
	}
	if data, err := os.ReadFile(filepath.Join(local, "release-notes", "references", "style.md")); err != nil || string(data) != "Short lines." {
		t.Errorf("extra file = %q, %v", data, err)
	}
	if got := lastCommit(t, local); got != "add release-notes" {
		t.Errorf("commit = %q", got)
	}
	r, _ := registry.Load()
	if !slices.Contains(r.Projects[w.ProjectID()].Skills, "release-notes") || slices.Contains(r.Global, "release-notes") {
		t.Errorf("registry = %+v, global %v", r.Projects[w.ProjectID()], r.Global)
	}
	// The other tools see it in the same session.
	if got, isErr := call(t, cs, "get_skill", map[string]any{"id": "release-notes"}); isErr || !strings.Contains(got, "Group changes by type.") {
		t.Errorf("get_skill: %s", got)
	}
}

func TestCreateSkillGlobal(t *testing.T) {
	w, local := createEnv(t)
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	text, out, isErr := createSkillCall(t, cs, map[string]any{
		"name": "commit-style", "description": "How to write commit messages.", "body": "Use the imperative.", "scope": "global",
	})
	if isErr {
		t.Fatal(text)
	}
	if out.Scope != "global" || !strings.Contains(text, "It is served in every folder.") {
		t.Errorf("result = %+v\n%s", out, text)
	}
	r, _ := registry.Load()
	if !slices.Contains(r.Global, "commit-style") || slices.Contains(r.Projects[w.ProjectID()].Skills, "commit-style") {
		t.Errorf("global = %v", r.Global)
	}
	if got := lastCommit(t, local); got != "add commit-style" {
		t.Errorf("commit = %q", got)
	}
}

func TestCreateSkillRegistersProject(t *testing.T) {
	createEnv(t)
	dir := filepath.Join(t.TempDir(), "web")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: Where{Dir: dir, From: "working directory"}}))
	args := map[string]any{"name": "web-deploy", "description": "Deploy the site.", "body": "Run make deploy.", "scope": "project"}
	text, out, isErr := createSkillCall(t, cs, args)
	if isErr {
		t.Fatal(text)
	}
	file := filepath.Join(dir, project.FileName)
	if out.ProjectFile != file || !strings.Contains(text, "Created "+file+" for this project. Tell the user to commit it") {
		t.Errorf("result = %+v\n%s", out, text)
	}
	p, err := project.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := registry.Load()
	if got := r.Projects[p.ID]; got == nil || !slices.Equal(got.Skills, []string{"web-deploy"}) {
		t.Errorf("project = %+v", got)
	}
	// The session now serves the new project.
	if got, _ := call(t, cs, "list_skills", nil); !strings.Contains(got, `"id":"web-deploy"`) || !strings.Contains(got, `"scope":"project"`) {
		t.Errorf("list_skills: %s", got)
	}
	// A second skill uses the same project and creates no file.
	args["name"] = "web-test"
	text, out, isErr = createSkillCall(t, cs, args)
	if isErr || out.ProjectFile != "" {
		t.Errorf("second create: %+v\n%s", out, text)
	}
	if again, _ := project.Find(dir); again.ID != p.ID {
		t.Errorf("ID changed from %s to %s", p.ID, again.ID)
	}
}

func TestCreateSkillNoFolder(t *testing.T) {
	createEnv(t)
	cs := connect(t, New(Options{Version: "dev", Log: discard}))
	text, _, isErr := createSkillCall(t, cs, map[string]any{"name": "x", "description": "d", "body": "b", "scope": "project"})
	if !isErr || !strings.Contains(text, `use scope "global"`) {
		t.Errorf("isError %v: %s", isErr, text)
	}
}

func TestCreateSkillErrors(t *testing.T) {
	w, local := createEnv(t)
	writeFiles(t, local, map[string]string{"pdf-2/SKILL.md": "---\nname: pdf-2\ndescription: Taken too.\n---\n"})
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	head := lastCommit(t, local)
	for name, c := range map[string]struct {
		args map[string]any
		want string
	}{
		"taken":        {map[string]any{"name": "pdf"}, `a skill named "pdf" already exists; use another name, such as "pdf-3"`},
		"invalid name": {map[string]any{"name": "My Release Notes!"}, `use a name such as "my-release-notes"`},
		"parent path":  {map[string]any{"name": "evil", "files": map[string]string{"../../escape.md": "x"}}, "invalid file path"},
		"absolute":     {map[string]any{"name": "evil", "files": map[string]string{"/tmp/escape.md": "x"}}, "invalid file path"},
		"git":          {map[string]any{"name": "evil", "files": map[string]string{".git/hooks/post-commit": "x"}}, ".git is not allowed"},
		"no scope":     {map[string]any{"name": "evil", "scope": ""}, "scope"},
		"bad scope":    {map[string]any{"name": "evil", "scope": "team"}, "scope"},
	} {
		args := map[string]any{"description": "d", "body": "b", "scope": "project"}
		for k, v := range c.args {
			args[k] = v
		}
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "create_skill", Arguments: args})
		var text string
		if err != nil {
			text = err.Error()
		} else {
			if !res.IsError {
				t.Errorf("%s: no error", name)
			}
			for _, c := range res.Content {
				if tc, ok := c.(*mcp.TextContent); ok {
					text += tc.Text
				}
			}
		}
		if !strings.Contains(text, c.want) {
			t.Errorf("%s: %s\nwant %q", name, text, c.want)
		}
	}
	if _, err := os.Stat(filepath.Join(local, "evil")); !os.IsNotExist(err) {
		t.Errorf("rejected skill was written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(local), "escape.md")); !os.IsNotExist(err) {
		t.Errorf("file escaped: %v", err)
	}
	if got := lastCommit(t, local); got != head {
		t.Errorf("commit = %q", got)
	}
}

func TestCreateSkillAnnotations(t *testing.T) {
	w, _ := createEnv(t)
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range res.Tools {
		if tool.Name != "create_skill" {
			continue
		}
		a := tool.Annotations
		if a == nil || a.ReadOnlyHint || a.DestructiveHint == nil || *a.DestructiveHint {
			t.Errorf("annotations = %+v", a)
		}
		if !strings.Contains(tool.Description, "ask them before you call this tool") {
			t.Errorf("description = %q", tool.Description)
		}
		data, _ := json.Marshal(tool.InputSchema)
		if !strings.Contains(string(data), `"enum":["project","global"]`) {
			t.Errorf("schema = %s", data)
		}
		return
	}
	t.Error("create_skill not listed")
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{
		"My Release Notes!": "my-release-notes",
		"--a__b--":          "a-b",
		"Ünïcode":           "n-code",
		"!!!":               "",
	} {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}
