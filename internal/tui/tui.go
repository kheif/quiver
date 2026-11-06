// Package tui is the interactive panel that opens when quiver runs without
// arguments in a terminal. It has three sections (Skills, Projects, Sources),
// each with a searchable list and a detail pane. It draws and sends keys; every
// read and every change goes through the Backend, which the CLI fills with the
// functions the commands use.
package tui

import (
	"errors"
	"fmt"
	"image/color"
	"io"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

// Mode is how a skill is assigned in one scope.
type Mode int

const (
	NotAssigned Mode = iota
	OnDemand
	AlwaysOn
)

var modeNames = [...]string{"Not assigned", "On demand", "Always on"}

func (m Mode) String() string { return modeNames[m] }

// Skill is one skill of the library and its assignments, as seen from the
// project the panel manages.
type Skill struct {
	ID, Name    string
	Source      string // "local" for the person's own skills
	Description string
	Local       bool   // it can be edited; skills from a source are read-only
	Pin         string // the commit a source skill is pinned to, if any

	Global, GlobalAlways   bool
	Project, ProjectAlways bool // in the managed project
	UsedIn                 []string
}

func modeOf(assigned, always bool) Mode {
	switch {
	case !assigned:
		return NotAssigned
	case always:
		return AlwaysOn
	}
	return OnDemand
}

// ProjectMode is the assignment in the managed project.
func (s Skill) ProjectMode() Mode { return modeOf(s.Project, s.ProjectAlways) }

// GlobalMode is the assignment for every folder.
func (s Skill) GlobalMode() Mode { return modeOf(s.Global, s.GlobalAlways) }

// Project is a registered project.
type Project struct {
	Name, ID string
	Skills   []string // IDs of the skills assigned to it directly, each once
	Paths    []string // folders where it was seen on this machine
	Path     string   // the first of Paths that exists, or ""
	Current  bool     // the folder quiver was started in
	Missing  bool     // it has folders, and none exists
	AlwaysOn []string // IDs of the skills assigned to it that are always-on
	Folders  int      // how many of Paths exist; more than one is a copy
}

// Source is a configured remote source of skills.
type Source struct {
	Name, Repo string
	Skills     int
	Commit     string   // short HEAD hash; empty when the clone is missing
	Updated    string   // date of that commit
	Pinned     []string // IDs of its skills that are pinned
}

// HealthItem is one result of the checks quiver doctor runs, or one line of
// the setup of the assistants.
type HealthItem struct {
	Level   string // error, warn or ok
	Title   string
	Details []string
}

// HealthReport is what the checks found. HookMissing is set when an installed
// assistant has no session-start hook, so always-on skills do not load there.
type HealthReport struct {
	Items       []HealthItem
	HookMissing bool
}

// Data is everything the panel shows.
type Data struct {
	// NoHome is set when Quiver has not been set up on this machine yet.
	NoHome bool
	// Project is the project the panel manages: the one chosen on the Projects
	// section, or the one of the folder quiver was started in. Nil when there
	// is none.
	Project *Project
	// Opened is the name of the project of the folder quiver was started in,
	// or "" outside a project.
	Opened   string
	Skills   []Skill
	Projects []Project
	Sources  []Source
	// Version and Home are shown in the help.
	Version, Home string
}

// Details is the file of a skill.
type Details struct{ Body, Raw string }

// Preview is what an update of a source would change.
type Preview struct {
	Source   string
	From, To string
	UpToDate bool
	// Ask is set when an assigned skill would change or go away, so the update
	// needs a yes.
	Ask   bool
	Lines []string
	// Diff is the patch of each assigned skill that changes, as quiver update
	// --diff prints it.
	Diff []string
}

// ErrDiverged is what Check returns, wrapped, when the history of a source
// was rewritten upstream. The panel then offers to take the remote history.
var ErrDiverged = errors.New("the remote history was rewritten")

// Backend is what the panel needs from the rest of quiver. Every change goes
// through the code the commands use.
type Backend interface {
	// Load reads everything. projectID names the project to manage; "" is the
	// one of the current folder.
	Load(projectID string) (Data, error)
	// Search returns the IDs of the library skills that match, best first.
	Search(text string) ([]string, error)
	Details(id string) (Details, error)
	// SetProject and SetGlobal move a skill to a mode in one scope and leave
	// the other scope alone. They return the warnings the commands print.
	SetProject(id, projectPath string, to Mode) (string, error)
	SetGlobal(id string, to Mode) (string, error)
	Edit(id string) (string, error)
	// Fork copies a source skill into the local ones and returns the new ID.
	Fork(id, name string) (newID, note string, err error)
	// Check fetches a source without applying it. Apply then applies the
	// result of the last Check of that source. With reset it takes the remote
	// history of a source whose history was rewritten.
	Check(source string, reset bool) (Preview, error)
	Apply(source string) (string, error)
	RemoveSource(name string) (string, error)
	RenameProject(id, name string) error
	ForgetProject(id string) error
	// SplitHere gives the project of the current folder its own ID.
	SplitHere() (string, error)
	// AddSource clones a repository as a source; name "" is the repository
	// owner.
	AddSource(repo, name string) (string, error)
	// NewSkill creates a local skill and assigns it to the project in the
	// folder at projectPath, or to none when that is "". Without a
	// description it opens the editor, so the caller gives it the terminal.
	NewSkill(name, description, projectPath string, global bool) (string, error)
	// CheckName and CheckImport tell early that a skill name or an import path
	// cannot work.
	CheckName(name string) error
	CheckImport(path string) error
	// Delete removes a local skill; force also unassigns it where it is
	// assigned. Pin keeps a source skill at its current commit; Unpin follows
	// the source again.
	Delete(id string, force bool) (string, error)
	Pin(id string) (string, error)
	Unpin(id string) (string, error)
	// Import moves the skills in path into the library and assigns them to
	// the project of the current folder. "" is the .claude/skills of that
	// project.
	Import(path string) (string, error)
	// Health runs the checks. It is slow (it asks the assistants), so the panel
	// asks for it on its own and not after every change.
	Health() (HealthReport, error)
	// Setup creates the Quiver home and registers Quiver in the assistants.
	Setup() (string, error)
	// Install and Uninstall set up or remove the MCP server and the hooks in
	// the assistants. Fix applies the safe fixes of quiver doctor.
	Install() (string, error)
	Uninstall() (string, error)
	Fix() (string, error)
}

type section int

const (
	secSkills section = iota
	secProjects
	secSources
	secHealth
	secCount
)

var sectionNames = [secCount]string{"Skills", "Projects", "Sources", "Health"}

type skillView int

const (
	viewHere skillView = iota
	viewProject
	viewGlobal
	viewAll
	viewCount
)

var skillViewNames = [viewCount]string{"Available here", "Project assignments", "Global assignments", "All skills"}

type sourceView int

const (
	srcAll sourceView = iota
	srcProject
	srcGlobal
	srcCount
)

var sourceViewNames = [srcCount]string{"All sources", "Used by project", "Used globally"}

type healthView int

const (
	healthProblems healthView = iota
	healthAll
	healthCount
)

var healthViewNames = [healthCount]string{"Problems", "All checks"}

type focus int

const (
	focusList focus = iota
	focusDetail
)

// The smallest terminal the layout works in, and the width from which the
// detail pane sits beside the list.
const (
	minWidth  = 40
	minHeight = 12
	wideWidth = 100
)

type (
	loadedMsg struct {
		data Data
		err  error
		keys [secCount]string
	}
	doneMsg struct {
		key  string // the pending operation that ended
		text string
		err  error
		keep string // skill ID to select after the reload
	}
	healthMsg struct {
		report HealthReport
		err    error
	}
	checkedMsg struct {
		source  string
		preview Preview
		err     error
	}
)

// confirmation is a question asked before a change.
type confirmation struct {
	prompt string
	run    func(m Model) (Model, tea.Cmd)
	offset int // the first line shown when the prompt is read in the body
}

// dialog is the choice of a mode for one scope of one skill.
type dialog struct {
	global bool
	id     string
	from   Mode
	cursor int
}

// shownLines are the lines of the summary, or of the patches after d.
func (p *previewState) shownLines() []string {
	if p.showDiff {
		return p.diff
	}
	return p.lines
}

// previewState is an update waiting for a yes.
type previewState struct {
	source   string
	lines    []string
	diff     []string
	showDiff bool
	offset   int
}

// Model is the Bubble Tea model of the panel.
type Model struct {
	be   Backend
	data Data
	// loadErr is set while the data cannot be read, so the panel says so
	// instead of showing an empty library.
	loadErr string

	sec       section
	scope     string // the project chosen on the Projects section; "" is the folder's
	skillView skillView
	srcView   sourceView
	cursor    [secCount]int
	shown     [secCount][]int // indexes into the data, after the view and the query
	// counts hold how many results each view has for the current query.
	skillCounts  [viewCount]int
	srcCounts    [srcCount]int
	healthView   healthView
	healthCounts [healthCount]int
	// health is the result of the last checks; it is empty until they ran.
	health       []HealthItem
	hookMissing  bool
	healthLoaded bool
	// loaded is set once the first load has ended; until then the panel says
	// it is loading instead of showing an empty library.
	loaded bool

	queries   [secCount]string
	filter    textinput.Model
	searching bool

	focus     focus
	rowCursor int // the selected row of the detail pane

	details map[string]Details
	reading bool
	vp      viewport.Model
	help    bool

	dlg     *dialog
	asking  *asking
	preview *previewState
	confirm *confirmation

	checks  map[string]string // what the last check of each source found
	pending map[string]string // running operations, by key, with what to say
	status  string
	failed  bool
	note    bool // the status is a hint, not the result of a change

	// bg is the terminal background, once the terminal has answered; nil
	// until then, and for terminals that never do.
	bg color.Color

	width, height int
	askWidth      int
}

// New returns the model; its first command loads the data.
func New(be Backend) Model {
	in := textinput.New()
	in.Prompt = "/ "
	return Model{
		be: be, filter: in, vp: viewport.New(),
		details: map[string]Details{}, checks: map[string]string{}, pending: map[string]string{},
		width: 80, height: 24,
	}
}

// Run opens the panel and returns when the person quits.
func Run(be Backend) error {
	_, err := tea.NewProgram(New(be)).Run()
	return err
}

// Init loads the data and asks the terminal for its background color, which
// only tints the selected row; the answer arrives later or never.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.load(m.selectedKeys()), tea.RequestBackgroundColor, m.checkHealth())
}

