package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

// new → add → list_skills shows it → remove → gone.
func TestNewAddRemove(t *testing.T) {
	e := newEnv(t)
	app := e.dir("app")
	e.quiver(app, "new", "hello", "--none", "--description", "Say hello to the user.")
	cs := e.mcp(app)
	mustNotContain(t, "list_skills before add", e.call(cs, "list_skills", nil), `"id":"hello"`)

	e.quiver(app, "add", "hello")
	mustContain(t, "list_skills after add", e.call(cs, "list_skills", nil), `"id":"hello"`)
	mustContain(t, "status", e.quiver(app, "status"), "hello  project")

	e.quiver(app, "remove", "hello")
	mustNotContain(t, "list_skills after remove", e.call(cs, "list_skills", nil), `"id":"hello"`)
}

// create_skill with project scope, then with global scope.
func TestCreateSkill(t *testing.T) {
	e := newEnv(t)
	app, other := e.dir("app"), e.dir("other")
	cs := e.mcp(app)
	e.call(cs, "create_skill", map[string]any{
		"name": "release-notes", "description": "Write release notes from the git log.",
		"body": "Group the changes by type.", "scope": "project",
	})
	e.call(cs, "create_skill", map[string]any{
		"name": "commit-style", "description": "Write commit messages in our style.",
		"body": "Use the imperative mood.", "scope": "global",
	})
	// The project skill is served only in app; the global one everywhere.
	mustContain(t, "status in app", e.quiver(app, "status"), "release-notes  project", "commit-style   global")
	got := e.quiver(other, "status")
	mustContain(t, "status in other", got, "commit-style  global")
	mustNotContain(t, "status in other", got, "release-notes")
	if _, err := os.Stat(filepath.Join(app, ".quiver.json")); err != nil {
		t.Errorf("create_skill did not register app: %v", err)
	}
	cs2 := e.mcp(other)
	mustContain(t, "get_skill from other", e.call(cs2, "get_skill", map[string]any{"id": "commit-style"}), "Use the imperative mood.")
}

// import from a project with .claude/skills.
func TestImport(t *testing.T) {
	e := newEnv(t)
	app := e.dir("app")
	e.write(app, map[string]string{
		".claude/skills/deploy/SKILL.md":       skillMD("deploy", "Deploy the app.", "Run make deploy."),
		".claude/skills/deploy/scripts/run.sh": "make deploy\n",
	})
	// Assign to the project, keep the original.
	got, err := e.run(app, "\nn\n", "import")
	if err != nil {
		t.Fatalf("%v\n%s", err, got)
	}
	mustContain(t, "import", got, "Imported deploy into the library and assigned it to this project.")
	if _, err := os.Stat(filepath.Join(app, ".claude", "skills", "deploy", "SKILL.md")); err != nil {
		t.Errorf("original deleted: %v", err)
	}
	cs := e.mcp(app)
	mustContain(t, "get_skill", e.call(cs, "get_skill", map[string]any{"id": "deploy"}), "Run make deploy.", "scripts/run.sh")
}

// source add → add anthropic/x → update → new content served.
func TestSourceUpdate(t *testing.T) {
	e := newEnv(t)
	// The source is a bare repository, like one on a git host.
	work := e.dir("work")
	e.git(work, "init", "--quiet", "--initial-branch=main")
	e.write(work, map[string]string{"skills/x/SKILL.md": skillMD("x", "Skill x.", "Version one.")})
	e.git(work, "add", "--all")
	e.git(work, "commit", "--quiet", "-m", "v1")
	bare := filepath.Join(e.dir("remote"), "anthropic.git")
	e.git(work, "clone", "--quiet", "--bare", work, bare)
	e.git(work, "remote", "add", "origin", bare)

	app := e.dir("app")
	e.quiver(app, "source", "add", bare, "--name", "anthropic")
	e.quiver(app, "add", "anthropic/x")
	cs := e.mcp(app)
	mustContain(t, "get_skill v1", e.call(cs, "get_skill", map[string]any{"id": "anthropic/x"}), "Version one.")

	e.write(work, map[string]string{"skills/x/SKILL.md": skillMD("x", "Skill x.", "Version two.")})
	e.git(work, "commit", "--quiet", "-am", "v2")
	e.git(work, "push", "--quiet", "origin", "main")
	// x is assigned, so update asks first; without an answer nothing changes.
	out, _ := e.run(app, "", "update")
	mustContain(t, "update preview", out, "changed  anthropic/x (assigned: project app)", "anthropic: not updated")
	mustContain(t, "get_skill unchanged", e.call(cs, "get_skill", map[string]any{"id": "anthropic/x"}), "Version one.")
	e.quiver(app, "update", "--yes")
	got := e.call(cs, "get_skill", map[string]any{"id": "anthropic/x"})
	mustContain(t, "get_skill v2", got, "Version two.")
	mustNotContain(t, "get_skill v2", got, "Version one.")
}

// A moved project keeps its skills; a copied one gets the PossibleCopy
// warning, and projects split separates it.
func TestMoveCopySplit(t *testing.T) {
	e := newEnv(t)
	e.quiver(e.home, "new", "pdf", "--none", "--description", "Fill PDF forms.")
	e.quiver(e.home, "new", "notes", "--none", "--description", "Keep notes.")
	app := e.dir("app")
	e.quiver(app, "add", "pdf")

	moved := filepath.Join(filepath.Dir(app), "app-moved")
	if err := os.Rename(app, moved); err != nil {
		t.Fatal(err)
	}
	got := e.quiver(moved, "status")
	mustContain(t, "status after move", got, "pdf  project")
	mustNotContain(t, "status after move", got, "more than one folder")
	// A command that writes records the new folder.
	e.quiver(moved, "add", "notes")

	cp := e.dir("app-copy")
	e.copyDir(moved, cp)
	mustContain(t, "status in copy", e.quiver(cp, "status"), "is in more than one folder", "quiver projects split")

	e.quiver(cp, "projects", "split")
	e.quiver(cp, "remove", "pdf")
	mustNotContain(t, "status in copy after split", e.quiver(cp, "status"), "pdf", "more than one folder")
	mustContain(t, "status in original after split", e.quiver(moved, "status"), "pdf    project", "notes  project")
}

// doctor --fix repairs what it safely can.
func TestDoctorFix(t *testing.T) {
	e := newEnv(t)
	app := e.dir("app")
	e.quiver(app, "new", "pdf", "--description", "Fill PDF forms.")
	// A deleted skill folder: an uncommitted change and a missing assigned ID.
	if err := os.RemoveAll(filepath.Join(e.qhome, "local", "pdf")); err != nil {
		t.Fatal(err)
	}
	e.write(filepath.Join(e.qhome, "sources", ".add-x-123"), map[string]string{"README.md": "left over"})

	got, _ := e.run(app, "", "doctor")
	mustContain(t, "doctor", got, "local/ has uncommitted changes", "project app: pdf", "folders left over")

	got, _ = e.run(app, "y\n", "doctor", "--fix")
	mustContain(t, "doctor --fix", got, "fixed: committed them", "fixed: unassigned 1 skill", "fixed: deleted them")

	// Only the Claude Code check can still fail: claude is not on PATH.
	got, _ = e.run(app, "", "doctor")
	mustNotContain(t, "doctor after --fix", got, "uncommitted", "project app: pdf", "left over", "error  library", "error  assigned")
}
