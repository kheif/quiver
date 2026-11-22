package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/homesync"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/tui"
)

// A Searchable skill counts as assigned wherever the registry is read.

func TestSourceRemoveCountsSearchable(t *testing.T) {
	dir := assignEnv(t)
	repo := skillRepo(t, "acme", "review", "deploy")
	mustRun(t, "", "source", "add", repo, "--name", "acme")
	mustRun(t, "", "add", "acme/review", "--searchable", "--global")
	mustRun(t, "", "add", "acme/deploy", "--searchable", "--project", dir, "--yes")

	out, err := runIn(t, "", "source", "remove", "acme")
	if err == nil || !strings.Contains(err.Error(), "global: acme/review") || !strings.Contains(err.Error(), "acme/deploy") {
		t.Fatalf("remove refused with %v\n%s", err, out)
	}
	out = mustRun(t, "", "source", "remove", "acme", "--force")
	wantOut(t, out, "Unassigned globally: acme/review", "acme/deploy", `Removed source "acme".`)
	if r := loadRegistry(t); len(r.Searchable) != 0 || len(r.Projects[onlyProject(t)].Searchable) != 0 {
		t.Errorf("registry %+v", r)
	}
}

func TestDeleteCountsSearchable(t *testing.T) {
	assignEnv(t, "notes")
	mustRun(t, "", "add", "notes", "--searchable", "--global")

	out, err := runIn(t, "", "delete", "notes", "--yes")
	if err == nil || !strings.Contains(err.Error(), "skill notes is assigned") {
		t.Fatalf("delete of a Searchable skill: %v\n%s", err, out)
	}
	mustRun(t, "", "delete", "notes", "--yes", "--force")
	if r := loadRegistry(t); len(r.Searchable) != 0 {
		t.Errorf("Searchable = %v", r.Searchable)
	}
}

func TestForkReplacesSearchable(t *testing.T) {
	dir := assignEnv(t)
	repo := skillRepo(t, "acme", "review")
	mustRun(t, "", "source", "add", repo, "--name", "acme")
	mustRun(t, "", "add", "acme/review", "--searchable", "--global")
	mustRun(t, "", "add", "acme/review", "--searchable", "--project", dir, "--yes")

	out := mustRun(t, "", "fork", "acme/review")
	wantOut(t, out, "Replaced acme/review with review globally.", "Replaced acme/review with review in project")
	r := loadRegistry(t)
	if !reflect.DeepEqual(r.Searchable, []string{"review"}) || !reflect.DeepEqual(r.Projects[onlyProject(t)].Searchable, []string{"review"}) {
		t.Errorf("registry %+v", r)
	}
}

func TestDoctorChecksSearchable(t *testing.T) {
	assignEnv(t, "pdf", "pdf-copy:pdf", "tdd")
	registered(t)
	mustRun(t, "", "add", "pdf", "pdf-copy", "--searchable", "--global")

	out := mustRun(t, "", "doctor")
	wantOut(t, out, "warn   served skills that share a name", "every folder: pdf, pdf-copy share the name")

	// An assigned skill that is not in the library is reported, at any level.
	err := registryUpdateSearchable("gone")
	if err != nil {
		t.Fatal(err)
	}
	out, _ = runIn(t, "", "doctor")
	wantOut(t, out, "assigned skills that are not in the library", "global: gone")
	mustRun(t, "y\n", "doctor", "--fix")
	if r := loadRegistry(t); len(r.Searchable) != 2 {
		t.Errorf("--fix left Searchable = %v", r.Searchable)
	}
}

func TestProjectsCountSearchable(t *testing.T) {
	dir := assignEnv(t, "pdf", "tdd", "notes")
	mustRun(t, "", "add", "pdf", "--project", dir, "--yes")
	mustRun(t, "", "add", "tdd", "notes", "--searchable", "--project", dir)

	out := mustRun(t, "", "projects")
	if !strings.Contains(out, "app") || !strings.Contains(out, "  3  ") {
		t.Errorf("projects counts one skill, not three:\n%s", out)
	}
	out = mustRun(t, "n\n", "projects", "forget", "app")
	wantOut(t, out, "and its 3 skill assignments?")
	if !strings.Contains(mustRun(t, "", "projects", "--json"), `"searchable": [`) {
		t.Error("projects --json has no searchable field")
	}
}

func TestSummaryAndCompletionCountSearchable(t *testing.T) {
	dir := assignEnv(t, "pdf", "tdd")
	t.Chdir(dir)
	mustRun(t, "", "add", "pdf", "tdd", "--searchable", "--global")
	out, _ := runIn(t, "")
	wantOut(t, out, "Served:  0 skills, 2 searchable\n", "Next:    quiver status\n")
	mustRun(t, "", "add", "pdf", "--global")
	out, _ = runIn(t, "")
	wantOut(t, out, "Served:  1 skill (1 global, 0 project), 1 searchable\n")

	// remove completes Searchable skills too.
	got := mustRun(t, "", "__complete", "remove", "--global", "")
	wantOut(t, got, "pdf\n", "tdd\n")
	got = mustRun(t, "", "__complete", "remove", "--project", dir, "")
	if strings.Contains(got, "tdd\n") {
		t.Errorf("a global skill is completed for the project:\n%s", got)
	}
}

