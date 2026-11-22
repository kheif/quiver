package cli

import (
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
)

// The hook passes the cue when the folder has a Searchable skill, alone when
// nothing is listed, and nothing when nothing is Searchable.
func TestHookPassesTheCueForSearchableSkills(t *testing.T) {
	dir := assignEnv(t, "pdf", "tdd")
	if out, errOut := runHookCmd(t, cwdInput(dir), "session-start", "claude-code"); out != "" || errOut != "" {
		t.Fatalf("an unassigned skill gave output: %q %q", out, errOut)
	}

	mustRun(t, "", "add", "pdf", "--searchable", "--global")
	out, errOut := runHookCmd(t, cwdInput(dir), "session-start", "claude-code")
	_, text := nestedContext(t, out)
	for _, want := range []string{"<quiver-search>", "call search_skills before you start", "No skills are listed in this folder"} {
		if !strings.Contains(text, want) {
			t.Errorf("cue alone: missing %q in\n%s", want, text)
		}
	}
	if errOut != "" || strings.Contains(text, "- pdf") || strings.Contains(text, "quiver-skills>\n") {
		t.Errorf("the cue alone names a skill or has a catalog: stderr %q\n%s", errOut, text)
	}

	// A listed skill: the cue comes after the catalog and no longer says that
	// nothing is listed.
	mustRun(t, "", "add", "tdd", "--global")
	out, _ = runHookCmd(t, cwdInput(dir), "session-start", "claude-code")
	_, text = nestedContext(t, out)
	if !strings.Contains(text, "- tdd: ") || strings.Index(text, "<quiver-search>") < strings.Index(text, "- tdd: ") || strings.Contains(text, "No skills are listed") {
		t.Errorf("catalog and cue:\n%s", text)
	}

	// Back to nothing Searchable: the old behaviour.
	mustRun(t, "", "add", "pdf", "--global")
	out, _ = runHookCmd(t, cwdInput(dir), "session-start", "claude-code")
	if _, text = nestedContext(t, out); strings.Contains(text, "quiver-search") {
		t.Errorf("a cue without a Searchable skill:\n%s", text)
	}
}

// What the user writes or imports is Searchable; a skill that is already
// assigned keeps its level.
func TestImportKeepsTheLevelOfAnAssignedSkill(t *testing.T) {
	f := listEnv(t)
	dir := importProject(t, f)
	t.Chdir(dir)
	mustRun(t, "", "add", "notes", "--project", dir, "--yes")
	out := mustRun(t, "", "import", "--yes", "--keep")
	wantOut(t, out, "notes is already in the library with the same content")
	r := loadRegistry(t)
	p := r.Projects["web-000003"]
	if r.ProjectLevel("web-000003", "notes") != registry.OnDemand {
		t.Errorf("notes: level %v, searchable %v, skills %v", r.ProjectLevel("web-000003", "notes"), p.Searchable, p.Skills)
	}
	if r.ProjectLevel("web-000003", "review") != registry.Searchable {
		t.Errorf("a newly imported skill is not Searchable: %v", r.ProjectLevel("web-000003", "review"))
	}
}

func TestNewSaysItIsSearchableAndHowToMoveIt(t *testing.T) {
	f := listEnv(t)
	out := mustRun(t, "", "new", "fresh", "--description", "A fresh skill.", "--project", f.app)
	wantOut(t, out, "Assigned to app as Searchable: fresh", "It is Searchable: not listed at session start, found by searching. quiver add fresh moves it to another level.")
	if registry.Searchable != loadRegistry(t).ProjectLevel("app-000001", "fresh") {
		t.Error("not Searchable")
	}
	help := mustRun(t, "", "new", "--help")
	wantOut(t, help, "Searchable", "quiver add <id>")
	help = mustRun(t, "", "import", "--help")
	wantOut(t, help, "Searchable", "keeps its")
	mustRun(t, "", "add", "fresh", "--project", f.app)
	if loadRegistry(t).ProjectLevel("app-000001", "fresh") != registry.OnDemand {
		t.Error("quiver add did not move it to on demand")
	}
}

// quiver fork is unchanged: the copy takes the place and level of the original.
func TestForkKeepsTheLevelOfTheOriginal(t *testing.T) {
	f := listEnv(t)
	_ = f
	mustRun(t, "", "add", "acme/deploy", "--searchable", "--global")
	mustRun(t, "", "fork", "acme/deploy", "--as", "my-deploy")
	if got := loadRegistry(t).GlobalLevel("my-deploy"); got != registry.Searchable {
		t.Errorf("the fork has level %v", got)
	}
}

// New and import offer the engine setup after the first Searchable skill.
func TestNewAndImportOfferTheSetupOnce(t *testing.T) {
	f := listEnv(t)
	h := setupHost(t)
	asTerminal(t)
	out := mustRun(t, "\n", "new", "fresh", "--description", "A fresh skill.", "--project", f.app)
	wantOut(t, out, "Set up search by meaning now?")
	if !library.LoadEngine().Offered || h.Requests() != 0 {
		t.Errorf("after no: %+v, %d requests", library.LoadEngine(), h.Requests())
	}
	out = mustRun(t, "y\n", "new", "second", "--description", "Another.", "--project", f.app)
	if strings.Contains(out, "Set up search") {
		t.Errorf("the offer was repeated:\n%s", out)
	}
	// A command that assigns nothing Searchable offers nothing.
	writeSearchFile(t, `{"version": 1, "engine": "words", "local": {"model": "tiny"}}`)
	out = mustRun(t, "y\n", "new", "loose", "--description", "Not assigned.", "--none")
	if strings.Contains(out, "Set up search") {
		t.Errorf("offered after --none:\n%s", out)
	}
}
