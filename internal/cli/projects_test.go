package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/project"
)

func TestWhere(t *testing.T) {
	f := listEnv(t)
	app, api := "project app ("+f.app+")\n", "project api ("+f.api+")\n"
	for _, c := range []struct{ id, want string }{
		{"tdd", api + app},
		{"acme/review", "global\n" + app},
		{"pdf", "global\n"},
		{"notes", "notes is not assigned anywhere.\n"},
	} {
		if got, _ := runIn(t, "", "where", c.id); got != c.want {
			t.Errorf("where %s:\ngot  %q\nwant %q", c.id, got, c.want)
		}
	}
	if _, err := runIn(t, "", "where", "nope"); err == nil || !strings.Contains(err.Error(), `skill "nope" not found`) {
		t.Errorf("unknown skill: %v", err)
	}
	// An assigned skill that left the library is still shown.
	if err := os.RemoveAll(filepath.Join(f.home, "local", "tdd")); err != nil {
		t.Fatal(err)
	}
	if got := f.run(t, "where", "tdd"); !strings.HasSuffix(got, "warning: tdd is assigned but not in the library, so it is not served\n") {
		t.Errorf("missing skill:\n%s", got)
	}
}

func TestProjectsGolden(t *testing.T) {
	f := listEnv(t)
	if err := os.RemoveAll(f.api); err != nil {
		t.Fatal(err)
	}
	golden(t, "projects", f.run(t, "projects"))
}

func TestProjectsEmpty(t *testing.T) {
	sourceEnv(t)
	if out, err := runIn(t, "", "projects"); err != nil || !strings.HasPrefix(out, "No projects.") {
		t.Errorf("got %q, %v", out, err)
	}
}

func TestProjectsRename(t *testing.T) {
	f := listEnv(t)
	// By name, then by ID.
	if got := f.run(t, "projects", "rename", "app", "Web App"); got != "Renamed project app-000001 to Web App.\n" {
		t.Errorf("got %q", got)
	}
	f.run(t, "projects", "rename", "api-000002", "Web App")
	if got := loadRegistry(t).Projects["api-000002"].Name; got != "Web App" {
		t.Errorf("name = %q", got)
	}
	// Both are named Web App now.
	if _, err := runIn(t, "", "projects", "rename", "Web App", "x"); err == nil || !strings.Contains(err.Error(), "use the ID") {
		t.Errorf("shared name: %v", err)
	}
	if _, err := runIn(t, "", "projects", "rename", "nope", "x"); err == nil {
		t.Error("unknown project: want error")
	}
}

func TestProjectsForget(t *testing.T) {
	f := listEnv(t)
	out, err := runIn(t, "n\n", "projects", "forget", "app")
	if err != nil || !strings.Contains(out, "and its 2 skill assignments? [y/N] Project not forgotten.") {
		t.Errorf("no: %q, %v", out, err)
	}
	if loadRegistry(t).Projects["app-000001"] == nil {
		t.Fatal("forgot after no")
	}
	if out, err := runIn(t, "y\n", "projects", "forget", "app"); err != nil || !strings.HasSuffix(out, "Forgot project app-000001. Its .quiver.json was not changed.\n") {
		t.Errorf("yes: %q, %v", out, err)
	}
	if loadRegistry(t).Projects["app-000001"] != nil {
		t.Error("app still registered")
	}
	if p, err := project.Find(f.app); err != nil || p.ID != "app-000001" {
		t.Errorf(".quiver.json: %+v, %v", p, err)
	}
	f.run(t, "projects", "forget", "--yes", "api-000002")
	if n := len(loadRegistry(t).Projects); n != 0 {
		t.Errorf("%d projects left", n)
	}
}

func TestProjectsSplit(t *testing.T) {
	f := listEnv(t)
	cp := projectDir(t, "app-copy", "app-000001")
	t.Chdir(cp)
	out := f.run(t, "projects", "split")
	p, err := project.Find(cp)
	if err != nil || p.ID == "app-000001" || !strings.HasPrefix(p.ID, "app-copy-") {
		t.Fatalf("after split: %+v, %v", p, err)
	}
	if want := " is now project " + p.ID + ", with the skills of app-000001."; !strings.Contains(out, want) {
		t.Errorf("output %q", out)
	}
	r := loadRegistry(t)
	if got := r.Projects[p.ID].Skills; !slices.Equal(got, []string{"acme/review", "tdd"}) {
		t.Errorf("copy skills = %v", got)
	}
	// Changing the copy does not change the original.
	f.run(t, "add", "notes")
	f.run(t, "remove", "tdd")
	r = loadRegistry(t)
	if got := r.Projects[p.ID].Skills; !slices.Equal(got, []string{"acme/review", "notes"}) {
		t.Errorf("copy skills = %v", got)
	}
	if got := r.Projects["app-000001"].Skills; !slices.Equal(got, []string{"acme/review", "tdd"}) {
		t.Errorf("original skills = %v", got)
	}
	// The original has nothing left to split from.
	t.Chdir(f.app)
	if _, err := runIn(t, "", "projects", "split"); err == nil || !strings.Contains(err.Error(), "nothing to split") {
		t.Errorf("split the original: %v", err)
	}
}
