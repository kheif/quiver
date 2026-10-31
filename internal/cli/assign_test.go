package cli

import (
	"errors"
	"io"
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
	t.Chdir(sub)
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

func TestRemoveProject(t *testing.T) {
	dir := assignEnv(t, "pdf", "tdd")
	home := os.Getenv("QUIVER_HOME")
	if _, err := runIn(t, "", "add", "pdf", "tdd", "--project", dir); err != nil {
		t.Fatal(err)
	}
	out, err := runIn(t, "", "remove", "pdf", "--project", dir)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := project.Find(dir)
	if got := loadRegistry(t).Projects[p.ID].Skills; !reflect.DeepEqual(got, []string{"tdd"}) {
		t.Errorf("skills = %v", got)
	}
	if !strings.Contains(out, "Removed from app: pdf") || !strings.Contains(out, "tdd  project") {
		t.Errorf("output:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(home, "local", "pdf", "SKILL.md")); err != nil {
		t.Error("skill left the library:", err)
	}
}

func TestRemoveGlobal(t *testing.T) {
	assignEnv(t, "pdf", "tdd")
	if _, err := runIn(t, "", "add", "pdf", "tdd", "--global"); err != nil {
		t.Fatal(err)
	}
	out, err := runIn(t, "", "remove", "tdd", "nope", "--global")
	if err != nil {
		t.Fatal(err)
	}
	if got := loadRegistry(t).Global; !reflect.DeepEqual(got, []string{"pdf"}) {
		t.Errorf("global = %v", got)
	}
	for _, want := range []string{"note: nope was not global", "Removed from every folder: tdd"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRemoveNotAssigned(t *testing.T) {
	dir := assignEnv(t, "pdf", "tdd")
	if _, err := runIn(t, "", "add", "tdd", "--global"); err != nil {
		t.Fatal(err)
	}
	if _, err := project.Register(dir); err != nil {
		t.Fatal(err)
	}
	// The project is not in the registry yet, and nope is not in the library.
	out, err := runIn(t, "", "remove", "pdf", "tdd", "nope", "--project", dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"note: pdf was not assigned to app", "note: tdd is global; remove it with --global", "note: nope was not assigned"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Removed") {
		t.Errorf("claims a removal:\n%s", out)
	}
}

func TestRemoveDeletedSkill(t *testing.T) {
	dir := assignEnv(t, "pdf")
	home := os.Getenv("QUIVER_HOME")
	if _, err := runIn(t, "", "add", "pdf", "--project", dir); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(home, "local", "pdf")); err != nil {
		t.Fatal(err)
	}
	if _, err := runIn(t, "", "remove", "pdf", "--project", dir); err != nil {
		t.Fatal(err)
	}
	p, _ := project.Find(dir)
	if got := loadRegistry(t).Projects[p.ID].Skills; len(got) != 0 {
		t.Errorf("skills = %v", got)
	}
}

func TestRemoveUnregistered(t *testing.T) {
	dir := assignEnv(t, "pdf")
	_, err := runIn(t, "", "remove", "pdf", "--project", dir)
	if err == nil || !strings.Contains(err.Error(), "not a Quiver project") {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, project.FileName)); err == nil {
		t.Error(".quiver.json was created")
	}
}

func TestAddAlwaysGlobal(t *testing.T) {
	assignEnv(t, "caveman", "pdf")
	out, err := runIn(t, "", "add", "caveman", "--global", "--always")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runIn(t, "", "add", "pdf", "--global"); err != nil {
		t.Fatal(err)
	}
	r := loadRegistry(t)
	if !reflect.DeepEqual(r.Global, []string{"caveman", "pdf"}) || !reflect.DeepEqual(r.AlwaysOn, []string{"caveman"}) {
		t.Errorf("global = %v, alwaysOn = %v", r.Global, r.AlwaysOn)
	}
	for _, want := range []string{"Assigned as always-on globally: caveman", "caveman  global  always-on"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestAddAlwaysProject(t *testing.T) {
	dir := assignEnv(t, "pdf", "tdd")
	out, err := runIn(t, "", "add", "tdd", "pdf", "--project", dir, "--always")
	if err != nil {
		t.Fatal(err)
	}
	p, _ := project.Find(dir)
	rp := loadRegistry(t).Projects[p.ID]
	if !reflect.DeepEqual(rp.Skills, []string{"pdf", "tdd"}) || !reflect.DeepEqual(rp.AlwaysOn, []string{"pdf", "tdd"}) {
		t.Errorf("project = %+v", rp)
	}
	if !strings.Contains(out, "Assigned as always-on to app: tdd, pdf") || !strings.Contains(out, "tdd  project  always-on") {
		t.Errorf("output:\n%s", out)
	}
}

func TestRemoveAlways(t *testing.T) {
	dir := assignEnv(t, "caveman", "pdf")
	if _, err := runIn(t, "", "add", "caveman", "--global", "--always"); err != nil {
		t.Fatal(err)
	}
	if _, err := runIn(t, "", "add", "pdf", "--project", dir, "--always"); err != nil {
		t.Fatal(err)
	}
	out, err := runIn(t, "", "remove", "pdf", "caveman", "--project", dir, "--always")
	if err != nil {
		t.Fatal(err)
	}
	p, _ := project.Find(dir)
	r := loadRegistry(t)
	if rp := r.Projects[p.ID]; !reflect.DeepEqual(rp.Skills, []string{"pdf"}) || len(rp.AlwaysOn) != 0 {
		t.Errorf("project = %+v", rp)
	}
	for _, want := range []string{"note: caveman is always-on globally; remove the mark with --global", "No longer always-on in app: pdf", "pdf      project"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	out, err = runIn(t, "", "remove", "caveman", "pdf", "--global", "--always")
	if err != nil {
		t.Fatal(err)
	}
	r = loadRegistry(t)
	if !reflect.DeepEqual(r.Global, []string{"caveman"}) || len(r.AlwaysOn) != 0 {
		t.Errorf("global = %v, alwaysOn = %v", r.Global, r.AlwaysOn)
	}
	for _, want := range []string{"note: pdf was not always-on globally", "No longer always-on in every folder: caveman"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

// fakeTerminal makes commands treat their input as a person at a terminal.
func fakeTerminal(t *testing.T) {
	t.Helper()
	old := interactive
	interactive = func(io.Reader) bool { return true }
	t.Cleanup(func() { interactive = old })
}

func TestAddConfirmsNewProject(t *testing.T) {
	dir := assignEnv(t, "pdf")
	fakeTerminal(t)
	question := "Register " + dir + " as a project? [Y/n] "
	out, err := runIn(t, "n\n", "add", "pdf", "--project", dir)
	if err != nil {
		t.Fatal(err)
	}
	if out != question+"Not registered.\n" {
		t.Errorf("no: %q", out)
	}
	if _, err := os.Stat(filepath.Join(dir, project.FileName)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf(".quiver.json written: %v", err)
	}
	if out, err = runIn(t, "y\n", "add", "pdf", "--project", dir); err != nil || !strings.HasPrefix(out, question+"Created ") {
		t.Errorf("yes: %v\n%s", err, out)
	}
	// A registered project and --yes are not asked about.
	if out, _ = runIn(t, "", "add", "pdf", "--project", dir); strings.Contains(out, "Register") {
		t.Errorf("asked again:\n%s", out)
	}
	other := filepath.Join(t.TempDir(), "other")
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err = runIn(t, "", "add", "pdf", "--project", other, "--yes"); err != nil || strings.Contains(out, "Register") || !strings.Contains(out, "Created ") {
		t.Errorf("--yes: %v\n%s", err, out)
	}
}

func TestNewConfirmsNewProject(t *testing.T) {
	dir := assignEnv(t)
	fakeTerminal(t)
	out, err := runIn(t, "n\n", "new", "notes", "--description", "Keep notes.", "--project", dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(out, "Not registered.\n") {
		t.Errorf("output:\n%s", out)
	}
	local, _ := library.LocalDir()
	if _, err := os.Stat(filepath.Join(local, "notes")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("skill created: %v", err)
	}
	if out, err = runIn(t, "", "new", "notes", "--description", "Keep notes.", "--project", dir, "-y"); err != nil || !strings.Contains(out, "Assigned to app: notes") {
		t.Errorf("--yes: %v\n%s", err, out)
	}
}

func TestImportConfirmsNewProject(t *testing.T) {
	dir := assignEnv(t)
	for _, name := range []string{"one", "two"} {
		writeFiles(t, filepath.Join(dir, ".claude", "skills", name), map[string]string{
			"SKILL.md": "---\nname: " + name + "\ndescription: Test skill.\n---\n",
		})
	}
	t.Chdir(dir)
	fakeTerminal(t)
	// one: project, do not register. two: project again, not asked again.
	out, err := runIn(t, "p\nn\n\n", "import")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	skipped := "Skipped: the folder is not a project. Assign it to every folder with g."
	if strings.Count(out, "Assign to (p)roject, (g)lobal, or (s)kip? [p] Register "+dir+" as a project? [Y/n] Not registered.\n"+skipped) != 1 || strings.Count(out, skipped) != 2 {
		t.Errorf("output:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, project.FileName)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf(".quiver.json written: %v", err)
	}
}

func TestSessionNote(t *testing.T) {
	assignEnv(t, "pdf")
	note := "Open assistant sessions see the change through list_skills"
	for _, c := range []struct {
		args []string
		want bool
	}{
		{[]string{"add", "pdf", "--global"}, true},
		{[]string{"remove", "pdf", "--global"}, true},
		{[]string{"remove", "pdf", "--global"}, false}, // nothing removed
	} {
		out, err := runIn(t, "", c.args...)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, note) != c.want {
			t.Errorf("%v: note %v:\n%s", c.args, !c.want, out)
		}
	}
}
