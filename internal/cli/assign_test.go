package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/setup"
)

// assignEnv sets up a Quiver home with local skills. Each entry is
// "<folder>" or "<folder>:<name>" and returns a project folder that is not
// registered yet.
func assignEnv(t *testing.T, skills ...string) string {
	t.Helper()
	home := sourceEnv(t)
	if err := setup.EnsureInit(); err != nil {
		t.Fatal(err)
	}
	for _, s := range skills {
		folder, name, ok := strings.Cut(s, ":")
		if !ok {
			name = folder
		}
		dir := filepath.Join(home, "local", folder)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		content := "---\nname: " + name + "\ndescription: Test skill.\n---\n"
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dir := filepath.Join(t.TempDir(), "app")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func loadRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	r, err := registry.Load()
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestAddGlobal(t *testing.T) {
	assignEnv(t, "pdf", "tdd")
	out, err := runIn(t, "", "add", "pdf", "tdd", "--global")
	if err != nil {
		t.Fatal(err)
	}
	if r := loadRegistry(t); !reflect.DeepEqual(r.Global, []string{"pdf", "tdd"}) {
		t.Errorf("global = %v", r.Global)
	}
	for _, want := range []string{"Assigned globally: pdf, tdd", "Served in every folder:", "pdf  global"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "warning") {
		t.Errorf("unexpected warning:\n%s", out)
	}
}

func TestAddNameClash(t *testing.T) {
	dir := assignEnv(t, "pdf", "pdf-old:pdf", "tdd")
	if _, err := runIn(t, "", "add", "pdf", "--global"); err != nil {
		t.Fatal(err)
	}
	out, err := runIn(t, "", "add", "pdf-old", "tdd", "--project", dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := `warning: skills pdf, pdf-old share the name "pdf"`; !strings.Contains(out, want) {
		t.Errorf("output missing %q:\n%s", want, out)
	}
	if strings.Count(out, "warning") != 1 {
		t.Errorf("want one warning:\n%s", out)
	}
}

func TestAddProjectRegisters(t *testing.T) {
	dir := assignEnv(t, "pdf", "tdd")
	if _, err := runIn(t, "", "add", "tdd", "--global"); err != nil {
		t.Fatal(err)
	}
	out, err := runIn(t, "", "add", "pdf", "--project", dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := project.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := loadRegistry(t)
	if got := r.Projects[p.ID]; got == nil || !reflect.DeepEqual(got.Skills, []string{"pdf"}) {
		t.Errorf("project = %+v", got)
	}
	for _, want := range []string{"Created " + filepath.Join(dir, project.FileName), "Assigned to app: pdf", "pdf  project", "tdd  global"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	// A second add reuses the ID and does not announce the file again.
	out, err = runIn(t, "", "add", "tdd", "--project", dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Created") {
		t.Errorf("file announced again:\n%s", out)
	}
	if again, _ := project.Find(dir); again.ID != p.ID {
		t.Errorf("ID changed from %s to %s", p.ID, again.ID)
	}
}

func TestAddCurrentFolder(t *testing.T) {
	dir := assignEnv(t, "pdf")
	sub := filepath.Join(dir, "src")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := project.Register(dir); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if _, err := runIn(t, "", "add", "pdf"); err != nil {
		t.Fatal(err)
	}
	p, _ := project.Find(dir)
	if got := loadRegistry(t).Projects[p.ID]; got == nil || !reflect.DeepEqual(got.Skills, []string{"pdf"}) {
		t.Errorf("project = %+v", got)
	}
	if _, err := os.Stat(filepath.Join(sub, project.FileName)); err == nil {
		t.Error("registered the subfolder")
	}
}

func TestAddUnknownChangesNothing(t *testing.T) {
	dir := assignEnv(t, "pdf")
	_, err := runIn(t, "", "add", "pdf", "pfd2", "nope", "--project", dir)
	if !errors.Is(err, library.ErrSkillNotFound) || !strings.Contains(err.Error(), `"nope"`) || !strings.Contains(err.Error(), `"pfd2"`) {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, project.FileName)); err == nil {
		t.Error(".quiver.json was created")
	}
	if r := loadRegistry(t); len(r.Projects) != 0 || len(r.Global) != 0 {
		t.Errorf("registry changed: %+v", r)
	}
}

func TestAddBadTarget(t *testing.T) {
	dir := assignEnv(t, "pdf")
	if _, err := runIn(t, "", "add", "pdf", "--project", filepath.Join(dir, "missing")); err == nil {
		t.Error("missing folder: want error")
	}
	if _, err := runIn(t, "", "add", "pdf", "--global", "--project", dir); err == nil {
		t.Error("--global with --project: want error")
	}
}
