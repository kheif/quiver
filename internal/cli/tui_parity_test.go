package cli

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/assistant/assistanttest"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/tui"
)

func TestParseChecksReadsWhatDoctorPrints(t *testing.T) {
	items := parseChecks(`ok     Quiver home layout: /h
warn   local/ has uncommitted changes
       ?? a/SKILL.md
       ?? b/SKILL.md
error  library skills that cannot be used
       local/x: missing frontmatter
-      registered with Claude Code (skipped: the claude command was not found)
`)
	if len(items) != 4 {
		t.Fatalf("items %+v", items)
	}
	if items[1].Level != "warn" || items[1].Title != "local/ has uncommitted changes" || !slices.Equal(items[1].Details, []string{"?? a/SKILL.md", "?? b/SKILL.md"}) {
		t.Errorf("warn %+v", items[1])
	}
	if items[2].Level != "error" || len(items[2].Details) != 1 {
		t.Errorf("error %+v", items[2])
	}
	if items[3].Level != "ok" || !strings.Contains(items[3].Title, "skipped") {
		t.Errorf("skipped check %+v", items[3])
	}
}

func TestTUIBackendHealthRunsTheChecksWithoutTheNetwork(t *testing.T) {
	assignEnv(t, "demo")
	assistanttest.Install(t)
	appHome(t)
	fakeExecutable(t, "/opt/homebrew/bin/quiver")
	t.Setenv(noUpdateCheckEnv, "")
	b := &tuiBackend{}
	r, err := b.Health()
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, it := range r.Items {
		titles = append(titles, it.Title)
	}
	joined := strings.Join(titles, "\n")
	for _, want := range []string{"Quiver home layout", "registry.json", "How always-on skills reach your assistants"} {
		if !strings.Contains(joined, want) {
			t.Errorf("no %q in:\n%s", want, joined)
		}
	}
	// The version check asks the network, so the panel does not run it.
	if strings.Contains(joined, "latest release") || strings.Contains(joined, "newer") {
		t.Errorf("the checks asked for the latest release:\n%s", joined)
	}
}

func TestTUIBackendHealthIsEmptyWithoutAHome(t *testing.T) {
	sourceEnv(t)
	r, err := (&tuiBackend{}).Health()
	if err != nil || len(r.Items) != 0 || homeExists() {
		t.Errorf("report %+v, err %v, home %v", r, err, homeExists())
	}
}

func TestTUIBackendSetupCreatesTheHomeAndRegistersTheAssistants(t *testing.T) {
	sourceEnv(t)
	f := assistanttest.Install(t)
	appHome(t)
	fakeExecutable(t, "/opt/homebrew/bin/quiver")
	t.Chdir(t.TempDir())
	b := &tuiBackend{}
	d, err := b.Load("")
	if err != nil || !d.NoHome {
		t.Fatalf("before setup: %+v, %v", d, err)
	}
	text, err := b.Setup()
	if err != nil || !strings.Contains(text, "Quiver is set up in") {
		t.Fatalf("text %q, err %v", text, err)
	}
	if !homeExists() {
		t.Error("the home was not created")
	}
	if f.User() != "/opt/homebrew/bin/quiver|mcp" {
		t.Errorf("the assistant was not registered: %q", f.User())
	}
	if d, err = b.Load(""); err != nil || d.NoHome {
		t.Errorf("after setup: %+v, %v", d, err)
	}
}

func TestTUIBackendInstallAndUninstallLikeTheCommands(t *testing.T) {
	assignEnv(t, "demo")
	f := assistanttest.Install(t)
	appHome(t)
	fakeExecutable(t, "/opt/homebrew/bin/quiver")
	t.Chdir(t.TempDir())
	b := &tuiBackend{}
	if text, err := b.Install(); err != nil || !strings.Contains(text, "Registered Quiver") || f.User() == "" {
		t.Fatalf("install: %q, %v, user %q", text, err, f.User())
	}
	if text, err := b.Uninstall(); err != nil || !strings.Contains(text, "Removed Quiver") || f.User() != "" {
		t.Fatalf("uninstall: %q, %v, user %q", text, err, f.User())
	}
}

