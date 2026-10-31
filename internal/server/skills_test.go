package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
)

// writeFiles writes files under root. Paths are slash-separated.
func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// skillsEnv builds a library and a registry: pdf is global, db-migration
// belongs to the project, and draft is in the library but not assigned.
// "gone" is assigned to the project but missing from the library. It
// returns the project to serve and the library's local folder.
func skillsEnv(t *testing.T) (Where, string) {
	t.Helper()
	root, _, p := projectDir(t)
	home := t.TempDir()
	t.Setenv("QUIVER_HOME", home)
	local := filepath.Join(home, "local")
	writeFiles(t, local, map[string]string{
		"pdf/SKILL.md":                         "---\nname: pdf\ndescription: Read, create and edit PDF files.\n---\nUse pdftk.\n",
		"db-migration/SKILL.md":                "---\nname: db-migration\ndescription: Rules for database migrations.\n---\nRead references/checklist.md first.\n",
		"db-migration/references/checklist.md": "1. Back up.\n",
		"db-migration/scripts/run.sh":          "#!/bin/sh\n",
		"db-migration/.notes":                  "hidden",
		"draft/SKILL.md":                       "---\nname: draft\ndescription: Not assigned anywhere.\n---\nDraft body.\n",
	})
	err := registry.Update(func(r *registry.Registry) error {
		_ = r.EnsureProject(p.ID, root)
		r.AssignGlobal("pdf")
		return r.Assign(p.ID, "db-migration", "gone")
	})
	if err != nil {
		t.Fatal(err)
	}
	return Where{Dir: root, From: ProjectEnv, Project: p}, local
}

func TestTools(t *testing.T) {
	w, _ := skillsEnv(t)
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	if want := []string{"create_skill", "get_skill", "get_skill_file", "list_skills", "update_skill"}; !slices.Equal(names, want) {
		t.Errorf("tools = %v, want %v", names, want)
	}
}

// listSkills calls list_skills and decodes its result.
func listSkills(t *testing.T, w Where) func() []SkillInfo {
	t.Helper()
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	return func() []SkillInfo {
		t.Helper()
		out, isError := call(t, cs, "list_skills", map[string]any{})
		if isError {
			t.Fatalf("list_skills failed: %s", out)
		}
		var got struct{ Skills []SkillInfo }
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		return got.Skills
	}
}

func TestListSkills(t *testing.T) {
	w, _ := skillsEnv(t)
	list := listSkills(t, w)
	want := []SkillInfo{
		{ID: "db-migration", Name: "db-migration", Description: "Rules for database migrations.", Scope: "project"},
		{ID: "pdf", Name: "pdf", Description: "Read, create and edit PDF files.", Scope: "global"},
	}
	if got := list(); !slices.Equal(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}

	// Skills assigned during the session show up.
	err := registry.Update(func(r *registry.Registry) error {
		return r.Assign(w.ProjectID(), "draft")
	})
	if err != nil {
		t.Fatal(err)
	}
	got := list()
	if len(got) != 3 || got[1].ID != "draft" || got[1].Scope != "project" {
		t.Errorf("after assign: %+v", got)
	}
}

// list_skills returns every skill with its whole description, however many
// there are: project skills first, then global ones, always-on marked.
func TestListSkillsMany(t *testing.T) {
	w, local := skillsEnv(t)
	desc := strings.Repeat("Use this skill when the task is about its area. ", 20)
	files := map[string]string{}
	var ids []string
	for i := range 200 {
		id := fmt.Sprintf("skill-%03d", i)
		ids = append(ids, id)
		files[id+"/SKILL.md"] = "---\nname: " + id + "\ndescription: " + desc + "\n---\n"
	}
	writeFiles(t, local, files)
	err := registry.Update(func(r *registry.Registry) error {
		r.AssignGlobal(ids[:100]...)
		if err := r.Assign(w.ProjectID(), ids[100:]...); err != nil {
			return err
		}
		return r.MarkAlwaysOnGlobal("skill-000")
	})
	if err != nil {
		t.Fatal(err)
	}
	got := listSkills(t, w)()
	if len(got) != 202 {
		t.Fatalf("%d skills", len(got))
	}
	for i, s := range got {
		project := i < 101 // db-migration and skill-100 to skill-199
		if project != (s.Scope == "project") {
			t.Errorf("%d: %s is %s", i, s.ID, s.Scope)
		}
		if strings.HasPrefix(s.ID, "skill-") && s.Description != strings.TrimSpace(desc) {
			t.Errorf("%s: description %q", s.ID, s.Description)
		}
		if s.AlwaysOn != (s.ID == "skill-000") {
			t.Errorf("%s: always-on %v", s.ID, s.AlwaysOn)
		}
	}
	if got[0].ID != "db-migration" || got[1].ID != "skill-100" || got[101].ID != "pdf" || got[102].ID != "skill-000" {
		t.Errorf("order: %s %s %s %s", got[0].ID, got[1].ID, got[101].ID, got[102].ID)
	}
}

func TestListSkillsNoProject(t *testing.T) {
	skillsEnv(t)
	got := listSkills(t, Where{Dir: t.TempDir()})()
	if len(got) != 1 || got[0].ID != "pdf" || got[0].Scope != "global" {
		t.Errorf("got %+v", got)
	}
}