// The review of sync lists Searchable skills that arrive and changes to
// their files, as for always-on skills.
func TestSyncReviewsSearchableChanges(t *testing.T) {
	a, b, remote, use := machines(t)
	localSkill(t, a, "rules", "House rules.")
	quiver(t, "sync", "init", remote)
	use(b)
	quiver(t, "sync", "init", remote)

	use(a)
	quiver(t, "add", "rules", "--searchable", "--global")
	quiver(t, "sync")
	use(b)
	out, err := runIn(t, "", "sync")
	if !errors.Is(err, homesync.ErrDeclined) || !strings.Contains(out, "becomes searchable (the assistant loads it without asking): rules") {
		t.Fatalf("err = %v\n%s", err, out)
	}
	if r := loadRegistry(t); len(r.Searchable) != 0 {
		t.Errorf("Searchable after a declined sync: %v", r.Searchable)
	}
	quiver(t, "sync", "--yes")
	if r := loadRegistry(t); !reflect.DeepEqual(r.Searchable, []string{"rules"}) {
		t.Errorf("Searchable after --yes: %v", r.Searchable)
	}

	use(a)
	if err := os.WriteFile(filepath.Join(a, "local", "rules", "SKILL.md"), []byte("---\nname: rules\ndescription: House rules.\n---\nNew text.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	quiver(t, "sync")
	use(b)
	out, err = runIn(t, "", "sync")
	if !errors.Is(err, homesync.ErrDeclined) || !strings.Contains(out, "searchable skill with changed files: rules") {
		t.Errorf("err = %v\n%s", err, out)
	}
}

func registryUpdateSearchable(ids ...string) error {
	return registry.Update(func(r *registry.Registry) error {
		r.AssignSearchableGlobal(ids...)
		return nil
	})
}

// The panel moves a skill to and from Searchable with the functions of add
// and remove, in one scope, and reads the level of each scope.
func TestTUIBackendSearchable(t *testing.T) {
	dir := assignEnv(t, "demo", "other")
	t.Chdir(dir)
	b := &tuiBackend{}
	mustRun(t, "", "add", "demo", "other", "--global", "--yes")
	mustRun(t, "", "add", "demo", "--always", "--yes")

	// Searchable in the project leaves the global scope alone and drops the
	// project's always-on mark.
	if _, err := b.SetProject("demo", dir, tui.Searchable); err != nil {
		t.Fatal(err)
	}
	d, err := b.Load("")
	if err != nil {
		t.Fatal(err)
	}
	var demo tui.Skill
	for _, s := range d.Skills {
		if s.ID == "demo" {
			demo = s
		}
	}
	if demo.ProjectMode() != tui.Searchable || demo.GlobalMode() != tui.OnDemand || demo.Project || demo.ProjectAlways || !demo.ProjectSearchable {
		t.Errorf("demo %+v", demo)
	}
	if !reflect.DeepEqual(d.Project.Searchable, []string{"demo"}) || len(d.Project.Skills) != 0 {
		t.Errorf("project %+v", d.Project)
	}
	if !reflect.DeepEqual(demo.SearchableIn, []string{d.Project.ID}) {
		t.Errorf("SearchableIn %v", demo.SearchableIn)
	}

	// Global: Searchable, then back to On demand, then to Always on.
	if _, err := b.SetGlobal("other", tui.Searchable); err != nil {
		t.Fatal(err)
	}
	if r := loadRegistry(t); !reflect.DeepEqual(r.Searchable, []string{"other"}) || !reflect.DeepEqual(r.Global, []string{"demo"}) {
		t.Errorf("registry %+v", r)
	}
	if _, err := b.SetGlobal("other", tui.AlwaysOn); err != nil {
		t.Fatal(err)
	}
	if r := loadRegistry(t); len(r.Searchable) != 0 || !reflect.DeepEqual(r.AlwaysOn, []string{"other"}) {
		t.Errorf("registry %+v", r)
	}
	// Searchable to Not assigned removes it from the scope.
	if _, err := b.SetProject("demo", dir, tui.NotAssigned); err != nil {
		t.Fatal(err)
	}
	if p := loadRegistry(t).Projects[onlyProject(t)]; len(p.Searchable) != 0 {
		t.Errorf("project %+v", p)
	}
}

func TestTUIBackendMakesASourceSearchable(t *testing.T) {
	assignEnv(t)
	b := &tuiBackend{}
	added, err := b.AddSource(skillRepo(t, "team", "pdf", "docx", "xlsx"), "acme")
	if err != nil || added.Skills != 3 || added.Source != "acme" {
		t.Fatalf("added %+v, err %v", added, err)
	}
	// A skill that is listed already keeps its level.
	mustRun(t, "", "add", "acme/pdf", "--global")
	text, err := b.MakeSearchable("acme")
	if err != nil || !strings.Contains(text, "Made Searchable globally: acme/docx, acme/xlsx") || !strings.Contains(text, "Undo: quiver remove --source acme --global") {
		t.Errorf("text %q, err %v", text, err)
	}
	r := loadRegistry(t)
	if !reflect.DeepEqual(r.Global, []string{"acme/pdf"}) || !reflect.DeepEqual(r.Searchable, []string{"acme/docx", "acme/xlsx"}) {
		t.Errorf("registry %+v", r)
	}
}