func (m Model) load(keys [secCount]string) tea.Cmd {
	scope := m.scope
	return func() tea.Msg {
		d, err := m.be.Load(scope)
		return loadedMsg{d, err, keys}
	}
}

// checkHealth runs the checks in the background.
func (m Model) checkHealth() tea.Cmd {
	be := m.be
	return func() tea.Msg {
		r, err := be.Health()
		return healthMsg{r, err}
	}
}

func (m Model) wide() bool { return m.width >= wideWidth }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.vp.SetWidth(max(msg.Width-4, 10))
		m.vp.SetHeight(max(msg.Height-2, 3))
		m.filter.SetWidth(max(msg.Width-28, 8))
		m.askWidth = max(msg.Width-12, 8)
		if m.asking != nil {
			m.asking.input.SetWidth(m.inputWidth(m.asking))
		}
		m.setReadContent()
	case loadedMsg:
		if msg.err != nil {
			m.loadErr = errText(msg.err)
			return m, nil
		}
		m.loadErr = ""
		m.data = cleanData(msg.data)
		m.details = map[string]Details{}
		if m.scope != "" && !slices.ContainsFunc(m.data.Projects, func(p Project) bool { return p.ID == m.scope }) {
			m.scope = ""
		}
		m.refilter(msg.keys, false)
		m.loaded = true
	case healthMsg:
		if msg.err != nil {
			m.healthLoaded = true
			m.health = []HealthItem{{Level: "error", Title: "The checks could not run", Details: []string{errText(msg.err)}}}
		} else {
			m.healthLoaded = true
			m.health, m.hookMissing = msg.report.Items, msg.report.HookMissing
		}
		for i := range m.health {
			m.health[i].Title = oneLine(m.health[i].Title)
			m.health[i].Details = cleanAll(m.health[i].Details)
		}
		keys := m.selectedKeys()
		// Health opens on its problems, or on every check when there are none.
		m.healthView = healthProblems
		if m.problems() == 0 {
			m.healthView = healthAll
		}
		m.refilter(keys, false)
	case doneMsg:
		delete(m.pending, msg.key)
		m.note = false
		m.failed = msg.err != nil
		m.status = oneLine(msg.text)
		if msg.err != nil {
			m.status = errText(msg.err)
		}
		if name, ok := strings.CutPrefix(msg.key, "apply:"); ok {
			m.checks[name] = "Up to date"
			if msg.err != nil {
				m.checks[name] = "Update failed"
			}
		}
		keys := m.selectedKeys()
		if msg.keep != "" {
			keys[secSkills] = msg.keep
		}
		switch msg.key {
		case "setup", "install", "uninstall", "fix":
			m.healthLoaded = false
			return m, tea.Batch(m.load(keys), m.checkHealth())
		}
		return m, m.load(keys)
	case checkedMsg:
		return m.checked(msg)
	case tea.BackgroundColorMsg:
		m.bg = msg.Color
	case tea.PasteMsg:
		switch {
		case m.searching:
			m.filter.SetValue(m.filter.Value() + oneLine(msg.Content))
			m.queryChanged()
		case m.asking != nil:
			m.asking.input.SetValue(m.asking.input.Value() + oneLine(msg.Content))
		}
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	return m, nil
}

