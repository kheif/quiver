package server

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/registry"
)

var testLimit = Limit{Max: 100000}

func sessionText(t *testing.T, projectID string) string {
	t.Helper()
	sc, err := BuildSessionContext(projectID, testLimit)
	if err != nil {
		t.Fatal(err)
	}
	return joined(sc.Parts)
}

func instructionsOf(t *testing.T, w Where) string {
	t.Helper()
	return connect(t, New(Options{Version: "dev", Log: discard, Where: w})).InitializeResult().Instructions
}

// The cue in every delivery path: the server instructions, the description of
// search_skills and the session-start text.
func TestCueInEveryDeliveryPath(t *testing.T) {
	w, _ := searchableSkillsEnv(t)
	if err := registry.Update(func(r *registry.Registry) error { return r.MarkAlwaysOnGlobal("pdf") }); err != nil {
		t.Fatal(err)
	}

	got := instructionsOf(t, w)
	// The sentence about list_skills stays: an assistant without a hook
	// never gets the block and depends on it.
	if !strings.Contains(got, "If there is no <quiver-skills> block, call the list_skills tool of this server before your first reply.") {
		t.Errorf("the list_skills sentence changed:\n%s", got)
	}
	want := usage + "Always-on skills: pdf" + ". If the conversation already has a <quiver-always-on> block, these skills are loaded: follow them. Otherwise call the get_skill tool of this server for each before your first reply and follow it for the whole conversation.\n\n" + searchCue + "\n\n" + createHint
	if got != want {
		t.Errorf("instructions:\n%s\nwant\n%s", got, want)
	}
	if strings.Contains(got, "draft") || len(got) > 4096 {
		t.Errorf("the instructions name a skill or are %d bytes", len(got))
	}

	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == "search_skills" && !strings.HasSuffix(tool.Description, searchCue) {
			t.Errorf("search_skills description: %s", tool.Description)
		}
		if tool.Name == "list_skills" && strings.Contains(tool.Description, "made them searchable") {
			t.Errorf("list_skills carries the cue: %s", tool.Description)
		}
	}

	// The session-start text: the always-on skills, the catalog, then the cue.
	text := sessionText(t, w.ProjectID())
	a, c, q := strings.Index(text, "<quiver-always-on"), strings.Index(text, skillsIntro), strings.Index(text, "<quiver-search")
	if a < 0 || c < a || q < c || !strings.Contains(text, searchCue) {
		t.Errorf("order always-on %d, catalog %d, cue %d:\n%s", a, c, q, text)
	}
	if strings.Contains(text, nothingListed) || strings.Contains(text, "draft") {
		t.Errorf("the cue names a skill or says nothing is listed:\n%s", text)
	}
}

func TestCueTextIsTheSecondWordingOfTheSpike(t *testing.T) {
	for _, want := range []string{
		"the topic of the conversation, not only the last message",
		"call search_skills before you start and again when the topic changes",
		"name the product, tool, service or channel the work is for, and what is being produced",
		"search once more from another angle",
		"Load a fitting skill with get_skill without asking, because the user chose it",
		"A plain question about the project needs no search.",
	} {
		if !strings.Contains(searchCue, want) {
			t.Errorf("the cue lacks %q", want)
		}
	}
}

func TestNoCueWhenNothingIsSearchable(t *testing.T) {
	w, _ := skillsEnv(t)
	if got := instructionsOf(t, w); got != usage+createHint {
		t.Errorf("instructions changed without a Searchable skill: %q", got)
	}
	if text := sessionText(t, w.ProjectID()); strings.Contains(text, "searchable") || strings.Contains(text, "quiver-search") {
		t.Errorf("a cue without a Searchable skill:\n%s", text)
	}
}

// A folder in which nothing is listed gets the cue alone, and it says so.
func TestCueAloneWhenNothingIsListed(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	writeFiles(t, filepath.Join(os.Getenv("QUIVER_HOME"), "local"), map[string]string{
		"draft/SKILL.md": "---\nname: draft\ndescription: Searchable only.\n---\nBody.\n",
	})
	if err := registry.Update(func(r *registry.Registry) error { r.AssignSearchableGlobal("draft"); return nil }); err != nil {
		t.Fatal(err)
	}
	sc, err := BuildSessionContext("", testLimit)
	if err != nil {
		t.Fatal(err)
	}
	if len(sc.Parts) != 1 || sc.Count != 1 {
		t.Fatalf("%d parts", len(sc.Parts))
	}
	text := sc.Parts[0]
	if !strings.Contains(text, "<quiver-search") || !strings.Contains(text, searchCue) || !strings.Contains(text, "No skills are listed in this folder") {
		t.Errorf("text:\n%s", text)
	}
	if strings.Contains(text, skillsIntro) || strings.Contains(text, "<quiver-always-on") {
		t.Errorf("a catalog or an always-on block:\n%s", text)
	}
	if n, err := PartsNeeded(testLimit); err != nil || n != 1 {
		t.Errorf("PartsNeeded %d, %v", n, err)
	}
	if got := instructionsOf(t, Where{}); !strings.Contains(got, nothingListed) {
		t.Errorf("instructions:\n%s", got)
	}
}

