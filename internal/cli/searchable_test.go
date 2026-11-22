package cli

import (
	"encoding/json"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/server"
)

func mustRun(t *testing.T, stdin string, args ...string) string {
	t.Helper()
	out, err := runIn(t, stdin, args...)
	if err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
	return out
}

func wantOut(t *testing.T, out string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("output missing %q:\n%s", w, out)
		}
	}
}

func TestAddSearchableProject(t *testing.T) {
	dir := assignEnv(t, "pdf", "tdd", "notes")
	mustRun(t, "", "add", "pdf", "tdd", "--project", dir, "--yes")
	mustRun(t, "", "add", "tdd", "--always", "--project", dir)

	// A move says what it did, and takes the always-on mark with it.
	out := mustRun(t, "", "add", "pdf", "tdd", "notes", "--searchable", "--project", dir)
	wantOut(t, out,
		"Made Searchable to app: pdf, tdd, notes",
		"note: pdf was on demand there; it is Searchable now",
		"note: tdd was always-on there; it is Searchable now",
		"Searchable in app (not listed; found by searching): notes, pdf, tdd",
		"(none)")
	if strings.Contains(out, "note: notes") {
		t.Errorf("a new skill is not a move:\n%s", out)
	}
	p := loadRegistry(t).Projects[onlyProject(t)]
	if len(p.Skills) != 0 || len(p.AlwaysOn) != 0 || !reflect.DeepEqual(p.Searchable, []string{"notes", "pdf", "tdd"}) {
		t.Errorf("project %+v", p)
	}
	if !strings.Contains(readHomeFile(t, registryDir(t), "registry.json"), `"version": 3`) {
		t.Error("registry.json is not version 3")
	}

	// Without the flag the skill moves back to On demand.
	out = mustRun(t, "", "add", "pdf", "--project", dir)
	wantOut(t, out, "note: pdf was Searchable there; it is on demand now")
	p = loadRegistry(t).Projects[onlyProject(t)]
	if !reflect.DeepEqual(p.Skills, []string{"pdf"}) || !reflect.DeepEqual(p.Searchable, []string{"notes", "tdd"}) {
		t.Errorf("project %+v", p)
	}
}

func registryDir(t *testing.T) string {
	t.Helper()
	path, err := registry.Path()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(path)
}

func onlyProject(t *testing.T) string {
	t.Helper()
	r := loadRegistry(t)
	if len(r.Projects) != 1 {
		t.Fatalf("projects: %+v", r.Projects)
	}
	for id := range r.Projects {
		return id
	}
	return ""
}

func TestAddSearchableGlobal(t *testing.T) {
	assignEnv(t, "pdf", "tdd")
	mustRun(t, "", "add", "pdf", "--global", "--always")
	mustRun(t, "", "add", "tdd", "--global")

	out := mustRun(t, "", "add", "pdf", "tdd", "--searchable", "--global")
	wantOut(t, out, "Made Searchable globally: pdf, tdd",
		"note: pdf was always-on there; it is Searchable now",
		"note: tdd was on demand there; it is Searchable now",
		"Searchable in every folder (not listed; found by searching): pdf, tdd")
	r := loadRegistry(t)
	if len(r.Global) != 0 || len(r.AlwaysOn) != 0 || !reflect.DeepEqual(r.Searchable, []string{"pdf", "tdd"}) {
		t.Errorf("registry %+v", r)
	}

	// On demand keeps an always-on mark, as before: a skill that was
	// Searchable has none, and an always-on skill stays always-on.
	mustRun(t, "", "add", "tdd", "--global", "--always")
	mustRun(t, "", "add", "tdd", "--global")
	r = loadRegistry(t)
	if !reflect.DeepEqual(r.Global, []string{"tdd"}) || !reflect.DeepEqual(r.AlwaysOn, []string{"tdd"}) || !reflect.DeepEqual(r.Searchable, []string{"pdf"}) {
		t.Errorf("registry %+v", r)
	}
}

