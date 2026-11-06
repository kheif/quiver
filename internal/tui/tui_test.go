package tui

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// fake is a backend that keeps the assignments the panel changes, so a test
// can follow a skill through several changes.
type fake struct {
	skills   []Skill // without assignments
	projects []Project
	sources  []Source
	glob     map[string]Mode
	proj     map[string]map[string]Mode

	calls   []string
	edited  []string
	loads   []string
	err     error
	loadErr error

	checks   map[string]Preview
	checkErr error
	applied  []string

	health      []HealthItem
	hookMissing bool
	noHome      bool
}

func (f *fake) projectByPath(path string) (string, bool) {
	for _, p := range f.projects {
		if slices.Contains(p.Paths, path) {
			return p.ID, true
		}
	}
	return "", false
}

func (f *fake) Load(projectID string) (Data, error) {
	f.loads = append(f.loads, projectID)
	if f.loadErr != nil {
		return Data{}, f.loadErr
	}
	d := Data{Version: "1.5.0", Home: "/home/q/.quiver", Sources: slices.Clone(f.sources), NoHome: f.noHome}
	here := ""
	for _, p := range f.projects {
		if p.Current {
			here = p.ID
			d.Opened = p.Name
		}
	}
	scope := projectID
	if !slices.ContainsFunc(f.projects, func(p Project) bool { return p.ID == scope }) {
		scope = here
	}
	for _, p := range f.projects {
		p.Skills = nil
		for id := range f.proj[p.ID] {
			p.Skills = append(p.Skills, id)
		}
		sort.Strings(p.Skills)
		d.Projects = append(d.Projects, p)
		if p.ID == scope {
			cp := p
			d.Project = &cp
		}
	}
	for _, s := range f.skills {
		g := f.glob[s.ID]
		s.Global, s.GlobalAlways = g != NotAssigned, g == AlwaysOn
		if scope != "" {
			pm := f.proj[scope][s.ID]
			s.Project, s.ProjectAlways = pm != NotAssigned, pm == AlwaysOn
		}
		d.Skills = append(d.Skills, s)
	}
	return d, nil
}

func (f *fake) Search(text string) ([]string, error) {
	var ids []string
	for _, s := range f.skills {
		hay := strings.ToLower(s.ID + " " + s.Description)
		ok := true
		for _, w := range strings.Fields(strings.ToLower(text)) {
			ok = ok && strings.Contains(hay, w)
		}
		if ok {
			ids = append(ids, s.ID)
		}
	}
	return ids, nil
}

func (f *fake) Details(id string) (Details, error) {
	return Details{Body: "Instructions of " + id + ".", Raw: "---\nname: " + id + "\n---\nInstructions of " + id + "."}, nil
}

func (f *fake) SetProject(id, path string, to Mode) (string, error) {
	f.calls = append(f.calls, fmt.Sprintf("project %s %s %v", id, path, to))
	if f.err != nil {
		return "", f.err
	}
	pid, ok := f.projectByPath(path)
	if !ok {
		return "", errors.New("no project in " + path)
	}
	if to == NotAssigned {
		delete(f.proj[pid], id)
	} else {
		f.proj[pid][id] = to
	}
	return "", nil
}

func (f *fake) SetGlobal(id string, to Mode) (string, error) {
	f.calls = append(f.calls, fmt.Sprintf("global %s %v", id, to))
	if f.err != nil {
		return "", f.err
	}
	if to == NotAssigned {
		delete(f.glob, id)
	} else {
		f.glob[id] = to
	}
	return "", nil
}

func (f *fake) Edit(id string) (string, error) {
	f.edited = append(f.edited, id)
	return "Saved " + id + ".", nil
}

func (f *fake) Fork(id, name string) (string, string, error) {
	f.calls = append(f.calls, "fork "+id+" "+name)
	return name, "Forked " + id + " into " + name + ".", f.err
}

func (f *fake) Check(source string, reset bool) (Preview, error) {
	f.calls = append(f.calls, "check "+source)
	if reset {
		f.calls = append(f.calls, "reset "+source)
	}
	if f.checkErr != nil {
		return Preview{}, f.checkErr
	}
	if p, ok := f.checks[source]; ok {
		return p, nil
	}
	return Preview{Source: source, From: "abc1234", To: "abc1234", UpToDate: true, Lines: []string{source + ": up to date (abc1234)"}}, nil
}

func (f *fake) Apply(source string) (string, error) {
	f.applied = append(f.applied, source)
	return source + " updated to def5678.", nil
}

func (f *fake) RemoveSource(name string) (string, error) {
	f.calls = append(f.calls, "remove-source "+name)
	return "Removed source " + name + ".", nil
}

func (f *fake) RenameProject(id, name string) error {
	f.calls = append(f.calls, "rename "+id+" "+name)
	for i := range f.projects {
		if f.projects[i].ID == id {
			f.projects[i].Name = name
		}
	}
	return nil
}

func (f *fake) ForgetProject(id string) error {
	f.calls = append(f.calls, "forget "+id)
	f.projects = slices.DeleteFunc(f.projects, func(p Project) bool { return p.ID == id })
	return nil
}

func (f *fake) AddSource(repo, name string) (string, error) {
	f.calls = append(f.calls, "add-source "+repo+" "+name)
	return "Added source " + name + " with 2 skills.", f.err
}

func (f *fake) NewSkill(name, description, projectPath string, global bool) (string, error) {
	f.calls = append(f.calls, "new "+name+" "+description+" "+projectPath)
	if global {
		f.calls = append(f.calls, "new-global "+name)
	}
	return "Created skill " + name + ".", f.err
}

func (f *fake) CheckName(name string) error {
	if strings.ContainsAny(name, " /") {
		return fmt.Errorf("invalid name %q: use lowercase letters, digits and dashes", name)
	}
	return nil
}

func (f *fake) CheckImport(path string) error {
	if path == "/nowhere" {
		return errors.New("no such file or folder: /nowhere")
	}
	return nil
}

func (f *fake) Delete(id string, force bool) (string, error) {
	f.calls = append(f.calls, fmt.Sprintf("delete %s force=%v", id, force))
	f.skills = slices.DeleteFunc(f.skills, func(s Skill) bool { return s.ID == id })
	return "Deleted skill " + id + ".", f.err
}

func (f *fake) Pin(id string) (string, error) {
	f.calls = append(f.calls, "pin "+id)
	return "Pinned " + id + ".", f.err
}

func (f *fake) Unpin(id string) (string, error) {
	f.calls = append(f.calls, "unpin "+id)
	return "Unpinned " + id + ".", f.err
}

func (f *fake) Import(path string) (string, error) {
	f.calls = append(f.calls, "import "+path)
	return "Imported 2 skills.", f.err
}

func (f *fake) Health() (HealthReport, error) {
	f.calls = append(f.calls, "health")
	return HealthReport{Items: slices.Clone(f.health), HookMissing: f.hookMissing}, nil
}

func (f *fake) Setup() (string, error) {
	f.calls = append(f.calls, "setup")
	f.noHome = false
	return "Quiver is set up.", f.err
}

