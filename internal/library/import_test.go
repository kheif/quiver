package library

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/registry"
)

// writeTree writes files (slash paths) under dir.
func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for p, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func skillMD(name, desc string) string {
	return "---\nname: " + name + "\ndescription: " + desc + "\n---\nBody.\n"
}

func TestFindSkills(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		".claude/skills/pdf/SKILL.md":     skillMD("pdf", "PDF."),
		".claude/skills/pdf/ref/SKILL.md": skillMD("inner", "Part of pdf."),
		".claude/skills/broken/SKILL.md":  "no frontmatter\n",
		".claude/skills/empty/notes.txt":  "not a skill",
		"tools/deploy/SKILL.md":           skillMD("deploy", "Deploy."),
		"node_modules/pkg/SKILL.md":       skillMD("pkg", "Vendored."),
		".git/SKILL.md":                   skillMD("git", "No."),
		"home/local/mine/SKILL.md":        skillMD("mine", "Library."),
		"unnamed/SKILL.md":                "---\ndescription: No name.\n---\n",
	})
	names := func(found []Found) []string {
		var out []string
		for _, f := range found {
			rel, _ := filepath.Rel(root, f.Dir)
			out = append(out, f.Name+"="+filepath.ToSlash(rel))
		}
		return out
	}
	found, err := FindSkills(root, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := names(found), []string{"broken=.claude/skills/broken", "pdf=.claude/skills/pdf"}; !slices.Equal(got, want) {
		t.Errorf("shallow = %v, want %v", got, want)
	}
	if found[0].Err == nil || found[1].Err != nil || found[1].Description != "PDF." {
		t.Errorf("shallow details = %+v", found)
	}
	found, err = FindSkills(root, true, filepath.Join(root, "home"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"broken=.claude/skills/broken", "pdf=.claude/skills/pdf", "deploy=tools/deploy", "unnamed=unnamed"}
	if got := names(found); !slices.Equal(got, want) {
		t.Errorf("deep = %v, want %v", got, want)
	}
}

func TestSetName(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"---\nname: old\ndescription: d\n---\nname: body\n", "---\nname: new\ndescription: d\n---\nname: body\n"},
		{"---\ndescription: d\n---\n", "---\nname: new\ndescription: d\n---\n"},
		{"no frontmatter\n", "no frontmatter\n"},
	} {
		if got := string(setName([]byte(c.in), "new")); got != c.want {
			t.Errorf("setName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestImportLocal(t *testing.T) {
	local := createEnv(t)
	src := filepath.Join(t.TempDir(), "pdf")
	writeTree(t, src, map[string]string{
		"SKILL.md":       skillMD("pdf", "PDF."),
		"scripts/run.sh": "echo hi\n",
		".git/HEAD":      "x",
	})
	if err := os.Symlink("/etc/hosts", filepath.Join(src, "hosts")); err != nil {
		t.Skipf("cannot create a symbolic link: %v", err) // Windows without developer mode
	}
	res, err := ImportLocal(src, "pdf", ScopeProject, "app-000001", "app")
	if err != nil {
		t.Fatal(err)
	}
	if res.Reused || len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "not a regular file") {
		t.Errorf("result = %+v", res)
	}
	files, _ := treeFiles(filepath.Join(local, "pdf"))
	if want := []string{"SKILL.md", filepath.Join("scripts", "run.sh")}; !slices.Equal(files, want) {
		t.Errorf("copied %v", files)
	}
	if got := gitOut(t, local, "log", "-1", "--format=%s"); got != "import pdf from app" {
		t.Errorf("commit = %q", got)
	}
	r, _ := registry.Load()
	if !slices.Contains(r.Projects["app-000001"].Skills, "pdf") {
		t.Errorf("not assigned: %v", r.Projects["app-000001"].Skills)
	}
	if _, err := os.Stat(filepath.Join(src, "SKILL.md")); err != nil {
		t.Errorf("original changed: %v", err)
	}

	// Same content again: assigned, not copied. .git is not compared, but
	// the link that was not copied is, so remove it first.
	if err := os.Remove(filepath.Join(src, "hosts")); err != nil {
		t.Fatal(err)
	}
	res, err = ImportLocal(src, "pdf", ScopeGlobal, "", "app")
	if err != nil || !res.Reused {
		t.Errorf("same content: %+v, %v", res, err)
	}
	if r, _ := registry.Load(); !slices.Contains(r.Global, "pdf") {
		t.Errorf("global = %v", r.Global)
	}

	// Different content under a taken name is refused, and nothing changes.
	writeTree(t, src, map[string]string{"SKILL.md": skillMD("pdf", "Another PDF.")})
	if _, err := ImportLocal(src, "pdf", ScopeGlobal, "", "app"); !errors.Is(err, ErrSkillExists) {
		t.Errorf("taken: %v", err)
	}
	if data, _ := os.ReadFile(filepath.Join(local, "pdf", "SKILL.md")); !strings.Contains(string(data), "description: PDF.") {
		t.Errorf("overwritten:\n%s", data)
	}

	// Under a new name, SKILL.md gets that name.
	if _, err := ImportLocal(src, "pdf-2", ScopeNone, "", "app"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(local, "pdf-2", "SKILL.md"))
	if want := skillMD("pdf-2", "Another PDF."); string(data) != want {
		t.Errorf("renamed SKILL.md = %q, want %q", data, want)
	}
}

func TestImportLocalInvalid(t *testing.T) {
	local := createEnv(t)
	src := filepath.Join(t.TempDir(), "bad")
	writeTree(t, src, map[string]string{"SKILL.md": "---\nname: bad\n---\n"})
	if _, err := ImportLocal(src, "bad", ScopeGlobal, "", "app"); err == nil || !strings.Contains(err.Error(), "description is missing") {
		t.Errorf("got %v", err)
	}
	if _, err := os.Stat(filepath.Join(local, "bad")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("left behind: %v", err)
	}
	if r, _ := registry.Load(); len(r.Global) != 0 {
		t.Errorf("assigned: %v", r.Global)
	}
}
