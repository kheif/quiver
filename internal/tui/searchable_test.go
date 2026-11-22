package tui

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// The scope setting has four states, in this order.
func TestScopeSettingHasFourStatesInOrder(t *testing.T) {
	f := fixture()
	m := goTo(t, keys(t, start(t, f), "right", "right", "right"), "pdf") // the whole library
	m = keys(t, m, "enter", "enter")
	scr := screen(m)
	last := -1
	for _, name := range []string{"Not assigned", "Searchable", "On demand", "Always on"} {
		i := strings.Index(scr[max(last, 0):], name)
		if i < 0 {
			t.Fatalf("%q missing or out of order in:\n%s", name, scr)
		}
		last += i + 1
	}
	contains(t, m, "1-4 pick")

	m = keys(t, m, "2", "enter")
	if f.callList() != "project pdf /code/app Searchable" {
		t.Fatalf("calls %q", f.callList())
	}
	contains(t, m, "✓ pdf: Searchable in app: not listed, found by searching.", "Applies from the next session")
	// The row shows the new state.
	contains(t, m, "Searchable")
}

// A change for every folder asks first, and says what goes with it.
func TestSearchableForEveryFolderAsksFirst(t *testing.T) {
	f := fixture()
	m := goTo(t, start(t, f), "commit-rules")
	m = keys(t, m, "enter", "down", "enter") // Global, always on
	m = keys(t, m, "2", "enter")
	contains(t, m, "Make commit-rules searchable in every folder? It is always-on there, and that mark goes too", "[y/N]")
	if len(f.calls) != 0 {
		t.Fatalf("changed before the answer: %v", f.calls)
	}
	m = keys(t, m, "n")
	if len(f.calls) != 0 {
		t.Fatalf("changed after no: %v", f.calls)
	}

	m = goTo(t, keys(t, m, "esc"), "demo")
	m = keys(t, m, "enter", "down", "enter", "2", "enter")
	contains(t, m, "Make demo searchable in every folder? It is no longer listed")
	m = keys(t, m, "y")
	if f.callList() != "global demo Searchable" {
		t.Fatalf("calls %q", f.callList())
	}
	contains(t, m, "✓ demo: Searchable in every folder: not listed, found by searching.")
}

// A Searchable skill is in the view of what is available here (marked), in
// the views of the project's and of the global assignments, and in the whole
// library.
func TestSearchableSkillsAreInTheViews(t *testing.T) {
	f := fixture()
	f.proj["app-1"]["pdf"] = Searchable
	f.glob["team/migrate-check"] = Searchable
	m := start(t, f)

	here := listed(m)
	for _, id := range []string{"pdf", "team/migrate-check"} {
		if !slices.Contains(here, id) {
			t.Errorf("%s is not available here: %v", id, here)
		}
	}
	contains(t, m, "Project · searchable", "Global · searchable")
	if want := [viewCount]int{5, 2, 3, 5}; fmt.Sprint(m.skillCounts) != fmt.Sprint(want) {
		t.Errorf("counts %v, want %v", m.skillCounts, want)
	}
	m = keys(t, m, "right") // the project's assignments
	if got := listed(m); !slices.Contains(got, "pdf") || slices.Contains(got, "team/migrate-check") {
		t.Errorf("project view: %v", got)
	}
	m = keys(t, m, "right") // global
	if got := listed(m); !slices.Contains(got, "team/migrate-check") || slices.Contains(got, "pdf") {
		t.Errorf("global view: %v", got)
	}
	m = keys(t, m, "right") // all
	if got := listed(m); len(got) != 5 {
		t.Errorf("all: %v", got)
	}
}

// The Sources and Projects sections count a Searchable skill as used.
func TestSearchableSkillsCountAsUsed(t *testing.T) {
	f := fixture()
	f.proj["app-1"]["team/migrate-check"] = Searchable
	f.proj["api-1"]["pdf"] = Searchable
	m := start(t, f)

	m = keys(t, m, "3")
	m = goTo(t, m, "team")
	contains(t, m, "team/migrate-check", "project app (searchable)")
	lacks(t, m, "not assigned")

	m = keys(t, m, "2")
	contains(t, m, "2 skills (1 searchable)", "1 skill (1 searchable)")
	m = goTo(t, m, "app-1")
	contains(t, m, "ASSIGNED DIRECTLY · 2", "team/migrate-check", "searchable")

	// A skill that is Searchable in another project is not unused.
	m = keys(t, m, "1", "right", "right", "right")
	m = goTo(t, m, "pdf")
	contains(t, m, "also used in api")
}