func (f *fake) Install() (string, error) {
	f.calls = append(f.calls, "install")
	f.hookMissing = false
	return "Registered Quiver in your assistants.", f.err
}

func (f *fake) Uninstall() (string, error) {
	f.calls = append(f.calls, "uninstall")
	return "Removed Quiver from your assistants.", f.err
}

func (f *fake) Fix() (string, error) {
	f.calls = append(f.calls, "fix")
	return "Applied 1 fix.", f.err
}

func (f *fake) SplitHere() (string, error) {
	f.calls = append(f.calls, "split")
	return "split done", nil
}

func (f *fake) callList() string { return strings.Join(f.calls, "; ") }

// fixture: the project "app" is opened here and has db-migration; "api"
// has nothing. demo is global and on demand; commit-rules is global and
// always on.
func fixture() *fake {
	return &fake{
		skills: []Skill{
			{ID: "commit-rules", Name: "commit-rules", Source: "local", Local: true, Description: "Commit message conventions"},
			{ID: "db-migration", Name: "db-migration", Source: "local", Local: true, Description: "Rules for database migrations"},
			{ID: "demo", Name: "demo", Source: "local", Local: true, Description: "Generates friendly greetings"},
			{ID: "pdf", Name: "pdf", Source: "local", Local: true, Description: "Read and fill PDF files"},
			{ID: "team/migrate-check", Name: "migrate-check", Source: "team", Description: "Checks a migration before merge", Pin: "abc1234"},
		},
		projects: []Project{
			{Name: "api", ID: "api-1", Paths: []string{"/code/api"}, Path: "/code/api"},
			{Name: "app", ID: "app-1", Paths: []string{"/code/app"}, Path: "/code/app", Current: true},
			{Name: "gone", ID: "gone-1", Paths: []string{"/code/gone"}, Missing: true},
			{Name: "elsewhere", ID: "else-1"},
		},
		sources: []Source{
			{Name: "team", Repo: "acme/skills", Skills: 1, Commit: "abc1234", Updated: "2026-10-01", Pinned: []string{"team/migrate-check@abc1234"}},
			{Name: "anthropic", Repo: "anthropics/skills", Skills: 9, Commit: "def5678"},
		},
		glob: map[string]Mode{"demo": OnDemand, "commit-rules": AlwaysOn},
		proj: map[string]map[string]Mode{"app-1": {"db-migration": OnDemand}, "api-1": {}, "gone-1": {}, "else-1": {}},
	}
}

// drive feeds msg to m and runs the commands it returns, like the Bubble
// Tea runtime does, until none are left.
func drive(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	queue := []tea.Msg{msg}
	for steps := 0; len(queue) > 0; steps++ {
		if steps > 50 {
			t.Fatal("too many messages")
		}
		next, cmd := m.Update(queue[0])
		queue = queue[1:]
		m = next.(Model)
		if cmd != nil {
			// A command that waits, such as the blink of the cursor, is a timer
			// the runtime would deliver later; the tests do not wait for it.
			// A batch is several commands that run side by side.
			cmds := []tea.Cmd{cmd}
			for len(cmds) > 0 {
				c := cmds[0]
				cmds = cmds[1:]
				if c == nil {
					continue
				}
				done := make(chan tea.Msg, 1)
				go func() { done <- c() }()
				select {
				case out := <-done:
					if _, quit := out.(tea.QuitMsg); quit {
						return m
					}
					if batch, ok := out.(tea.BatchMsg); ok {
						cmds = append(cmds, batch...)
					} else if out != nil {
						queue = append(queue, out)
					}
				case <-time.After(50 * time.Millisecond):
				}
			}
		}
	}
	return m
}

func keyMsg(k string) tea.KeyPressMsg {
	switch k {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "pgup":
		return tea.KeyPressMsg{Code: tea.KeyPgUp}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	case "ctrl+a":
		return tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl}
	case "ctrl+u":
		return tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}
	case " ":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	}
	return tea.KeyPressMsg{Code: []rune(k)[0], Text: k}
}

func keys(t *testing.T, m Model, ks ...string) Model {
	t.Helper()
	for _, k := range ks {
		m = drive(t, m, keyMsg(k))
	}
	return m
}

// typed sends each character of s as a key.
func typed(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		m = drive(t, m, keyMsg(string(r)))
	}
	return m
}

func startSized(t *testing.T, f *fake, w, h int) Model {
	t.Helper()
	m := New(f)
	m = drive(t, m, tea.WindowSizeMsg{Width: w, Height: h})
	m = drive(t, m, m.checkHealth()())
	f.calls = slices.DeleteFunc(f.calls, func(c string) bool { return c == "health" })
	return drive(t, m, m.load(m.selectedKeys())())
}

func start(t *testing.T, f *fake) Model { return startSized(t, f, 100, 40) }

// screen is what the person sees, without styles.
func screen(m Model) string { return ansi.Strip(m.render()) }

// contains checks that the screen shows each text, also when the text was
// wrapped over several lines.
func contains(t *testing.T, m Model, wants ...string) {
	t.Helper()
	v := screen(m)
	flat := strings.Join(strings.Fields(v), " ")
	for _, w := range wants {
		if !strings.Contains(v, w) && !strings.Contains(flat, strings.Join(strings.Fields(w), " ")) {
			t.Errorf("screen lacks %q:\n%s", w, v)
		}
	}
}

func lacks(t *testing.T, m Model, nots ...string) {
	t.Helper()
	v := screen(m)
	for _, w := range nots {
		if strings.Contains(v, w) {
			t.Errorf("screen has %q:\n%s", w, v)
		}
	}
}

// listed returns the IDs of the skills of the current view, in order.
func listed(m Model) []string {
	var ids []string
	for _, i := range m.shown[secSkills] {
		ids = append(ids, m.data.Skills[i].ID)
	}
	return ids
}

func selected(m Model) string { return m.keyAt(m.sec, m.cursor[m.sec]) }

// goTo moves the cursor to the skill with the given ID.
func goTo(t *testing.T, m Model, id string) Model {
	t.Helper()
	for i := 0; selected(m) != id && i < 200; i++ {
		m = keys(t, m, "down")
	}
	if selected(m) != id {
		t.Fatalf("%s is not listed: %v", id, listed(m))
	}
	return m
}

func TestOpensOnSkillsShowingWhatIsAvailableHere(t *testing.T) {
	m := start(t, fixture())
	contains(t, m, "quiver", "project app", "1 Skills", "2 Projects", "3 Sources",
		"‹Here 3›", "Project 1", "Global 2", "All 5",
		"5 in library", "Available here · 3 skills", "NAME", "AVAILABLE HERE")
	// Global and project assignments in one list, each skill once.
	if got := listed(m); fmt.Sprint(got) != "[commit-rules db-migration demo]" {
		t.Errorf("listed %v", got)
	}
	contains(t, m, "commit-rules", "Global · always on", "db-migration", "Project", "demo", "Global")
	lacks(t, m, "G P ●", "Overview", "pdf", "migrate-check")
	// No technical ID in the header, and no box.
	lacks(t, m, "app-1", "Managing:")
}

