package tui

import (
	"fmt"
	"image/color"
	"os"
	"slices"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// One green accent on the normal text of the terminal. The selected row, the
// current section and the focus are also marked with characters, so they show
// without color.
var (
	accent     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	accentBold = accent.Bold(true)
	bold       = lipgloss.NewStyle().Bold(true)
	faint      = lipgloss.NewStyle().Faint(true)
)

func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

// clip cuts s to w cells; wrap breaks it into lines of at most w cells.
func clip(s string, w int) string { return ansi.Truncate(s, max(w, 0), "…") }
func wrap(s string, w int) string { return ansi.Wrap(s, max(w, 1), "") }

// clipPath cuts a path to w cells from the left, so the folder name at its
// end stays.
func clipPath(s string, w int) string {
	if n := ansi.StringWidth(s) - w; n > 0 && w > 1 {
		return ansi.TruncateLeft(s, n+1, "…")
	}
	return s
}

// pad fills s with spaces up to w cells.
func pad(s string, w int) string {
	if n := w - ansi.StringWidth(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

// spread puts left and right on one line of w cells, right at the end.
func spread(left, right string, w int) string {
	gap := w - ansi.StringWidth(left) - ansi.StringWidth(right)
	if gap < 1 {
		return clip(left, w)
	}
	return left + strings.Repeat(" ", gap) + right
}

// fit returns exactly n lines: s cut to n lines, or padded with empty ones.
func fit(s []string, n int) []string {
	if len(s) > n {
		return s[:max(n, 0)]
	}
	for len(s) < n {
		s = append(s, "")
	}
	return s
}

func rule(w int) string { return faint.Render(strings.Repeat("─", max(w, 0))) }

// tilde shortens a folder under the home folder to ~/….
func tilde(path string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && home != "/" {
		if rest, ok := strings.CutPrefix(path, home+string(os.PathSeparator)); ok {
			return "~/" + rest
		}
	}
	return path
}

// roomy reports whether the terminal is tall enough for the spacious layout:
// blank lines between the parts, rows of two lines and bordered choices.
func (m Model) roomy() bool { return m.height >= 30 }

// mix returns a blended with b, b by the share t.
func mix(a, b color.Color, t float64) color.Color {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	f := func(x, y uint32) uint8 { return uint8(float64(x>>8)*(1-t) + float64(y>>8)*t) }
	return color.RGBA{f(ar, br), f(ag, bg), f(ab, bb), 255}
}

// band is the style of the selected row: the terminal's background with a
// little green. Until the terminal has told its background there is none,
// and the row is only bold and green.
func (m Model) band(extra lipgloss.Style) lipgloss.Style {
	if m.bg == nil {
		return extra.Foreground(lipgloss.Color("2"))
	}
	return extra.Background(mix(m.bg, color.RGBA{86, 168, 118, 255}, 0.22))
}

// bar is the style of the key line: the background, a little lighter or
// darker. Without a known background it has none.
func (m Model) bar() lipgloss.Style {
	if m.bg == nil {
		return lipgloss.NewStyle()
	}
	r, g, b, _ := m.bg.RGBA()
	if (r+g+b)/3 < 0x8000 {
		return lipgloss.NewStyle().Background(lipgloss.Lighten(m.bg, 0.06))
	}
	return lipgloss.NewStyle().Background(lipgloss.Darken(m.bg, 0.06))
}

// statusLines is what the status area says, wrapped to at most three lines,
// so a long message is not cut at the first one.
func (m Model) statusLines(cw int) []string {
	var text string
	switch {
	case m.confirm != nil:
		if m.confirmInBody(cw) {
			question, _, _ := strings.Cut(m.confirm.prompt, "\n")
			return []string{bold.Render(clip(question+" [y/N]", cw))}
		}
		ls := m.wrapStatus(m.confirm.prompt+" [y/N]", cw)
		for i, l := range ls {
			ls[i] = bold.Render(l)
		}
		return ls
	case m.status == "" && len(m.pending) > 0:
		labels := make([]string, 0, len(m.pending))
		for _, l := range m.pending {
			labels = append(labels, l)
		}
		sort.Strings(labels)
		text = "… " + strings.ToUpper(labels[0][:1]) + labels[0][1:]
	case m.status == "":
		return []string{""}
	case m.failed:
		text = "✗ " + m.status
	case m.note:
		text = m.status
	default:
		text = "✓ " + m.status
	}
	return m.wrapStatus(text, cw)
}

func (m Model) wrapStatus(text string, cw int) []string {
	ls := strings.Split(wrap(text, cw), "\n")
	if len(ls) > 3 {
		ls = append(ls[:2], clip(ls[2]+" "+strings.Join(ls[3:], " "), cw))
	}
	for i, l := range ls {
		ls[i] = clip(l, cw)
	}
	return ls
}

// margin is the empty space left and right of everything.
const margin = 2

// contentWidth is the width of the parts, inside the margins.
func (m Model) contentWidth() int { return max(m.width, minWidth) - 2*margin }

// headerHeight is the project line, the sections and their rule, with a
// blank line after each of the first and last when roomy.
func (m Model) headerHeight() int {
	if m.roomy() {
		return 5
	}
	return 3
}

// bodyHeight is the number of lines between the header and the rule at the
// bottom; the rule, the status and the key line take the rest.
func (m Model) bodyHeight() int {
	return max(m.height-m.headerHeight()-2-len(m.statusLines(m.contentWidth())), 1)
}

func (m Model) render() string {
	if m.width < minWidth || m.height < minHeight {
		return wrap(fmt.Sprintf("Terminal too small (%dx%d); quiver needs %dx%d. Make the window bigger, or press q to quit.", m.width, m.height, minWidth, minHeight), max(m.width, 1))
	}
	cw := m.contentWidth()
	if m.reading {
		return m.renderRead(cw)
	}
	h := m.bodyHeight()
	rows := []string{m.topLine(cw)}
	if m.roomy() {
		rows = append(rows, "")
	}
	rows = append(rows, m.sectionsLine(cw), m.sectionsRule(cw))
	if m.roomy() {
		rows = append(rows, "")
	}
	for i, r := range rows {
		rows[i] = strings.Repeat(" ", margin) + clip(r, cw)
	}
	for _, l := range fit(m.bodyLines(cw, h), h) {
		rows = append(rows, strings.Repeat(" ", margin)+clip(l, cw))
	}
	rows = append(rows, strings.Repeat(" ", margin)+rule(cw))
	for _, l := range m.statusLines(cw) {
		rows = append(rows, strings.Repeat(" ", margin)+l)
	}
	rows = append(rows, m.footerBar())
	return strings.Join(rows, "\n")
}

// topLine is the small signature and the project the panel manages.
func (m Model) topLine(cw int) string {
	left := accentBold.Render("quiver ›") + "   "
	p := m.data.Project
	// room is what is left of the line for the folder of the project.
	room := func() int { return cw - ansi.StringWidth(left) - 3 - 27 }
	switch {
	case p == nil:
		left += bold.Render("no project") + faint.Render("   global skills only")
	case p.Current:
		left += "project " + bold.Render(p.Name)
		if p.Path != "" {
			left += faint.Render("   " + clipPath(tilde(p.Path), room()))
		}
	default:
		left += "managing project " + bold.Render(p.Name)
		if p.Path != "" {
			left += faint.Render("   " + clipPath(tilde(p.Path), room()))
		}
		if m.data.Opened != "" {
			left += faint.Render("   · terminal opened in " + m.data.Opened)
		} else {
			left += faint.Render("   · terminal opened outside a project")
		}
	}
	if n := m.problems(); n > 0 && m.sec != secHealth {
		return spread(left, bold.Render("! "+plural(n, "problem", "problems"))+faint.Render(" · 4 Health"), cw)
	}
	return left
}

func tabLabel(i int) string { return fmt.Sprintf("%d %s", i+1, sectionNames[i]) }

const tabGap = 3

func (m Model) sectionsLine(cw int) string {
	var tabs []string
	for i := range sectionNames {
		if section(i) == m.sec {
			tabs = append(tabs, accentBold.Render(tabLabel(i)))
		} else {
			tabs = append(tabs, tabLabel(i))
		}
	}
	return clip(strings.Join(tabs, strings.Repeat(" ", tabGap)), cw)
}

// sectionsRule is the rule under the tabs, thicker under the current one.
func (m Model) sectionsRule(cw int) string {
	start := 0
	for i := 0; i < int(m.sec); i++ {
		start += len(tabLabel(i)) + tabGap
	}
	n := len(tabLabel(int(m.sec)))
	left := min(start, cw)
	mid := min(n, cw-left)
	return faint.Render(strings.Repeat("─", left)) + accentBold.Render(strings.Repeat("━", mid)) + faint.Render(strings.Repeat("─", max(cw-left-mid, 0)))
}

// bodyLines is the area under the tabs: the two panes, or a dialog that
// takes their place.
func (m Model) bodyLines(cw, h int) []string {
	switch {
	case m.confirm != nil && m.confirmInBody(cw):
		return m.confirmLines(cw, h)
	case !m.loaded && m.loadErr == "":
		return []string{faint.Render("Loading…")}
	case m.loaded && m.data.NoHome && m.loadErr == "" && !m.help:
		return m.setupLines(cw)
	case m.loadErr != "":
		out := []string{bold.Render("Quiver cannot read its data."), ""}
		out = append(out, strings.Split(wrap(m.loadErr, cw), "\n")...)
		return append(out, "", "Fix the problem, or run quiver doctor. q quits.")
	case m.help:
		return m.helpLines(cw)
	case m.dlg != nil && !m.popover(cw):
		return m.dialogLines(cw)
	case m.asking != nil:
		return m.askLines(cw)
	case m.preview != nil:
		return m.previewLines(cw, h)
	}
	if m.wide() {
		lw := min(max(cw*44/100, 36), 64)
		const gap = "  │  "
		rw := cw - lw - ansi.StringWidth(gap)
		if m.dlg != nil && !m.popover(rw) {
			return m.dialogLines(cw)
		}
		left, right := m.listLines(lw, h), m.detailLines(rw, h)
		out := make([]string, h)
		for i := range out {
			out[i] = pad(clip(left[i], lw), lw) + faint.Render(gap) + right[i]
		}
		return out
	}
	if m.focus == focusDetail {
		return m.detailLines(cw, h)
	}
	return m.listLines(cw, h)
}

func (m Model) listLines(w, h int) []string {
	return fit(m.listPane(w, h), h)
}

func (m Model) detailLines(w, h int) []string {
	switch m.sec {
	case secProjects:
		return fit(m.projectDetail(w, h), h)
	case secSources:
		return fit(m.sourceDetail(w, h), h)
	case secHealth:
		return fit(m.healthDetail(w, h), h)
	}
	return fit(m.skillDetail(w, h), h)
}

// --- the list ------------------------------------------------------------

// listRow is one row of a list: a title, a second line of detail that only
// the spacious layout shows, and what the row is in the current view.
type listRow struct {
	title, sub, scope string
	group             string // what the row belongs to; rows of one group sit under a header
	path              bool   // sub is a folder: cut it from the left
	tone              int
}

// listPane is the title, the views, the search line, the column header and
// the rows.
func (m Model) listPane(w, h int) []string {
	var rows []listRow
	var left, right, noun, what string
	switch m.sec {
	case secProjects:
		rows, left, right, noun, what = m.projectRows(), "NAME / FOLDER", "SKILLS", "project", "Projects"
	case secSources:
		rows, left, right, noun, what = m.sourceRows(), "NAME / REPOSITORY", "STATUS", "source", "Sources"
	case secHealth:
		rows, left, right, noun, what = m.healthRows(), "CHECK", "RESULT", "check", "Health"
	default:
		rows, left, right, noun, what = m.skillRows(), "NAME / SOURCE", m.scopeHeader(), "skill", "Skills"
		if strings.TrimSpace(m.queries[secSkills]) == "" {
			left = "NAME"
		}
	}
	if !m.roomy() {
		left = strings.SplitN(left, " /", 2)[0]
	}

	var out []string
	if m.roomy() {
		out = append(out, spread(bold.Render(what), faint.Render(m.countText()), w), m.viewsLine(w), m.searchLine(w, false), rule(w), "")
	} else {
		out = append(out, m.viewsLine(w), m.searchLine(w, true))
	}
	if len(rows) == 0 {
		return append(out, m.emptyLines(w)...)
	}
	return append(out, m.renderRows(rows, left, right, noun, w, h-len(out))...)
}

// scopeHeader names the column that says where a skill is assigned.
func (m Model) scopeHeader() string {
	if m.skillView == viewHere {
		return "AVAILABLE HERE"
	}
	return "ASSIGNED"
}

// countText is what the title line says about the results.
func (m Model) countText() string {
	switch m.sec {
	case secProjects:
		return fmt.Sprintf("%d registered", len(m.data.Projects))
	case secSources:
		return fmt.Sprintf("%d configured", len(m.data.Sources))
	case secHealth:
		return fmt.Sprintf("%d checks", len(m.health))
	}
	return fmt.Sprintf("%d in library", len(m.data.Skills))
}

// viewsLine shows the views of the section with their result counts; the
// current one is marked.
func (m Model) viewsLine(w int) string {
	switch m.sec {
	case secProjects:
		return faint.Render(fmt.Sprintf("All projects %d", len(m.data.Projects)))
	case secSources:
		return m.tabs(w, sourceViewNames[:], []string{"All", "Project", "Global"}, m.srcCounts[:], int(m.srcView))
	case secHealth:
		return m.tabs(w, healthViewNames[:], []string{"Problems", "All"}, m.healthCounts[:], int(m.healthView))
	}
	return m.tabs(w, skillViewNames[:], []string{"Here", "Project", "Global", "All"}, m.skillCounts[:], int(m.skillView))
}

func (m Model) tabs(w int, long, short []string, counts []int, cur int) string {
	build := func(names []string) string {
		var parts []string
		for i, n := range names {
			label := fmt.Sprintf("%s %d", n, counts[i])
			if i == cur {
				parts = append(parts, accentBold.Underline(true).Render("‹"+label+"›"))
			} else {
				parts = append(parts, faint.Render(label))
			}
		}
		return strings.Join(parts, "  ")
	}
	if s := build(long); ansi.StringWidth(s) <= w {
		return s
	}
	return clip(build(short), w)
}

// searchLine is the search field. With counts, it also says how many rows
// the list has.
func (m Model) searchLine(w int, counts bool) string {
	q := m.queries[m.sec]
	var left string
	switch {
	case m.searching:
		left = m.filter.View()
	case q != "":
		left = "/ " + q + faint.Render("  (esc clears)")
	default:
		left = faint.Render("/ Search " + strings.ToLower(sectionNames[m.sec]) + "…")
	}
	if !counts {
		return left
	}
	n := len(m.shown[m.sec])
	var right string
	switch m.sec {
	case secProjects:
		right = plural(n, "project", "projects")
	case secSources:
		right = plural(n, "source", "sources")
	case secHealth:
		right = healthViewNames[m.healthView] + ": " + plural(n, "check", "checks")
	default:
		right = skillViewNames[m.skillView] + ": " + plural(n, "result", "results")
	}
	return spread(left, faint.Render(right), w)
}

func (m Model) skillRows() []listRow {
	var rows []listRow
	// Without a search the list is grouped by where the skills come from; with
	// one it is in the order of the best match, and the group is in the ID.
	grouped := strings.TrimSpace(m.queries[secSkills]) == ""
	for _, i := range m.shown[secSkills] {
		s := m.data.Skills[i]
		r := listRow{title: s.ID, scope: m.scopeLabel(s)}
		switch {
		case (s.Global && s.GlobalAlways) || (s.Project && s.ProjectAlways):
			r.tone = toneStrong
		case !s.Project && !s.Global:
			r.tone = toneQuiet
		}
		if grouped {
			r.group = skillGroup(s)
			r.title = strings.TrimPrefix(s.ID, s.Source+"/")
		}
		rows = append(rows, r)
	}
	return rows
}

// skillGroup is the header a skill sits under: its source, or "Your skills".
func skillGroup(s Skill) string {
	if s.Local || s.Source == "" || s.Source == "local" {
		return "Your skills"
	}
	return s.Source
}

func (m Model) projectRows() []listRow {
	var rows []listRow
	for _, i := range m.shown[secProjects] {
		p := m.data.Projects[i]
		sub, isPath := "not seen on this machine", false
		if len(p.Paths) > 0 {
			sub, isPath = tilde(p.Paths[0]), true
		}
		var tags []string
		if m.data.Project != nil && p.ID == m.data.Project.ID {
			tags = append(tags, "managing")
		}
		if p.Current {
			tags = append(tags, "here")
		}
		if p.Missing {
			tags = append(tags, "missing")
		}
		if p.Folders > 1 {
			tags = append(tags, fmt.Sprintf("in %d folders", p.Folders))
		}
		scope := plural(len(p.Skills), "skill", "skills")
		if n := len(p.AlwaysOn); n > 0 {
			scope += fmt.Sprintf(" (%d always on)", n)
		}
		if len(tags) > 0 {
			scope += " · " + strings.Join(tags, " · ")
		}
		rows = append(rows, listRow{title: p.Name, sub: sub, scope: scope, path: isPath})
	}
	return rows
}

func (m Model) sourceRows() []listRow {
	var rows []listRow
	for _, i := range m.shown[secSources] {
		s := m.data.Sources[i]
		rows = append(rows, listRow{title: s.Name, sub: s.Repo, scope: m.sourceStatus(s)})
	}
	return rows
}

// Tones of the right column of a row.
const (
	toneNormal = iota
	toneStrong // always on: the thing that matters most
	toneQuiet  // not used here: stays out of the way
)

// listEntry is one thing in the list: a group header, or a row.
type listEntry struct {
	row    int // index into the rows; -1 for a header
	header string
	lines  int
}

// renderRows draws the column header and as many entries as fit around the
// cursor. Rows of one group sit under a header that says what they belong to,
// so a list reads as blocks, not as one long column. The selected row has a
// band when the list has the keyboard.
func (m Model) renderRows(rows []listRow, left, right, noun string, w, h int) []string {
	roomy := m.roomy()
	twoLines := roomy && m.sec != secSkills
	rowH, tail := 1, 0
	if twoLines {
		rowH = 2
		// A blank line between the rows, when there are few enough to afford it.
		if m.sec != secHealth && len(rows)*3 <= max(h-3, 1)*2 {
			rowH = 3
		}
	}
	if roomy {
		tail = 2
	}
	// Rows under a group header are indented, so the header reads as a heading.
	ind := 2
	if len(rows) > 0 && rows[0].group != "" {
		ind = 4
	}
	lead := strings.Repeat(" ", ind-2)
	avail := w - ind
	scopeW, titleW := ansi.StringWidth(right), ansi.StringWidth(left)
	counts := map[string]int{}
	for _, r := range rows {
		scopeW = max(scopeW, ansi.StringWidth(r.scope))
		titleW = max(titleW, ansi.StringWidth(r.title))
		counts[r.group]++
	}
	scopeW = min(scopeW, avail*3/5)
	titleW = min(titleW, max(avail-scopeW-2, 8))
	scopeW = min(scopeW, max(avail-titleW-2, 0))

	out := []string{faint.Render(strings.Repeat(" ", ind) + pad(left, titleW) + "  " + clip(right, scopeW))}

	// The entries: a header before the first row of each group, and a blank
	// line before every header but the first when there is room.
	var entries []listEntry
	prev, at := "", -1
	for i, r := range rows {
		if r.group != "" && r.group != prev {
			lines := 1
			if len(entries) > 0 && roomy {
				lines = 2
			}
			entries = append(entries, listEntry{row: -1, header: r.group, lines: lines})
		}
		prev = r.group
		if i == m.cursor[m.sec] {
			at = len(entries)
		}
		entries = append(entries, listEntry{row: i, lines: rowH})
	}
	budget := max(h-1-tail, 1)

	// The first entry shown is the one nearest the top that still lets the
	// cursor entry fit; the header of its group comes with it when it fits.
	start := max(at, 0)
	used := 0
	if at >= 0 {
		used = entries[at].lines
	}
	for start > 0 && used+entries[start-1].lines <= budget {
		start--
		used += entries[start].lines
	}
	if start > 0 && entries[start].row >= 0 && entries[start-1].row < 0 && used+entries[start-1].lines <= budget {
		start--
	}
	end, used := start, 0
	for end < len(entries) && used+entries[end].lines <= budget {
		used += entries[end].lines
		end++
	}
	listFocus := m.focus == focusList
	shownRows := 0
	for _, e := range entries[start:end] {
		if e.row >= 0 {
			shownRows++
		}
		if e.row < 0 {
			if e.lines == 2 {
				out = append(out, "")
			}
			label := strings.ToUpper(e.header[:1]) + e.header[1:]
			head := accentBold.Render(label) + faint.Render(fmt.Sprintf("  %d ", counts[e.header]))
			out = append(out, head+faint.Render(strings.Repeat("─", max(w-ansi.StringWidth(label)-len(fmt.Sprint(counts[e.header]))-4, 0))))
			continue
		}
		r := rows[e.row]
		scope := clip(r.scope, scopeW)
		first := pad(clip(r.title, titleW), titleW) + "  " + scope
		var second string
		if twoLines {
			second = clip(r.sub, avail)
			if r.path {
				second = clipPath(r.sub, avail)
			}
		}
		switch {
		case e.row == m.cursor[m.sec] && listFocus:
			out = append(out, m.band(bold).Render(pad(lead+"› "+clip(first, avail), w)))
			if twoLines {
				out = append(out, m.band(faint).Render(pad(lead+"  "+second, w)))
			}
		case e.row == m.cursor[m.sec]:
			out = append(out, lead+faint.Render("› ")+bold.Render(clip(first, avail)))
			if twoLines {
				out = append(out, lead+"  "+faint.Render(second))
			}
		default:
			styled := scope
			switch r.tone {
			case toneStrong:
				styled = accent.Render(scope)
			case toneQuiet:
				styled = faint.Render(scope)
			}
			title := pad(clip(r.title, titleW), titleW)
			if twoLines {
				title = bold.Render(title)
			}
			out = append(out, lead+"  "+title+"  "+styled)
			if twoLines {
				out = append(out, lead+"  "+faint.Render(second))
			}
		}
		if rowH == 3 {
			out = append(out, "")
		}
	}
	if roomy {
		more := "end of list"
		if rest := len(rows) - (rowsBefore(entries, start) + shownRows); rest > 0 {
			more = fmt.Sprintf("%d more below", rest)
		}
		what := plural(len(rows), noun, noun+"s")
		if m.sec == secSkills {
			what = skillViewNames[m.skillView] + " · " + what
		}
		if q := strings.TrimSpace(m.queries[m.sec]); q != "" {
			what += " match " + fmt.Sprintf("%q", q)
		}
		out = append(out, "", faint.Render("  "+what+" · "+more))
	}
	return out
}

// rowsBefore is the number of rows before entry i.
func rowsBefore(entries []listEntry, i int) int {
	n := 0
	for _, e := range entries[:i] {
		if e.row >= 0 {
			n++
		}
	}
	return n
}

// elsewhere returns the names of the other projects that use the skill, so a
// skill that no one uses here is not taken for unused.
func (m Model) elsewhere(s Skill) []string {
	var names []string
	for _, id := range s.UsedIn {
		if m.data.Project != nil && id == m.data.Project.ID {
			continue
		}
		name := id
		for _, p := range m.data.Projects {
			if p.ID == id {
				name = p.Name
			}
		}
		names = append(names, name)
	}
	return names
}

// usedText lists up to four project names.
func usedText(names []string) string {
	if len(names) > 4 {
		return strings.Join(names[:4], ", ") + fmt.Sprintf(" and %d more", len(names)-4)
	}
	return strings.Join(names, ", ")
}

// scopeLabel says in words where a skill is assigned, as seen from the managed
// project and from every folder.
func (m Model) scopeLabel(s Skill) string {
	withLoad := true
	var base string
	switch {
	case s.Project && s.Global:
		base = "Project + global"
	case s.Project:
		base = "Project"
	case s.Global:
		base = "Global"
	case len(m.elsewhere(s)) > 0:
		base, withLoad = "Other projects", false
	default:
		base = "Unassigned"
	}
	if withLoad && ((s.Global && s.GlobalAlways) || (s.Project && s.ProjectAlways)) {
		base += " · always on"
	}
	return base
}

// emptyLines tells the empty states apart: an empty library, an empty view
// and a search without results.
func (m Model) emptyLines(w int) []string {
	var ls []string
	switch m.sec {
	case secProjects:
		ls = []string{"No projects. Run quiver add inside a folder to register it."}
		if len(m.data.Projects) > 0 {
			ls = []string{fmt.Sprintf("No project matches %q.", strings.TrimSpace(m.queries[secProjects]))}
		}
	case secSources:
		ls = m.emptySources()
	case secHealth:
		ls = m.emptyHealth()
	default:
		ls = m.emptySkills()
	}
	var out []string
	for _, l := range ls {
		out = append(out, strings.Split(wrap("  "+l, w), "\n")...)
	}
	return out
}

func (m Model) emptySkills() []string {
	q := strings.TrimSpace(m.queries[secSkills])
	switch {
	case len(m.data.Skills) == 0:
		return []string{"The library is empty.", "Press n to create a skill, i to import skills,", "or 3 then a to add a source."}
	case q != "":
		ls := []string{"No matches in " + strings.ToLower(skillViewNames[m.skillView]) + "."}
		if m.canSearchAll() {
			ls = append(ls, fmt.Sprintf("Press a to search all skills for %q (ctrl+a while typing).", q))
		}
		return ls
	}
	switch m.skillView {
	case viewHere:
		return []string{"No skills are available here yet.", "Switch to All skills (press →) and assign one."}
	case viewProject:
		if m.data.Project == nil {
			return []string{"No project selected.", "Choose one in Projects (press 2)."}
		}
		return []string{"Nothing assigned to " + m.data.Project.Name + " directly."}
	}
	return []string{"No skills are assigned globally."}
}

// emptySources tells a library without remote sources from a view or a
// search with no sources.
func (m Model) emptySources() []string {
	local := 0
	for _, s := range m.data.Skills {
		if s.Local {
			local++
		}
	}
	switch {
	case len(m.data.Sources) == 0:
		ls := []string{"No remote sources.", "Press a to add one (a git repository of skills)."}
		if local > 0 {
			ls = append(ls, fmt.Sprintf("Your %s are local and need no source.", plural(local, "skill", "skills")))
		}
		return ls
	case strings.TrimSpace(m.queries[secSources]) != "":
		return []string{fmt.Sprintf("No source matches %q.", strings.TrimSpace(m.queries[secSources]))}
	case m.srcView == srcProject && m.data.Project == nil:
		return []string{"No project selected.", "Choose one in Projects (press 2)."}
	case m.srcView == srcProject:
		return []string{"No source has skills assigned to " + m.data.Project.Name + " directly."}
	}
	return []string{"No source has skills assigned globally."}
}

// --- the detail ----------------------------------------------------------

// label is a small heading in capitals; the block that has the keyboard is
// in the accent and says what the keys do.
func (m Model) label(text, hint string, focused bool) string {
	if focused {
		return accentBold.Render(text) + faint.Render("   "+hint)
	}
	return faint.Render(text)
}

// actionLines draws the selectable rows of the detail pane as a plain list.
func (m Model) actionLines(w int) []string {
	rows := m.actions()
	labelW := 0
	for _, a := range rows {
		labelW = max(labelW, ansi.StringWidth(a.label))
	}
	labelW = min(labelW, max(w/2, 12))
	var out []string
	for i, a := range rows {
		mark := "  "
		if m.focus == focusDetail && i == m.rowCursor {
			mark = accentBold.Render("› ")
		}
		line := pad(clip(a.label, labelW), labelW)
		if a.value != "" {
			v := a.value
			switch {
			case !a.ok:
				v = faint.Render(v)
			case v == AlwaysOn.String():
				v = accent.Render(v)
			}
			line += "  " + v
		}
		out = append(out, mark+clip(line, w-2))
	}
	return out
}

// choiceWidths are the widths of the choice boxes: the text of a mode, and
// the box with its borders.
const (
	choiceText = 12 // "Not assigned"
	choiceBox  = choiceText + 4 + 2
)

// popover reports whether the assignment choice opens as a menu under the
// row, in a detail pane of width w. It needs the spacious layout; otherwise
// the choice takes the place of the panes.
func (m Model) popover(w int) bool {
	if !m.roomy() || m.sec != secSkills {
		return false
	}
	labelW := 0
	for _, a := range m.actions() {
		labelW = max(labelW, ansi.StringWidth(a.label))
	}
	return w >= 2+labelW+2+choiceBox
}

// choiceLines draws the assignment rows as a label and a bordered choice,
// like a drop-down; the focused one has an accent border. While a choice is
// being made, its menu opens under the row.
func (m Model) choiceLines(w int) []string {
	rows := m.actions()
	labelW := 0
	for _, a := range rows {
		labelW = max(labelW, ansi.StringWidth(a.label))
	}
	if !m.popover(w) {
		return m.actionLines(w)
	}
	inner := choiceBox - 2
	var out []string
	for i, a := range rows {
		open := m.dlg != nil && (i == 0) == !m.dlg.global
		focused := open || m.focus == focusDetail && i == m.rowCursor
		edge := faint
		if focused {
			edge = accentBold
		}
		text := pad(clip(a.value, choiceText), choiceText)
		switch {
		case !a.ok:
			text = faint.Render(text)
		case a.value == AlwaysOn.String():
			text = accent.Render(text)
		case focused:
			text = bold.Render(text)
		}
		indent := strings.Repeat(" ", w-choiceBox)
		mark := "  "
		name := pad(a.label, labelW)
		if focused {
			mark, name = accentBold.Render("› "), bold.Render(name)
		}
		out = append(out,
			indent+edge.Render("╭"+strings.Repeat("─", inner)+"╮"),
			pad(mark+name, w-choiceBox)+edge.Render("│")+" "+text+" "+edge.Render("▾")+" "+edge.Render("│"),
			indent+edge.Render("╰"+strings.Repeat("─", inner)+"╯"))
		if open {
			out = append(out, m.menuLines(indent, inner)...)
		}
	}
	return out
}

// menuLines is the open list of modes under a choice: the one the cursor is
// on has a marker, the current one a check.
func (m Model) menuLines(indent string, inner int) []string {
	d := m.dlg
	out := make([]string, 0, len(modeNames)+2)
	out = append(out, indent+accentBold.Render("╭"+strings.Repeat("─", inner)+"╮"))
	for i, name := range modeNames {
		mark, check := " ", " "
		text := pad(name, choiceText)
		if i == d.cursor {
			mark, text = "›", bold.Render(text)
		}
		if Mode(i) == d.from {
			check = "✓"
		}
		out = append(out, indent+accentBold.Render("│")+accent.Render(mark)+" "+text+" "+accent.Render(check)+accentBold.Render("│"))
	}
	return append(out, indent+accentBold.Render("╰"+strings.Repeat("─", inner)+"╯"))
}

func (m Model) skillDetail(w, h int) []string {
	s, ok := m.selectedSkill()
	if !ok {
		return []string{faint.Render("No skill selected.")}
	}
	roomy := m.roomy()
	src := s.Source + " · read-only"
	if s.Local {
		src = s.Source + " · editable"
	}
	var out []string
	if roomy {
		out = append(out, spread(bold.Render(clip(s.ID, max(w-ansi.StringWidth(src)-2, 8))), faint.Render(src), w), "")
		if s.Description != "" {
			out = append(out, strings.Split(wrap(s.Description, w), "\n")...)
			out = append(out, "")
		}
		var notes []string
		if s.Pin != "" {
			notes = append(notes, "pinned to "+s.Pin)
		}
		if !s.Local {
			notes = append(notes, "f forks it into a local copy you can edit")
		}
		if names := m.elsewhere(s); len(names) > 0 {
			notes = append(notes, "also used in "+usedText(names))
		}
		if len(notes) > 0 {
			out = append(out, strings.Split(faint.Render(wrap(strings.Join(notes, " · "), w)), "\n")...)
			out = append(out, "")
		}
	} else {
		out = append(out, bold.Render(clip(s.ID, w)), faint.Render(clip("Source: "+src, w)))
		if s.Pin != "" {
			out = append(out, faint.Render(clip("Pinned to "+s.Pin, w)))
		}
		if !s.Local {
			out = append(out, faint.Render(clip("f forks it into a local copy you can edit", w)))
		}
		if names := m.elsewhere(s); len(names) > 0 {
			out = append(out, faint.Render(clip("Also used in "+usedText(names), w)))
		}
		out = append(out, "")
	}
	out = append(out, m.label("ASSIGNMENTS", "↑↓ choose · enter change · esc back", m.focus == focusDetail))
	if roomy {
		out = append(out, m.choiceLines(w)...)
		out = append(out, rule(w), m.label("PREVIEW · SKILL.md", "", false))
	} else {
		out = append(out, m.actionLines(w)...)
		out = append(out, "")
		if s.Description != "" {
			out = append(out, strings.Split(wrap(s.Description, w), "\n")...)
			out = append(out, "")
		}
	}
	d, err := m.detailsOf(s.ID)
	if err != nil {
		return append(out, errText(err))
	}
	more := false
	for _, l := range instructionLines(s, d, roomy) {
		for _, wl := range strings.Split(wrap(l, w), "\n") {
			if len(out) >= h-1 {
				more = true
				break
			}
			if strings.HasPrefix(strings.TrimLeft(wl, " "), "#") {
				wl = accent.Render(wl)
			}
			out = append(out, wl)
		}
		if more {
			break
		}
	}
	if more {
		out = append(out, faint.Render("… press r to read the whole SKILL.md"))
	}
	return out
}

// instructionLines is the start of the instructions: not the description
// again, and with paragraphs kept when there is room.
func instructionLines(s Skill, d Details, paragraphs bool) []string {
	var out []string
	for _, l := range strings.Split(d.Body, "\n") {
		l = strings.TrimRight(l, " ")
		if strings.TrimSpace(l) == "" {
			if paragraphs && len(out) > 0 && out[len(out)-1] != "" {
				out = append(out, "")
			}
			continue
		}
		if len(out) == 0 && (strings.TrimSpace(l) == s.Description || strings.TrimLeft(l, "# ") == s.Name && s.Name != "" && !paragraphs) {
			continue
		}
		out = append(out, l)
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

// limited returns up to n lines of items, with a last line that counts the
// rest.
func limited(items []string, n int, w int) []string {
	if n <= 0 {
		return nil
	}
	if len(items) <= n {
		return clipAll(items, w)
	}
	out := clipAll(items[:max(n-1, 0)], w)
	return append(out, faint.Render(fmt.Sprintf("  … and %d more", len(items)-len(out))))
}

func clipAll(items []string, w int) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = clip(s, w)
	}
	return out
}

func (m Model) projectDetail(w, h int) []string {
	p, ok := m.selectedProject()
	if !ok {
		return []string{faint.Render("No project selected.")}
	}
	roomy := m.roomy()
	var out []string
	if roomy {
		out = append(out, spread(bold.Render(clip(p.Name, max(w-len(p.ID)-5, 8))), faint.Render("ID "+p.ID), w), "")
	} else {
		out = append(out, bold.Render(clip(p.Name, w)), faint.Render(clip("ID "+p.ID, w)))
	}
	out = append(out, m.label("FOLDERS", "", false))
	if len(p.Paths) == 0 {
		out = append(out, "  "+faint.Render("not seen on this machine"))
	}
	for _, path := range p.Paths {
		tail := ""
		if p.Missing {
			tail = "  · no longer exists"
		}
		out = append(out, "  "+clipPath(tilde(path), w-2-ansi.StringWidth(tail))+faint.Render(tail))
	}
	if p.Folders > 1 {
		out = append(out, faint.Render(clip(fmt.Sprintf("  The same project ID is in %d folders. They share one skill list; press s in a copy to split it.", p.Folders), w)))
	}
	out = append(out, "", m.label("ACTIONS", "↑↓ choose · enter run · esc back", m.focus == focusDetail))
	out = append(out, m.actionLines(w)...)

	direct := map[string]bool{}
	for _, id := range p.Skills {
		direct[id] = true
	}
	var inherited []string
	for _, s := range m.data.Skills {
		if s.Global && !direct[s.ID] {
			inherited = append(inherited, "  "+s.ID)
		}
	}
	sorted := slices.Clone(p.Skills)
	sort.Strings(sorted)
	inLibrary := map[string]bool{}
	for _, s := range m.data.Skills {
		inLibrary[s.ID] = true
	}
	items := make([]string, len(sorted))
	for i, id := range sorted {
		items[i] = "  " + id
		switch {
		case !inLibrary[id]:
			items[i] += faint.Render("  not in the library")
		case slices.Contains(p.AlwaysOn, id):
			items[i] += accent.Render("  always on")
		}
	}
	room := h - len(out) - 4
	out = append(out, "", m.label(fmt.Sprintf("ASSIGNED DIRECTLY · %d", len(items)), "", false))
	out = append(out, limited(items, max(room/2, 1), w)...)
	if len(inherited) > 0 {
		out = append(out, "", m.label(fmt.Sprintf("AVAILABLE THROUGH GLOBAL ASSIGNMENTS · %d", len(inherited)), "", false))
		out = append(out, limited(inherited, max(h-len(out), 1), w)...)
	}
	return out
}

func (m Model) sourceStatus(s Source) string {
	if st, ok := m.checks[s.Name]; ok {
		return st
	}
	if s.Commit == "" {
		return "Clone missing or damaged"
	}
	return "Not checked"
}

func (m Model) sourceDetail(w, h int) []string {
	s, ok := m.selectedSource()
	if !ok {
		return []string{faint.Render("No source selected.")}
	}
	status := m.sourceStatus(s)
	var out []string
	if m.roomy() {
		out = append(out, spread(bold.Render(clip(s.Name, max(w-ansi.StringWidth(status)-2, 8))), faint.Render(status), w), "")
	} else {
		out = append(out, bold.Render(clip(s.Name, w)), faint.Render(clip(status, w)))
	}
	kv := func(k, v string) string { return clip(faint.Render(pad(k, 12))+v, w) }
	out = append(out, kv("Repository", s.Repo))
	if s.Commit != "" {
		commit := s.Commit
		if s.Updated != "" {
			commit += " · " + s.Updated
		}
		out = append(out, kv("Commit", commit))
	}
	if len(s.Pinned) > 0 {
		out = append(out, kv("Pinned", strings.Join(s.Pinned, ", ")))
	}
	out = append(out, "", m.label("ACTIONS", "↑↓ choose · enter run · esc back", m.focus == focusDetail))
	out = append(out, m.actionLines(w)...)

	var items []string
	for _, sk := range m.data.Skills {
		if sk.Source != s.Name {
			continue
		}
		var scopes []string
		if sk.Project && m.data.Project != nil {
			scopes = append(scopes, "project "+m.data.Project.Name)
		}
		if sk.Global {
			scopes = append(scopes, "global")
		}
		if names := m.elsewhere(sk); len(names) > 0 {
			verb := "used in "
			if len(scopes) > 0 {
				verb = "also in "
			}
			scopes = append(scopes, verb+usedText(names))
		}
		if len(scopes) == 0 {
			scopes = []string{"not assigned"}
		}
		items = append(items, "  "+sk.ID+"  "+faint.Render(strings.Join(scopes, " · ")))
	}
	out = append(out, "", m.label(fmt.Sprintf("SKILLS · %d", len(items)), "", false))
	if len(items) > 0 {
		out = append(out, faint.Render(clip("  Assign them in Skills: press 1, then → for all skills.", w)))
	}
	return append(out, limited(items, max(h-len(out), 1), w)...)
}

// --- dialogs -------------------------------------------------------------

func (m Model) dialogLines(cw int) []string {
	d := m.dlg
	where, title := "every folder", "Every folder"
	if !d.global && m.data.Project != nil {
		where, title = m.data.Project.Name, m.projectLabel()
	}
	out := []string{bold.Render(clip(title+" — "+d.id, cw)), faint.Render(clip("The other scope is not changed.", cw)), ""}
	for i, name := range modeNames {
		line := fmt.Sprintf("%d  %s", i+1, name)
		if Mode(i) == d.from {
			line += faint.Render("  (current)")
		}
		if i == d.cursor {
			out = append(out, accentBold.Render("› ")+bold.Render(line))
		} else {
			out = append(out, "  "+line)
		}
	}
	return append(out, "",
		faint.Render(clip("Not assigned: "+d.id+" is not available in "+where+".", cw)),
		faint.Render(clip("On demand: the assistant loads it when the task fits.", cw)),
		faint.Render(clip("Always on: loaded before the first reply, from the next session.", cw)))
}

// confirmInBody reports whether the question is too long for the status line:
// it has several lines, or wraps to more than three. It is then read in the
// body, where it scrolls.
func (m Model) confirmInBody(cw int) bool {
	p := m.confirm.prompt + " [y/N]"
	return strings.Contains(p, "\n") || len(strings.Split(wrap(p, cw), "\n")) > 3
}

// confirmLines is a long question, from the line the person scrolled to.
func (m Model) confirmLines(cw, h int) []string {
	c := m.confirm
	all := strings.Split(wrap(c.prompt, cw), "\n")
	room := max(h-3, 1) // the last line says how much is left
	c.offset = min(c.offset, max(len(all)-room, 0))
	out := []string{m.label("CONFIRM", "", true), ""}
	end := min(c.offset+room, len(all))
	out = append(out, all[c.offset:end]...)
	if end < len(all) {
		out = append(out, faint.Render(fmt.Sprintf("… %d more lines, ↓ to read on", len(all)-end)))
	}
	return out
}

func (m Model) askLines(cw int) []string {
	a := m.asking
	out := []string{m.label(a.title, "", true), "", a.input.View(), ""}
	return append(out, strings.Split(faint.Render(wrap(a.hint, cw)), "\n")...)
}

func (m Model) previewLines(cw, h int) []string {
	p := m.preview
	out := []string{m.label("UPDATE · "+strings.ToUpper(p.source), "these changes are waiting for your review", true), ""}
	room := max(h-len(out), 1)
	lines := p.shownLines()
	end := min(p.offset+room, len(lines))
	for _, l := range lines[min(p.offset, len(lines)):end] {
		out = append(out, clip(l, cw))
	}
	return out
}

func (m Model) helpLines(cw int) []string {
	out := []string{m.label("KEYS", "", false)}
	add := func(keys, what string) { out = append(out, clip("  "+bold.Render(pad(keys, 16))+what, cw)) }
	add("1 2 3 4", "Skills, Projects, Sources, Health")
	add("↑ ↓  j k", "move; pgup pgdn home end jump")
	add("/", "search this section; enter keeps, esc clears")
	switch m.sec {
	case secSkills:
		add("← →  h l", "switch view: available here, project, global, all")
		add("enter  space", "edit the project and global assignments")
		add("r", "read the whole SKILL.md")
		add("e  d", "edit a local skill; delete it")
		add("f  p", "fork a source skill; pin it to its commit, or unpin")
		add("n  i", "new skill; import skills from a folder")
	case secProjects:
		add("enter", "details and actions")
		add("m  r  f", "manage skills, rename, forget")
		add("s", "split a copied folder (project of this folder)")
	case secHealth:
		add("← →  h l", "switch view: problems, all checks")
		add("i  x", "set up Quiver in the assistants; remove it")
		add("f  r", "apply the safe fixes of quiver doctor; check again")
	default:
		add("← →  h l", "switch view: all, used by project, used globally")
		add("a", "add a source (a git repository of skills)")
		add("u  U", "check one source for updates, or all")
		add("x", "remove the source")
	}
	add("tab", "move between the list and its detail")
	if m.sec == secSkills {
		add("On demand", "loaded when the task fits")
		add("Always on", "loaded before the first reply; needs the hook (Health)")
	}
	add("esc  ?  q", "back; help; quit")
	out = append(out, "", m.label("ABOUT", "", false), "  Quiver "+m.data.Version)
	if m.data.Home != "" {
		out = append(out, clip("  Library "+m.data.Home, cw))
	}
	return out
}

// --- keys ----------------------------------------------------------------

// footerKeys are the few keys that work now, each "key what it does". The
// last two stay when the line is too narrow; the others are dropped from the
// end.
func (m Model) footerKeys(cw int) []string {
	var keys []string
	essential := []string{"? help", "q quit"}
	switch {
	case m.confirm != nil:
		keys, essential = []string{"y yes", "n cancel"}, nil
		if m.confirmInBody(cw) {
			keys = append(keys, "↑↓ read")
		}
	case m.preview != nil:
		keys, essential = []string{"y apply the update", "n cancel", "↑↓ scroll"}, nil
		if len(m.preview.diff) > 0 {
			what := "d show the changes"
			if m.preview.showDiff {
				what = "d show the summary"
			}
			keys = append(keys, what)
		}
	case m.dlg != nil:
		keys, essential = []string{"↑↓ choose", "1-3 pick", "enter apply", "esc cancel"}, nil
	case m.asking != nil:
		keys, essential = []string{"enter " + m.asking.verb, "esc cancel"}, nil
	case m.searching:
		keys, essential = []string{"enter keep results", "esc clear search", "↑↓ move"}, nil
		if m.canSearchAll() {
			keys = append([]string{"ctrl+a search all skills"}, keys...)
		}
	case m.help:
		keys, essential = []string{"esc close help"}, nil
	case m.loadErr != "":
		keys, essential = []string{"q quit"}, nil
	case !m.loaded:
		keys, essential = []string{"q quit"}, nil
	case m.data.NoHome:
		keys, essential = []string{"enter set up Quiver", "q quit"}, nil
	case m.focus == focusDetail:
		keys = m.detailKeys()
	default:
		keys = m.listKeys()
	}
	const sep = 3
	width := func(ks []string) int {
		n := 0
		for _, k := range append(slices.Clone(ks), essential...) {
			n += ansi.StringWidth(k) + sep
		}
		return n
	}
	for len(keys) > 0 && width(keys) > cw+sep {
		keys = keys[:len(keys)-1]
	}
	return append(keys, essential...)
}

// footerBar is the key line: each key in bold, what it does dim, on a bar of
// its own.
func (m Model) footerBar() string {
	cw := m.contentWidth()
	var parts []string
	for _, k := range m.footerKeys(cw) {
		key, what, _ := strings.Cut(k, " ")
		parts = append(parts, bold.Render(key)+" "+faint.Render(what))
	}
	line := strings.Repeat(" ", margin) + clip(strings.Join(parts, "   "), cw)
	return m.bar().Render(pad(line, m.width))
}

// footer is the plain text of the key line.
func (m Model) footer(cw int) string { return strings.Join(m.footerKeys(cw), "   ") }

func (m Model) listKeys() []string {
	switch m.sec {
	case secProjects:
		keys := []string{"↑↓ move", "/ search", "enter details", "m manage", "r rename", "f forget"}
		if p, ok := m.selectedProject(); ok && p.Current && p.Folders > 1 {
			keys = append(keys, "s split")
		}
		return keys
	case secSources:
		return []string{"↑↓ move", "←→ view", "/ search", "a add source", "enter details", "u check updates", "U check all", "x remove"}
	case secHealth:
		return []string{"↑↓ move", "←→ view", "/ search", "i set up assistants", "f fix", "r check again", "x remove from assistants"}
	}
	keys := []string{"←→ view", "/ search", "enter assign", "n new", "i import"}
	if s, ok := m.selectedSkill(); ok {
		switch {
		case s.Local:
			keys = append(keys, "e edit", "d delete")
		case s.Pin != "":
			keys = append(keys, "f fork", "p unpin")
		default:
			keys = append(keys, "f fork", "p pin")
		}
		keys = append(keys, "r read")
	}
	if m.canSearchAll() {
		keys = append([]string{"a search all skills"}, keys...)
	}
	return keys
}

func (m Model) detailKeys() []string {
	rows := m.actions()
	enter := "enter run"
	if m.rowCursor < len(rows) {
		a := rows[m.rowCursor]
		switch m.sec {
		case secSkills:
			switch {
			case m.rowCursor == 0 && m.data.Project != nil:
				enter = "enter change " + m.data.Project.Name + " assignment"
			case m.rowCursor == 0:
				enter = "enter choose a project"
			default:
				enter = "enter change global assignment"
			}
		default:
			enter = "enter " + strings.ToLower(a.label)
		}
	}
	keys := []string{"↑↓ choose", enter, "esc back to list"}
	if m.sec == secSkills {
		keys = append(keys, "r read")
	}
	return keys
}

func (m Model) renderRead(cw int) string {
	var id string
	if s, ok := m.selectedSkill(); ok {
		id = s.ID
	}
	head := spread(bold.Render("SKILL.md · "+id), faint.Render(fmt.Sprintf("%3.0f%%", m.vp.ScrollPercent()*100)), cw)
	rows := []string{head}
	rows = append(rows, fit(strings.Split(m.vp.View(), "\n"), m.height-2)...)
	rows = append(rows, faint.Render("esc back  ↑↓ pgup pgdn scroll  home end"))
	for i, r := range rows {
		rows[i] = strings.Repeat(" ", margin) + clip(r, cw)
	}
	return strings.Join(rows, "\n")
}