func TestAddSearchableErrors(t *testing.T) {
	assignEnv(t, "pdf")
	for label, args := range map[string][]string{
		"with --always":            {"add", "pdf", "--searchable", "--always", "--global"},
		"--source without it":      {"add", "--source", "acme", "--global"},
		"--source and IDs":         {"add", "pdf", "--source", "acme", "--searchable", "--global"},
		"unknown source":           {"add", "--source", "nope", "--searchable", "--global"},
		"unknown skill":            {"add", "nope", "--searchable", "--global"},
		"remove --source and IDs":  {"remove", "pdf", "--source", "acme", "--global"},
		"remove --source --always": {"remove", "--source", "acme", "--always", "--global"},
		"remove unknown source":    {"remove", "--source", "nope", "--global"},
	} {
		if out, err := runIn(t, "", args...); err == nil {
			t.Errorf("%s: no error:\n%s", label, out)
		}
	}
	if r := loadRegistry(t); len(r.Global)+len(r.Searchable) != 0 {
		t.Errorf("an error changed the registry: %+v", r)
	}
}

func TestAddSourceSearchable(t *testing.T) {
	dir := assignEnv(t, "pdf")
	repo := skillRepo(t, "acme", "review", "deploy", "lint", "plan")
	mustRun(t, "", "source", "add", repo, "--name", "acme")
	mustRun(t, "", "add", "acme/review", "--global")
	mustRun(t, "", "add", "acme/lint", "--global", "--always")

	// Listed and always-on skills keep their level; the rest is Searchable.
	out := mustRun(t, "", "add", "--source", "acme", "--searchable", "--global")
	wantOut(t, out, "Made Searchable globally: acme/deploy, acme/plan")
	r := loadRegistry(t)
	if !reflect.DeepEqual(r.Global, []string{"acme/lint", "acme/review"}) || !reflect.DeepEqual(r.AlwaysOn, []string{"acme/lint"}) ||
		!reflect.DeepEqual(r.Searchable, []string{"acme/deploy", "acme/plan"}) {
		t.Errorf("registry %+v", r)
	}
	// Again: nothing is unassigned any more.
	wantOut(t, mustRun(t, "", "add", "--source", "acme", "--searchable", "--global"), "assigned in every folder already; nothing changed")

	// In a project: the project scope decides, so a skill that is global is
	// still unassigned there, and the global scope is not touched.
	out = mustRun(t, "", "add", "--source", "acme", "--searchable", "--project", dir, "--yes")
	wantOut(t, out, "Made Searchable to app: acme/deploy, acme/lint, acme/plan, acme/review")
	p := loadRegistry(t).Projects[onlyProject(t)]
	if len(p.Skills) != 0 || len(p.Searchable) != 4 {
		t.Errorf("project %+v", p)
	}

	// A source without the flag is an error, and nothing changes.
	if _, err := runIn(t, "", "add", "--source", "acme", "--global"); err == nil {
		t.Error("--source without --searchable")
	}
}

func TestRemoveFromEveryLevel(t *testing.T) {
	dir := assignEnv(t, "pdf", "tdd", "notes")
	mustRun(t, "", "add", "pdf", "tdd", "notes", "--project", dir, "--yes")
	mustRun(t, "", "add", "pdf", "--always", "--project", dir)
	mustRun(t, "", "add", "tdd", "--searchable", "--project", dir)

	out := mustRun(t, "", "remove", "pdf", "tdd", "--project", dir)
	wantOut(t, out, "Removed from app: pdf, tdd")
	p := loadRegistry(t).Projects[onlyProject(t)]
	if !reflect.DeepEqual(p.Skills, []string{"notes"}) || len(p.AlwaysOn) != 0 || len(p.Searchable) != 0 {
		t.Errorf("project %+v", p)
	}

	mustRun(t, "", "add", "pdf", "--searchable", "--global")
	mustRun(t, "", "add", "tdd", "--global", "--always")
	out = mustRun(t, "", "remove", "pdf", "tdd", "--global")
	wantOut(t, out, "Removed from every folder: pdf, tdd")
	r := loadRegistry(t)
	if len(r.Global) != 0 || len(r.AlwaysOn) != 0 || len(r.Searchable) != 0 {
		t.Errorf("registry %+v", r)
	}

	// A skill that is Searchable globally is not removed by the project scope.
	mustRun(t, "", "add", "pdf", "--searchable", "--global")
	out = mustRun(t, "", "remove", "pdf", "--project", dir)
	wantOut(t, out, "note: pdf is Searchable globally; remove it with --global")
}

