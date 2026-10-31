package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/skill"
)

func TestFork(t *testing.T) {
	home := sourceEnv(t)
	bare := bareSource(t, "acme", "pdf", "review")
	app := projectDir(t, "app", "app-000001")
	for _, args := range [][]string{
		{"add", "acme/pdf", "--global", "--always"},
		{"add", "acme/pdf", "acme/review", "--project", app},
	} {
		if out, err := runIn(t, "", args...); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	head, err := exec.Command("git", "-C", filepath.Join(home, "sources", "acme"), "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	commit := strings.TrimSpace(string(head))

	out, err := runIn(t, "", "fork", "acme/pdf")
	if err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(home, "local")
	for _, want := range []string{
		"Forked acme/pdf into pdf (" + filepath.Join(local, "pdf") + ").",
		"Replaced acme/pdf with pdf globally.",
		"Replaced acme/pdf with pdf in project app (" + app + ").",
		"Change it with quiver edit pdf.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	r := loadRegistry(t)
	p := r.Projects["app-000001"]
	if !reflect.DeepEqual(r.Global, []string{"pdf"}) || !reflect.DeepEqual(r.AlwaysOn, []string{"pdf"}) ||
		!reflect.DeepEqual(p.Skills, []string{"acme/review", "pdf"}) {
		t.Errorf("registry: global %v, always-on %v, project %v", r.Global, r.AlwaysOn, p.Skills)
	}
	s, err := skill.Load(filepath.Join(local, "pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Frontmatter[library.ForkedFromField]; got != "acme/pdf@"+commit {
		t.Errorf("forkedFrom = %v", got)
	}
	if got := commits(t, home)[0]; got != "fork pdf from acme/pdf" {
		t.Errorf("commit = %q", got)
	}

	// The copy is what projects get: edits show, upstream changes do not.
	edited := strings.Replace(string(readTestFile(t, filepath.Join(local, "pdf", "SKILL.md"))), "description: Test skill.", "description: Edited.", 1)
	if err := os.WriteFile(filepath.Join(local, "pdf", "SKILL.md"), []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	push(t, bare, map[string]string{"skills/pdf/SKILL.md": "---\nname: pdf\ndescription: Upstream.\n---\n"}, false)
	if out, err := runIn(t, "", "update"); err != nil || strings.Contains(out, "warning") {
		t.Fatalf("update: %v\n%s", err, out)
	}
	l, err := library.Load()
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{"pdf": "Edited.", "acme/pdf": "Upstream."} {
		if s, err := l.Get(id); err != nil || s.Description != want {
			t.Errorf("%s: %+v, %v", id, s, err)
		}
	}
	if out, err := runIn(t, "", "where", "acme/pdf"); err != nil || out != "acme/pdf is not assigned anywhere.\n" {
		t.Errorf("where: %v\n%s", err, out)
	}
}

func TestForkAs(t *testing.T) {
	home := sourceEnv(t)
	bareSource(t, "acme", "review")
	out, err := runIn(t, "", "fork", "acme/review", "--as", "team-review")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "acme/review was not assigned anywhere; assign the copy with quiver add team-review.") {
		t.Errorf("output:\n%s", out)
	}
	s, err := skill.Load(filepath.Join(home, "local", "team-review"))
	if err != nil || s.Name != "team-review" {
		t.Errorf("copy: %+v, %v", s, err)
	}
	if !strings.HasPrefix(s.Frontmatter[library.ForkedFromField].(string), "acme/review@") {
		t.Errorf("forkedFrom = %v", s.Frontmatter[library.ForkedFromField])
	}
}

func TestForkErrors(t *testing.T) {
	sourceEnv(t)
	bareSource(t, "acme", "review")
	if _, err := runIn(t, "", "fork", "acme/review"); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"fork", "acme/review"}, "choose another name with --as"},
		{[]string{"fork", "review"}, "review is already a local skill"},
		{[]string{"fork", "acme/nope"}, "not found"},
		{[]string{"fork", "acme/review", "--as", "Bad Name"}, "choose a name with --as"},
	} {
		if _, err := runIn(t, "", c.args...); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%v: got %v, want %q", c.args, err, c.want)
		}
	}
}

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
