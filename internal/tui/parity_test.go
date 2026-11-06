package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// healthFixture is a library whose setup has one error, one warning and two
// good checks.
func healthFixture() *fake {
	f := fixture()
	f.health = []HealthItem{
		{Level: "ok", Title: "Quiver home layout: /home/q/.quiver"},
		{Level: "warn", Title: "no Quiver session-start hook in Claude Code; run: quiver install"},
		{Level: "error", Title: "library skills that cannot be used", Details: []string{"local/broken: missing frontmatter"}},
		{Level: "ok", Title: "registry.json"},
	}
	f.hookMissing = true
	return f
}

func TestHealthListsProblemsFirstAndTheHeaderCountsThem(t *testing.T) {
	m := start(t, healthFixture())
	contains(t, m, "! 2 problems", "4 Health")
	m = keys(t, m, "4")
	contains(t, m, "Health", "‹Problems 2›", "All checks 4", "CHECK", "RESULT")
	if got := healthListed(m); len(got) != 2 || got[0] != "library skills that cannot be used" {
		t.Errorf("problems %v, the error first", got)
	}
	lacks(t, m, "! 2 problems")
	contains(t, m, "missing frontmatter", "i  set up the MCP server and hooks")
	m = keys(t, m, "right")
	if got := healthListed(m); len(got) != 4 || got[3] != "registry.json" {
		t.Errorf("every check %v, the good ones last", got)
	}
}

// healthListed is the titles of the checks in the list.
func healthListed(m Model) []string {
	var out []string
	for _, i := range m.shown[secHealth] {
		out = append(out, m.health[i].Title)
	}
	return out
}

func TestHealthOpensOnEveryCheckWhenThereAreNoProblems(t *testing.T) {
	f := fixture()
	f.health = []HealthItem{{Level: "ok", Title: "registry.json"}}
	m := keys(t, start(t, f), "4")
	contains(t, m, "‹All checks 1›", "registry.json")
	lacks(t, m, "problems")
}

func TestHealthSaysItIsCheckingUntilTheChecksEnd(t *testing.T) {
	m := New(healthFixture())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})
	m = drive(t, m, m.load(m.selectedKeys())())
	m = keys(t, m, "4")
	contains(t, m, "Checking your setup…")
}

func TestHealthInstallAsksBeforeItEditsTheAssistants(t *testing.T) {
	f := healthFixture()
	m := keys(t, start(t, f), "4", "i")
	contains(t, m, "Register the MCP server and the session-start hooks", ".quiver-backup", "[y/N]")
	m = keys(t, m, "n")
	if len(f.calls) != 0 {
		t.Fatalf("installed before the answer: %v", f.calls)
	}
	m = keys(t, m, "i", "y")
	if f.callList() != "install; health" {
		t.Errorf("calls %q", f.callList())
	}
	contains(t, m, "✓ Registered Quiver in your assistants.")
}

func TestHealthFixUninstallAndCheckAgain(t *testing.T) {
	f := healthFixture()
	m := keys(t, start(t, f), "4", "f")
	contains(t, m, "Apply the safe fixes of quiver doctor?")
	m = keys(t, m, "y")
	m = keys(t, m, "x")
	contains(t, m, "Remove the MCP server and the hooks of Quiver from every assistant?")
	keys(t, m, "y", "r")
	if f.callList() != "fix; health; uninstall; health; health" {
		t.Errorf("calls %q", f.callList())
	}
}

func TestAlwaysOnSaysWhenTheHookIsMissing(t *testing.T) {
	f := healthFixture()
	m := start(t, f)
	m = goTo(t, m, "db-migration")
	m = keys(t, m, "enter", "enter", "down", "down", "enter")
	contains(t, m, "✓ db-migration: Always on in app.", "an assistant has no session-start hook", "Press 4, then i")
	lacks(t, m, "Applies from the next session")

	f = healthFixture()
	f.hookMissing = false
	m = goTo(t, start(t, f), "db-migration")
	m = keys(t, m, "enter", "enter", "down", "down", "enter")
	contains(t, m, "Applies from the next session.")
	lacks(t, m, "no session-start hook")
}

func TestFirstRunOffersToSetUpQuiver(t *testing.T) {
	f := fixture()
	f.noHome = true
	m := start(t, f)
	contains(t, m, "Quiver is not set up on this machine yet.", "enter set up Quiver", "q quit")
	lacks(t, m, "The library is empty")
	m = keys(t, m, "enter")
	contains(t, m, "Set up Quiver?", "/home/q/.quiver", ".quiver-backup", "[y/N]")
	m = keys(t, m, "n")
	if len(f.calls) != 0 {
		t.Fatalf("set up before the answer: %v", f.calls)
	}
	m = keys(t, m, "enter", "y")
	if !strings.HasPrefix(f.callList(), "setup") {
		t.Errorf("calls %q", f.callList())
	}
	contains(t, m, "✓ Quiver is set up.", "1 Skills")
	lacks(t, m, "not set up")
}