func TestSelectedRowIsMarkedWithACharacter(t *testing.T) {
	m := start(t, fixture())
	if !strings.Contains(screen(m), "› commit-rules") {
		t.Errorf("no marker on the selected row:\n%s", screen(m))
	}
}

func TestViewsAreSeparateAndTheQueryStaysWhenSwitching(t *testing.T) {
	m := start(t, fixture())
	m = keys(t, m, "right")
	if got := listed(m); fmt.Sprint(got) != "[db-migration]" {
		t.Errorf("project assignments: %v", got)
	}
	contains(t, m, "‹Project 1›", "Project assignments · 1 skill")
	m = keys(t, m, "right")
	if got := listed(m); fmt.Sprint(got) != "[commit-rules demo]" {
		t.Errorf("global assignments: %v", got)
	}
	m = keys(t, m, "right")
	if len(listed(m)) != 5 {
		t.Errorf("all skills: %v", listed(m))
	}
	m = keys(t, m, "right") // wraps
	if m.skillView != viewHere {
		t.Errorf("view %v", m.skillView)
	}
	m = keys(t, m, "/")
	m = typed(t, m, "demo")
	m = keys(t, m, "enter", "right", "right", "right")
	contains(t, m, "/ demo", "‹All 1›", "Global 1")
	m = keys(t, m, "left")
	if got := listed(m); fmt.Sprint(got) != "[demo]" || m.queries[secSkills] != "demo" {
		t.Errorf("query lost: %v %q", got, m.queries[secSkills])
	}
}

func TestTypingInSearchNeverRunsCommands(t *testing.T) {
	f := fixture()
	m := start(t, f)
	m = keys(t, m, "/")
	m = typed(t, m, "qgajk1 2e")
	if m.filter.Value() != "qgajk1 2e" || m.sec != secSkills || m.confirm != nil || m.dlg != nil || len(f.calls) != 0 || len(f.edited) != 0 {
		t.Errorf("typing ran something: filter %q sec %v calls %v", m.filter.Value(), m.sec, f.calls)
	}
	contains(t, m, "No matches in available here.")
	m = keys(t, m, "esc")
	if m.searching || m.filter.Value() != "" {
		t.Error("esc did not clear the search")
	}
}

func TestLettersOutsideSearchGiveAHintAndChangeNothing(t *testing.T) {
	f := fixture()
	m := keys(t, start(t, f), "m", "z", "x")
	contains(t, m, "/ Search")
	if len(f.calls) != 0 {
		t.Errorf("calls %v", f.calls)
	}
}

func TestPasteGoesIntoTheSearchOnOneLine(t *testing.T) {
	m := keys(t, start(t, fixture()), "/")
	m = drive(t, m, tea.PasteMsg{Content: "pdf\nfiles\x1b[2J"})
	if got := m.filter.Value(); got != "pdf files [2J" {
		t.Errorf("search %q", got)
	}
}

func TestEmptyViewEmptyLibraryAndEmptySearchAreDifferentStates(t *testing.T) {
	f := fixture()
	f.glob = map[string]Mode{}
	f.proj["app-1"] = map[string]Mode{}
	m := startSized(t, f, 80, 24)
	contains(t, m, "No skills are available here yet.")
	m = keys(t, m, "right")
	contains(t, m, "Nothing assigned to app directly.")
	m = keys(t, m, "right")
	contains(t, m, "No skills are assigned globally.")

	empty := startSized(t, &fake{projects: fixture().projects, glob: map[string]Mode{}, proj: map[string]map[string]Mode{}}, 80, 24)
	contains(t, empty, "The library is empty.", "Press n to create a skill")
	empty = keys(t, empty, "/")
	empty = typed(t, empty, "zzz")
	contains(t, empty, "The library is empty.")
	lacks(t, empty, "No matches")

	m = keys(t, startSized(t, fixture(), 80, 24), "right", "/")
	m = typed(t, m, "pdf")
	contains(t, m, "No matches in project assignments.", "ctrl+a search all skills")
}

func TestNoMatchesOffersTheSameQueryInTheWholeLibrary(t *testing.T) {
	m := keys(t, startSized(t, fixture(), 80, 24), "right", "/")
	m = typed(t, m, "pdf")
	m = keys(t, m, "enter")
	contains(t, m, "Press a to search all skills", "a search all skills")
	m = keys(t, m, "a")
	if m.skillView != viewAll || fmt.Sprint(listed(m)) != "[pdf]" || m.queries[secSkills] != "pdf" {
		t.Errorf("view %v listed %v query %q", m.skillView, listed(m), m.queries[secSkills])
	}
	// The same from inside the search box.
	m = keys(t, start(t, fixture()), "right", "/")
	m = typed(t, m, "pdf")
	m = keys(t, m, "ctrl+a")
	if m.skillView != viewAll || m.searching || fmt.Sprint(listed(m)) != "[pdf]" {
		t.Errorf("ctrl+a: view %v searching %v listed %v", m.skillView, m.searching, listed(m))
	}
}

func TestDetailShowsBothAssignmentsSourceAndEditability(t *testing.T) {
	m := goTo(t, start(t, fixture()), "demo")
	contains(t, m, "demo", "local · editable", "ASSIGNMENTS",
		"This project", "Not assigned", "Every folder", "On demand", "Generates friendly greetings", "Instructions of demo.")
	// The description is not repeated by the preview.
	if n := strings.Count(screen(m), "Generates friendly greetings"); n != 1 {
		t.Errorf("description shown %d times", n)
	}
	m = keys(t, m, "right", "right", "right")
	m = goTo(t, m, "team/migrate-check")
	contains(t, m, "team · read-only", "pinned to abc1234", "f forks it")
}

func TestTheAcceptanceScenarioAlwaysOnInOneProjectOnly(t *testing.T) {
	f := fixture()
	m := start(t, f)
	// Choose the api project to manage.
	m = keys(t, m, "2")
	m = goTo(t, m, "api-1")
	m = keys(t, m, "enter")
	contains(t, m, "ACTIONS", "Manage skills", "Rename", "Forget")
	m = keys(t, m, "enter") // Manage skills
	if m.sec != secSkills || m.scope != "api-1" || m.skillView != viewProject {
		t.Fatalf("sec %v scope %q view %v", m.sec, m.scope, m.skillView)
	}
	contains(t, m, "managing project api", "terminal opened in app", "Nothing assigned to api directly.")

	// Find demo in the global assignments.
	m = keys(t, m, "right", "/")
	m = typed(t, m, "demo")
	m = keys(t, m, "enter")
	contains(t, m, "Global assignments · 1 skill")
	if selected(m) != "demo" {
		t.Fatalf("selected %q", selected(m))
	}
	// Edit the assignments: the project row first.
	m = keys(t, m, "enter")
	contains(t, m, "Project · api", "Not assigned", "enter change api assignment")
	m = keys(t, m, "enter")
	contains(t, m, "Project · api", "› Not assigned", "On demand", "Always on", "✓")
	m = keys(t, m, "down", "down", "enter")
	if f.callList() != "project demo /code/api Always on" {
		t.Fatalf("calls %q", f.callList())
	}
	contains(t, m, "✓ demo: Always on in api. Global unchanged (on demand).", "Applies from the next session")
	// Global is still on demand; the row shows both modes.
	contains(t, m, "Project · api", "Always on", "Every folder", "On demand")

	// Available here shows the one demo row.
	m = keys(t, m, "esc", "esc", "left", "left")
	if m.skillView != viewHere || fmt.Sprint(listed(m)) != "[commit-rules demo]" {
		t.Errorf("view %v listed %v", m.skillView, listed(m))
	}
	if n := strings.Count(screen(m), "demo"); n < 1 {
		t.Error("demo not shown")
	}

	// Remove the api assignment: it stays available globally.
	m = goTo(t, m, "demo")
	m = keys(t, m, "enter", "enter", "up", "up", "enter")
	if !strings.HasSuffix(f.callList(), "project demo /code/api Not assigned") {
		t.Fatalf("calls %q", f.callList())
	}
	contains(t, m, "demo: Removed from api. Still available globally.")
	m = keys(t, m, "esc", "right")
	if slices.Contains(listed(m), "demo") {
		t.Errorf("demo is still in the project assignments: %v", listed(m))
	}
	m = keys(t, m, "right")
	if !slices.Contains(listed(m), "demo") {
		t.Errorf("demo left the global assignments: %v", listed(m))
	}
	m = keys(t, m, "left", "left", "left")
	if !slices.Contains(listed(m), "demo") {
		t.Errorf("demo is not available here: %v", listed(m))
	}
}

