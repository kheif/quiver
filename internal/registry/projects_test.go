package registry

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/project"
)

// folder makes a folder under a fake $HOME, with a .quiver.json holding id
// if id is not empty.
func folder(t *testing.T, home, name, id string) string {
	t.Helper()
	dir := filepath.Join(home, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if id != "" {
		if err := project.SetID(dir, id); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func fakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestEnsureProjectNew(t *testing.T) {
	home := fakeHome(t)
	dir := folder(t, home, "app", "app-1")
	r := Default()
	if w := r.EnsureProject("app-1", dir); w != nil {
		t.Fatal(w)
	}
	p := r.Projects["app-1"]
	if p.Name != "app" || !reflect.DeepEqual(p.Paths, []string{dir}) || p.Skills == nil {
		t.Errorf("got %+v", p)
	}
	if w := r.EnsureProject("app-1", dir); w != nil || len(r.Projects["app-1"].Paths) != 1 {
		t.Errorf("second call: %v, %+v", w, r.Projects["app-1"])
	}
}

func TestEnsureProjectMoved(t *testing.T) {
	home := fakeHome(t)
	r := Default()
	_ = r.EnsureProject("app-1", filepath.Join(home, "gone"))
	dir := folder(t, home, "new-place", "app-1")
	if w := r.EnsureProject("app-1", dir); w != nil {
		t.Fatal(w)
	}
	if got := r.Projects["app-1"].Paths; !reflect.DeepEqual(got, []string{dir}) {
		t.Errorf("paths %v", got)
	}
}

func TestEnsureProjectOldPathHasOtherID(t *testing.T) {
	home := fakeHome(t)
	old := folder(t, home, "old", "other-2")
	r := Default()
	_ = r.EnsureProject("app-1", old)
	dir := folder(t, home, "new", "app-1")
	if w := r.EnsureProject("app-1", dir); w != nil {
		t.Fatal(w)
	}
}

func TestEnsureProjectCopied(t *testing.T) {
	home := fakeHome(t)
	a := folder(t, home, "a", "app-1")
	b := folder(t, home, "b", "app-1")
	r := Default()
	_ = r.EnsureProject("app-1", a)
	w := r.EnsureProject("app-1", b)
	if w == nil || !reflect.DeepEqual(w.Paths, []string{a, b}) || !strings.Contains(w.Error(), "projects split") {
		t.Errorf("got %v", w)
	}
	if got := r.Projects["app-1"].Paths; len(got) != 2 {
		t.Errorf("paths %v", got)
	}
}

func TestEnsureProjectSymlinkIsSameFolder(t *testing.T) {
	home := fakeHome(t)
	dir := folder(t, home, "real", "app-1")
	link := filepath.Join(home, "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Skip(err)
	}
	r := Default()
	_ = r.EnsureProject("app-1", dir)
	if w := r.EnsureProject("app-1", link); w != nil {
		t.Errorf("got %v", w)
	}
}

func TestRenameAndForget(t *testing.T) {
	r := withProject()
	if err := r.RenameProject("app", "My App"); err != nil || r.Projects["app"].Name != "My App" {
		t.Errorf("rename: %v", err)
	}
	if err := r.RenameProject("app", " "); err == nil {
		t.Error("empty name: want error")
	}
	if err := r.RenameProject("nope", "x"); err == nil {
		t.Error("unknown: want error")
	}
	if err := r.ForgetProject("app"); err != nil || r.Projects["app"] != nil {
		t.Errorf("forget: %v", err)
	}
	if err := r.ForgetProject("app"); err == nil {
		t.Error("forget twice: want error")
	}
}

func TestLookup(t *testing.T) {
	r := Default()
	r.Projects["app-1"] = &Project{Name: "app"}
	r.Projects["web-1"] = &Project{Name: "web"}
	r.Projects["web-2"] = &Project{Name: "web"}
	r.Projects["x-1"] = &Project{Name: "app-1"}
	for ref, want := range map[string]string{"app-1": "app-1", "app": "app-1", "web-2": "web-2"} {
		if got, err := r.Lookup(ref); err != nil || got != want {
			t.Errorf("Lookup(%q) = %q, %v; want %q", ref, got, err, want)
		}
	}
	if _, err := r.Lookup("web"); err == nil || !strings.Contains(err.Error(), "web-1, web-2") {
		t.Errorf("shared name: %v", err)
	}
	if _, err := r.Lookup("nope"); err == nil {
		t.Error("unknown: want error")
	}
}

func TestSplitProject(t *testing.T) {
	home := fakeHome(t)
	setHome(t)
	orig := folder(t, home, "app", "app-1")
	cp := folder(t, home, "app-copy", "app-1")
	err := Update(func(r *Registry) error {
		_ = r.EnsureProject("app-1", orig)
		r.Projects["app-1"].Skills = []string{"pdf", "tdd"}
		r.Projects["app-1"].AlwaysOn = []string{"pdf"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	oldID, p, err := SplitProject(cp)
	if err != nil {
		t.Fatal(err)
	}
	if oldID != "app-1" || !strings.HasPrefix(p.ID, "app-copy-") || p.Root != cp {
		t.Errorf("split = %s, %+v", oldID, p)
	}
	if found, err := project.Find(cp); err != nil || found.ID != p.ID {
		t.Errorf(".quiver.json: %+v, %v", found, err)
	}
	r, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := &Project{Name: "app-copy", Paths: []string{cp}, Skills: []string{"pdf", "tdd"}, AlwaysOn: []string{"pdf"}}
	if got := r.Projects[p.ID]; !reflect.DeepEqual(got, want) {
		t.Errorf("new project = %+v", got)
	}
	if got := r.Projects["app-1"].Paths; !reflect.DeepEqual(got, []string{orig}) {
		t.Errorf("old paths = %v", got)
	}
	// The original folder is now the only one with app-1.
	if _, _, err := SplitProject(orig); !errors.Is(err, ErrNotCopied) {
		t.Errorf("split the original: %v", err)
	}
}

func TestSplitProjectUnknown(t *testing.T) {
	home := fakeHome(t)
	setHome(t)
	dir := folder(t, home, "app", "app-1")
	if _, _, err := SplitProject(dir); err == nil || !strings.Contains(err.Error(), "not in the registry") {
		t.Errorf("got %v", err)
	}
}

// gitRun runs git in dir with a fixed identity.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=Test", "-c", "user.email=test@example.com"}, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// Worktrees of one repository share the committed .quiver.json: they are
// one project, not copies.
func TestEnsureProjectWorktrees(t *testing.T) {
	home := fakeHome(t)
	setHome(t)
	main := folder(t, home, "app", "app-1")
	gitRun(t, main, "init", "--quiet")
	gitRun(t, main, "add", ".quiver.json")
	gitRun(t, main, "commit", "--quiet", "-m", "init")
	wt := filepath.Join(home, "app-feature")
	gitRun(t, main, "worktree", "add", "--quiet", wt)
	err := Update(func(r *Registry) error {
		_ = r.EnsureProject("app-1", main)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	r, _ := Load()
	if w := r.EnsureProject("app-1", wt); w != nil {
		t.Errorf("worktree reported as a copy: %v", w)
	}
	if got := r.Projects["app-1"].Paths; len(got) != 2 {
		t.Errorf("paths = %v", got)
	}
	if err := Write(r); err != nil {
		t.Fatal(err)
	}
	if _, _, err := SplitProject(wt); !errors.Is(err, ErrWorktree) {
		t.Errorf("split in a worktree: %v", err)
	}
	// A plain copy of the folder is still a copy.
	cp := folder(t, home, "app-copy", "app-1")
	if w := r.EnsureProject("app-1", cp); w == nil {
		t.Error("copy not reported")
	}
}