func TestFirstRunQuits(t *testing.T) {
	f := fixture()
	f.noHome = true
	m := start(t, f)
	if _, cmd := m.Update(keyMsg("q")); cmd == nil {
		t.Error("q did not quit")
	}
}

func TestTheFirstScreenSaysLoadingBeforeTheDataArrives(t *testing.T) {
	m := New(fixture())
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	contains(t, m, "Loading…")
	lacks(t, m, "The library is empty")
}

func TestASkillUsedInOtherProjectsIsNotCalledUnassigned(t *testing.T) {
	f := fixture()
	for i := range f.skills {
		if f.skills[i].ID == "pdf" {
			f.skills[i].UsedIn = []string{"api-1", "else-1"}
		}
	}
	m := goTo(t, keys(t, start(t, f), "right", "right", "right"), "pdf")
	contains(t, m, "pdf Other projects", "also used in api, elsewhere")

	// The same skill in the source view of its own source.
	f = fixture()
	f.skills[4].UsedIn = []string{"api-1"}
	m = goTo(t, keys(t, start(t, f), "3"), "team")
	contains(t, m, "team/migrate-check", "used in api")
	lacks(t, m, "not assigned")
}

func TestProjectsShowAlwaysOnAndCopiesAndMissingSkills(t *testing.T) {
	f := fixture()
	f.projects[1].AlwaysOn = []string{"db-migration"}
	f.projects[1].Folders = 2
	f.proj["app-1"]["ghost/missing"] = OnDemand
	m := keys(t, start(t, f), "2")
	m = goTo(t, m, "app-1")
	contains(t, m, "2 skills (1 always on)", "in 2 folders",
		"The same project ID is in 2 folders", "db-migration  always on",
		"ghost/missing  not in the library", "s split")
}

func TestDeletingALocalSkillListsWhereItIsAssigned(t *testing.T) {
	f := fixture()
	for i := range f.skills {
		if f.skills[i].ID == "db-migration" {
			f.skills[i].UsedIn = []string{"app-1", "api-1"}
		}
	}
	m := goTo(t, start(t, f), "db-migration")
	contains(t, m, "d delete")
	m = keys(t, m, "d")
	contains(t, m, "Delete skill db-migration from the library?", "It is assigned in: app, api",
		"unassigns it there", "git history still has the skill", "[y/N]")
	m = keys(t, m, "n")
	if len(f.calls) != 0 {
		t.Fatalf("deleted before the answer: %v", f.calls)
	}
	m = keys(t, m, "d", "y")
	if f.callList() != "delete db-migration force=true" {
		t.Errorf("calls %q", f.callList())
	}
	contains(t, m, "✓ Deleted skill db-migration.")
	if got := listed(m); fmt.Sprint(got) != "[commit-rules demo]" {
		t.Errorf("listed %v", got)
	}
}

func TestDeletingAnUnassignedSkillNeedsNoForce(t *testing.T) {
	f := fixture()
	m := goTo(t, keys(t, start(t, f), "right", "right", "right"), "pdf")
	m = keys(t, m, "d")
	contains(t, m, "It is not assigned anywhere.")
	keys(t, m, "y")
	if f.callList() != "delete pdf force=false" {
		t.Errorf("calls %q", f.callList())
	}
}

func TestASourceSkillCannotBeDeletedOnItsOwn(t *testing.T) {
	f := fixture()
	m := goTo(t, keys(t, start(t, f), "right", "right", "right"), "team/migrate-check")
	m = keys(t, m, "d")
	contains(t, m, "cannot be deleted on its own", "Removing the source")
	if m.confirm != nil || len(f.calls) != 0 {
		t.Errorf("confirm %v, calls %v", m.confirm, f.calls)
	}
}

func TestPinAndUnpinASourceSkill(t *testing.T) {
	f := fixture()
	m := goTo(t, keys(t, start(t, f), "right", "right", "right"), "team/migrate-check")
	contains(t, m, "p unpin")
	m = keys(t, m, "p")
	contains(t, m, "Serve team/migrate-check from the current version of source team again?", "[y/N]")
	keys(t, m, "y")
	if f.callList() != "unpin team/migrate-check" {
		t.Errorf("calls %q", f.callList())
	}
	f = fixture()
	f.skills[4].Pin = ""
	m = goTo(t, keys(t, start(t, f), "right", "right", "right"), "team/migrate-check")
	contains(t, m, "p pin")
	m = keys(t, m, "p")
	contains(t, m, "Keep team/migrate-check at its current commit of source team?", "until you press p")
	keys(t, m, "y")
	if f.callList() != "pin team/migrate-check" {
		t.Errorf("calls %q", f.callList())
	}
}