func (m Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if k == "space" {
		k = " "
	}
	if k == "ctrl+c" || (k == "q" && (m.width < minWidth || m.height < minHeight)) {
		return m, tea.Quit
	}
	switch {
	case m.confirm != nil:
		return m.confirmKey(k)
	case m.preview != nil:
		return m.previewKey(k)
	case m.dlg != nil:
		return m.dialogKey(k)
	case m.asking != nil:
		return m.askKey(msg, k)
	case m.reading:
		return m.readKey(msg, k)
	case m.help:
		m.help = false
		return m, nil
	case m.searching:
		return m.searchKey(msg, k)
	case m.loadErr != "":
		if k == "q" {
			return m, tea.Quit
		}
		return m, nil
	case m.loaded && m.data.NoHome:
		return m.setupKey(k)
	}
	return m.normalKey(msg, k)
}

func (m Model) confirmKey(k string) (tea.Model, tea.Cmd) {
	c := m.confirm
	switch k {
	case "up", "k":
		c.offset = max(c.offset-1, 0)
		return m, nil
	case "down", "j":
		c.offset++
		return m, nil
	case "pgup":
		c.offset = max(c.offset-max(m.bodyHeight()-3, 1), 0)
		return m, nil
	case "pgdown":
		c.offset += max(m.bodyHeight()-3, 1)
		return m, nil
	}
	m.confirm = nil
	if k == "y" || k == "Y" {
		return c.run(m)
	}
	m.say("Cancelled.", false, true)
	return m, nil
}

// runningText names what is running, for a question about quitting.
func (m Model) runningText() string {
	labels := make([]string, 0, len(m.pending))
	for _, l := range m.pending {
		labels = append(labels, l)
	}
	slices.Sort(labels)
	return strings.ToUpper(labels[0][:1]) + labels[0][1:]
}

// say sets the status line.
func (m *Model) say(text string, failed, note bool) {
	m.status, m.failed, m.note = text, failed, note
}

func (m Model) fail(text string) Model {
	m.say(text, true, false)
	return m
}