func TestRemoveSource(t *testing.T) {
	dir := assignEnv(t, "pdf")
	repo := skillRepo(t, "acme", "review", "deploy")
	other := skillRepo(t, "beta", "one")
	mustRun(t, "", "source", "add", repo, "--name", "acme")
	mustRun(t, "", "source", "add", other, "--name", "beta")
	mustRun(t, "", "add", "acme/review", "beta/one", "pdf", "--global")
	mustRun(t, "", "add", "acme/deploy", "--searchable", "--global")
	mustRun(t, "", "add", "acme/review", "acme/deploy", "--project", dir, "--yes")

	// Without --global only the project is changed.
	out := mustRun(t, "", "remove", "--source", "acme", "--project", dir)
	wantOut(t, out, "Removed from app: acme/deploy, acme/review")
	r := loadRegistry(t)
	if len(r.Global) != 3 || len(r.Searchable) != 1 || len(r.Projects[onlyProject(t)].Skills) != 0 {
		t.Errorf("registry %+v", r)
	}

	// With --global every level of the global scope, and no other source.
	out = mustRun(t, "", "remove", "--source", "acme", "--global")
	wantOut(t, out, "Removed from every folder: acme/deploy, acme/review")
	r = loadRegistry(t)
	if !reflect.DeepEqual(r.Global, []string{"beta/one", "pdf"}) || len(r.Searchable) != 0 {
		t.Errorf("registry %+v", r)
	}
	wantOut(t, mustRun(t, "", "remove", "--source", "acme", "--global"), "No skill of source acme is assigned in every folder.")
}

// searchableEnv is the list fixture with one more skill that is Searchable
// in every folder (acme/deploy) and notes that is Searchable in app.
func searchableEnv(t *testing.T) fixture {
	t.Helper()
	f := listEnv(t)
	mustRun(t, "", "add", "acme/deploy", "--searchable", "--global")
	mustRun(t, "", "add", "notes", "--searchable", "--project", f.app)
	t.Setenv(server.ProjectEnv, "")
	t.Chdir(f.app)
	return f
}

func TestListAndWhereShowTheLevel(t *testing.T) {
	f := searchableEnv(t)

	var l struct {
		Skills []listedSkill `json:"skills"`
	}
	if err := json.Unmarshal([]byte(mustRun(t, "", "list", "--json")), &l); err != nil {
		t.Fatal(err)
	}
	byID := map[string]listedSkill{}
	for _, s := range l.Skills {
		byID[s.ID] = s
	}
	if d := byID["acme/deploy"]; d.Global || len(d.Projects) != 0 || !d.Searchable.Global || len(d.Searchable.Projects) != 0 {
		t.Errorf("acme/deploy %+v", d)
	}
	if n := byID["notes"]; n.Global || len(n.Projects) != 0 || n.Searchable.Global || !reflect.DeepEqual(n.Searchable.Projects, []string{"app-000001"}) {
		t.Errorf("notes %+v", n)
	}
	// The served fields are as before.
	if r := byID["acme/review"]; !r.Global || !reflect.DeepEqual(r.Projects, []string{"app-000001"}) || r.Searchable.Global {
		t.Errorf("acme/review %+v", r)
	}
	out := mustRun(t, "", "list")
	wantOut(t, out, "searchable (global)", "searchable (1 project)")

	// A Searchable skill is assigned: --unassigned leaves it out.
	un := mustRun(t, "", "list", "--unassigned")
	if strings.Contains(un, "acme/deploy") || strings.Contains(un, "notes") {
		t.Errorf("--unassigned lists a Searchable skill:\n%s", un)
	}

	wantOut(t, mustRun(t, "", "where", "acme/deploy"), "searchable, global")
	wantOut(t, mustRun(t, "", "where", "notes"), "searchable, project app ("+f.app+")")
	var w whereJSON
	if err := json.Unmarshal([]byte(mustRun(t, "", "where", "notes", "--json")), &w); err != nil {
		t.Fatal(err)
	}
	if w.Global || len(w.Projects) != 0 || w.Searchable.Global || len(w.Searchable.Projects) != 1 || w.Searchable.Projects[0].ID != "app-000001" {
		t.Errorf("where notes %+v", w)
	}
	if out := mustRun(t, "", "where", "pdf"); strings.Contains(out, "searchable") {
		t.Errorf("a served skill:\n%s", out)
	}
}

