package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/skill"
)

// fakeEditor replaces the editor with fn for the test. fn gets the path
// and may change the file. It returns how often the editor ran.
func fakeEditor(t *testing.T, fn func(path string) error) *int {
	t.Helper()
	calls := new(int)
	orig := runEditor
	runEditor = func(path string) error {
		*calls++
		return fn(path)
	}
	t.Cleanup(func() { runEditor = orig })
	return calls
}

// commits returns the commit subjects in local/, newest first.
func commits(t *testing.T, home string) []string {
	t.Helper()
	out, err := exec.Command("git", "-C", filepath.Join(home, "local"), "log", "--format=%s").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}

func TestNewScopes(t *testing.T) {
	f := listEnv(t)
	calls := fakeEditor(t, func(string) error { t.Error("editor opened"); return nil })
	dir := filepath.Join(t.TempDir(), "web")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"new", "g", "--description", "Global one.", "--global"}, "Assigned globally: g"},
		{[]string{"new", "p", "--description", "Project one.", "--project", f.app}, "Assigned to app: p"},
		{[]string{"new", "w", "--description", "New project.", "--project", dir}, "Created " + filepath.Join(dir, project.FileName)},
		{[]string{"new", "n", "--description", "No scope.", "--none"}, "Not assigned. Assign it with: quiver add n"},
	} {
		out, err := runIn(t, "", c.args...)
		if err != nil {
			t.Fatalf("%v: %v", c.args, err)
		}
		if !strings.Contains(out, c.want) || !strings.Contains(out, "Created skill "+c.args[1]) {
			t.Errorf("%v: output missing %q:\n%s", c.args, c.want, out)
		}
	}
	if *calls != 0 {
		t.Errorf("editor ran %d times", *calls)
	}
	r := loadRegistry(t)
	if !reflect.DeepEqual(r.Global, []string{"acme/review", "g", "pdf"}) {
		t.Errorf("global = %v", r.Global)
	}
	if got := r.Projects["app-000001"].Skills; !reflect.DeepEqual(got, []string{"acme/review", "p", "tdd"}) {
		t.Errorf("app = %v", got)
	}
	p, err := project.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Projects[p.ID].Skills; !reflect.DeepEqual(got, []string{"w"}) {
		t.Errorf("web = %v", got)
	}
	if got := commits(t, f.home)[:4]; !reflect.DeepEqual(got, []string{"add n", "add w", "add p", "add g"}) {
		t.Errorf("commits = %v", got)
	}
	s, err := skill.Load(filepath.Join(f.home, "local", "g"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "\n# G\n\nInstructions go here.\n"; s.Body != want {
		t.Errorf("body = %q", s.Body)
	}
}

func TestNewCurrentFolder(t *testing.T) {
	f := listEnv(t)
	t.Chdir(f.api)
	if _, err := runIn(t, "", "new", "api-docs", "--description", "Write API docs."); err != nil {
		t.Fatal(err)
	}
	if got := loadRegistry(t).Projects["api-000002"].Skills; !reflect.DeepEqual(got, []string{"api-docs", "tdd"}) {
		t.Errorf("api = %v", got)
	}
}

func TestNewNameClash(t *testing.T) {
	f := listEnv(t)
	// The global acme/review is named review.
	out, err := runIn(t, "", "new", "review", "--description", "My review.", "--project", f.app)
	if err != nil {
		t.Fatal(err)
	}
	if want := `warning: skills acme/review, review share the name "review"`; !strings.Contains(out, want) {
		t.Errorf("output missing %q:\n%s", want, out)
	}
}

func TestNewRejects(t *testing.T) {
	f := listEnv(t)
	other := filepath.Join(t.TempDir(), "other")
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	for args, want := range map[string]string{
		"new pdf --description x --none":                    "skill pdf already exists; change it with: quiver edit pdf",
		"new ../evil --description x --none":                `invalid name "../evil"`,
		"new Bad --description x --none":                    `invalid name "Bad"`,
		"new x --description x --global --none":             "none",
		"new pdf --description x --project " + other:        "already exists",
		"new y --description x --project " + other + "/nop": "nop",
	} {
		if _, err := runIn(t, "", strings.Fields(args)...); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v, want %q", args, err, want)
		}
	}
	// A taken name does not register the folder.
	if _, err := project.Find(other); err == nil {
		t.Error("folder registered for a failed new")
	}
	if got := commits(t, f.home)[0]; got != "Initialize Quiver skills" {
		t.Errorf("last commit = %q", got)
	}
}