func (m Model) readKey(msg tea.KeyPressMsg, k string) (tea.Model, tea.Cmd) {
	switch k {
	case "esc", "enter", "q", "r":
		m.reading = false
		return m, nil
	case "home":
		m.vp.GotoTop()
		return m, nil
	case "end":
		m.vp.GotoBottom()
		return m, nil
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m Model) searchKey(msg tea.KeyPressMsg, k string) (tea.Model, tea.Cmd) {
	switch k {
	case "esc":
		m.filter.SetValue("")
		m.filter.Blur()
		m.searching = false
		m.queryChanged()
		return m, nil
	case "enter":
		m.filter.Blur()
		m.searching = false
		return m, nil
	case "up", "down", "pgup", "pgdown":
		m.move(k)
		return m, nil
	case "ctrl+a":
		if m.canSearchAll() {
			m.skillView = viewAll
			m.filter.Blur()
			m.searching = false
			m.refilter(m.selectedKeys(), false)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.queryChanged()
	return m, cmd
}

// queryChanged rebuilds the lists after the search text changed. The cursor
// goes to the best match.
func (m *Model) queryChanged() {
	m.queries[m.sec] = m.filter.Value()
	m.refilter([secCount]string{}, true)
}

// canSearchAll reports whether the Skills section shows a search with no
// results that a wider view might have.
func (m Model) canSearchAll() bool {
	return m.sec == secSkills && m.skillView != viewAll && len(m.shown[secSkills]) == 0 && strings.TrimSpace(m.queries[secSkills]) != ""
}

func (m Model) normalKey(msg tea.KeyPressMsg, k string) (tea.Model, tea.Cmd) {
	switch k {
	case "q":
		if len(m.pending) > 0 {
			m.confirm = &confirmation{
				prompt: m.runningText() + " is still running. Quit anyway? It stops half way.",
				run:    func(m Model) (Model, tea.Cmd) { return m, tea.Quit },
			}
			return m, nil
		}
		return m, tea.Quit
	case "?":
		m.help = true
		return m, nil
	case "1", "2", "3", "4":
		return m.goTo(section(k[0] - '1')), nil
	case "/":
		m.focus = focusList
		m.searching = true
		return m, m.filter.Focus()
	case "tab", "shift+tab":
		if m.hasDetail() {
			if m.focus == focusList {
				m.focus, m.rowCursor = focusDetail, 0
			} else {
				m.focus = focusList
			}
		}
		return m, nil
	case "esc":
		switch {
		case m.focus == focusDetail:
			m.focus = focusList
		case m.queries[m.sec] != "":
			m.filter.SetValue("")
			m.queryChanged()
		}
		return m, nil
	}
	if m.focus == focusDetail {
		if next, cmd, ok := m.detailKey(k); ok {
			return next, cmd
		}
	} else if next, cmd, ok := m.listKey(k); ok {
		return next, cmd
	}
	if next, cmd, ok := m.shortcut(k); ok {
		return next, cmd
	}
	if msg.Text != "" {
		m.say("Press / to search, ? for the keys.", false, true)
	}
	return m, nil
}

func (m Model) goTo(s section) Model {
	m.queries[m.sec] = m.filter.Value()
	m.sec = s
	m.filter.SetValue(m.queries[s])
	m.focus = focusList
	m.help = false
	m.setReadContent()
	return m
}

// hasDetail reports whether the selected row has a detail pane to move to.
func (m Model) hasDetail() bool { return len(m.shown[m.sec]) > 0 }

// listKey runs the keys of a focused list.
func (m Model) listKey(k string) (Model, tea.Cmd, bool) {
	switch k {
	case "up", "k", "down", "j", "pgup", "pgdown", "ctrl+u", "ctrl+d", "home", "end":
		m.move(k)
		return m, nil, true
	case "left", "h", "right", "l":
		d := 1
		if k == "left" || k == "h" {
			d = -1
		}
		return m.cycleView(d), nil, true
	case "enter", " ":
		if m.hasDetail() {
			m.focus, m.rowCursor = focusDetail, 0
		} else if k == "enter" {
			m.say("Nothing is selected.", false, true)
		}
		return m, nil, true
	}
	return m, nil, false
}

// cycleView moves to the next or previous view of the section.
func (m Model) cycleView(d int) Model {
	keys := m.selectedKeys()
	switch m.sec {
	case secSkills:
		m.skillView = skillView((int(m.skillView) + d + int(viewCount)) % int(viewCount))
	case secSources:
		m.srcView = sourceView((int(m.srcView) + d + int(srcCount)) % int(srcCount))
	case secHealth:
		m.healthView = healthView((int(m.healthView) + d + int(healthCount)) % int(healthCount))
	default:
		return m
	}
	m.refilter(keys, false)
	return m
}

// move changes the cursor of the current list for a navigation key.
func (m *Model) move(k string) {
	n := len(m.shown[m.sec])
	if n == 0 {
		return
	}
	step := max(m.bodyHeight()-3, 1)
	c := m.cursor[m.sec]
	switch k {
	case "up", "k":
		c--
	case "down", "j":
		c++
	case "pgup", "ctrl+u":
		c -= step
	case "pgdown", "ctrl+d":
		c += step
	case "home":
		c = 0
	case "end":
		c = n - 1
	}
	m.cursor[m.sec] = min(max(c, 0), n-1)
	m.setReadContent()
}

// action is one selectable row of the detail pane.
type action struct {
	label, value string
	ok           bool
}

// actions returns the rows of the detail pane of the selected item.
func (m Model) actions() []action {
	switch m.sec {
	case secSkills:
		s, ok := m.selectedSkill()
		if !ok {
			return nil
		}
		project := action{label: "This project", value: "No project"}
		if p := m.data.Project; p != nil {
			project = action{label: m.projectLabel(), value: s.ProjectMode().String(), ok: true}
		}
		return []action{project, {label: "Every folder", value: s.GlobalMode().String(), ok: true}}
	case secProjects:
		p, ok := m.selectedProject()
		if !ok {
			return nil
		}
		out := []action{{label: "Manage skills", ok: true}, {label: "Rename", ok: true}, {label: "Forget", ok: true}}
		if p.Current {
			out = append(out, action{label: "Split from a copy", ok: true})
		}
		return out
	case secSources:
		if _, ok := m.selectedSource(); !ok {
			return nil
		}
		return []action{{label: "Check for updates", ok: true}, {label: "Remove source", ok: true}}
	}
	return nil
}

// healthRank orders the checks: what needs action first.
func healthRank(level string) int {
	switch level {
	case "error":
		return 0
	case "warn":
		return 1
	}
	return 2
}

func (m Model) selectedHealth() (HealthItem, bool) {
	if m.sec != secHealth || m.cursor[secHealth] >= len(m.shown[secHealth]) {
		return HealthItem{}, false
	}
	return m.health[m.shown[secHealth][m.cursor[secHealth]]], true
}

// projectLabel names the project row of a skill: "This project" for the one
// of the folder quiver was opened in, otherwise the project's name.
func (m Model) projectLabel() string {
	p := m.data.Project
	if p == nil || p.Current {
		return "This project"
	}
	return "Project · " + p.Name
}

// detailKey runs the keys of the focused detail pane.
func (m Model) detailKey(k string) (Model, tea.Cmd, bool) {
	rows := m.actions()
	switch k {
	case "up", "k":
		m.rowCursor = max(m.rowCursor-1, 0)
		return m, nil, true
	case "down", "j":
		m.rowCursor = min(m.rowCursor+1, max(len(rows)-1, 0))
		return m, nil, true
	case "enter", " ":
		if m.rowCursor < len(rows) {
			next, cmd := m.activate(m.rowCursor)
			return next, cmd, true
		}
		return m, nil, true
	}
	return m, nil, false
}

// activate runs row i of the detail pane.
func (m Model) activate(i int) (Model, tea.Cmd) {
	switch m.sec {
	case secSkills:
		s, ok := m.selectedSkill()
		if !ok {
			return m, nil
		}
		if i == 0 && m.data.Project == nil {
			return m.fail("No project selected. Choose one in Projects (press 2), or run quiver add in a project folder."), nil
		}
		d := &dialog{global: i == 1, id: s.ID, from: s.ProjectMode()}
		if i == 1 {
			d.from = s.GlobalMode()
		}
		d.cursor = int(d.from)
		m.dlg = d
	case secProjects:
		return m.projectAction(i)
	case secSources:
		if i == 0 {
			return m.checkSource()
		}
		return m.askRemoveSource()
	}
	return m, nil
}

// shortcut runs the letter keys of each section; they work from the list and
// from the detail pane.
func (m Model) shortcut(k string) (Model, tea.Cmd, bool) {
	switch m.sec {
	case secSkills:
		switch k {
		case "r":
			if _, ok := m.selectedSkill(); ok {
				m.reading = true
				m.setReadContent()
				m.vp.GotoTop()
			}
			return m, nil, true
		case "e":
			next, cmd := m.edit()
			return next, cmd, true
		case "f":
			next, cmd := m.askFork()
			return next, cmd, true
		case "d":
			next, cmd := m.askDelete()
			return next, cmd, true
		case "p":
			next, cmd := m.askPin()
			return next, cmd, true
		case "n":
			next, cmd := m.askNewSkill()
			return next, cmd, true
		case "i":
			next, cmd := m.askImport()
			return next, cmd, true
		case "a":
			if m.canSearchAll() {
				m.skillView = viewAll
				m.refilter(m.selectedKeys(), false)
				return m, nil, true
			}
		}
	case secProjects:
		idx := map[string]int{"m": 0, "r": 1, "f": 2, "s": 3}
		if i, ok := idx[k]; ok {
			if i < len(m.actions()) {
				next, cmd := m.projectAction(i)
				return next, cmd, true
			}
			return m, nil, true
		}
	case secHealth:
		switch k {
		case "i":
			next, cmd := m.askInstall()
			return next, cmd, true
		case "x":
			next, cmd := m.askUninstall()
			return next, cmd, true
		case "f":
			next, cmd := m.askFix()
			return next, cmd, true
		case "r":
			m.healthLoaded = false
			m.say("Checking again…", false, true)
			return m, tea.Batch(m.load(m.selectedKeys()), m.checkHealth()), true
		}
	case secSources:
		switch k {
		case "u":
			next, cmd := m.checkSource()
			return next, cmd, true
		case "U":
			next, cmd := m.checkAllSources()
			return next, cmd, true
		case "a":
			next, cmd := m.askAddSource()
			return next, cmd, true
		case "x":
			next, cmd := m.askRemoveSource()
			return next, cmd, true
		}
	}
	return m, nil, false
}

func (m Model) selectedSkill() (Skill, bool) {
	if m.sec != secSkills || m.cursor[secSkills] >= len(m.shown[secSkills]) {
		return Skill{}, false
	}
	return m.data.Skills[m.shown[secSkills][m.cursor[secSkills]]], true
}

func (m Model) selectedProject() (Project, bool) {
	if m.sec != secProjects || m.cursor[secProjects] >= len(m.shown[secProjects]) {
		return Project{}, false
	}
	return m.data.Projects[m.shown[secProjects][m.cursor[secProjects]]], true
}

func (m Model) selectedSource() (Source, bool) {
	if m.sec != secSources || m.cursor[secSources] >= len(m.shown[secSources]) {
		return Source{}, false
	}
	return m.data.Sources[m.shown[secSources][m.cursor[secSources]]], true
}

// keyAt is the identity of the item under cursor c of section s.
func (m Model) keyAt(s section, c int) string {
	if c < 0 || c >= len(m.shown[s]) {
		return ""
	}
	i := m.shown[s][c]
	switch s {
	case secSkills:
		return m.data.Skills[i].ID
	case secProjects:
		return m.data.Projects[i].ID
	case secHealth:
		return m.health[i].Title
	}
	return m.data.Sources[i].Name
}

// selectedKeys is the identity of the selected item of every section, so a
// reload can put each cursor back on it.
func (m Model) selectedKeys() [secCount]string {
	var keys [secCount]string
	for s := section(0); s < secCount; s++ {
		keys[s] = m.keyAt(s, m.cursor[s])
	}
	return keys
}

// inView reports whether the skill belongs to view v.
func inView(s Skill, v skillView) bool {
	switch v {
	case viewHere:
		return s.Global || s.Project
	case viewProject:
		return s.Project
	case viewGlobal:
		return s.Global
	}
	return true
}

// sourceUse returns the skills of a source that are assigned globally and to
// the managed project.
func (m Model) sourceUse(name string) (global, project []Skill) {
	for _, s := range m.data.Skills {
		if s.Source != name {
			continue
		}
		if s.Global {
			global = append(global, s)
		}
		if s.Project {
			project = append(project, s)
		}
	}
	return global, project
}

// refilter rebuilds the lists of every section from the views and the
// queries. Each cursor goes back to the item named in keys when it is still
// shown, and otherwise stays at its row, which is then the nearest one. With
// reset, the cursor of the current section goes to the first result.
func (m *Model) refilter(keys [secCount]string, reset bool) {
	m.queries[m.sec] = m.filter.Value()

	// Skills: the results of the query, counted for every view.
	text := strings.TrimSpace(m.queries[secSkills])
	order := make([]int, 0, len(m.data.Skills))
	if text == "" {
		for i := range m.data.Skills {
			order = append(order, i)
		}
		// Grouped by source, your own skills first, as the list shows them.
		slices.SortStableFunc(order, func(a, b int) int {
			ga, gb := skillGroup(m.data.Skills[a]), skillGroup(m.data.Skills[b])
			switch {
			case ga == gb:
				return 0
			case ga == "Your skills":
				return -1
			case gb == "Your skills":
				return 1
			}
			return strings.Compare(ga, gb)
		})
	} else {
		ids, err := m.be.Search(text)
		if err != nil {
			m.say(errText(err), true, false)
		}
		for _, id := range ids {
			if i := slices.IndexFunc(m.data.Skills, func(s Skill) bool { return s.ID == id }); i >= 0 {
				order = append(order, i)
			}
		}
	}
	m.skillCounts = [viewCount]int{}
	m.shown[secSkills] = m.shown[secSkills][:0]
	for _, i := range order {
		for v := skillView(0); v < viewCount; v++ {
			if inView(m.data.Skills[i], v) {
				m.skillCounts[v]++
			}
		}
		if inView(m.data.Skills[i], m.skillView) {
			m.shown[secSkills] = append(m.shown[secSkills], i)
		}
	}

	// Projects: by name, ID or folder.
	text = strings.ToLower(strings.TrimSpace(m.queries[secProjects]))
	m.shown[secProjects] = m.shown[secProjects][:0]
	for i, p := range m.data.Projects {
		if text == "" || strings.Contains(strings.ToLower(p.Name+" "+p.ID+" "+strings.Join(p.Paths, " ")), text) {
			m.shown[secProjects] = append(m.shown[secProjects], i)
		}
	}

	// Sources: by name or repository, in the three views.
	text = strings.ToLower(strings.TrimSpace(m.queries[secSources]))
	m.srcCounts = [srcCount]int{}
	m.shown[secSources] = m.shown[secSources][:0]
	for i, s := range m.data.Sources {
		if text != "" && !strings.Contains(strings.ToLower(s.Name+" "+s.Repo), text) {
			continue
		}
		global, project := m.sourceUse(s.Name)
		in := [srcCount]bool{true, len(project) > 0, len(global) > 0}
		for v := range in {
			if in[v] {
				m.srcCounts[v]++
			}
		}
		if in[m.srcView] {
			m.shown[secSources] = append(m.shown[secSources], i)
		}
	}

	// Health: the checks, problems first, in the two views.
	text = strings.ToLower(strings.TrimSpace(m.queries[secHealth]))
	m.healthCounts = [healthCount]int{}
	m.shown[secHealth] = m.shown[secHealth][:0]
	var health []int
	for i, h := range m.health {
		if text == "" || strings.Contains(strings.ToLower(h.Title+" "+strings.Join(h.Details, " ")), text) {
			health = append(health, i)
		}
	}
	slices.SortStableFunc(health, func(a, b int) int {
		return healthRank(m.health[a].Level) - healthRank(m.health[b].Level)
	})
	for _, i := range health {
		problem := healthRank(m.health[i].Level) < 2
		m.healthCounts[healthAll]++
		if problem {
			m.healthCounts[healthProblems]++
		}
		if problem || m.healthView == healthAll {
			m.shown[secHealth] = append(m.shown[secHealth], i)
		}
	}

	for s := section(0); s < secCount; s++ {
		c := min(m.cursor[s], max(len(m.shown[s])-1, 0))
		switch {
		case s == m.sec && reset:
			c = 0
		case keys[s] != "":
			for at := range m.shown[s] {
				if m.keyAt(s, at) == keys[s] {
					c = at
				}
			}
		}
		m.cursor[s] = c
	}
	if len(m.shown[m.sec]) == 0 {
		m.focus = focusList
	}
	m.rowCursor = min(m.rowCursor, max(len(m.actions())-1, 0))
	m.setReadContent()
}

// detailsOf returns the file of a skill, read once per load.
func (m Model) detailsOf(id string) (Details, error) {
	if d, ok := m.details[id]; ok {
		return d, nil
	}
	d, err := m.be.Details(id)
	if err != nil {
		return Details{}, err
	}
	d = Details{Body: lines(d.Body), Raw: lines(d.Raw)}
	m.details[id] = d
	return d, nil
}

func (m *Model) setReadContent() {
	if s, ok := m.selectedSkill(); ok {
		d, err := m.detailsOf(s.ID)
		if err != nil {
			d.Raw = errText(err)
		}
		m.vp.SetContent(wrap(d.Raw, max(m.width-4, 10)))
	}
}

// --- changes -----------------------------------------------------------

// start marks an operation as running. It reports false when the same one is
// already running, so a repeated key does not start it twice.
func (m *Model) start(key, label string) bool {
	if _, busy := m.pending[key]; busy {
		m.say("Already running: "+label, false, true)
		return false
	}
	m.pending[key] = label
	m.say("", false, false)
	return true
}

func (m Model) dialogKey(k string) (tea.Model, tea.Cmd) {
	d := m.dlg
	switch k {
	case "esc", "q":
		m.dlg = nil
	case "up", "k":
		d.cursor = max(d.cursor-1, 0)
	case "down", "j":
		d.cursor = min(d.cursor+1, len(modeNames)-1)
	case "1", "2", "3":
		d.cursor = int(k[0] - '1')
	case "enter", " ":
		m.dlg = nil
		s, ok := m.skillByID(d.id)
		if !ok {
			return m, nil
		}
		to := Mode(d.cursor)
		if to == d.from {
			m.say(fmt.Sprintf("%s: no change, it is already %s.", d.id, strings.ToLower(to.String())), false, true)
			return m, nil
		}
		return m.setMode(s, d.global, to)
	}
	return m, nil
}

func (m Model) skillByID(id string) (Skill, bool) {
	for _, s := range m.data.Skills {
		if s.ID == id {
			return s, true
		}
	}
	return Skill{}, false
}

// setMode moves a skill to a mode in one scope. A change for every folder
// asks first.
func (m Model) setMode(s Skill, global bool, to Mode) (Model, tea.Cmd) {
	if !global {
		if m.data.Project == nil {
			return m.fail("No project selected."), nil
		}
		if m.data.Project.Path == "" {
			return m.fail(fmt.Sprintf("Project %s has no folder on this machine, so its skills cannot be changed here.", m.data.Project.Name)), nil
		}
		return m.runSet(s, false, to)
	}
	from := s.GlobalMode()
	var q string
	switch {
	case to == NotAssigned && from == AlwaysOn:
		q = fmt.Sprintf("Remove %s from every folder? It is always-on there, and that mark goes too.", s.ID)
	case to == NotAssigned:
		q = fmt.Sprintf("Remove %s from every folder?", s.ID)
	case to == OnDemand && from == AlwaysOn:
		q = fmt.Sprintf("Stop loading %s before the first reply in every folder? It stays available.", s.ID)
	case to == OnDemand:
		q = fmt.Sprintf("Make %s available in every folder?", s.ID)
	default:
		q = fmt.Sprintf("Make %s available in every folder and load it before the first reply?", s.ID)
	}
	m.confirm = &confirmation{prompt: q, run: func(m Model) (Model, tea.Cmd) { return m.runSet(s, true, to) }}
	return m, nil
}

// runSet starts the change; its message names the skill and what changed.
func (m Model) runSet(s Skill, global bool, to Mode) (Model, tea.Cmd) {
	scope := "global"
	if !global {
		scope = m.data.Project.ID
	}
	key := "set:" + s.ID + ":" + scope
	if !m.start(key, "changing "+s.ID) {
		return m, nil
	}
	var proj, path string
	if p := m.data.Project; p != nil {
		proj, path = p.Name, p.Path
	}
	text := s.ID + ": " + setMessage(s, proj, global, to, m.hookMissing)
	be := m.be
	return m, func() tea.Msg {
		var warn string
		var err error
		if global {
			warn, err = be.SetGlobal(s.ID, to)
		} else {
			warn, err = be.SetProject(s.ID, path, to)
		}
		if err != nil {
			return doneMsg{key: key, err: err, keep: s.ID}
		}
		if warn != "" {
			text += " " + warn
		}
		return doneMsg{key: key, text: text, keep: s.ID}
	}
}

// setMessage says what a change did, from the state before it.
func setMessage(s Skill, project string, global bool, to Mode, hookMissing bool) string {
	var b strings.Builder
	where, otherWhere, other, from := "in "+project, "globally", s.GlobalMode(), s.ProjectMode()
	if global {
		where, otherWhere, other, from = "in every folder", "in "+project, s.ProjectMode(), s.GlobalMode()
	}
	switch to {
	case NotAssigned:
		if global {
			b.WriteString("Removed globally.")
		} else {
			b.WriteString("Removed from " + project + ".")
		}
		if other != NotAssigned {
			b.WriteString(" Still available " + otherWhere + ".")
		}
	case OnDemand:
		b.WriteString("On demand " + where + ".")
	case AlwaysOn:
		b.WriteString("Always on " + where + ".")
	}
	if to != NotAssigned && other != NotAssigned {
		label := "Global"
		if global {
			label = "Project " + project
		}
		b.WriteString(" " + label + " unchanged (" + strings.ToLower(other.String()) + ").")
	}
	switch {
	case to == AlwaysOn && hookMissing:
		b.WriteString(" It does not load before the first reply yet: an assistant has no session-start hook. Press 4, then i.")
	case to == AlwaysOn || from == AlwaysOn:
		b.WriteString(" Applies from the next session.")
	}
	return b.String()
}

// execFunc runs fn while Bubble Tea has given the terminal away.
type execFunc struct{ fn func() error }

func (e *execFunc) Run() error          { return e.fn() }
func (e *execFunc) SetStdin(io.Reader)  {}
func (e *execFunc) SetStdout(io.Writer) {}
func (e *execFunc) SetStderr(io.Writer) {}

// execProcess gives the terminal to c. Tests replace it.
var execProcess = tea.Exec

// edit opens a local skill in the editor.
func (m Model) edit() (Model, tea.Cmd) {
	s, ok := m.selectedSkill()
	if !ok {
		return m, nil
	}
	if !s.Local {
		m.say(fmt.Sprintf("%s comes from source %s and is read-only. Press f to fork it into a local copy you can edit.", s.ID, s.Source), false, true)
		return m, nil
	}
	var text string
	id := s.ID
	return m, execProcess(&execFunc{func() (err error) {
		text, err = m.be.Edit(id)
		return err
	}}, func(err error) tea.Msg { return doneMsg{text: text, err: err, keep: id} })
}

// askImport asks where the skills to import are, then confirms: the CLI
// deletes the originals too.
func (m Model) askImport() (Model, tea.Cmd) {
	return m.ask(&asking{
		title: "IMPORT SKILLS", prompt: "Folder or file: ", verb: "next",
		hint: "A folder of skills, one SKILL.md or a markdown file. Leave it empty for the .claude/skills of the folder Quiver was opened in.",
		submit: func(m Model, path string) (Model, tea.Cmd) {
			if err := m.be.CheckImport(path); err != nil {
				return m.fail(errText(err)), nil
			}
			from := "the .claude/skills of this folder"
			if path != "" {
				from = path
			}
			m.confirm = &confirmation{
				prompt: fmt.Sprintf("Import the skills in %s into the library? Each is assigned to the project of this folder, and its original is deleted unless it is committed in a project's git (as quiver import --yes).", from),
				run: func(m Model) (Model, tea.Cmd) {
					return m.runProject("import", "importing", func() (string, error) { return m.be.Import(path) })
				},
			}
			return m, nil
		},
	})
}

// askAddSource asks for a repository and a name, then clones it.
func (m Model) askAddSource() (Model, tea.Cmd) {
	return m.ask(&asking{
		title: "ADD SOURCE", prompt: "Repository: ", verb: "next",
		hint: "owner/repo (GitHub), host/owner/repo, a git URL or a local folder. Its skills give instructions to your assistant and can include scripts: add only sources you trust.",
		submit: func(m Model, repo string) (Model, tea.Cmd) {
			if repo == "" {
				return m.fail("A repository cannot be empty."), nil
			}
			return m.ask(&asking{
				title: "ADD SOURCE · " + repo, prompt: "Name: ", verb: "clone",
				hint: "The skills get IDs like <name>/<skill>. Leave it empty to use the repository owner.",
				submit: func(m Model, name string) (Model, tea.Cmd) {
					m.sec = secSources
					return m.runProject("source:"+repo, "cloning "+repo, func() (string, error) { return m.be.AddSource(repo, name) })
				},
			})
		},
	})
}

// --- projects ------------------------------------------------------------

// projectAction runs row i of a project: manage, rename, forget, split.
func (m Model) projectAction(i int) (Model, tea.Cmd) {
	p, ok := m.selectedProject()
	if !ok {
		return m, nil
	}
	switch i {
	case 0:
		m.scope = p.ID
		if p.Current {
			m.scope = ""
		}
		m.skillView = viewProject
		m.queries[secSkills] = ""
		m = m.goTo(secSkills)
		m.cursor[secSkills] = 0
		m.say("", false, false)
		return m, m.load([secCount]string{})
	case 1:
		id := p.ID
		return m.ask(&asking{
			title: "RENAME PROJECT", prompt: "Name: ", verb: "rename", value: p.Name,
			hint: "The name is only a label; the project keeps its ID and skills.",
			submit: func(m Model, name string) (Model, tea.Cmd) {
				if name == "" {
					return m.fail("A project name cannot be empty."), nil
				}
				for _, other := range m.data.Projects {
					if other.ID != id && other.Name == name {
						return m.fail(fmt.Sprintf("Another project is already named %s (%s). Names must differ, or quiver projects commands cannot tell them apart.", name, other.ID)), nil
					}
				}
				return m.runProject("rename:"+id, "renaming", func() (string, error) {
					return "Renamed project to " + name + ".", m.be.RenameProject(id, name)
				})
			},
		})
	case 2:
		q := fmt.Sprintf("Forget project %s [%s] and its %s?", p.Name, p.ID, plural(len(p.Skills), "skill assignment", "skill assignments"))
		q += " The folder and its .quiver.json stay; using the folder again registers it again with no skills."
		m.confirm = &confirmation{prompt: q, run: func(m Model) (Model, tea.Cmd) {
			return m.runProject("forget:"+p.ID, "forgetting "+p.Name, func() (string, error) {
				return "Forgot project " + p.Name + ".", m.be.ForgetProject(p.ID)
			})
		}}
	case 3:
		m.confirm = &confirmation{
			prompt: fmt.Sprintf("Give %s its own project ID? Use this for a folder that was copied with its .quiver.json; it starts with the skills of the original.", p.Name),
			run:    func(m Model) (Model, tea.Cmd) { return m.runProject("split", "splitting "+p.Name, m.be.SplitHere) },
		}
	}
	return m, nil
}

func (m Model) runProject(key, label string, fn func() (string, error)) (Model, tea.Cmd) {
	if !m.start(key, label) {
		return m, nil
	}
	return m, func() tea.Msg {
		text, err := fn()
		return doneMsg{key: key, text: text, err: err}
	}
}

// asking is a one-line question that takes the place of the panes: rename a
// project, name a new skill, give a repository.
type asking struct {
	title  string
	prompt string
	hint   string
	verb   string // the enter key in the footer
	value  string
	input  textinput.Model
	// submit gets the trimmed answer. It may ask the next question.
	submit func(m Model, answer string) (Model, tea.Cmd)
}

// inputWidth is the room for the text of a question: the width of the line
// less its prompt, since the prompt is drawn inside the field.
func (m Model) inputWidth(a *asking) int {
	return max(m.askWidth-len([]rune(a.prompt)), 4)
}

// ask opens a question.
func (m Model) ask(a *asking) (Model, tea.Cmd) {
	a.input = textinput.New()
	a.input.Prompt = a.prompt
	a.input.SetValue(a.value)
	if m.askWidth > 0 {
		a.input.SetWidth(m.inputWidth(a))
	}
	m.asking = a
	return m, a.input.Focus()
}

func (m Model) askKey(msg tea.KeyPressMsg, k string) (tea.Model, tea.Cmd) {
	a := m.asking
	switch k {
	case "esc":
		m.asking = nil
		return m, nil
	case "enter":
		m.asking = nil
		return a.submit(m, strings.TrimSpace(a.input.Value()))
	}
	var cmd tea.Cmd
	a.input, cmd = a.input.Update(msg)
	return m, cmd
}

// --- sources -------------------------------------------------------------

// checkSource fetches the selected source and shows what an update would do.
func (m Model) checkSource() (Model, tea.Cmd) {
	s, ok := m.selectedSource()
	if !ok {
		return m, nil
	}
	return m.check(s.Name, false)
}

// check fetches a source. With reset it takes the remote history of a source
// whose history was rewritten.
func (m Model) check(name string, reset bool) (Model, tea.Cmd) {
	if !m.start("check:"+name, "checking "+name) {
		return m, nil
	}
	be := m.be
	return m, func() tea.Msg {
		p, err := be.Check(name, reset)
		return checkedMsg{name, p, err}
	}
}

// checkAllSources checks every source, one after the other. It applies
// nothing: each source still waits for its own yes.
func (m Model) checkAllSources() (Model, tea.Cmd) {
	if len(m.data.Sources) == 0 {
		m.say("No sources. Press a to add one.", false, true)
		return m, nil
	}
	if !m.start("check:*", "checking every source") {
		return m, nil
	}
	be := m.be
	names := make([]string, len(m.data.Sources))
	for i, s := range m.data.Sources {
		names[i] = s.Name
	}
	return m, func() tea.Msg {
		var parts []string
		for _, n := range names {
			p, err := be.Check(n, false)
			switch {
			case err != nil:
				parts = append(parts, n+": "+firstLine(errText(err)))
			case p.UpToDate:
				parts = append(parts, n+": up to date")
			default:
				parts = append(parts, n+": changes available, press u on it")
			}
		}
		return doneMsg{key: "check:*", text: strings.Join(parts, "; ")}
	}
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

// checked handles the result of a check: an update that changes no assigned
// skill is applied at once, as quiver update does; one that does is shown
// first.
func (m Model) checked(msg checkedMsg) (tea.Model, tea.Cmd) {
	delete(m.pending, "check:"+msg.source)
	if errors.Is(msg.err, ErrDiverged) {
		m.checks[msg.source] = "History rewritten"
		name := msg.source
		m.confirm = &confirmation{
			prompt: fmt.Sprintf("The history of source %s was rewritten upstream (a force-push). Take the remote history, as quiver update %s --reset does? Skills of a source are read-only here, so nothing of yours is lost; changes to assigned skills are shown before they apply.", name, name),
			run:    func(m Model) (Model, tea.Cmd) { return m.check(name, true) },
		}
		return m, nil
	}
	if msg.err != nil {
		m.checks[msg.source] = "Check failed"
		m.say(msg.source+": "+errText(msg.err), true, false)
		return m, nil
	}
	p := msg.preview
	switch {
	case p.UpToDate:
		m.checks[msg.source] = "Up to date (" + p.To + ")"
		m.say(fmt.Sprintf("%s is up to date (%s).", msg.source, p.To), false, false)
	case !p.Ask:
		m.checks[msg.source] = "Applying"
		return m.apply(msg.source)
	default:
		m.checks[msg.source] = "Changes available"
		m.preview = &previewState{source: msg.source, lines: cleanLines(p.Lines), diff: cleanLines(p.Diff)}
	}
	return m, nil
}

func (m Model) apply(source string) (Model, tea.Cmd) {
	if !m.start("apply:"+source, "updating "+source) {
		return m, nil
	}
	be := m.be
	return m, func() tea.Msg {
		text, err := be.Apply(source)
		return doneMsg{key: "apply:" + source, text: text, err: err}
	}
}

func (m Model) previewKey(k string) (tea.Model, tea.Cmd) {
	p := m.preview
	switch k {
	case "y", "Y":
		m.preview = nil
		m.checks[p.source] = "Applying"
		return m.apply(p.source)
	case "n", "N", "esc", "q":
		m.preview = nil
		m.say(p.source+" was not updated. Press u on it to review the changes again.", false, true)
	case "d":
		if len(p.diff) > 0 {
			p.showDiff, p.offset = !p.showDiff, 0
		}
	case "up", "k":
		p.offset = max(p.offset-1, 0)
	case "down", "j":
		p.offset = min(p.offset+1, max(len(p.shownLines())-1, 0))
	case "pgup":
		p.offset = max(p.offset-m.bodyHeight(), 0)
	case "pgdown":
		p.offset = min(p.offset+m.bodyHeight(), max(len(p.shownLines())-1, 0))
	}
	return m, nil
}

// askRemoveSource asks before a source goes, and says which assignments go
// with it.
func (m Model) askRemoveSource() (Model, tea.Cmd) {
	s, ok := m.selectedSource()
	if !ok {
		return m, nil
	}
	global, _ := m.sourceUse(s.Name)
	var affected []string
	// A few skills share a line; more get a line each, so the list scrolls
	// instead of wrapping without an indent.
	add := func(where string, ids []string) {
		if len(ids) <= 4 {
			affected = append(affected, where+": "+strings.Join(ids, ", "))
			return
		}
		affected = append(affected, where+":")
		for _, id := range ids {
			affected = append(affected, "  "+id)
		}
	}
	if len(global) > 0 {
		ids := make([]string, len(global))
		for i, g := range global {
			ids[i] = g.ID
		}
		add("globally", ids)
	}
	for _, p := range m.data.Projects {
		var ids []string
		for _, id := range p.Skills {
			if strings.HasPrefix(id, s.Name+"/") {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			add("in "+p.Name, ids)
		}
	}
	q := fmt.Sprintf("Remove source %s and its %s?", s.Name, plural(s.Skills, "skill", "skills"))
	if len(affected) > 0 {
		q += "\nThese assignments are removed with it:"
		for _, a := range affected {
			q += "\n  " + a
		}
	} else {
		q += " None of its skills is assigned."
	}
	name := s.Name
	m.confirm = &confirmation{prompt: q, run: func(m Model) (Model, tea.Cmd) {
		return m.runProject("remove:"+name, "removing "+name, func() (string, error) { return m.be.RemoveSource(name) })
	}}
	return m, nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// cleanLines cleans every line of a text that came from outside, keeping the
// indent of the update report.
func cleanLines(ls []string) []string {
	out := make([]string, len(ls))
	for i, l := range ls {
		indent := len(l) - len(strings.TrimLeft(l, " "))
		out[i] = strings.Repeat(" ", min(indent, 12)) + oneLine(l)
	}
	return out
}

// cleanData cleans every text that came from a skill, a source or a folder.
func cleanData(d Data) Data {
	cleanProject := func(p Project) Project {
		p.Name, p.ID, p.Path = oneLine(p.Name), oneLine(p.ID), oneLine(p.Path)
		p.Paths = cleanAll(p.Paths)
		p.Skills = cleanAll(p.Skills)
		return p
	}
	if d.Project != nil {
		p := cleanProject(*d.Project)
		d.Project = &p
	}
	d.Opened, d.Version, d.Home = oneLine(d.Opened), oneLine(d.Version), oneLine(d.Home)
	skills := make([]Skill, len(d.Skills))
	for i, s := range d.Skills {
		s.ID, s.Name, s.Source = oneLine(s.ID), oneLine(s.Name), oneLine(s.Source)
		s.Description, s.Pin = oneLine(s.Description), oneLine(s.Pin)
		skills[i] = s
	}
	d.Skills = skills
	projects := make([]Project, len(d.Projects))
	for i, p := range d.Projects {
		projects[i] = cleanProject(p)
	}
	d.Projects = projects
	sources := make([]Source, len(d.Sources))
	for i, s := range d.Sources {
		s.Name, s.Repo, s.Commit, s.Updated = oneLine(s.Name), oneLine(s.Repo), oneLine(s.Commit), oneLine(s.Updated)
		s.Pinned = cleanAll(s.Pinned)
		sources[i] = s
	}
	d.Sources = sources
	return d
}

func cleanAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = oneLine(s)
	}
	return out
}
