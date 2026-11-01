package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/setup"
)

// machines sets up two Quiver homes, a and b, and an empty bare remote.
// use switches QUIVER_HOME between them.
func machines(t *testing.T) (a, b, remote string, use func(home string)) {
	t.Helper()
	a = sourceEnv(t)
	b = filepath.Join(t.TempDir(), "q")
	remote = filepath.Join(t.TempDir(), "home.git")
	gitIn(t, ".", "init", "--quiet", "--bare", remote)
	use = func(home string) {
		t.Helper()
		t.Setenv("QUIVER_HOME", home)
		if err := setup.EnsureInit(); err != nil {
			t.Fatal(err)
		}
	}
	use(a)
	return a, b, remote, use
}

// localSkill writes a local skill named name into home.
func localSkill(t *testing.T, home, name, description string) {
	t.Helper()
	dir := filepath.Join(home, "local", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// quiver runs quiver and fails the test on an error.
func quiver(t *testing.T, args ...string) string {
	t.Helper()
	out, err := runIn(t, "", args...)
	if err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
	return out
}

func TestSyncBetweenMachines(t *testing.T) {
	a, b, remote, use := machines(t)
	app := projectDir(t, "app", "app-000001")

	// Machine A: a new skill, assigned to a project, then the first sync.
	localSkill(t, a, "notes", "Take notes.")
	quiver(t, "add", "notes", "--project", app)
	out := quiver(t, "sync", "init", remote)
	if !strings.Contains(out, "The Quiver home now syncs with "+remote+" (branch main).") {
		t.Errorf("init output:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(a, "local", ".git")); !os.IsNotExist(err) {
		t.Errorf("local/.git still exists: %v", err)
	}
	if setup.Missing(a) != nil {
		t.Errorf("home looks broken after init: %v", setup.Missing(a))
	}

	// Machine B joins: it gets the skill and the assignment.
	use(b)
	localSkill(t, b, "draft", "Write drafts.")
	quiver(t, "add", "draft", "--global")
	out = quiver(t, "sync", "init", remote)
	if !strings.Contains(out, "Merged the changes of both machines in registry.json.") {
		t.Errorf("join output:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(b, "local", "notes", "SKILL.md")); err != nil {
		t.Error(err)
	}
	r := loadRegistry(t)
	if p := r.Projects["app-000001"]; p == nil || !reflect.DeepEqual(p.Skills, []string{"notes"}) || len(p.Paths) != 0 {
		t.Errorf("project on b: %+v", p)
	}
	if !reflect.DeepEqual(r.Global, []string{"draft"}) {
		t.Errorf("global on b: %v", r.Global)
	}

	// Back on A: a sync brings B's skill and assignment, and the project
	// folder A knows stays.
	use(a)
	out = quiver(t, "sync")
	if !strings.Contains(out, "Got the changes from "+remote) {
		t.Errorf("sync output on a:\n%s", out)
	}
	r = loadRegistry(t)
	if !reflect.DeepEqual(r.Global, []string{"draft"}) || !reflect.DeepEqual(r.Projects["app-000001"].Paths, []string{app}) {
		t.Errorf("registry on a: global %v, paths %v", r.Global, r.Projects["app-000001"].Paths)
	}
	if out := quiver(t, "sync"); !strings.Contains(out, "Already in sync.") {
		t.Errorf("second sync:\n%s", out)
	}
	// Commands that commit into local/ keep working in the synced home;
	// registry.json is committed by the next sync.
	quiver(t, "delete", "draft", "--yes", "--force")
	if status := gitOut(t, a, "status", "--short"); status != "M registry.json" {
		t.Errorf("status after delete: %q", status)
	}
	if log := gitOut(t, a, "log", "-1", "--format=%s"); log != "delete draft" {
		t.Errorf("last commit: %q", log)
	}
}

func TestSyncMergesAssignments(t *testing.T) {
	a, b, remote, use := machines(t)
	localSkill(t, a, "one", "One.")
	localSkill(t, a, "two", "Two.")
	localSkill(t, a, "three", "Three.")
	quiver(t, "add", "one", "--global")
	quiver(t, "sync", "init", remote)
	use(b)
	quiver(t, "sync", "init", remote)

	// Both machines change the assignments before they sync again.
	quiver(t, "add", "two", "--global")
	quiver(t, "remove", "one", "--global")
	quiver(t, "sync")
	use(a)
	quiver(t, "add", "three", "--global", "--always")
	out := quiver(t, "sync")
	if !strings.Contains(out, "Merged the changes of both machines in registry.json.") || !strings.Contains(out, "Sent this machine's changes") {
		t.Errorf("sync output:\n%s", out)
	}
	r := loadRegistry(t)
	if !reflect.DeepEqual(r.Global, []string{"three", "two"}) || !reflect.DeepEqual(r.AlwaysOn, []string{"three"}) {
		t.Errorf("global %v, always-on %v", r.Global, r.AlwaysOn)
	}
	use(b)
	quiver(t, "sync")
	if r := loadRegistry(t); !reflect.DeepEqual(r.Global, []string{"three", "two"}) {
		t.Errorf("global on b: %v", r.Global)
	}
}

func TestSyncStopsOnSkillConflict(t *testing.T) {
	a, b, remote, use := machines(t)
	localSkill(t, a, "notes", "Take notes.")
	quiver(t, "sync", "init", remote)
	use(b)
	quiver(t, "sync", "init", remote)
	localSkill(t, b, "notes", "Take notes on b.")
	quiver(t, "sync")

	use(a)
	localSkill(t, a, "notes", "Take notes on a.")
	before := gitOut(t, a, "rev-parse", "HEAD")
	_, err := runIn(t, "", "sync")
	if err == nil || !strings.Contains(err.Error(), "cannot be merged automatically:\n  local/notes/SKILL.md") ||
		!strings.Contains(err.Error(), "Nothing was pulled or pushed") {
		t.Fatalf("err = %v", err)
	}
	if setup.Synced(a) && gitOut(t, a, "status", "--short") != "" {
		t.Error("the conflict was left in the working tree")
	}
	if got := gitOut(t, a, "log", "-1", "--format=%P", "HEAD"); !strings.Contains(got, before) {
		t.Errorf("HEAD moved off the local commit: parent %s, want %s", got, before)
	}
	data, _ := os.ReadFile(filepath.Join(a, "local", "notes", "SKILL.md"))
	if !strings.Contains(string(data), "Take notes on a.") {
		t.Errorf("local change lost:\n%s", data)
	}
}

func TestSyncInitConflictChangesNothing(t *testing.T) {
	a, b, remote, use := machines(t)
	localSkill(t, a, "notes", "Take notes on a.")
	quiver(t, "sync", "init", remote)
	use(b)
	localSkill(t, b, "notes", "Take notes on b.")
	gitIn(t, filepath.Join(b, "local"), "add", "-A")
	gitIn(t, filepath.Join(b, "local"), "-c", "user.name=t", "-c", "user.email=t@t", "commit", "--quiet", "-m", "add notes")
	_, err := runIn(t, "", "sync", "init", remote)
	if err == nil || !strings.Contains(err.Error(), "quiver sync init changed nothing") {
		t.Fatalf("err = %v", err)
	}
	if setup.Synced(b) {
		t.Error("b is a repository after the failed init")
	}
	if got := gitOut(t, filepath.Join(b, "local"), "log", "-1", "--format=%s"); got != "add notes" {
		t.Errorf("local/ history of b: %q", got)
	}
	data, _ := os.ReadFile(filepath.Join(b, "local", "notes", "SKILL.md"))
	if !strings.Contains(string(data), "on b.") {
		t.Errorf("b's skill changed:\n%s", data)
	}
}

func TestSyncClonesMissingSources(t *testing.T) {
	_, b, remote, use := machines(t)
	bareSource(t, "acme", "pdf")
	quiver(t, "add", "acme/pdf", "--global")
	quiver(t, "sync", "init", remote)

	use(b)
	out := quiver(t, "sync", "init", remote)
	if !strings.Contains(out, "Downloaded source acme.") {
		t.Errorf("output:\n%s", out)
	}
	l, err := library.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Get("acme/pdf"); err != nil {
		t.Errorf("acme/pdf is not in b's library; warnings %v", l.Warnings)
	}
	if r := loadRegistry(t); !reflect.DeepEqual(r.Global, []string{"acme/pdf"}) {
		t.Errorf("global on b: %v", r.Global)
	}
}

func TestSyncNeedsInit(t *testing.T) {
	sourceEnv(t)
	_, err := runIn(t, "", "sync")
	if err == nil || !strings.Contains(err.Error(), "is not synced yet: run quiver sync init <git-remote>") {
		t.Errorf("err = %v", err)
	}
}

func TestRegistryMergeDropsOrphanAlwaysOn(t *testing.T) {
	base := []byte(`{"version":2,"global":["a"],"alwaysOn":[],"projects":{}}`)
	ours := []byte(`{"version":2,"global":[],"alwaysOn":[],"projects":{}}`)
	theirs := []byte(`{"version":2,"global":["a"],"alwaysOn":["a"],"projects":{}}`)
	r, err := registry.Merge(base, ours, theirs)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Global) != 0 || len(r.AlwaysOn) != 0 {
		t.Errorf("global %v, always-on %v", r.Global, r.AlwaysOn)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}