// After a source with skills is added, the panel asks once whether to make
// them Searchable.
func TestAddSourceOffersSearchable(t *testing.T) {
	add := func(f *fake) Model {
		m := keys(t, start(t, f), "3")
		m = keys(t, m, "a")
		m = typed(t, m, "acme/skills")
		m = keys(t, m, "enter")
		m = typed(t, m, "acme")
		return keys(t, m, "enter")
	}

	f := fixture()
	f.addedSkills = 2
	m := add(f)
	contains(t, m, "Make the 2 skills of acme searchable in every folder? The assistant will then load them without asking when a search finds them.", "quiver remove --source acme --global", "[y/N]")
	if f.callList() != "add-source acme/skills acme" {
		t.Errorf("calls %q", f.callList())
	}
	m = keys(t, m, "y")
	if f.callList() != "add-source acme/skills acme; searchable acme" {
		t.Errorf("calls %q", f.callList())
	}
	contains(t, m, "✓ Made the skills of acme searchable.")

	// No is the default.
	f = fixture()
	f.addedSkills = 2
	m = add(f)
	m = keys(t, m, "enter")
	if f.callList() != "add-source acme/skills acme" || m.confirm != nil {
		t.Errorf("calls %q, confirm %v", f.callList(), m.confirm)
	}
	contains(t, m, "Cancelled.")

	// A source without skills is not asked about, and neither is a failure.
	f = fixture()
	if m = add(f); m.confirm != nil {
		t.Error("asked about a source without skills")
	}
}

// The panel searches the whole library once and keeps, in every view, the
// results that belong to it.
func TestSearchResultsAreIntersectedWithEveryView(t *testing.T) {
	f := fixture()
	f.glob["pdf"] = Searchable
	m := keys(t, start(t, f), "/")
	m = typed(t, m, "migrat")
	found, _ := f.SearchWords("migrat")
	if len(found) != 2 {
		t.Fatalf("the engine found %v", found)
	}
	for v := skillView(0); v < viewCount; v++ {
		var want []string
		for _, id := range found {
			i := slices.IndexFunc(m.data.Skills, func(s Skill) bool { return s.ID == id })
			if inView(m.data.Skills[i], v) {
				want = append(want, id)
			}
		}
		if m.skillCounts[v] != len(want) {
			t.Errorf("view %d: count %d, want %d (%v)", v, m.skillCounts[v], len(want), want)
		}
		m.skillView = v
		m.refilter([secCount]string{}, true)
		var got []string
		for _, i := range m.shown[secSkills] {
			got = append(got, m.data.Skills[i].ID)
		}
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("view %d shows %v, want %v", v, got, want)
		}
	}
}

// shownIDs are the skills the Skills section lists, in order.
func shownIDs(m Model) []string {
	var out []string
	for _, i := range m.shown[secSkills] {
		out = append(out, m.data.Skills[i].ID)
	}
	return out
}

// A key never runs the engine: the words results show at once, and the engine
// runs as a command after the pause.
func TestSearchEngineRunsAsACommandNotOnTheKeystroke(t *testing.T) {
	f := fixture()
	f.fused = map[string][]string{"migrat": {"team/migrate-check", "pdf", "db-migration"}}
	m := keys(t, start(t, f), "right", "right", "right") // the whole library
	m = keys(t, m, "/")
	m = typed(t, m, "migra")
	f.mu.Lock()
	f.engineCalls = nil
	f.mu.Unlock()

	next, cmd := m.Update(keyMsg("t"))
	m = next.(Model)
	if len(f.engineCalls) != 0 {
		t.Fatalf("the engine ran on the keystroke: %v", f.engineCalls)
	}
	if got := shownIDs(m); !slices.Equal(got, []string{"db-migration", "team/migrate-check"}) {
		t.Errorf("words results first: %v", got)
	}
	if cmd == nil {
		t.Fatal("no command for the engine")
	}
	// The command asks the engine and answers with the fused results, which
	// replace the words ones.
	var reply searchedMsg
	found := false
	var run func(tea.Cmd)
	run = func(c tea.Cmd) {
		switch out := c().(type) {
		case searchedMsg:
			reply, found = out, true
		case tea.BatchMsg:
			for _, c := range out {
				if c != nil {
					run(c)
				}
			}
		}
	}
	run(cmd)
	if !found || reply.text != "migrat" || len(f.engineCalls) != 1 || f.engineCalls[0] != "migrat" {
		t.Fatalf("reply %+v (%v), engine calls %v", reply, found, f.engineCalls)
	}
	next, _ = m.Update(reply)
	m = next.(Model)
	if got := shownIDs(m); !slices.Equal(got, []string{"team/migrate-check", "pdf", "db-migration"}) {
		t.Errorf("fused results: %v", got)
	}
}