func TestGlobalChangesAskFirstAndSayWhatGoes(t *testing.T) {
	f := fixture()
	m := goTo(t, start(t, f), "commit-rules")
	m = keys(t, m, "enter", "down", "enter") // Global
	contains(t, m, "Every folder", "› Always on", "✓")
	m = keys(t, m, "1", "enter")
	contains(t, m, "Remove commit-rules from every folder? It is always-on there, and that mark goes too. [y/N]")
	if len(f.calls) != 0 {
		t.Fatalf("changed before the answer: %v", f.calls)
	}
	m = keys(t, m, "n")
	contains(t, m, "Cancelled.")
	if len(f.calls) != 0 {
		t.Fatalf("a no changed state: %v", f.calls)
	}
	m = keys(t, m, "enter", "1", "enter", "y")
	if f.callList() != "global commit-rules Not assigned" {
		t.Errorf("calls %q", f.callList())
	}
	contains(t, m, "✓ commit-rules: Removed globally.", "Applies from the next session")
}

func TestChoosingTheCurrentModeChangesNothing(t *testing.T) {
	f := fixture()
	m := goTo(t, start(t, f), "db-migration")
	m = keys(t, m, "enter", "enter", "enter")
	contains(t, m, "no change, it is already on demand")
	if len(f.calls) != 0 {
		t.Errorf("calls %v", f.calls)
	}
}

func TestTheRowThatLeavesTheViewMovesTheSelectionToANeighbour(t *testing.T) {
	f := fixture()
	m := goTo(t, start(t, f), "db-migration")
	m = keys(t, m, "enter", "enter", "1", "enter") // db-migration leaves the project
	if got := listed(m); fmt.Sprint(got) != "[commit-rules demo]" {
		t.Fatalf("listed %v", got)
	}
	if selected(m) != "demo" {
		t.Errorf("selected %q, want the next row", selected(m))
	}
	contains(t, m, "db-migration: Removed from app.")
	// The last row goes: the one before it is selected.
	m = goTo(t, keys(t, m, "esc"), "demo")
	m = keys(t, m, "enter", "down", "enter", "1", "enter", "y")
	if selected(m) != "commit-rules" {
		t.Errorf("selected %q", selected(m))
	}
}

func TestNoProjectMakesTheProjectRowSayWhatToDo(t *testing.T) {
	f := fixture()
	for i := range f.projects {
		f.projects[i].Current = false
	}
	m := start(t, f)
	contains(t, m, "no project", "global skills only")
	m = keys(t, m, "right", "right") // global assignments
	m = keys(t, m, "enter")
	contains(t, m, "This project", "No project", "enter choose a project")
	m = keys(t, m, "enter")
	contains(t, m, "No project selected. Choose one in Projects (press 2)")
	if len(f.calls) != 0 || m.dlg != nil {
		t.Errorf("calls %v", f.calls)
	}
}

func TestAProjectWithoutAFolderCannotBeChanged(t *testing.T) {
	f := fixture()
	m := keys(t, start(t, f), "2")
	m = goTo(t, m, "gone-1")
	m = keys(t, m, "enter", "enter") // manage
	contains(t, m, "managing project gone")
	m = keys(t, m, "right", "right")
	m = goTo(t, m, "pdf")
	m = keys(t, m, "enter", "enter", "2", "enter")
	contains(t, m, "Project gone has no folder on this machine")
	if len(f.calls) != 0 {
		t.Errorf("calls %v", f.calls)
	}
}

func TestChangesThatFailSayWhy(t *testing.T) {
	f := fixture()
	f.err = errors.New("open /very/long/path/registry.json.tmp: permission denied")
	m := goTo(t, start(t, f), "db-migration")
	m = keys(t, m, "enter", "enter", "3", "enter")
	contains(t, m, "✗ permission denied (open /very/long/path/registry.json.tmp)")
	lacks(t, m, "✓")
}

func TestARunningOperationIsNotStartedTwice(t *testing.T) {
	f := fixture()
	f.checks = map[string]Preview{"team": {Source: "team", From: "a", To: "b", Ask: true, Lines: []string{"team: a..b"}}}
	m := keys(t, start(t, f), "3")
	m = goTo(t, m, "team")
	m, cmd := m.checkSource()
	if cmd == nil || m.pending["check:team"] == "" {
		t.Fatal("the check did not start")
	}
	contains(t, m, "… Checking team")
	m2, cmd2 := m.checkSource()
	if cmd2 != nil || !strings.Contains(m2.status, "Already running") {
		t.Errorf("a second check started: %v %q", cmd2 != nil, m2.status)
	}
}

func TestEditRunsForLocalSkillsAndKeepsTheSelection(t *testing.T) {
	execProcess = func(c tea.ExecCommand, cb tea.ExecCallback) tea.Cmd {
		return func() tea.Msg { return cb(c.Run()) }
	}
	t.Cleanup(func() { execProcess = tea.Exec })
	f := fixture()
	m := goTo(t, start(t, f), "demo")
	m = keys(t, m, "e")
	if fmt.Sprint(f.edited) != "[demo]" {
		t.Errorf("edited %v", f.edited)
	}
	contains(t, m, "✓ Saved demo.")
	if selected(m) != "demo" {
		t.Errorf("selected %q after the editor", selected(m))
	}
	if len(f.loads) < 2 {
		t.Errorf("the list was not reloaded: %v", f.loads)
	}
}