func TestOnlySourceSkillsCanBePinned(t *testing.T) {
	f := fixture()
	m := keys(t, goTo(t, start(t, f), "demo"), "p")
	contains(t, m, "Only skills of a source can be pinned")
	if m.confirm != nil {
		t.Error("asked to pin a local skill")
	}
}

func TestARewrittenHistoryOffersTheReset(t *testing.T) {
	f := fixture()
	f.checkErr = fmt.Errorf("%w: team", ErrDiverged)
	m := goTo(t, keys(t, start(t, f), "3"), "team")
	m = keys(t, m, "u")
	contains(t, m, "The history of source team was rewritten upstream", "quiver update team --reset", "[y/N]")
	lacks(t, m, "✗")
	m = keys(t, m, "n")
	if f.callList() != "check team" {
		t.Errorf("reset without a yes: %q", f.callList())
	}
	f.calls = nil
	f.checkErr = nil
	keys(t, m, "u")
	f.calls = nil
	f.checkErr = fmt.Errorf("%w: team", ErrDiverged)
	m = keys(t, m, "u")
	f.checkErr = nil
	keys(t, m, "y")
	if !strings.Contains(f.callList(), "reset team") {
		t.Errorf("calls %q", f.callList())
	}
}

func TestThePreviewShowsTheDiffOnRequest(t *testing.T) {
	f := fixture()
	f.checks = map[string]Preview{"team": {
		Source: "team", From: "a", To: "b", Ask: true,
		Lines: []string{"team: a..b", "  changed  team/migrate-check (assigned: project api)"},
		Diff:  []string{"diff --git a/SKILL.md b/SKILL.md", "-old line", "+new line"},
	}}
	m := goTo(t, keys(t, start(t, f), "3"), "team")
	m = keys(t, m, "u")
	contains(t, m, "changed  team/migrate-check", "d show the changes")
	lacks(t, m, "+new line")
	m = keys(t, m, "d")
	contains(t, m, "+new line", "-old line", "d show the summary")
	lacks(t, m, "changed  team/migrate-check")
	m = keys(t, m, "d")
	contains(t, m, "changed  team/migrate-check")
}

func TestAPreviewWithoutDiffDoesNotOfferIt(t *testing.T) {
	f := fixture()
	f.checks = map[string]Preview{"team": {Source: "team", From: "a", To: "b", Ask: true, Lines: []string{"team: a..b"}}}
	m := keys(t, goTo(t, keys(t, start(t, f), "3"), "team"), "u")
	lacks(t, m, "d show the changes")
}

func TestALongQuestionIsReadInTheBodyAndScrolls(t *testing.T) {
	f := fixture()
	f.projects[0].Skills = nil
	for i := 0; i < 60; i++ {
		id := fmt.Sprintf("team/skill-%02d", i)
		f.skills = append(f.skills, Skill{ID: id, Source: "team", Description: "x"})
		f.proj["api-1"][id] = OnDemand
	}
	f.sources[0].Skills = 61
	m := goTo(t, keys(t, startSized(t, f, 80, 24), "3"), "team")
	m = keys(t, m, "x")
	contains(t, m, "CONFIRM", "Remove source team and its 61 skills?", "These assignments are removed with it:", "↑↓ read")
	contains(t, m, "more lines")
	m = keys(t, m, "pgdown", "pgdown", "pgdown", "pgdown", "pgdown")
	contains(t, m, "skill-59")
	lacks(t, m, "more lines")
	if m.confirm == nil {
		t.Fatal("scrolling cancelled the question")
	}
	m = keys(t, m, "n")
	if m.confirm != nil || len(f.calls) != 0 {
		t.Errorf("confirm %v, calls %v", m.confirm, f.calls)
	}
}

func TestQuittingWhileSomethingRunsAsksFirst(t *testing.T) {
	f := fixture()
	m := keys(t, start(t, f), "3")
	m, cmd := m.checkSource()
	if cmd == nil {
		t.Fatal("no check started")
	}
	m = keys(t, m, "q")
	contains(t, m, "Checking team is still running. Quit anyway?", "[y/N]")
	if _, quit := m.Update(keyMsg("y")); quit == nil {
		t.Error("y did not quit")
	}
	m = keys(t, m, "n")
	if m.confirm != nil {
		t.Error("n did not cancel")
	}
}