func TestShowAndStatusShowTheLevel(t *testing.T) {
	searchableEnv(t)

	out := mustRun(t, "", "show", "acme/deploy")
	wantOut(t, out, "Assigned: nowhere", "Searchable: every folder")
	var sh shownSkill
	if err := json.Unmarshal([]byte(mustRun(t, "", "show", "notes", "--json")), &sh); err != nil {
		t.Fatal(err)
	}
	if len(sh.Assigned) != 0 || len(sh.Searchable) != 1 || sh.Searchable[0].Project != "app-000001" {
		t.Errorf("show notes: assigned %+v, searchable %+v", sh.Assigned, sh.Searchable)
	}

	out = mustRun(t, "", "status")
	wantOut(t, out, "Searchable in app (not listed; found by searching): acme/deploy, notes")
	var st statusJSON
	if err := json.Unmarshal([]byte(mustRun(t, "", "status", "--json")), &st); err != nil {
		t.Fatal(err)
	}
	want := []searchableSkill{{ID: "acme/deploy", Scope: "global", InLibrary: true}, {ID: "notes", Scope: "project", InLibrary: true}}
	if !reflect.DeepEqual(st.Searchable, want) {
		t.Errorf("status searchable %+v", st.Searchable)
	}
	for _, s := range st.Skills {
		if s.ID == "acme/deploy" || s.ID == "notes" {
			t.Errorf("%s is served in status", s.ID)
		}
	}
}

func TestSearchShowsTheLevel(t *testing.T) {
	searchableEnv(t)
	// A name finds a skill at every level: the search covers the library.
	levels := map[string]foundSkill{}
	for _, q := range []string{"deploy", "notes", "review", "pdf", "tdd"} {
		var res struct {
			Skills []foundSkill `json:"skills"`
		}
		if err := json.Unmarshal([]byte(mustRun(t, "", "search", q, "--json")), &res); err != nil {
			t.Fatal(err)
		}
		for _, s := range res.Skills {
			levels[s.ID] = s
		}
	}
	for id, want := range map[string][2]string{
		"acme/deploy": {"", "searchable"},
		"notes":       {"", "searchable"},
		"acme/review": {"global", "on-demand"},
		"pdf":         {"global", "always-on"},
		"tdd":         {"project", "on-demand"},
	} {
		if got := levels[id]; got.Here != want[0] || got.Level != want[1] {
			t.Errorf("%s: here %q level %q, want %v", id, got.Here, got.Level, want)
		}
	}
	wantOut(t, mustRun(t, "", "search", "deploy"), "searchable")
}