func TestSourceSkillsAreForkedNotEdited(t *testing.T) {
	f := fixture()
	m := keys(t, start(t, f), "right", "right", "right")
	m = goTo(t, m, "team/migrate-check")
	contains(t, m, "f fork", "p unpin")
	lacks(t, m, "e edit", "d delete")
	m = keys(t, m, "e")
	contains(t, m, "read-only", "Press f to fork")
	if len(f.edited) != 0 {
		t.Errorf("edited %v", f.edited)
	}
	m = keys(t, m, "f")
	contains(t, m, "FORK · team/migrate-check", "Name: migrate-check", "takes the original's place wherever team/migrate-check is assigned")
	m = keys(t, m, "enter")
	if f.callList() != "fork team/migrate-check migrate-check" {
		t.Errorf("calls %q", f.callList())
	}
	contains(t, m, "✓ Forked team/migrate-check into migrate-check.")
}

func TestForkTakesAnotherNameAndRefusesABadOne(t *testing.T) {
	f := fixture()
	m := goTo(t, keys(t, start(t, f), "right", "right", "right"), "team/migrate-check")
	m = keys(t, m, "f")
	m = typed(t, keys(t, m, "ctrl+u"), "bad name")
	m = keys(t, m, "enter")
	contains(t, m, "✗ invalid name")
	if len(f.calls) != 0 {
		t.Errorf("forked with a bad name: %q", f.callList())
	}
	m = typed(t, keys(t, m, "f", "ctrl+u"), "mine")
	keys(t, m, "enter")
	if f.callList() != "fork team/migrate-check mine" {
		t.Errorf("calls %q", f.callList())
	}
}

func TestForkRefusesAnExistingLocalName(t *testing.T) {
	f := fixture()
	m := goTo(t, keys(t, start(t, f), "right", "right", "right"), "team/migrate-check")
	m = keys(t, m, "f")
	m = typed(t, keys(t, m, "ctrl+u"), "demo")
	m = keys(t, m, "enter")
	contains(t, m, "✗ a skill named demo already exists")
	if len(f.calls) != 0 {
		t.Errorf("calls %q", f.callList())
	}
}

func TestReadingTheWholeFile(t *testing.T) {
	m := keys(t, goTo(t, start(t, fixture()), "demo"), "r")
	if !m.reading {
		t.Fatal("not reading")
	}
	contains(t, m, "SKILL.md · demo", "name: demo", "Instructions of demo.", "esc back")
	m = keys(t, m, "esc")
	if m.reading || selected(m) != "demo" {
		t.Error("esc did not return to the row")
	}
}

func TestWideTerminalPutsTheDetailBesideTheList(t *testing.T) {
	m := startSized(t, fixture(), 140, 40)
	lines := strings.Split(screen(m), "\n")
	var both bool
	for _, l := range lines {
		if strings.Contains(l, "commit-rules") && strings.Contains(l, "│") && strings.Contains(l, "commit-rules") {
			both = true
		}
	}
	if !both {
		t.Errorf("no list row beside the detail:\n%s", screen(m))
	}
	contains(t, m, "local · editable", "ASSIGNMENTS", "This project", "Every folder")
	// Both panes start at the top of the body.
	var first string
	for _, l := range lines {
		if strings.Contains(l, "NAME") && strings.Contains(l, "AVAILABLE HERE") {
			first = l
		}
	}
	if first == "" {
		t.Errorf("no list header:\n%s", screen(m))
	}
}

func TestOneSkillShowsItsPreviewNextToTheRow(t *testing.T) {
	f := fixture()
	f.skills = f.skills[:1]
	m := startSized(t, f, 140, 40)
	contains(t, m, "commit-rules", "Instructions of commit-rules.")
	for _, l := range strings.Split(screen(m), "\n") {
		if strings.Contains(l, "commit-rules") && strings.Contains(l, "│") {
			return
		}
	}
	t.Errorf("preview not beside the row:\n%s", screen(m))
}

func TestNarrowTerminalOpensTheDetailWithEnterAndEscReturnsToTheSameRowAndSearch(t *testing.T) {
	m := startSized(t, fixture(), 80, 24)
	lacks(t, m, "ASSIGNMENTS", "│")
	m = keys(t, m, "right", "right", "right", "/")
	m = typed(t, m, "migr")
	m = keys(t, m, "enter")
	want := selected(m)
	m = keys(t, m, "down", "enter")
	contains(t, m, "ASSIGNMENTS", "This project", "Every folder")
	lacks(t, m, "ASSIGNED")
	m = keys(t, m, "esc")
	contains(t, m, "ASSIGNED", "/ migr")
	lacks(t, m, "ASSIGNMENTS")
	if selected(m) == "" || m.queries[secSkills] != "migr" {
		t.Errorf("selected %q (was %q), query %q", selected(m), want, m.queries[secSkills])
	}
	// The detail opened for the selected row.
	m = keys(t, m, "enter")
	contains(t, m, selected(m))
}

