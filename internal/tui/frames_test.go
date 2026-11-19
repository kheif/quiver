package tui

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestWriteFrames records what the panel shows, step by step, for the
// website's tour (web/src/data/panel.json). It runs the real model on a sample
// library, so the pictures cannot drift from the panel. It writes nothing
// unless FRAMES_OUT names the file:
//
//	FRAMES_OUT=../../web/src/data/panel.json go test -run TestWriteFrames ./internal/tui
func TestWriteFrames(t *testing.T) {
	out := os.Getenv("FRAMES_OUT")
	if out == "" {
		t.Skip("set FRAMES_OUT to write the website's frames")
	}
	f := &fake{
		skills: []Skill{
			{ID: "reply-style", Name: "reply-style", Source: "local", Local: true, Description: "Answer first, then the reason. No filler."},
			{ID: "release-notes", Name: "release-notes", Source: "local", Local: true, Description: "Write release notes from the commits since the last tag"},
			{ID: "k8s-debug", Name: "k8s-debug", Source: "local", Local: true, Description: "Debug a failing pod: events, logs, probes"},
			{ID: "db-migration", Name: "db-migration", Source: "local", Local: true, Description: "Rules for safe database migrations"},
			{ID: "openapi-lint", Name: "openapi-lint", Source: "local", Local: true, Description: "Check an OpenAPI file against our API conventions"},
			{ID: "anthropic/pdf", Name: "pdf", Source: "anthropic", Description: "Read, fill and create PDF files"},
			{ID: "anthropic/docx", Name: "docx", Source: "anthropic", Description: "Create and edit Word documents"},
			{ID: "team/migrate-check", Name: "migrate-check", Source: "team", Description: "Checks a migration before merge"},
		},
		projects: []Project{
			{Name: "api", ID: "api-1", Paths: []string{"~/code/api"}, Path: "~/code/api", Current: true, Folders: 1, Skills: []string{"k8s-debug", "openapi-lint", "db-migration"}, AlwaysOn: []string{"db-migration"}},
			{Name: "checkout-web", ID: "web-1", Paths: []string{"~/code/checkout-web"}, Path: "~/code/checkout-web", Folders: 1, Skills: []string{"release-notes"}},
		},
		sources: []Source{
			{Name: "anthropic", Repo: "anthropics/skills", Skills: 19, Commit: "def5678", Updated: "2026-11-03"},
			{Name: "team", Repo: "acme/skills", Skills: 6, Commit: "abc1234", Updated: "2026-10-28"},
		},
		glob: map[string]Mode{"reply-style": AlwaysOn, "anthropic/pdf": OnDemand},
		proj: map[string]map[string]Mode{"api-1": {"k8s-debug": OnDemand, "openapi-lint": OnDemand, "db-migration": AlwaysOn}, "web-1": {"release-notes": OnDemand}},
		health: []HealthItem{
			{Level: "ok", Title: "Library and registry are readable"},
			{Level: "ok", Title: "Claude Code: MCP server and session-start hook are set up"},
			{Level: "ok", Title: "Codex: MCP server and session-start hook are set up"},
			{Level: "warn", Title: "Cursor: the session-start hook is missing", Details: []string{"Always-on skills do not load by themselves there. Press i to set it up."}},
		},
		updates: map[string]bool{"anthropic": true},
		checks: map[string]Preview{"anthropic": {Source: "anthropic", From: "def5678", To: "9a1b2c3", Ask: true,
			Lines: []string{"anthropic/pdf changed: SKILL.md", "anthropic/docx changed: scripts/convert.py (needs review)"}}},
	}
	// The sample backend answers every skill's text with a stub.
	bodies := strings.NewReplacer(
		"Instructions of db-migration.", "Never drop a column in the same release that stops using it.",
		"Instructions of reply-style.", "Answer first, then the reason.",
	)

	type event struct {
		Key     string `json:"key,omitempty"`     // pressed to get this screen; empty for the first
		ID      string `json:"id,omitempty"`      // names a screen the docs show on its own
		Caption string `json:"caption,omitempty"` // what the story says from here on
		Wait    int    `json:"wait"`              // milliseconds to stay on this screen
		Frame   []int  `json:"frame"`             // the screen's lines, as indexes into lines
	}
	var (
		events []event
		lines  []string
		index  = map[string]int{}
	)
	m := startSized(t, f, 100, 22)
	// press sends the keys one at a time and records the screen after each; the
	// last one gets the id, the caption and the wait.
	record := func(key, id, caption string, wait int) {
		var frame []int
		for _, line := range strings.Split(bodies.Replace(m.render()), "\n") {
			i, ok := index[line]
			if !ok {
				i = len(lines)
				index[line] = i
				lines = append(lines, line)
			}
			frame = append(frame, i)
		}
		events = append(events, event{Key: key, ID: id, Caption: caption, Wait: wait, Frame: frame})
	}
	press := func(key string, wait int) {
		m = keys(t, m, key)
		record(key, "", "", wait)
	}
	last := func(id, caption string) { events[len(events)-1].ID, events[len(events)-1].Caption = id, caption }

	record("", "skills", "Run quiver. Your whole library is here, grouped by where each skill comes from. The right side shows how the selected skill is assigned.", 2600)
	press("down", 500)
	press("down", 900)
	press("/", 700)
	last("", "Press / and type. The list narrows as you type, in any word order.")
	for _, r := range "mig" {
		press(string(r), 260)
	}
	events[len(events)-1].Wait, events[len(events)-1].ID = 2200, "search"
	press("enter", 400)
	press("enter", 700)
	press("enter", 2800)
	last("assign", "Give a skill to this project and to every folder. The two settings are independent, each On demand or Always on.")
	press("esc", 300)
	press("esc", 300)
	press("esc", 500)
	press("2", 2600)
	last("projects", "Projects show what each project gets. Choose one to manage it from anywhere.")
	press("down", 900)
	press("3", 2600)
	last("sources", "Sources are your skill repositories. Quiver checks them in the background and tells you when one has changes.")
	press("u", 3400)
	last("update", "Nothing is pulled until you have seen what changes. Scripts are marked as needing review.")
	press("esc", 300)
	press("4", 3000)
	last("health", "Health shows how always-on skills reach each assistant, and what to fix when one does not get them.")
	press("1", 1200)

	b, err := json.Marshal(map[string]any{"cols": 100, "rows": 22, "lines": lines, "events": events})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