func TestTUIBackendFixAppliesTheSafeFixes(t *testing.T) {
	home := assignEnv(t, "demo")
	_ = home
	assistanttest.Install(t)
	appHome(t)
	fakeExecutable(t, "/opt/homebrew/bin/quiver")
	t.Chdir(t.TempDir())
	b := &tuiBackend{}
	if text, err := b.Fix(); err != nil {
		t.Fatalf("%q, %v", text, err)
	}
	// local/ was not committed; the fix commits it, so the next check is quiet.
	r, _ := b.Health()
	for _, it := range r.Items {
		if strings.Contains(it.Title, "uncommitted changes") {
			t.Errorf("still uncommitted: %+v", it)
		}
	}
}

func TestTUIBackendDeletesALocalSkillLikeQuiverDelete(t *testing.T) {
	dir := assignEnv(t, "demo", "keep")
	t.Chdir(dir)
	b := &tuiBackend{}
	if _, err := b.SetProject("demo", dir, tui.OnDemand); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Delete("demo", false); err == nil || !strings.Contains(err.Error(), "assigned") {
		t.Fatalf("deleting an assigned skill without force: %v", err)
	}
	text, err := b.Delete("demo", true)
	if err != nil || !strings.Contains(text, "Deleted skill demo.") || !strings.Contains(text, "Unassigned from app.") {
		t.Fatalf("text %q, err %v", text, err)
	}
	if r := loadRegistry(t); len(r.ProjectsUsing("demo")) != 0 {
		t.Errorf("still assigned: %v", r.ProjectsUsing("demo"))
	}
	if text, err = b.Delete("keep", false); err != nil || !strings.Contains(text, "Deleted skill keep.") {
		t.Errorf("unassigned skill: %q, %v", text, err)
	}
	if _, err := b.Delete("acme/pdf", true); err == nil {
		t.Error("a source skill was deleted on its own")
	}
}

func TestTUIBackendPinsAndUnpinsASourceSkill(t *testing.T) {
	sourceEnv(t)
	bareSource(t, "acme", "pdf")
	b := &tuiBackend{}
	text, err := b.Pin("acme/pdf")
	if err != nil || !strings.HasPrefix(text, "Pinned acme/pdf to ") {
		t.Fatalf("pin: %q, %v", text, err)
	}
	d, err := b.Load("")
	if err != nil {
		t.Fatal(err)
	}
	pinned := false
	for _, s := range d.Skills {
		pinned = pinned || (s.ID == "acme/pdf" && s.Pin != "")
	}
	if !pinned || len(d.Sources[0].Pinned) != 1 {
		t.Errorf("the pin is not in the data: %+v", d.Sources)
	}
	text, err = b.Unpin("acme/pdf")
	if err != nil || !strings.Contains(text, "Unpinned acme/pdf") {
		t.Fatalf("unpin: %q, %v", text, err)
	}
	if _, err := b.Pin("nope/skill"); err == nil {
		t.Error("an unknown skill was pinned")
	}
}

func TestTUIBackendCheckSaysARewrittenHistoryAndResetsOnRequest(t *testing.T) {
	sourceEnv(t)
	bare := bareSource(t, "acme", "pdf")
	push(t, bare, map[string]string{"README.md": "# rewritten"}, true, "--amend")
	b := &tuiBackend{}
	_, err := b.Check("acme", false)
	if !errors.Is(err, tui.ErrDiverged) {
		t.Fatalf("err %v", err)
	}
	p, err := b.Check("acme", true)
	if err != nil || p.UpToDate {
		t.Fatalf("reset check: %+v, %v", p, err)
	}
	if !strings.Contains(strings.Join(p.Lines, "\n"), "reset to the rewritten remote history") {
		t.Errorf("lines %v", p.Lines)
	}
	if _, err := b.Apply("acme"); err != nil {
		t.Fatal(err)
	}
	if p, err := b.Check("acme", false); err != nil || !p.UpToDate {
		t.Errorf("after the reset: %+v, %v", p, err)
	}
}

