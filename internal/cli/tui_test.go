package cli

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/setup"
	"github.com/kheif/quiver/internal/tui"
)

// fakeScreen makes quiver alone think it runs in a terminal and records
// the backend that would drive the panel.
func fakeScreen(t *testing.T, isTerminal bool) *tui.Backend {
	t.Helper()
	var got tui.Backend
	oldTerm, oldRun := terminal, runTUI
	terminal = func(io.Writer) (int, bool) { return 80, isTerminal }
	runTUI = func(be tui.Backend) error { got = be; return nil }
	t.Cleanup(func() { terminal, runTUI = oldTerm, oldRun })
	fakeTerminal(t)
	return &got
}

func TestBareQuiverOpensTheScreenInATerminal(t *testing.T) {
	assignEnv(t, "pdf")
	got := fakeScreen(t, true)
	out, err := runIn(t, "")
	if err != nil {
		t.Fatal(err)
	}
	if *got == nil || out != "" {
		t.Errorf("backend = %v, output %q", *got, out)
	}
}

func TestBareQuiverPrintsHelpWhenOutputIsNotATerminal(t *testing.T) {
	assignEnv(t, "pdf")
	got := fakeScreen(t, false)
	out, err := runIn(t, "")
	if err != nil {
		t.Fatal(err)
	}
	if *got != nil || !strings.Contains(out, "Usage:") {
		t.Errorf("backend = %v, output:\n%s", *got, out)
	}
}

func TestBareQuiverWithoutAHomeOpensTheScreenWhichOffersSetup(t *testing.T) {
	sourceEnv(t)
	got := fakeScreen(t, true)
	out, err := runIn(t, "")
	if err != nil {
		t.Fatal(err)
	}
	if *got == nil || out != "" {
		t.Fatalf("backend = %v, output:\n%s", *got, out)
	}
	d, err := (*got).Load("")
	if err != nil || !d.NoHome || d.Home == "" {
		t.Fatalf("data %+v, err %v", d, err)
	}
	if homeExists() {
		t.Error("opening the screen created the home")
	}
}

type marks struct{ global, globalAlways, project, projectAlways bool }

func marksOf(d tui.Data) map[string]marks {
	out := map[string]marks{}
	for _, s := range d.Skills {
		out[s.ID] = marks{s.Global, s.GlobalAlways, s.Project, s.ProjectAlways}
	}
	return out
}

func TestTUIBackendMovesASkillInOneScopeAndLeavesTheOtherAlone(t *testing.T) {
	dir := assignEnv(t, "demo")
	t.Chdir(dir)
	b := &tuiBackend{}
	d, err := b.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if d.Project != nil || d.Opened != "" || len(d.Skills) != 1 || !d.Skills[0].Local || d.Version == "" || d.Home == "" {
		t.Fatalf("data %+v", d)
	}

	// demo starts global and on demand.
	if _, err := b.SetGlobal("demo", tui.OnDemand); err != nil {
		t.Fatal(err)
	}
	// Always on in the project registers the folder, assigns the skill and marks
	// it there only; the global setting does not change.
	if _, err := b.SetProject("demo", dir, tui.AlwaysOn); err != nil {
		t.Fatal(err)
	}
	if d, err = b.Load(""); err != nil {
		t.Fatal(err)
	}
	if d.Project == nil || !d.Project.Current || d.Project.Path == "" || d.Opened != d.Project.Name {
		t.Fatalf("project %+v opened %q", d.Project, d.Opened)
	}
	if got := marksOf(d)["demo"]; got != (marks{global: true, project: true, projectAlways: true}) {
		t.Fatalf("marks %+v", got)
	}
	if d.Skills[0].ProjectMode() != tui.AlwaysOn || d.Skills[0].GlobalMode() != tui.OnDemand {
		t.Errorf("modes %v %v", d.Skills[0].ProjectMode(), d.Skills[0].GlobalMode())
	}
	if out, err := runIn(t, "", "status"); err != nil || !strings.Contains(out, "demo") {
		t.Errorf("status after the panel's change:\n%s %v", out, err)
	}

	// Always on to on demand keeps the assignment and drops only the mark.
	if _, err := b.SetProject("demo", dir, tui.OnDemand); err != nil {
		t.Fatal(err)
	}
	d, _ = b.Load("")
	if got := marksOf(d)["demo"]; got != (marks{global: true, project: true}) {
		t.Errorf("after on demand: %+v", got)
	}
	// Global always on does not touch the project.
	if _, err := b.SetGlobal("demo", tui.AlwaysOn); err != nil {
		t.Fatal(err)
	}
	d, _ = b.Load("")
	if got := marksOf(d)["demo"]; got != (marks{global: true, globalAlways: true, project: true}) {
		t.Errorf("after global always on: %+v", got)
	}
	// Removing the project assignment leaves the global one, mark included.
	if _, err := b.SetProject("demo", dir, tui.NotAssigned); err != nil {
		t.Fatal(err)
	}
	d, _ = b.Load("")
	if got := marksOf(d)["demo"]; got != (marks{global: true, globalAlways: true}) {
		t.Errorf("after removing from the project: %+v", got)
	}
	// Removing the global assignment drops its mark too; asking for the mode
	// it already has changes nothing.
	if _, err := b.SetGlobal("demo", tui.NotAssigned); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SetGlobal("demo", tui.NotAssigned); err != nil {
		t.Fatal(err)
	}
	r := loadRegistry(t)
	if len(r.Global) != 0 || len(r.AlwaysOn) != 0 {
		t.Errorf("registry: global %v always-on %v", r.Global, r.AlwaysOn)
	}
	if _, err := b.SetGlobal("missing", tui.OnDemand); err == nil {
		t.Error("an unknown skill did not fail")
	}
}

