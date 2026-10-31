package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/registry"
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

// A registry.json synced from another machine names projects this machine
// has not seen, so they have no paths in its state.
func TestProjectsNotSeenOnThisMachine(t *testing.T) {
	f := listEnv(t)
	if err := os.Remove(filepath.Join(f.home, registry.StateFile)); err != nil {
		t.Fatal(err)
	}
	if got := f.run(t, "projects"); !strings.Contains(got, "app   app-000001  2       (not seen on this machine)") {
		t.Errorf("projects:\n%s", got)
	}
	if got := f.run(t, "where", "tdd"); got != "project api (not seen on this machine)\nproject app (not seen on this machine)\n" {
		t.Errorf("where:\n%s", got)
	}
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

func TestProjectsJSON(t *testing.T) {
	f := listEnv(t)
	if err := os.RemoveAll(f.api); err != nil {
		t.Fatal(err)
	}
	out, err := runIn(t, "", "projects", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got struct{ Projects []projectJSON }
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	want := []projectJSON{
		{ID: "api-000002", Name: "api", Skills: []string{"tdd"}, Paths: []string{f.api}, Missing: []string{f.api}},
		{ID: "app-000001", Name: "app", Skills: []string{"acme/review", "tdd"}, Paths: []string{f.app}, Missing: []string{}},
	}
	if !reflect.DeepEqual(got.Projects, want) {
		t.Errorf("got  %+v\nwant %+v", got.Projects, want)
	}

	sourceEnv(t)
	if out, err := runIn(t, "", "projects", "--json"); err != nil || out != "{\n  \"projects\": []\n}\n" {
		t.Errorf("empty: %v %q", err, out)
	}
}

func TestWhereJSON(t *testing.T) {
	f := listEnv(t)
	for _, c := range []struct {
		id   string
		want whereJSON
	}{
		{"acme/review", whereJSON{ID: "acme/review", InLibrary: true, Global: true, Projects: []projectJSON{
			{ID: "app-000001", Name: "app", Skills: []string{"acme/review", "tdd"}, Paths: []string{f.app}, Missing: []string{}},
		}}},
		{"notes", whereJSON{ID: "notes", InLibrary: true, Projects: []projectJSON{}}},
	} {
		out, err := runIn(t, "", "where", c.id, "--json")
		if err != nil {
			t.Fatal(err)
		}
		var got whereJSON
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("where %s:\ngot  %+v\nwant %+v", c.id, got, c.want)
		}
	}
	if _, err := runIn(t, "", "where", "nope", "--json"); err == nil {
		t.Error("unknown skill: no error")
	}
}
