package tui

import (
	"fmt"
	"slices"
	"strings"
	"testing"
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
	found, _ := f.Search("migrat")
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