func TestTUIBackendCheckGivesTheDiffOfAnAssignedSkill(t *testing.T) {
	sourceEnv(t)
	bare := bareSource(t, "acme", "pdf", "review")
	if err := registry.Update(func(r *registry.Registry) error {
		r.AssignGlobal("acme/pdf")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	push(t, bare, map[string]string{
		"skills/pdf/SKILL.md":    "---\nname: pdf\ndescription: Updated pdf.\n---\n",
		"skills/review/SKILL.md": "---\nname: review\ndescription: Updated review.\n---\n",
	}, false)
	p, err := (&tuiBackend{}).Check("acme", false)
	if err != nil || !p.Ask {
		t.Fatalf("%+v, %v", p, err)
	}
	diff := strings.Join(p.Diff, "\n")
	if !strings.Contains(diff, "+description: Updated pdf.") {
		t.Errorf("no patch of the assigned skill:\n%s", diff)
	}
	if strings.Contains(diff, "Updated review") {
		t.Errorf("the patch of an unassigned skill is in:\n%s", diff)
	}
}

func TestTUIBackendForkTakesAName(t *testing.T) {
	sourceEnv(t)
	bareSource(t, "acme", "pdf")
	b := &tuiBackend{}
	id, _, err := b.Fork("acme/pdf", "my-pdf")
	if err != nil || id != "my-pdf" {
		t.Fatalf("id %q, err %v", id, err)
	}
	_, _, err = b.Fork("acme/pdf", "my-pdf")
	if err == nil || strings.Contains(err.Error(), "--as") {
		t.Errorf("a name that is taken: %v, mentions --as", err)
	}
}

func TestTUIBackendNewSkillCanBeGlobal(t *testing.T) {
	dir := assignEnv(t)
	t.Chdir(dir)
	b := &tuiBackend{}
	text, err := b.NewSkill("notes", "Write notes.", "", true)
	if err != nil || !strings.Contains(text, "Created skill notes.") {
		t.Fatalf("%q, %v", text, err)
	}
	if r := loadRegistry(t); !slices.Contains(r.Global, "notes") {
		t.Errorf("global %v", r.Global)
	}
}

func TestTUIBackendNewSkillMessagesAreSentences(t *testing.T) {
	dir := assignEnv(t)
	t.Chdir(dir)
	text, err := (&tuiBackend{}).NewSkill("notes", "Write notes.", dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, "notes Assigned") || !strings.HasSuffix(text, ".") {
		t.Errorf("messages run together: %q", text)
	}
}

func TestTUIBackendChecksNamesAndImportPathsEarly(t *testing.T) {
	b := &tuiBackend{}
	for _, bad := range []string{"Bad", "a b", "-lead", "a/b", strings.Repeat("x", 80)} {
		if err := b.CheckName(bad); err == nil {
			t.Errorf("name %q was accepted", bad)
		}
	}
	if err := b.CheckName("release-notes"); err != nil {
		t.Error(err)
	}
	if err := b.CheckImport(""); err != nil {
		t.Errorf("an empty path is the project's .claude/skills: %v", err)
	}
	if err := b.CheckImport(filepath.Join(t.TempDir(), "missing")); err == nil || !strings.Contains(err.Error(), "no such file or folder") {
		t.Errorf("a missing path: %v", err)
	}
	if err := b.CheckImport(t.TempDir()); err != nil {
		t.Error(err)
	}
}

func TestTUIBackendImportSaysWhatItDidInOneLine(t *testing.T) {
	dir := assignEnv(t)
	t.Chdir(dir)
	for _, name := range []string{"deploy", "lint"} {
		skillDir := filepath.Join(dir, ".claude", "skills", name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: "+name+"\ndescription: Does "+name+".\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	text, err := (&tuiBackend{}).Import("")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(text, "Imported 2 skills.") || strings.Contains(text, "Found") || strings.Contains(text, ".claude/skills") {
		t.Errorf("text %q", text)
	}
	if text, err = (&tuiBackend{}).Import(""); err != nil || !strings.Contains(text, "No skills found") {
		t.Errorf("nothing left: %q, %v", text, err)
	}
}

func TestTUIBackendLoadTellsAlwaysOnAndCopiesOfAProject(t *testing.T) {
	dir := assignEnv(t, "demo", "other")
	t.Chdir(dir)
	b := &tuiBackend{}
	if _, err := b.SetProject("demo", dir, tui.AlwaysOn); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SetProject("other", dir, tui.OnDemand); err != nil {
		t.Fatal(err)
	}
	d, err := b.Load("")
	if err != nil || d.Project == nil {
		t.Fatalf("%+v, %v", d, err)
	}
	var p tui.Project
	for _, x := range d.Projects {
		if x.ID == d.Project.ID {
			p = x
		}
	}
	if !slices.Equal(p.AlwaysOn, []string{"demo"}) || p.Folders != 1 {
		t.Errorf("always-on %v, folders %d", p.AlwaysOn, p.Folders)
	}
}

func TestListTextHasNoTerminalControlCharacters(t *testing.T) {
	got := cut("Read\x1b[2J PDFs\x07 and\nmore", 80)
	if strings.ContainsAny(got, "\x1b\x07") || got != "Read[2J PDFs and more" {
		t.Errorf("%q", got)
	}
}