// Global-only and project-only Searchable skills each give the cue, in their
// own folders.
func TestCueForGlobalAndProjectScopes(t *testing.T) {
	w, _ := skillsEnv(t)
	other := "elsewhere"
	if err := registry.Update(func(r *registry.Registry) error {
		return r.AssignSearchable(w.ProjectID(), "draft")
	}); err != nil {
		t.Fatal(err)
	}
	if text := sessionText(t, w.ProjectID()); !strings.Contains(text, searchCue) {
		t.Errorf("no cue in the project with a project-only Searchable skill:\n%s", text)
	}
	if text := sessionText(t, other); strings.Contains(text, "quiver-search") {
		t.Errorf("cue in a folder where nothing is searchable:\n%s", text)
	}
	if got := instructionsOf(t, w); !strings.Contains(got, searchCue) {
		t.Errorf("instructions of the project lack the cue")
	}
	w.Project = nil
	if got := instructionsOf(t, w); strings.Contains(got, searchCue) {
		t.Errorf("instructions of a folder without the project have the cue")
	}

	if err := registry.Update(func(r *registry.Registry) error {
		r.AssignSearchableGlobal("draft")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if text := sessionText(t, other); !strings.Contains(text, searchCue) {
		t.Errorf("no cue for a global Searchable skill:\n%s", text)
	}
}

// A skill that is Searchable in one scope and listed in the other is listed in
// the folder: it is not searched, and does not make the cue by itself.
func TestCueIgnoresASkillThatIsListedInTheOtherScope(t *testing.T) {
	w, _ := skillsEnv(t)
	if err := registry.Update(func(r *registry.Registry) error {
		r.AssignSearchableGlobal("pdf") // pdf is listed in the project... and global listed
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// pdf is global and listed: moving it to Searchable globally takes it off the list.
	r, _ := registry.Load()
	if len(r.SearchableServed(w.ProjectID())) != 1 {
		t.Fatalf("searchable served %v", r.SearchableServed(w.ProjectID()))
	}
	// Now list it in the project too: the project's list wins in the folder.
	if err := registry.Update(func(r *registry.Registry) error { return r.Assign(w.ProjectID(), "pdf") }); err != nil {
		t.Fatal(err)
	}
	if text := sessionText(t, w.ProjectID()); strings.Contains(text, "quiver-search") {
		t.Errorf("a listed skill alone made the cue:\n%s", text)
	}
	// With another Searchable skill the cue is there, and pdf is not searched.
	if err := registry.Update(func(r *registry.Registry) error { r.AssignSearchableGlobal("draft"); return nil }); err != nil {
		t.Fatal(err)
	}
	if text := sessionText(t, w.ProjectID()); !strings.Contains(text, searchCue) {
		t.Errorf("no cue:\n%s", text)
	}
	set, _ := searchableSet(mustLibrary(t), mustRegistry(t), w.ProjectID())
	for _, s := range set {
		if s.ID == "pdf" {
			t.Errorf("the listed skill is searched")
		}
	}
}

func mustRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	r, err := registry.Load()
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// The check reads the lists of registry.json only: with a skills folder that
// cannot be read it still gives the cue.
func TestCueReadsNoSkillFile(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("file modes are not enforced here")
	}
	t.Setenv("QUIVER_HOME", t.TempDir())
	local := filepath.Join(os.Getenv("QUIVER_HOME"), "local")
	writeFiles(t, local, map[string]string{"draft/SKILL.md": "---\nname: draft\ndescription: Searchable only.\n---\nBody.\n"})
	if err := registry.Update(func(r *registry.Registry) error { r.AssignSearchableGlobal("draft"); return nil }); err != nil {
		t.Fatal(err)
	}
	r := mustRegistry(t)
	if err := os.Chmod(local, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(local, 0o755) })
	if got := cue(r, ""); got != searchCue+nothingListed {
		t.Errorf("cue %q", got)
	}
}

// create_skill is unchanged: a skill the assistant writes is on demand in the
// scope it is created in, and the result says how to make it searchable.
func TestCreateSkillStaysOnDemandAndSaysHowToMakeItSearchable(t *testing.T) {
	w, _ := createEnv(t)
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	text, out, isErr := createSkillCall(t, cs, map[string]any{"name": "notes", "description": "Keep notes.", "body": "Notes.", "scope": "global"})
	if isErr {
		t.Fatalf("create_skill: %s", text)
	}
	r := mustRegistry(t)
	if r.GlobalLevel("notes") != registry.OnDemand {
		t.Errorf("level %v, not on demand", r.GlobalLevel("notes"))
	}
	if !strings.Contains(text, "quiver add "+out.ID+" --searchable") {
		t.Errorf("the result does not say how to make it searchable:\n%s", text)
	}
}