func TestRenameRefusesAnotherProjectsName(t *testing.T) {
	f := fixture()
	m := goTo(t, keys(t, start(t, f), "2"), "api-1")
	m = keys(t, m, "r", "ctrl+u")
	m = typed(t, m, "app")
	m = keys(t, m, "enter")
	contains(t, m, "✗ Another project is already named app (app-1)")
	if len(f.calls) != 0 {
		t.Errorf("renamed: %v", f.calls)
	}
}

func TestNewSkillChecksTheNameBeforeItAsksForMore(t *testing.T) {
	f := fixture()
	m := typed(t, keys(t, start(t, f), "n"), "bad name")
	m = keys(t, m, "enter")
	contains(t, m, "✗ invalid name")
	if m.asking != nil {
		t.Error("the questions went on after a bad name")
	}
	m = typed(t, keys(t, m, "n"), "demo")
	m = keys(t, m, "enter")
	contains(t, m, "✗ a skill named demo already exists")
}

func TestNewSkillCanGoToEveryFolderOrNowhere(t *testing.T) {
	for answer, want := range map[string]string{
		"global": "new notes Write them. ; new-global notes",
		"g":      "new notes Write them. ; new-global notes",
		"none":   "new notes Write them. ",
	} {
		f := fixture()
		m := typed(t, keys(t, start(t, f), "n"), "notes")
		m = typed(t, keys(t, m, "enter"), "Write them.")
		m = keys(t, m, "enter", "ctrl+u")
		m = typed(t, m, answer)
		keys(t, m, "enter")
		if f.callList() != want {
			t.Errorf("%s: calls %q, want %q", answer, f.callList(), want)
		}
	}
	f := fixture()
	m := typed(t, keys(t, start(t, f), "n"), "notes")
	m = typed(t, keys(t, m, "enter"), "Write them.")
	m = keys(t, m, "enter", "ctrl+u")
	m = typed(t, m, "somewhere")
	m = keys(t, m, "enter")
	contains(t, m, `✗ Type project, global or none, not "somewhere".`)
	if len(f.calls) != 0 {
		t.Errorf("calls %q", f.callList())
	}
}

func TestImportChecksThePathBeforeItConfirms(t *testing.T) {
	f := fixture()
	m := typed(t, keys(t, start(t, f), "i"), "/nowhere")
	m = keys(t, m, "enter")
	contains(t, m, "✗ no such file or folder: /nowhere")
	if m.confirm != nil || len(f.calls) != 0 {
		t.Errorf("confirm %v, calls %v", m.confirm, f.calls)
	}
}

func TestAFailingBackendIsShownWithItsReasonForDelete(t *testing.T) {
	f := fixture()
	f.err = errors.New("skill demo is assigned: run again with --force")
	m := keys(t, goTo(t, start(t, f), "demo"), "d", "y")
	contains(t, m, "✗ skill demo is assigned")
}

func TestLongProjectFoldersKeepTheirEnd(t *testing.T) {
	f := fixture()
	long := "/very/long/path/that/goes/on/and/on/and/on/and/on/and/on/and/on/down/to/my-project"
	f.projects[1].Paths = []string{long}
	f.projects[1].Path = long
	m := startSized(t, f, 80, 24)
	contains(t, m, "my-project")
	m = keys(t, m, "2")
	contains(t, m, "my-project")
}

func TestTheSkillListIsGroupedBySourceAndFlatWhileSearching(t *testing.T) {
	f := fixture()
	f.skills = append(f.skills, Skill{ID: "acme/zip", Name: "zip", Source: "acme", Description: "Zip files"})
	m := keys(t, start(t, f), "right", "right", "right")
	contains(t, m, "Your skills  4", "Acme  1", "Team  1", "zip")
	// Your own skills first, then the sources by name.
	if got := listed(m); fmt.Sprint(got) != "[commit-rules db-migration demo pdf acme/zip team/migrate-check]" {
		t.Errorf("order %v", got)
	}
	m = typed(t, keys(t, m, "/"), "zip")
	lacks(t, m, "Your skills", "Acme  1")
	contains(t, m, "acme/zip")
}

func TestProjectsAndSourcesSeparateTheirRowsWhenThereIsRoom(t *testing.T) {
	m := keys(t, start(t, fixture()), "2")
	lines := strings.Split(screen(m), "\n")
	for i, l := range lines {
		if strings.Contains(l, "/code/api") && i+1 < len(lines) && !strings.Contains(strings.SplitN(lines[i+1], "│", 2)[0], "  ") {
			t.Errorf("no blank line after the first project:\n%s", screen(m))
		}
	}
}