func TestNewOpensEditor(t *testing.T) {
	f := listEnv(t)
	fakeEditor(t, func(path string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.Contains(string(data), "description: TODO — one sentence") {
			t.Errorf("template:\n%s", data)
		}
		edited := strings.Replace(string(data), "TODO — one sentence on when the assistant should use this skill.", "Write release notes.", 1)
		return os.WriteFile(path, []byte(edited), 0o644)
	})
	out, err := runIn(t, "", "new", "release-notes", "--none")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Saved release-notes.") || strings.Contains(out, "TODO") {
		t.Errorf("output:\n%s", out)
	}
	if got := commits(t, f.home)[:2]; !reflect.DeepEqual(got, []string{"edit release-notes", "add release-notes"}) {
		t.Errorf("commits = %v", got)
	}
	s, _ := skill.Load(filepath.Join(f.home, "local", "release-notes"))
	if s.Description != "Write release notes." || !strings.Contains(s.Body, "# Release notes") {
		t.Errorf("skill = %q\n%s", s.Description, s.Body)
	}
}

func TestNewEditorUnchanged(t *testing.T) {
	listEnv(t)
	fakeEditor(t, func(string) error { return nil })
	out, err := runIn(t, "", "new", "later", "--none")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"No changes.", "warning: the description of later still starts with TODO"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestEdit(t *testing.T) {
	f := listEnv(t)
	fakeEditor(t, func(path string) error {
		return os.WriteFile(path, []byte("---\nname: pdf\ndescription: Fill PDF forms.\n---\n"), 0o644)
	})
	out, err := runIn(t, "", "edit", "pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Saved pdf.") {
		t.Errorf("output:\n%s", out)
	}
	if got := commits(t, f.home)[0]; got != "edit pdf" {
		t.Errorf("last commit = %q", got)
	}
}

func TestEditInvalid(t *testing.T) {
	f := listEnv(t)
	fakeEditor(t, func(path string) error {
		return os.WriteFile(path, []byte("---\nname: pdf\n---\n"), 0o644)
	})
	_, err := runIn(t, "", "edit", "pdf")
	if err == nil || !strings.Contains(err.Error(), "description is missing") || !strings.Contains(err.Error(), "not committed; fix it with: quiver edit pdf") {
		t.Errorf("err = %v", err)
	}
	// The broken skill is left out of the library, but edit still opens it.
	fixed := fakeEditor(t, func(path string) error {
		return os.WriteFile(path, []byte("---\nname: pdf\ndescription: Fixed.\n---\n"), 0o644)
	})
	if _, err := runIn(t, "", "edit", "pdf"); err != nil || *fixed != 1 {
		t.Fatalf("err = %v, editor ran %d times", err, *fixed)
	}
	if got := commits(t, f.home)[0]; got != "edit pdf" {
		t.Errorf("last commit = %q", got)
	}
}

func TestEditRejects(t *testing.T) {
	listEnv(t)
	calls := fakeEditor(t, func(string) error { return nil })
	for id, want := range map[string]string{
		"acme/review": "read-only: acme/review comes from source acme; only local skills can be edited; to change it, make a local copy with: quiver fork acme/review",
		"nope":        `skill "nope" not found`,
		"../q":        `skill "../q" not found`,
		"broken/..":   `skill "broken/.." not found`,
	} {
		if _, err := runIn(t, "", "edit", id); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v, want %q", id, err, want)
		}
	}
	if *calls != 0 {
		t.Errorf("editor ran %d times", *calls)
	}
}
