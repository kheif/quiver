package library

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/setup"
	"github.com/kheif/quiver/internal/skill"
)

// createEnv sets up a Quiver home with a local git repository and a known
// project app-000001.
func createEnv(t *testing.T) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), "q")
	t.Setenv("QUIVER_HOME", home)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	if _, err := setup.Init(); err != nil {
		t.Fatal(err)
	}
	err := registry.Update(func(r *registry.Registry) error {
		_ = r.EnsureProject("app-000001", filepath.Join(t.TempDir(), "app"))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(home, "local")
}

// gitOut runs git in dir and returns its trimmed output.
func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestCreateLocalScopes(t *testing.T) {
	local := createEnv(t)
	for _, c := range []struct {
		name  string
		scope Scope
	}{
		{"none-skill", ScopeNone},
		{"global-skill", ScopeGlobal},
		{"project-skill", ScopeProject},
	} {
		warnings, err := CreateLocal(c.name, "Use it: when \"quoted\".", "# Body", nil, c.scope, "app-000001")
		if err != nil || len(warnings) != 0 {
			t.Fatalf("%s: %v, %v", c.name, warnings, err)
		}
		s, err := skill.Load(filepath.Join(local, c.name))
		if err != nil {
			t.Fatal(err)
		}
		if s.Name != c.name || s.Description != `Use it: when "quoted".` || s.Body != "\n# Body\n" {
			t.Errorf("%s: skill = %q, %q, %q", c.name, s.Name, s.Description, s.Body)
		}
		if got := gitOut(t, local, "log", "-1", "--format=%s"); got != "add "+c.name {
			t.Errorf("%s: commit = %q", c.name, got)
		}
	}
	if st := gitOut(t, local, "status", "--porcelain"); st != "" {
		t.Errorf("uncommitted changes:\n%s", st)
	}
	r, err := registry.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Global, []string{"global-skill"}) {
		t.Errorf("global = %v", r.Global)
	}
	if got := r.Projects["app-000001"].Skills; !reflect.DeepEqual(got, []string{"project-skill"}) {
		t.Errorf("project skills = %v", got)
	}
}

func TestCreateLocalFiles(t *testing.T) {
	local := createEnv(t)
	files := map[string]string{"references/guide.md": "guide", "template.txt": "t"}
	if _, err := CreateLocal("pdf", "Work with PDFs.", "", files, ScopeNone, ""); err != nil {
		t.Fatal(err)
	}
	for p, want := range files {
		got, err := os.ReadFile(filepath.Join(local, "pdf", filepath.FromSlash(p)))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v", p, got, err)
		}
	}
	if got := gitOut(t, local, "ls-files", "pdf"); got != "pdf/SKILL.md\npdf/references/guide.md\npdf/template.txt" {
		t.Errorf("committed files:\n%s", got)
	}
}

func TestCreateLocalRejects(t *testing.T) {
	local := createEnv(t)
	if _, err := CreateLocal("pdf", "Work with PDFs.", "", nil, ScopeGlobal, ""); err != nil {
		t.Fatal(err)
	}
	head := gitOut(t, local, "rev-parse", "HEAD")
	cases := map[string]struct {
		name    string
		files   map[string]string
		scope   Scope
		project string
		want    string
	}{
		"duplicate":      {"pdf", nil, ScopeNone, "", "skill already exists"},
		"bad name":       {"My Skill", nil, ScopeNone, "", "invalid name"},
		"slash in name":  {"a/b", nil, ScopeNone, "", "invalid name"},
		"parent dir":     {"x", map[string]string{"../escape.md": ""}, ScopeNone, "", "invalid file path"},
		"nested parent":  {"x", map[string]string{"a/../../escape.md": ""}, ScopeNone, "", "invalid file path"},
		"absolute":       {"x", map[string]string{"/tmp/escape.md": ""}, ScopeNone, "", "invalid file path"},
		"backslash":      {"x", map[string]string{`..\escape.md`: ""}, ScopeNone, "", "invalid file path"},
		"empty path":     {"x", map[string]string{"": ""}, ScopeNone, "", "invalid file path"},
		"git folder":     {"x", map[string]string{".git/config": ""}, ScopeNone, "", ".git is not allowed"},
		"skill file":     {"x", map[string]string{"SKILL.md": ""}, ScopeNone, "", "SKILL.md is written"},
		"unknown proj":   {"x", nil, ScopeProject, "nope", "unknown project"},
		"unknown scope":  {"x", nil, Scope("team"), "", "unknown scope"},
		"no description": {"x", nil, ScopeNone, "", "description"},
	}
	for name, c := range cases {
		desc := "Something."
		if name == "no description" {
			desc = " "
		}
		_, err := CreateLocal(c.name, desc, "", c.files, c.scope, c.project)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want %q", name, err, c.want)
		}
	}
	if _, err := CreateLocal("pdf", "d", "", nil, ScopeNone, ""); !errors.Is(err, ErrSkillExists) {
		t.Errorf("duplicate err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(local, "x")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("rejected skill was written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(local), "escape.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file escaped the skill folder: %v", err)
	}
	if got := gitOut(t, local, "rev-parse", "HEAD"); got != head {
		t.Error("a rejected skill was committed")
	}
	entries, _ := os.ReadDir(local)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".new-") {
			t.Errorf("temp folder left behind: %s", e.Name())
		}
	}
	r, _ := registry.Load()
	if !reflect.DeepEqual(r.Global, []string{"pdf"}) || len(r.Projects["app-000001"].Skills) != 0 {
		t.Errorf("registry changed: %+v", r)
	}
}

func TestCommitEdit(t *testing.T) {
	local := createEnv(t)
	if _, err := CreateLocal("pdf", "Work with PDFs.", "", nil, ScopeNone, ""); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(local, "pdf", "SKILL.md")
	if err := os.WriteFile(path, []byte("---\nname: pdf\ndescription: Fill PDF forms.\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitEdit("pdf"); err != nil {
		t.Fatal(err)
	}
	if got := gitOut(t, local, "log", "-1", "--format=%s"); got != "edit pdf" {
		t.Errorf("commit = %q", got)
	}
	// An invalid skill is not committed.
	if err := os.WriteFile(path, []byte("---\nname: pdf\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitEdit("pdf"); err == nil || !strings.Contains(err.Error(), "description is missing") {
		t.Errorf("err = %v", err)
	}
	if got := gitOut(t, local, "status", "--porcelain"); got != "M pdf/SKILL.md" {
		t.Errorf("status = %q", got)
	}
}