func TestResizingWhileTheDetailIsOpenKeepsTheFocus(t *testing.T) {
	m := startSized(t, fixture(), 140, 40)
	m = keys(t, m, "enter")
	if m.focus != focusDetail {
		t.Fatal("no detail focus")
	}
	m = drive(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	contains(t, m, "ASSIGNMENTS", "enter change app assignment")
	lacks(t, m, "AVAILABLE HERE")
	m = drive(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	contains(t, m, "NAME", "ASSIGNMENTS")
	if m.focus != focusDetail {
		t.Error("the focus was lost")
	}
}

func TestFooterNamesWhatEachKeyDoes(t *testing.T) {
	m := goTo(t, startSized(t, fixture(), 140, 40), "demo")
	contains(t, m, "enter assign", "r read", "e edit", "d delete", "/ search")
	lacks(t, m, "space assign")
	m = keys(t, m, "enter")
	contains(t, m, "enter change app assignment", "esc back to list")
	m = keys(t, m, "down")
	contains(t, m, "enter change global assignment")
	m = keys(t, m, "enter")
	contains(t, m, "enter apply", "esc cancel")
}

func TestTabMovesBetweenListAndDetail(t *testing.T) {
	m := start(t, fixture())
	m = keys(t, m, "tab")
	if m.focus != focusDetail {
		t.Error("tab did not reach the detail")
	}
	m = keys(t, m, "tab")
	if m.focus != focusList {
		t.Error("tab did not return")
	}
	m = keys(t, m, "enter", "shift+tab")
	if m.focus != focusList {
		t.Error("shift+tab did not return")
	}
}

func TestEscReturnsOneContextAtATime(t *testing.T) {
	m := goTo(t, start(t, fixture()), "demo")
	m = keys(t, m, "enter", "enter")
	if m.dlg == nil {
		t.Fatal("no dialog")
	}
	m = keys(t, m, "esc")
	if m.dlg != nil || m.focus != focusDetail {
		t.Errorf("dialog %v focus %v", m.dlg, m.focus)
	}
	if m.rowCursor != 0 {
		t.Errorf("row %d", m.rowCursor)
	}
	m = keys(t, m, "esc")
	if m.focus != focusList || selected(m) != "demo" {
		t.Errorf("focus %v selected %q", m.focus, selected(m))
	}
}

func TestSearchWithNoResultsClearsTheDetail(t *testing.T) {
	m := startSized(t, fixture(), 140, 40)
	contains(t, m, "ASSIGNMENTS")
	m = keys(t, m, "/")
	m = typed(t, m, "zzz")
	contains(t, m, "No skill selected.")
	lacks(t, m, "ASSIGNMENTS", "Instructions of")
}

func TestProjectsListDetailAndActions(t *testing.T) {
	f := fixture()
	f.proj["app-1"]["demo"] = OnDemand
	m := keys(t, start(t, f), "2")
	contains(t, m, "2 Projects", "NAME / FOLDER", "app", "api", "gone", "elsewhere", "managing", "missing", "4 registered")
	m = goTo(t, m, "app-1")
	contains(t, m, "ID app-1", "/code/app", "ASSIGNED DIRECTLY · 2", "  db-migration", "  demo",
		"AVAILABLE THROUGH GLOBAL ASSIGNMENTS · 1", "  commit-rules", "Manage skills", "Rename", "Forget", "Split from a copy")
	m = goTo(t, m, "gone-1")
	contains(t, m, "no longer exists")
	m = goTo(t, m, "else-1")
	contains(t, m, "not seen on this machine")
	lacks(t, m, "Split from a copy")
	// Search by name.
	m = keys(t, m, "/")
	m = typed(t, m, "ap")
	if len(m.shown[secProjects]) != 2 {
		t.Errorf("shown %v", m.shown[secProjects])
	}
}

func TestRenameAsksForANameAndCancels(t *testing.T) {
	f := fixture()
	m := goTo(t, keys(t, start(t, f), "2"), "api-1")
	m = keys(t, m, "r")
	contains(t, m, "RENAME PROJECT", "Name: api")
	m = keys(t, m, "esc")
	if m.asking != nil || len(f.calls) != 0 {
		t.Error("esc did not cancel")
	}
	m = keys(t, m, "r")
	for range "api" {
		m = keys(t, m, "backspace")
	}
	m = typed(t, m, "web")
	m = keys(t, m, "enter")
	if f.callList() != "rename api-1 web" {
		t.Errorf("calls %q", f.callList())
	}
	contains(t, m, "✓ Renamed project to web.")
}

func TestForgetExplainsTheEffectBeforeAsking(t *testing.T) {
	f := fixture()
	m := goTo(t, keys(t, start(t, f), "2"), "app-1")
	m = keys(t, m, "f")
	contains(t, m, "Forget project app [app-1] and its 1 skill assignment?", "The folder and its .quiver.json stay", "[y/N]")
	m = keys(t, m, "x")
	if len(f.calls) != 0 {
		t.Errorf("calls %v", f.calls)
	}
	m = keys(t, m, "f", "y")
	if f.callList() != "forget app-1" {
		t.Errorf("calls %q", f.callList())
	}
	contains(t, m, "✓ Forgot project app.")
	if m.scope != "" {
		t.Errorf("scope %q", m.scope)
	}
}

func TestSplitIsOnlyForTheProjectOfThisFolder(t *testing.T) {
	f := fixture()
	m := goTo(t, keys(t, start(t, f), "2"), "api-1")
	m = keys(t, m, "s")
	if m.confirm != nil {
		t.Error("split offered for another project")
	}
	m = goTo(t, m, "app-1")
	m = keys(t, m, "s")
	contains(t, m, "Give app its own project ID?")
	m = keys(t, m, "y")
	if f.callList() != "split" {
		t.Errorf("calls %q", f.callList())
	}
}

func TestChoosingAProjectDoesNotChangeTheRegistry(t *testing.T) {
	f := fixture()
	m := goTo(t, keys(t, start(t, f), "2"), "api-1")
	m = keys(t, m, "m")
	if len(f.calls) != 0 || m.scope != "api-1" || m.data.Project.Name != "api" {
		t.Errorf("calls %v scope %q", f.calls, m.scope)
	}
}

func TestSourcesNeverSayUpToDateBeforeACheck(t *testing.T) {
	f := fixture()
	m := keys(t, start(t, f), "3")
	contains(t, m, "3 Sources", "NAME / REPOSITORY", "team", "anthropic", "Not checked", "‹All 2›", "Project 0", "Global 0")
	lacks(t, m, "Up to date")
	m = goTo(t, m, "team")
	contains(t, m, "Repository acme/skills", "Not checked", "Commit abc1234 · 2026-10-01", "Pinned team/migrate-check@abc1234",
		"Check for updates", "Remove source", "SKILLS · 1", "team/migrate-check", "not assigned")
	m = keys(t, m, "u")
	if f.callList() != "check team" {
		t.Errorf("calls %q", f.callList())
	}
	contains(t, m, "✓ team is up to date (abc1234).", "Up to date (abc1234)")
}

func TestSourceViewsFollowTheAssignments(t *testing.T) {
	f := fixture()
	f.glob["team/migrate-check"] = OnDemand
	f.proj["app-1"]["anthropic/pdf"] = OnDemand
	f.skills = append(f.skills, Skill{ID: "anthropic/pdf", Name: "pdf", Source: "anthropic", Description: "x"})
	m := keys(t, start(t, f), "3")
	contains(t, m, "‹All 2›", "Project 1", "Global 1")
	m = keys(t, m, "right")
	if len(m.shown[secSources]) != 1 || m.data.Sources[m.shown[secSources][0]].Name != "anthropic" {
		t.Errorf("used by project: %v", m.shown[secSources])
	}
	contains(t, m, "‹Project 1›", "project app")
	m = keys(t, m, "right")
	if len(m.shown[secSources]) != 1 || m.data.Sources[m.shown[secSources][0]].Name != "team" {
		t.Errorf("used globally: %v", m.shown[secSources])
	}
	contains(t, m, "global")
}

func TestSourcesShowARemoteFreeLibraryHonestly(t *testing.T) {
	f := fixture()
	f.sources = nil
	m := keys(t, startSized(t, f, 80, 24), "3")
	contains(t, m, "No remote sources.", "Press a to add one", "Your 4 skills are local and need no source.")
	lacks(t, m, "Up to date")
}

func TestAnUpdateThatChangesAnAssignedSkillShowsAPreviewAndWaitsForYes(t *testing.T) {
	f := fixture()
	f.checks = map[string]Preview{"team": {Source: "team", From: "abc1234", To: "def5678", Ask: true,
		Lines: []string{"team: abc1234..def5678", "  changed  team/migrate-check (assigned: global)", "           M SKILL.md  +2 -1"}}}
	m := keys(t, start(t, f), "3")
	m = goTo(t, m, "team")
	m = keys(t, m, "u")
	if m.preview == nil {
		t.Fatal("no preview")
	}
	contains(t, m, "UPDATE · TEAM", "team: abc1234..def5678", "changed  team/migrate-check (assigned: global)", "y apply the update")
	if len(f.applied) != 0 {
		t.Fatal("applied before the yes")
	}
	m = keys(t, m, "n")
	contains(t, m, "team was not updated.")
	if len(f.applied) != 0 {
		t.Fatal("applied after a no")
	}
	m = keys(t, m, "u", "y")
	if fmt.Sprint(f.applied) != "[team]" {
		t.Errorf("applied %v", f.applied)
	}
	contains(t, m, "✓ team updated to def5678.", "Up to date")
}

func TestAnUpdateThatChangesNothingAssignedIsAppliedAtOnceAsQuiverUpdateDoes(t *testing.T) {
	f := fixture()
	f.checks = map[string]Preview{"team": {Source: "team", From: "a", To: "b", Lines: []string{"team: a..b"}}}
	m := goTo(t, keys(t, start(t, f), "3"), "team")
	m = keys(t, m, "u")
	if m.preview != nil || fmt.Sprint(f.applied) != "[team]" {
		t.Errorf("preview %v applied %v", m.preview, f.applied)
	}
}

func TestACheckThatFailsDoesNotLockTheScreenOrClaimUpToDate(t *testing.T) {
	f := fixture()
	f.checkErr = errors.New("could not resolve host")
	m := goTo(t, keys(t, start(t, f), "3"), "team")
	m = keys(t, m, "u")
	contains(t, m, "✗ team: could not resolve host", "Check failed")
	lacks(t, m, "Up to date")
	m = keys(t, m, "1")
	contains(t, m, "1 Skills")
}

func TestCheckAllChecksEverySourceAndAppliesNothing(t *testing.T) {
	f := fixture()
	f.checks = map[string]Preview{"team": {Source: "team", From: "a", To: "b", Ask: true, Lines: []string{"x"}}}
	m := keys(t, start(t, f), "3", "U")
	if f.callList() != "check team; check anthropic" && f.callList() != "check anthropic; check team" {
		t.Errorf("calls %q", f.callList())
	}
	contains(t, m, "team: changes available, press u on it", "anthropic: up to date")
	if len(f.applied) != 0 || m.preview != nil {
		t.Error("check all applied or opened a preview")
	}
}

func TestRemovingASourceShowsTheAssignmentsThatGo(t *testing.T) {
	f := fixture()
	f.glob["team/migrate-check"] = OnDemand
	f.proj["api-1"]["team/migrate-check"] = OnDemand
	m := goTo(t, keys(t, start(t, f), "3"), "team")
	m = keys(t, m, "x")
	contains(t, m, "Remove source team and its 1 skill?", "These assignments are removed with it:", "globally: team/migrate-check", "in api: team/migrate-check", "[y/N]")
	m = keys(t, m, "n")
	if len(f.calls) != 0 {
		t.Fatalf("calls %v", f.calls)
	}
	keys(t, m, "x", "y")
	if f.callList() != "remove-source team" {
		t.Errorf("calls %q", f.callList())
	}
	m = goTo(t, keys(t, start(t, fixture()), "3"), "anthropic")
	m = keys(t, m, "x")
	contains(t, m, "None of its skills is assigned.")
}

func TestHelpIsAboutTheCurrentViewAndShowsTheVersion(t *testing.T) {
	m := keys(t, start(t, fixture()), "?")
	contains(t, m, "KEYS", "switch view", "edit the project and global assignments", "ABOUT", "Quiver 1.5.0", "Library /home/q/.quiver")
	m = keys(t, m, "x")
	lacks(t, m, "ABOUT")
	m = keys(t, keys(t, m, "3"), "?")
	contains(t, m, "check one source for updates", "remove the source")
	lacks(t, m, "read the whole SKILL.md")
}

func TestLoadErrorIsShownInsteadOfAnEmptyLibrary(t *testing.T) {
	f := fixture()
	f.loadErr = errors.New("decode /home/q/registry.json: invalid character")
	m := start(t, f)
	contains(t, m, "Quiver cannot read its data.", "invalid character (decode /home/q/registry.json)", "run quiver doctor")
	lacks(t, m, "The library is empty")
	m = keys(t, m, "g", "/", "x")
	contains(t, m, "Quiver cannot read its data.")
	if _, cmd := m.Update(keyMsg("q")); cmd == nil {
		t.Error("q did not quit")
	}
}

func TestSkillTextCannotAlterTheScreen(t *testing.T) {
	f := fixture()
	f.skills[0].Description = "first line\nsecond line\r\x1b[31mred\x1b[0m \u202Ereversed\x07\tend"
	f.skills[3].ID = "evil\x1b]0;title\x07"
	f.skills[3].Name = "evil"
	f.glob["evil\x1b]0;title\x07"] = OnDemand
	for _, size := range [][2]int{{80, 24}, {40, 12}, {140, 40}} {
		m := startSized(t, f, size[0], size[1])
		for _, press := range []string{"1", "right", "right", "right"} {
			m = keys(t, m, press)
			out := m.render()
			for _, bad := range []string{"\x1b[31m", "\x1b]0;", "\x07", "\u202e", "\r"} {
				if strings.Contains(out, bad) {
					t.Errorf("%v: output holds %q", size, bad)
				}
			}
			if n := len(strings.Split(out, "\n")); n != size[1] {
				t.Errorf("%v: %d lines", size, n)
			}
		}
	}
}

func TestLayoutFillsTheTerminalExactly(t *testing.T) {
	f := fixture()
	for i := 0; i < 120; i++ {
		f.skills = append(f.skills, Skill{ID: fmt.Sprintf("extra-%03d-with-a-rather-long-name-こんにちは世界", i), Source: "local", Local: true, Description: strings.Repeat("long description ", 12)})
	}
	f.projects[0].Name = strings.Repeat("プロジェクト", 12)
	for _, size := range [][2]int{{40, 12}, {50, 15}, {80, 24}, {100, 30}, {140, 40}, {250, 60}} {
		for _, presses := range [][]string{nil, {"right", "right", "right"}, {"2"}, {"2", "enter"}, {"3"}, {"3", "enter"}, {"?"}, {"enter"}, {"enter", "enter"}, {"r"}, {"/", "x"}, {"/", "z", "z", "z"}} {
			m := startSized(t, f, size[0], size[1])
			m = keys(t, m, presses...)
			lines := strings.Split(m.render(), "\n")
			if len(lines) != size[1] {
				t.Errorf("%v %v: %d lines", size, presses, len(lines))
			}
			for _, l := range lines {
				if w := ansi.StringWidth(l); w > size[0] {
					t.Errorf("%v %v: line of %d cells: %q", size, presses, w, ansi.Strip(l))
				}
			}
		}
	}
}

func TestPreviewAndConfirmPromptsFitTheScreen(t *testing.T) {
	f := fixture()
	var ls []string
	for i := 0; i < 60; i++ {
		ls = append(ls, fmt.Sprintf("  changed  team/skill-%d (assigned: global)", i))
	}
	f.checks = map[string]Preview{"team": {Source: "team", From: "a", To: "b", Ask: true, Lines: ls}}
	for _, size := range [][2]int{{40, 12}, {80, 24}, {140, 40}} {
		m := startSized(t, f, size[0], size[1])
		m = keys(t, m, "3")
		m = goTo(t, m, "team")
		m = keys(t, m, "u")
		if n := len(strings.Split(m.render(), "\n")); n != size[1] {
			t.Errorf("%v preview: %d lines", size, n)
		}
		m = keys(t, m, "pgdown", "down", "n", "x")
		if n := len(strings.Split(m.render(), "\n")); n != size[1] {
			t.Errorf("%v confirm: %d lines", size, n)
		}
	}
}

func TestTooSmallTerminalSaysSo(t *testing.T) {
	m := startSized(t, fixture(), 30, 8)
	out := strings.Join(strings.Fields(m.render()), " ")
	if !strings.Contains(out, "Terminal too small") || !strings.Contains(out, "press q to quit") {
		t.Errorf("%q", out)
	}
	for _, line := range strings.Split(m.render(), "\n") {
		if len([]rune(line)) > 30 {
			t.Errorf("line wider than the terminal: %q", line)
		}
	}
	if _, cmd := m.Update(keyMsg("q")); cmd == nil {
		t.Error("q did not quit")
	}
}

func TestQuitWorksOnTheTooSmallScreenWithADialogOpen(t *testing.T) {
	m := start(t, fixture())
	m = keys(t, m, "n", "a", "b")
	m = drive(t, m, tea.WindowSizeMsg{Width: 20, Height: 5})
	if _, cmd := m.Update(keyMsg("q")); cmd == nil {
		t.Error("q did not quit while a dialog was open")
	}
}

func TestFooterKeepsHelpAndQuitWhenNarrow(t *testing.T) {
	m := startSized(t, fixture(), 40, 12)
	line := ansi.Strip(m.footer(38))
	if !strings.HasSuffix(line, "? help   q quit") || ansi.StringWidth(line) > 38 {
		t.Errorf("footer %q", line)
	}
	wide := ansi.Strip(startSized(t, fixture(), 140, 40).footer(138))
	if !strings.Contains(wide, "enter assign") || !strings.HasSuffix(wide, "? help   q quit") {
		t.Errorf("footer %q", wide)
	}
}

func TestNavigationKeys(t *testing.T) {
	f := fixture()
	for i := 0; i < 100; i++ {
		f.skills = append(f.skills, Skill{ID: fmt.Sprintf("extra-%03d", i), Source: "local", Description: "x"})
	}
	m := keys(t, startSized(t, f, 80, 24), "right", "right", "right")
	if len(listed(m)) != 105 {
		t.Fatalf("listed %d", len(listed(m)))
	}
	m = keys(t, m, "end")
	if selected(m) != "team/migrate-check" {
		t.Errorf("end: %q", selected(m))
	}
	m = keys(t, m, "home", "pgdown")
	if m.cursor[secSkills] < 5 {
		t.Errorf("pgdown: cursor %d", m.cursor[secSkills])
	}
	m = keys(t, m, "pgup", "j", "k", "down", "up")
	if m.cursor[secSkills] != 0 {
		t.Errorf("cursor %d", m.cursor[secSkills])
	}
}

func TestQuit(t *testing.T) {
	m := start(t, fixture())
	for _, k := range []string{"q", "ctrl+c"} {
		if _, cmd := m.Update(keyMsg(k)); cmd == nil || fmt.Sprintf("%T", cmd()) != "tea.QuitMsg" {
			t.Errorf("%s did not quit", k)
		}
	}
	// ctrl+c quits in every mode.
	for _, setup := range [][]string{{"/"}, {"enter", "enter"}, {"?"}, {"r"}} {
		mm := keys(t, m, setup...)
		if _, cmd := mm.Update(keyMsg("ctrl+c")); cmd == nil {
			t.Errorf("ctrl+c did not quit after %v", setup)
		}
	}
}

func TestSectionKeysWorkFromEverySection(t *testing.T) {
	m := start(t, fixture())
	for _, c := range []struct {
		key  string
		want section
	}{{"2", secProjects}, {"3", secSources}, {"1", secSkills}} {
		if m = keys(t, m, c.key); m.sec != c.want {
			t.Errorf("%s: section %v", c.key, m.sec)
		}
	}
	// Each section keeps its own search.
	m = keys(t, m, "/")
	m = typed(t, m, "demo")
	m = keys(t, m, "enter", "2")
	contains(t, m, "/ Search")
	m = keys(t, m, "1")
	contains(t, m, "/ demo")
}

func TestAddSourceAsksForRepositoryAndName(t *testing.T) {
	f := fixture()
	m := keys(t, start(t, f), "3")
	contains(t, m, "a add source")
	m = keys(t, m, "a")
	contains(t, m, "ADD SOURCE", "Repository:", "add only sources you trust")
	m = typed(t, m, "acme/skills")
	m = keys(t, m, "enter")
	contains(t, m, "ADD SOURCE · acme/skills", "Name:")
	m = typed(t, m, "acme")
	m = keys(t, m, "enter")
	if f.callList() != "add-source acme/skills acme" {
		t.Errorf("calls %q", f.callList())
	}
	contains(t, m, "✓ Added source acme with 2 skills.")
}

func TestAddSourceCanBeCancelledAndNeedsARepository(t *testing.T) {
	f := fixture()
	m := keys(t, keys(t, start(t, f), "3"), "a")
	m = keys(t, m, "enter")
	contains(t, m, "A repository cannot be empty.")
	m = keys(t, keys(t, m, "a"), "esc")
	if m.asking != nil || len(f.calls) != 0 {
		t.Error("esc did not cancel")
	}
}

func TestNewSkillWithDescriptionIsCreatedInTheProject(t *testing.T) {
	f := fixture()
	m := start(t, f)
	m = keys(t, m, "n")
	contains(t, m, "NEW SKILL", "Name:")
	m = typed(t, m, "release-notes")
	m = keys(t, m, "enter")
	contains(t, m, "NEW SKILL · release-notes", "Description:")
	m = typed(t, m, "Write release notes.")
	m = keys(t, m, "enter")
	contains(t, m, "Assign to: project", "project: on demand in app")
	m = keys(t, m, "enter")
	if f.callList() != "new release-notes Write release notes. /code/app" {
		t.Errorf("calls %q", f.callList())
	}
	contains(t, m, "✓ Created skill release-notes.")
}

func TestNewSkillWithoutDescriptionOpensTheEditor(t *testing.T) {
	execProcess = func(c tea.ExecCommand, cb tea.ExecCallback) tea.Cmd {
		return func() tea.Msg { return cb(c.Run()) }
	}
	t.Cleanup(func() { execProcess = tea.Exec })
	f := fixture()
	m := keys(t, start(t, f), "n")
	m = typed(t, m, "notes")
	keys(t, m, "enter", "enter", "enter")
	if len(f.calls) != 1 || !strings.HasPrefix(f.calls[0], "new notes  ") {
		t.Errorf("calls %q", f.callList())
	}
}

func TestImportConfirmsBeforeItMovesFiles(t *testing.T) {
	f := fixture()
	m := keys(t, start(t, f), "i")
	contains(t, m, "IMPORT SKILLS", "Folder or file:")
	m = typed(t, m, "~/pack")
	m = keys(t, m, "enter")
	contains(t, m, "Import the skills in ~/pack", "its original is deleted", "[y/N]")
	if len(f.calls) != 0 {
		t.Fatal("imported before the answer")
	}
	m = keys(t, m, "y")
	if f.callList() != "import ~/pack" {
		t.Errorf("calls %q", f.callList())
	}
	contains(t, m, "✓ Imported 2 skills.")
}