func TestGetSkill(t *testing.T) {
	w, local := skillsEnv(t)
	dir := filepath.Join(local, "db-migration")
	outside := filepath.Join(t.TempDir(), "secret.md")
	writeFiles(t, filepath.Dir(outside), map[string]string{"secret.md": "secret"})
	for link, target := range map[string]string{"leak.md": outside, "alias.md": "references/checklist.md", "broken.md": "missing.md"} {
		if err := os.Symlink(target, filepath.Join(dir, link)); err != nil {
			t.Fatal(err)
		}
	}
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))

	out, isError := call(t, cs, "get_skill", map[string]any{"id": "db-migration"})
	if isError {
		t.Fatal(out)
	}
	for _, want := range []string{"Read references/checklist.md first.", "references/checklist.md", "scripts/run.sh", "alias.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"leak.md", "broken.md", ".notes", "name: db-migration", "not assigned"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("unexpected %q in:\n%s", unwanted, out)
		}
	}
	if want := "Scripts and files referenced by this skill are in " + dir + ". Use absolute paths."; !strings.HasSuffix(strings.TrimSpace(out), want) {
		t.Errorf("does not end with %q:\n%s", want, out)
	}

	// A library skill that is not assigned here is returned with a note.
	out, isError = call(t, cs, "get_skill", map[string]any{"id": "draft"})
	if isError || !strings.Contains(out, "Draft body.") || !strings.Contains(out, "not assigned") {
		t.Errorf("draft: %q, isError %v", out, isError)
	}

	// Unknown IDs fail with suggestions.
	out, isError = call(t, cs, "get_skill", map[string]any{"id": "pdff"})
	if !isError || !strings.Contains(out, `"pdff" not found`) || !strings.Contains(out, "pdf") {
		t.Errorf("unknown: %q, isError %v", out, isError)
	}
}

func TestGetSkillFile(t *testing.T) {
	w, local := skillsEnv(t)
	dir := filepath.Join(local, "db-migration")
	outside := filepath.Join(t.TempDir(), "secret.md")
	writeFiles(t, filepath.Dir(outside), map[string]string{"secret.md": "secret"})
	writeFiles(t, dir, map[string]string{
		"assets/logo.png": "\x89PNG\x00\x01",
		"limit.txt":       strings.Repeat("a", 256*1024),
		"big.txt":         strings.Repeat("a", 256*1024+1),
		"latin1.txt":      "caf\xe9",
	})
	for link, target := range map[string]string{"leak.md": outside, "alias.md": "references/checklist.md", "up": ".."} {
		if err := os.Symlink(target, filepath.Join(dir, link)); err != nil {
			t.Fatal(err)
		}
	}
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	get := func(id, path string) (string, bool) {
		return call(t, cs, "get_skill_file", map[string]any{"id": id, "path": path})
	}

	for _, path := range []string{"references/checklist.md", "alias.md", "references/../references/checklist.md"} {
		if out, isError := get("db-migration", path); isError || out != "1. Back up.\n" {
			t.Errorf("%s: %q, isError %v", path, out, isError)
		}
	}
	if out, isError := get("db-migration", "limit.txt"); isError || len(out) != 256*1024 {
		t.Errorf("limit.txt: %d bytes, isError %v", len(out), isError)
	}
	if out, isError := get("draft", "SKILL.md"); isError || !strings.Contains(out, "Draft body.") {
		t.Errorf("unassigned skill: %q, isError %v", out, isError)
	}

	// Large and binary files come back as a path and a size.
	real := mustReal(t, dir)
	for path, size := range map[string]string{"assets/logo.png": "6 bytes", "big.txt": "262145 bytes", "latin1.txt": "4 bytes"} {
		out, isError := get("db-migration", path)
		if isError || !strings.Contains(out, filepath.Join(real, filepath.FromSlash(path))) || !strings.Contains(out, size) || strings.Contains(out, "aaaa") || strings.Contains(out, "PNG") {
			t.Errorf("%s: %q, isError %v", path, out, isError)
		}
	}

	for path, want := range map[string]string{
		"../pdf/SKILL.md":       "outside the skill folder",
		"/etc/hosts":            "outside the skill folder",
		"leak.md":               "outside the skill folder",
		"up/pdf/SKILL.md":       "outside the skill folder",
		"":                      "outside the skill folder",
		".notes":                "hidden",
		"references":            "not a file",
		"references/missing.md": "not found",
	} {
		if out, isError := get("db-migration", path); !isError || !strings.Contains(out, want) || strings.Contains(out, "secret") {
			t.Errorf("%q: %q, isError %v, want error %q", path, out, isError, want)
		}
	}
	if out, isError := get("nope", "SKILL.md"); !isError || !strings.Contains(out, `"nope" not found`) {
		t.Errorf("unknown skill: %q, isError %v", out, isError)
	}
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	args = append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

// TestGetSkillAfterUpdate checks that get_skill returns what "quiver update"
// pulled during the session.
func TestGetSkillAfterUpdate(t *testing.T) {
	w, _ := skillsEnv(t)
	repo := t.TempDir()
	write := func(body string) {
		writeFiles(t, repo, map[string]string{"skills/review/SKILL.md": "---\nname: review\ndescription: Review code.\n---\n" + body})
		gitIn(t, repo, "add", "-A")
		gitIn(t, repo, "commit", "--quiet", "-m", body)
	}
	gitIn(t, repo, "init", "--quiet")
	write("Old rules.\n")
	st, err := library.StageSource("acme", repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Keep(); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	if out, _ := call(t, cs, "get_skill", map[string]any{"id": "acme/review"}); !strings.Contains(out, "Old rules.") {
		t.Fatalf("before update: %q", out)
	}

	write("New rules.\n")
	if _, err := library.UpdateSource("acme", false); err != nil {
		t.Fatal(err)
	}
	if out, _ := call(t, cs, "get_skill", map[string]any{"id": "acme/review"}); !strings.Contains(out, "New rules.") {
		t.Errorf("after update: %q", out)
	}
}
