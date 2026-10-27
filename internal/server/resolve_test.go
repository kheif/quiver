package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kheif/quiver/internal/project"
)

// projectDir returns a folder registered as a project, with a subfolder.
func projectDir(t *testing.T) (root, sub string, p *project.Project) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv(ProjectEnv, "")
	root = filepath.Join(t.TempDir(), "app")
	sub = filepath.Join(root, "src", "lib")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	p, err := project.Register(root)
	if err != nil {
		t.Fatal(err)
	}
	return root, sub, p
}

func TestResolveFromEnv(t *testing.T) {
	root, _, p := projectDir(t)
	t.Chdir(t.TempDir())
	t.Setenv(ProjectEnv, root)
	w, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if w.ProjectID() != p.ID || w.Dir != root || w.From != ProjectEnv {
		t.Errorf("got %+v", w)
	}
}

func TestResolveSubfolder(t *testing.T) {
	root, sub, p := projectDir(t)
	t.Chdir(sub)
	w, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if w.ProjectID() != p.ID || w.Project.Root != root || w.From != "working directory" {
		t.Errorf("got %+v", w)
	}
	// Dir stays the folder the server started in.
	if real, _ := filepath.EvalSymlinks(w.Dir); real != mustReal(t, sub) {
		t.Errorf("dir %s, want %s", w.Dir, sub)
	}
}

func TestResolveNoProject(t *testing.T) {
	projectDir(t)
	dir := t.TempDir()
	t.Chdir(dir)
	w, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if w.Project != nil || w.ProjectID() != "" || mustReal(t, w.Dir) != mustReal(t, dir) {
		t.Errorf("got %+v", w)
	}
	if _, err := os.Stat(filepath.Join(dir, project.FileName)); err == nil {
		t.Error("resolve created .quiver.json")
	}
}

func TestResolveErrors(t *testing.T) {
	root, _, _ := projectDir(t)
	t.Setenv(ProjectEnv, filepath.Join(root, "missing"))
	if w, err := Resolve(); err == nil || w.Project != nil {
		t.Errorf("missing folder: %+v, %v", w, err)
	}
	if err := os.WriteFile(filepath.Join(root, project.FileName), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(ProjectEnv, root)
	if w, err := Resolve(); err == nil || w.Project != nil {
		t.Errorf("broken .quiver.json: %+v, %v", w, err)
	}
}

func mustReal(t *testing.T, p string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return real
}
