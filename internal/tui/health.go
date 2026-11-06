package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// problems counts the checks that need action.
func (m Model) problems() int {
	n := 0
	for _, h := range m.health {
		if healthRank(h.Level) < 2 {
			n++
		}
	}
	return n
}

// setupKey runs the keys of the first-run screen: Quiver has no home yet.
func (m Model) setupKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case "q":
		return m, tea.Quit
	case "enter":
		m.confirm = &confirmation{
			prompt: fmt.Sprintf("Set up Quiver? This creates %s and registers the MCP server and the session-start hooks in the assistants it finds. Their config files are saved with the suffix .quiver-backup first.", m.data.Home),
			run: func(m Model) (Model, tea.Cmd) {
				return m.runProject("setup", "setting up Quiver", m.be.Setup)
			},
		}
	}
	return m, nil
}

func (m Model) setupLines(cw int) []string {
	out := []string{bold.Render("Quiver is not set up on this machine yet."), ""}
	for _, l := range []string{
		"Quiver keeps your assistant skills in one library" + homeText(m.data.Home) + " and serves them to Claude Code, Codex, Cursor and other assistants.",
		"",
		"Setting up creates the library and registers Quiver in the assistants it finds. It changes nothing else, and your own skill folders stay as they are.",
	} {
		out = append(out, strings.Split(wrap(l, cw), "\n")...)
	}
	return append(out, "", accentBold.Render("enter")+" set up Quiver   "+accentBold.Render("q")+" quit")
}

func homeText(home string) string {
	if home == "" {
		return ""
	}
	return " (" + tilde(home) + ")"
}

// askInstall confirms before it edits the configuration of the assistants.
func (m Model) askInstall() (Model, tea.Cmd) {
	m.confirm = &confirmation{
		prompt: "Register the MCP server and the session-start hooks in every assistant Quiver finds? Their config files are saved with the suffix .quiver-backup first (as quiver install).",
		run: func(m Model) (Model, tea.Cmd) {
			return m.runProject("install", "setting up the assistants", m.be.Install)
		},
	}
	return m, nil
}

func (m Model) askUninstall() (Model, tea.Cmd) {
	m.confirm = &confirmation{
		prompt: "Remove the MCP server and the hooks of Quiver from every assistant? Your library and skills stay (as quiver uninstall). Always-on skills stop loading.",
		run: func(m Model) (Model, tea.Cmd) {
			return m.runProject("uninstall", "removing Quiver from the assistants", m.be.Uninstall)
		},
	}
	return m, nil
}

func (m Model) askFix() (Model, tea.Cmd) {
	m.confirm = &confirmation{
		prompt: "Apply the safe fixes of quiver doctor? They create missing parts of the home, commit uncommitted changes in local/ and delete leftovers of interrupted source changes. Fixes that need a decision are left to quiver doctor --fix.",
		run: func(m Model) (Model, tea.Cmd) {
			return m.runProject("fix", "fixing", m.be.Fix)
		},
	}
	return m, nil
}

func (m Model) healthRows() []listRow {
	var rows []listRow
	for _, i := range m.shown[secHealth] {
		h := m.health[i]
		group := "Fine"
		switch h.Level {
		case "error":
			group = "Errors"
		case "warn":
			group = "Warnings"
		}
		r := listRow{title: h.Title, scope: h.Level, group: group}
		if healthRank(h.Level) == 2 {
			r.tone = toneQuiet
		}
		rows = append(rows, r)
	}
	return rows
}

func (m Model) emptyHealth() []string {
	switch {
	case !m.healthLoaded:
		return []string{"Checking your setup…"}
	case len(m.health) == 0:
		return []string{"No checks ran."}
	case strings.TrimSpace(m.queries[secHealth]) != "":
		return []string{fmt.Sprintf("No check matches %q.", strings.TrimSpace(m.queries[secHealth]))}
	}
	return []string{"No problems found.", "Press → to see every check."}
}

func (m Model) healthDetail(w, h int) []string {
	it, ok := m.selectedHealth()
	if !ok {
		return []string{faint.Render("No check selected.")}
	}
	var out []string
	if m.roomy() {
		out = append(out, spread(bold.Render(clip(it.Title, max(w-len(it.Level)-2, 8))), faint.Render(it.Level), w), "")
	} else {
		out = append(out, bold.Render(clip(it.Title, w)), faint.Render(it.Level))
	}
	for _, d := range it.Details {
		out = append(out, strings.Split(wrap(d, w), "\n")...)
	}
	if len(it.Details) > 0 {
		out = append(out, "")
	}
	out = append(out, m.label("ACTIONS", "", false))
	for _, l := range []string{
		"i  set up the MCP server and hooks in the assistants",
		"f  apply the safe fixes of quiver doctor",
		"x  remove Quiver from the assistants",
		"r  check again",
	} {
		out = append(out, clip("  "+l, w))
	}
	return fit(out, h)
}