func TestOfferSearchable(t *testing.T) {
	ids := []string{"acme/a", "acme/b"}
	cases := []struct {
		name     string
		terminal bool
		flag     bool
		input    string
		asked    bool
		want     []string // Searchable after
	}{
		{"yes", true, false, "y\n", true, ids},
		{"yes in full", true, false, "yes\n", true, ids},
		{"no", true, false, "n\n", true, nil},
		{"default", true, false, "\n", true, nil},
		{"no input", true, false, "", true, nil},
		{"no terminal", false, false, "y\n", false, nil},
		{"flag", false, true, "", false, ids},
		{"flag in a terminal", true, true, "", false, ids},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assignEnv(t)
			old := interactive
			interactive = func(io.Reader) bool { return c.terminal }
			t.Cleanup(func() { interactive = old })
			var out strings.Builder
			if err := offerSearchable(strings.NewReader(c.input), &out, "acme", ids, c.flag); err != nil {
				t.Fatal(err)
			}
			if asked := strings.Contains(out.String(), "searchable in every folder? The assistant will then load them without asking when a search finds them. [y/N]"); asked != c.asked {
				t.Errorf("asked = %v:\n%s", asked, out.String())
			}
			if got := loadRegistry(t).Searchable; !reflect.DeepEqual(got, c.want) {
				t.Errorf("Searchable = %v, want %v", got, c.want)
			}
			undo := "To undo: quiver remove --source acme --global"
			if strings.Contains(out.String(), undo) != (c.want != nil) {
				t.Errorf("undo command:\n%s", out.String())
			}
		})
	}
	// A source without skills is not asked about.
	assignEnv(t)
	fakeTerminal(t)
	var out strings.Builder
	if err := offerSearchable(strings.NewReader("y\n"), &out, "acme", nil, false); err != nil || out.Len() != 0 {
		t.Errorf("no skills: %v %q", err, out.String())
	}
}

func TestSourceAddSearchable(t *testing.T) {
	assignEnv(t)
	repo := skillRepo(t, "acme", "review", "deploy")

	// No terminal and no flag: nothing is assigned.
	out := mustRun(t, "y\n", "source", "add", repo, "--name", "acme")
	wantOut(t, out, "Read them with quiver show <id> before you assign them.")
	if strings.Contains(out, "searchable") || len(loadRegistry(t).Searchable) != 0 {
		t.Errorf("assigned without a yes:\n%s", out)
	}

	// --searchable says yes, and the printed command undoes exactly that.
	repo2 := skillRepo(t, "beta", "one", "two", "three")
	out = mustRun(t, "", "source", "add", repo2, "--name", "beta", "--searchable")
	wantOut(t, out, "Made 3 skills Searchable in every folder.", "To undo: quiver remove --source beta --global")
	if got := loadRegistry(t).Searchable; !reflect.DeepEqual(got, []string{"beta/one", "beta/three", "beta/two"}) {
		t.Errorf("Searchable = %v", got)
	}
	mustRun(t, "", "add", "acme/review", "--global")
	mustRun(t, "", "remove", "--source", "beta", "--global")
	r := loadRegistry(t)
	if len(r.Searchable) != 0 || !reflect.DeepEqual(r.Global, []string{"acme/review"}) {
		t.Errorf("registry %+v", r)
	}
}

func TestUpdateWithSearchableSkills(t *testing.T) {
	sourceEnv(t)
	bare := bareSource(t, "acme", "pdf", "review")
	mustRun(t, "", "add", "acme/pdf", "--searchable", "--global")

	// A skill that the source adds later is not Searchable.
	push(t, bare, map[string]string{"skills/new/SKILL.md": "---\nname: new\ndescription: New.\n---\n"}, false)
	out := mustRun(t, "", "update")
	wantOut(t, out, "added    acme/new")
	if strings.Contains(out, "Apply the changes") {
		t.Errorf("asked about a skill nobody assigned:\n%s", out)
	}
	if got := loadRegistry(t).Searchable; !reflect.DeepEqual(got, []string{"acme/pdf"}) {
		t.Errorf("Searchable = %v", got)
	}

	// A change to a Searchable skill is previewed and waits for a yes, with
	// one question for the source.
	push(t, bare, map[string]string{
		"skills/pdf/SKILL.md": "---\nname: pdf\ndescription: Updated.\n---\n",
		"skills/new/SKILL.md": "---\nname: new\ndescription: Changed too.\n---\n",
	}, false)
	out = mustRun(t, "n\n", "update")
	wantOut(t, out, "changed  acme/pdf (assigned: searchable)\n           M SKILL.md  +1 -1\n", "changed  acme/new\n", "Apply the changes to acme? [y/N]", "acme: not updated")
	if n := strings.Count(out, "Apply the changes"); n != 1 {
		t.Errorf("%d questions:\n%s", n, out)
	}
	out = mustRun(t, "y\n", "update")
	wantOut(t, out, "acme: updated to")
}