func TestTUIBackendShowsAnotherProjectAndChangesItByItsFolder(t *testing.T) {
	dir := assignEnv(t, "pdf", "tdd")
	t.Chdir(dir)
	b := &tuiBackend{}
	other := filepath.Join(t.TempDir(), "api")
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SetProject("pdf", dir, tui.OnDemand); err != nil {
		t.Fatal(err)
	}
	// From this folder's project, assign in another one by its folder.
	if _, err := b.SetProject("tdd", other, tui.OnDemand); err != nil {
		t.Fatal(err)
	}
	d, err := b.Load("")
	if err != nil || len(d.Projects) != 2 {
		t.Fatalf("projects %+v, err %v", d.Projects, err)
	}
	var otherID string
	for _, p := range d.Projects {
		if p.Path == other {
			otherID = p.ID
		}
		if len(p.Skills) != 1 {
			t.Errorf("project %s has skills %v", p.Name, p.Skills)
		}
	}
	if otherID == "" || !d.Project.Current {
		t.Fatalf("projects %+v", d.Projects)
	}
	// Managing the other project changes the managed project, not the opened one.
	d, err = b.Load(otherID)
	if err != nil || d.Project.ID != otherID || d.Project.Current || d.Opened == d.Project.Name {
		t.Fatalf("project %+v opened %q, err %v", d.Project, d.Opened, err)
	}
	got := marksOf(d)
	if !got["tdd"].project || got["pdf"].project {
		t.Errorf("marks of the other project: %+v", got)
	}
	// A project that is gone falls back to the folder's, with no error.
	if d, err = b.Load("no-such-project"); err != nil || !d.Project.Current {
		t.Errorf("fallback: %+v, %v", d.Project, err)
	}

	// A folder that no longer exists is an error, not a new project.
	if err := os.RemoveAll(other); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SetProject("pdf", other, tui.OnDemand); err == nil {
		t.Error("assigning in a folder that is gone did not fail")
	}
	if _, err := os.Stat(other); err == nil {
		t.Error("the folder was created again")
	}
	if d, _ = b.Load(""); !slices.ContainsFunc(d.Projects, func(p tui.Project) bool { return p.Missing }) {
		t.Errorf("a project whose folder is gone is not marked: %+v", d.Projects)
	}
}