// A reply for a query that is no longer the current one is dropped.
func TestSearchDropsAStaleReply(t *testing.T) {
	f := fixture()
	m := keys(t, start(t, f), "/")
	m = typed(t, m, "migrat")
	before := shownIDs(m)
	next, _ := m.Update(searchedMsg{text: "mig", ids: []string{"pdf"}})
	m = next.(Model)
	if got := shownIDs(m); !slices.Equal(got, before) {
		t.Errorf("a stale reply changed the list: %v, was %v", got, before)
	}
	if m.fusedText == "mig" {
		t.Error("a stale reply was kept")
	}
	// A failed search leaves the words results.
	next, _ = m.Update(searchedMsg{text: "migrat", err: errors.New("no engine")})
	if got := shownIDs(next.(Model)); !slices.Equal(got, before) {
		t.Errorf("a failed search changed the list: %v", got)
	}
}

// A search that waited behind a newer key never runs.
func TestSearchThatWaitedBehindANewerKeyDoesNotRun(t *testing.T) {
	f := fixture()
	m := keys(t, start(t, f), "/")
	m.searchPause = 20 * time.Millisecond
	f.mu.Lock()
	f.engineCalls = nil
	f.mu.Unlock()
	var cmds []tea.Cmd
	for _, k := range []string{"m", "i", "g"} {
		next, cmd := m.Update(keyMsg(k))
		m = next.(Model)
		cmds = append(cmds, cmd)
	}
	var replies []searchedMsg
	var run func(tea.Cmd)
	run = func(c tea.Cmd) {
		switch out := c().(type) {
		case searchedMsg:
			replies = append(replies, out)
		case tea.BatchMsg:
			for _, c := range out {
				if c != nil {
					run(c)
				}
			}
		}
	}
	for _, c := range cmds {
		if c != nil {
			run(c)
		}
	}
	if len(replies) != 1 || replies[0].text != "mig" {
		t.Errorf("replies %+v", replies)
	}
	if len(f.engineCalls) != 1 {
		t.Errorf("the engine ran %d times: %v", len(f.engineCalls), f.engineCalls)
	}
}

// The results of the engine stay while a view, a reload or the query of another
// section changes: the engine is asked once for the text, and every view takes
// its part of the result.
func TestFusedResultsSurviveViewReloadAndOtherQueries(t *testing.T) {
	f := fixture()
	f.glob["pdf"] = Searchable
	f.fused = map[string][]string{"migrat": {"team/migrate-check", "pdf", "db-migration"}}
	m := keys(t, start(t, f), "/")
	m = typed(t, m, "migrat")
	calls := len(f.engineCalls)
	if calls == 0 {
		t.Fatal("the engine was never asked")
	}

	// Another view takes its part of the same result, in the order of the engine.
	m = keys(t, m, "enter") // leaves the field and keeps the query
	for _, v := range []skillView{viewProject, viewGlobal, viewAll} {
		m.skillView = v
		m.refilter([secCount]string{}, true)
		var want []string
		for _, id := range f.fused["migrat"] {
			i := slices.IndexFunc(m.data.Skills, func(s Skill) bool { return s.ID == id })
			if inView(m.data.Skills[i], v) {
				want = append(want, id)
			}
		}
		if got := shownIDs(m); !slices.Equal(got, want) {
			t.Errorf("view %d: %v, want %v", v, got, want)
		}
	}
	// A reload and a query typed in another section do not ask again.
	data, _ := f.Load("")
	next, _ := m.Update(loadedMsg{data: data, keys: m.selectedKeys()})
	m = next.(Model)
	m = keys(t, m, "2", "/")
	m = typed(t, m, "ap")
	if len(f.engineCalls) != calls {
		t.Errorf("the engine was asked again: %v", f.engineCalls[calls:])
	}
	if m.fusedText != "migrat" {
		t.Errorf("fused text %q", m.fusedText)
	}
}