func TestTUIBackendCountsEachProjectSkillOnce(t *testing.T) {
	dir := assignEnv(t, "pdf")
	t.Chdir(dir)
	b := &tuiBackend{}
	if _, err := b.SetProject("pdf", dir, tui.AlwaysOn); err != nil {
		t.Fatal(err)
	}
	if err := registry.Update(func(r *registry.Registry) error {
		for _, p := range r.Projects {
			p.Skills = append(p.Skills, "pdf")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	d, err := b.Load("")
	if err != nil || len(d.Projects) != 1 || len(d.Projects[0].Skills) != 1 {
		t.Errorf("projects %+v, err %v", d.Projects, err)
	}
}

func TestTUIBackendSearchAndDetails(t *testing.T) {
	assignEnv(t, "pdf", "tdd:test-driven")
	b := &tuiBackend{}
	if _, err := b.Search("pdf"); err == nil {
		t.Error("search before Load did not fail")
	}
	if _, err := b.Load(""); err != nil {
		t.Fatal(err)
	}
	ids, err := b.Search("test driven")
	if err != nil || len(ids) != 1 || ids[0] != "tdd" {
		t.Errorf("ids %v, err %v", ids, err)
	}
	d, err := b.Details("pdf")
	if err != nil || !strings.Contains(d.Raw, "name: pdf") || strings.Contains(d.Body, "name: pdf") {
		t.Errorf("details %+v, err %v", d, err)
	}
	if _, err := b.Details("missing"); err == nil {
		t.Error("details of a missing skill did not fail")
	}
}

func TestTUIBackendEditUsesTheEditCommand(t *testing.T) {
	assignEnv(t, "pdf")
	orig := runEditor
	runEditor = func(path string) error {
		return os.WriteFile(path, []byte("---\nname: pdf\ndescription: Changed.\n---\n"), 0o644)
	}
	t.Cleanup(func() { runEditor = orig })
	b := &tuiBackend{}
	text, err := b.Edit("pdf")
	if err != nil || text != "Saved pdf." {
		t.Errorf("text %q, err %v", text, err)
	}
	if _, err := b.Load(""); err != nil {
		t.Fatal(err)
	}
	if ids, _ := b.Search("Changed"); len(ids) != 1 {
		t.Errorf("the edit was not committed to the library: %v", ids)
	}
}

func TestTUIBackendCheckAndApplyFollowQuiverUpdate(t *testing.T) {
	sourceEnv(t)
	bare := bareSource(t, "acme", "pdf", "review")
	if err := registry.Update(func(r *registry.Registry) error {
		r.AssignGlobal("acme/pdf")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	b := &tuiBackend{}
	if _, err := b.Apply("acme"); err == nil {
		t.Error("apply without a check did not fail")
	}
	p, err := b.Check("acme", false)
	if err != nil || !p.UpToDate || p.Ask {
		t.Fatalf("up to date: %+v, %v", p, err)
	}

	// A change to a skill that is not assigned needs no yes.
	push(t, bare, map[string]string{"skills/review/SKILL.md": "---\nname: review\ndescription: Updated review.\n---\n"}, false)
	if p, err = b.Check("acme", false); err != nil || p.UpToDate || p.Ask {
		t.Fatalf("unassigned change: %+v, %v", p, err)
	}
	if text, err := b.Apply("acme"); err != nil || !strings.Contains(text, "acme updated to") {
		t.Fatalf("%q, %v", text, err)
	}
	l, err := library.Load()
	if err != nil {
		t.Fatal(err)
	}
	if s, err := l.Get("acme/review"); err != nil || s.Description != "Updated review." {
		t.Errorf("after update: %+v, %v", s, err)
	}

	// A change to an assigned skill asks, and is not applied by Check.
	push(t, bare, map[string]string{"skills/pdf/SKILL.md": "---\nname: pdf\ndescription: Updated pdf.\n---\n"}, false)
	if p, err = b.Check("acme", false); err != nil || !p.Ask || !strings.Contains(strings.Join(p.Lines, "\n"), "acme/pdf") {
		t.Fatalf("assigned change: %+v, %v", p, err)
	}
	if l, _ = library.Load(); l != nil {
		if s, err := l.Get("acme/pdf"); err != nil || s.Description == "Updated pdf." {
			t.Errorf("an assigned skill changed before the yes: %+v, %v", s, err)
		}
	}
	if _, err := b.Apply("acme"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Check("nope", false); err == nil {
		t.Error("an unknown source did not fail")
	}
}

func TestTUIBackendRemovesASourceAndSaysWhatWentWithIt(t *testing.T) {
	sourceEnv(t)
	bareSource(t, "acme", "pdf")
	if err := registry.Update(func(r *registry.Registry) error {
		r.AssignGlobal("acme/pdf")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	text, err := (&tuiBackend{}).RemoveSource("acme")
	if err != nil || !strings.Contains(text, "Removed source acme") || !strings.Contains(text, "1 skill globally") {
		t.Errorf("%q, %v", text, err)
	}
	if r := loadRegistry(t); len(r.Global) != 0 {
		t.Errorf("global %v", r.Global)
	}
}

func TestTUIBackendRenamesForgetsAndRefusesToSplitAProjectThatIsNotACopy(t *testing.T) {
	dir := assignEnv(t, "pdf")
	t.Chdir(dir)
	b := &tuiBackend{}
	if _, err := b.SetProject("pdf", dir, tui.OnDemand); err != nil {
		t.Fatal(err)
	}
	d, _ := b.Load("")
	id := d.Project.ID
	if err := b.RenameProject(id, "web"); err != nil {
		t.Fatal(err)
	}
	if d, _ = b.Load(""); d.Project.Name != "web" {
		t.Errorf("name %q", d.Project.Name)
	}
	if _, err := b.SplitHere(); err == nil {
		t.Error("splitting a project that is not a copy did not fail")
	}
	if err := b.ForgetProject(id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, project.FileName)); err != nil {
		t.Errorf(".quiver.json was removed: %v", err)
	}
	if d, _ = b.Load(""); len(d.Projects) != 0 {
		t.Errorf("projects %+v", d.Projects)
	}
	if err := b.ForgetProject(id); err == nil {
		t.Error("forgetting an unknown project did not fail")
	}
}

func TestTUIBackendForksASourceSkill(t *testing.T) {
	sourceEnv(t)
	bareSource(t, "acme", "pdf")
	if err := registry.Update(func(r *registry.Registry) error {
		r.AssignGlobal("acme/pdf")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	b := &tuiBackend{}
	id, note, err := b.Fork("acme/pdf", "")
	if err != nil || id != "pdf" || !strings.Contains(note, "took the original's place") {
		t.Fatalf("id %q, note %q, err %v", id, note, err)
	}
	if r := loadRegistry(t); !slices.Equal(r.Global, []string{"pdf"}) {
		t.Errorf("global %v", r.Global)
	}
	if _, _, err := b.Fork("pdf", ""); err == nil {
		t.Error("forking a local skill did not fail")
	}
}

func TestTUIBackendAddsASourceLikeQuiverSourceAdd(t *testing.T) {
	sourceEnv(t)
	if err := setup.EnsureInit(); err != nil {
		t.Fatal(err)
	}
	b := &tuiBackend{}
	text, err := b.AddSource(skillRepo(t, "team", "pdf", "docx"), "acme")
	if err != nil || !strings.Contains(text, "Added source acme with 2 skills") {
		t.Fatalf("text %q, err %v", text, err)
	}
	d, err := b.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Sources) != 1 || d.Sources[0].Name != "acme" || len(d.Skills) != 2 {
		t.Errorf("sources %+v, skills %d", d.Sources, len(d.Skills))
	}
	if _, err := b.AddSource(skillRepo(t, "team", "pdf"), "acme"); err == nil {
		t.Error("a second source with the same name was added")
	}
}

func TestTUIBackendRefusesASourceWithoutSkills(t *testing.T) {
	sourceEnv(t)
	b := &tuiBackend{}
	if _, err := b.AddSource(skillRepo(t, "empty"), "empty"); err == nil || !strings.Contains(err.Error(), "no skills found") {
		t.Errorf("err %v", err)
	}
	if d, _ := b.Load(""); len(d.Sources) != 0 {
		t.Errorf("sources %+v", d.Sources)
	}
}

func TestTUIBackendCreatesASkillInTheProject(t *testing.T) {
	dir := assignEnv(t)
	b := &tuiBackend{}
	text, err := b.NewSkill("release-notes", "Write release notes.", dir, false)
	if err != nil || !strings.Contains(text, "Created skill release-notes.") || !strings.Contains(text, "Assigned to app: release-notes") {
		t.Fatalf("text %q, err %v", text, err)
	}
	r := loadRegistry(t)
	p, err := project.Find(dir)
	if err != nil || !slices.Contains(r.Projects[p.ID].Skills, "release-notes") {
		t.Errorf("registry %+v, err %v", r.Projects, err)
	}
	// Without a project it is only created.
	text, err = b.NewSkill("loose", "Stand-alone.", "", false)
	if err != nil || !strings.Contains(text, "Not assigned") {
		t.Errorf("text %q, err %v", text, err)
	}
	if _, err := b.NewSkill("loose", "Again.", "", false); err == nil {
		t.Error("an existing name was reused")
	}
}

func TestTUIBackendCreatesASkillInTheEditor(t *testing.T) {
	assignEnv(t)
	orig := runEditor
	runEditor = func(path string) error {
		return os.WriteFile(path, []byte("---\nname: notes\ndescription: Take notes.\n---\nBody.\n"), 0o644)
	}
	t.Cleanup(func() { runEditor = orig })
	b := &tuiBackend{}
	text, err := b.NewSkill("notes", "", "", false)
	if err != nil || !strings.Contains(text, "Saved notes.") {
		t.Fatalf("text %q, err %v", text, err)
	}
}

func TestTUIBackendImportsAFolderOfSkills(t *testing.T) {
	assignEnv(t)
	pack := filepath.Join(t.TempDir(), "pack")
	if err := os.MkdirAll(filepath.Join(pack, "greeter"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: greeter\ndescription: Say hello.\n---\nHi.\n"
	if err := os.WriteFile(filepath.Join(pack, "greeter", "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	b := &tuiBackend{}
	text, err := b.Import(pack)
	if err != nil || text != "Imported 1 skill." {
		t.Fatalf("text %q, err %v", text, err)
	}
	if _, err := b.Load(""); err != nil {
		t.Fatal(err)
	}
	if ids, _ := b.Search("hello"); len(ids) != 1 {
		t.Errorf("the skill is not in the library: %v", ids)
	}
}
