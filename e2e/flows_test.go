package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/embed/embedtest"
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
	mustContain(t, "import", got, "Imported deploy into the library and made it Searchable in this project.")
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

// source add --searchable, source add without a terminal, add --searchable,
// list, where and remove --source, through the binary.
func TestSearchableAssignment(t *testing.T) {
	e := newEnv(t)
	source := func(name string, skills ...string) string {
		work := e.dir(name)
		e.git(work, "init", "--quiet", "--initial-branch=main")
		files := map[string]string{}
		for _, s := range skills {
			files["skills/"+s+"/SKILL.md"] = skillMD(s, "Skill "+s+".", "Text of "+s+".")
		}
		e.write(work, files)
		e.git(work, "add", "--all")
		e.git(work, "commit", "--quiet", "-m", "v1")
		return work
	}
	app := e.dir("app")

	// A yes at no terminal is not read: without the flag nothing is assigned.
	out, err := e.run(app, "y\n", "source", "add", source("plain", "p1", "p2"), "--name", "plain")
	if err != nil {
		t.Fatalf("source add: %v\n%s", err, out)
	}
	mustContain(t, "source add", out, "Read them with quiver show <id> before you assign them.")
	mustNotContain(t, "source add", out, "Made", "To undo")
	mustContain(t, "list --unassigned", e.quiver(app, "list", "--unassigned"), "plain/p1", "plain/p2")

	// --searchable says yes and prints the command that undoes it.
	out = e.quiver(app, "source", "add", source("tools", "t1", "t2"), "--name", "tools", "--searchable")
	mustContain(t, "source add --searchable", out, "Made 2 skills Searchable in every folder.", "To undo: quiver remove --source tools --global")
	got := e.quiver(app, "list")
	mustContain(t, "list", got, "searchable (global)")
	mustContain(t, "where", e.quiver(app, "where", "tools/t1"), "searchable, global")
	mustNotContain(t, "list --unassigned", e.quiver(app, "list", "--unassigned"), "tools/")

	// add --searchable in a project; a listed skill keeps its level under --source.
	e.quiver(app, "add", "plain/p1")
	out = e.quiver(app, "add", "--source", "plain", "--searchable")
	mustContain(t, "add --source", out, "Made Searchable to app: plain/p2")
	mustContain(t, "status", e.quiver(app, "status"), "plain/p1  project", "Searchable in app (not listed; found by searching): plain/p2, tools/t1, tools/t2")
	e.quiver(app, "add", "plain/p1", "--searchable")
	mustContain(t, "where after the move", e.quiver(app, "where", "plain/p1"), "searchable, project app")

	// remove --source: the project first, then every folder.
	mustContain(t, "remove --source", e.quiver(app, "remove", "--source", "plain"), "Removed from app: plain/p1, plain/p2")
	mustContain(t, "where after remove", e.quiver(app, "where", "plain/p1"), "not assigned anywhere")
	e.quiver(app, "remove", "--source", "tools", "--global")
	mustContain(t, "list after undo", e.quiver(app, "list", "--unassigned"), "tools/t1", "tools/t2", "plain/p1")
}

func TestSearchEngine(t *testing.T) {
	e := newEnv(t)
	app := e.dir("app")
	e.write(filepath.Join(e.qhome, "local"), map[string]string{
		"deploy-check/SKILL.md": skillMD("deploy-check", "Check a deploy.", "Check."),
	})
	mustContain(t, "engine", e.quiver(app, "engine"), "Search engine: words")
	mustContain(t, "engine use", e.quiver(app, "engine", "use", "words"), "Search engine: words")
	mustContain(t, "search", e.quiver(app, "search", "deploy"), "deploy-check")
	mustContain(t, "search --engine", e.quiver(app, "search", "--engine", "words", "deploy"), "deploy-check")
	mustContain(t, "status", e.quiver(app, "status"), "Search engine: words")
	got, _ := e.run(app, "", "doctor")
	mustContain(t, "doctor", got, "search engine: words")

	// A file of a newer version is refused: search still runs as words, and
	// the file is left as it is.
	file := filepath.Join(e.qhome, "state", "search.json")
	const newer = `{"version": 2, "engine": "local"}`
	e.write(filepath.Dir(file), map[string]string{"search.json": newer})
	msg := "search.json has version 2, but this quiver supports only up to 1: update quiver"
	mustContain(t, "engine with a newer file", e.quiver(app, "engine"), msg)
	mustContain(t, "status with a newer file", e.quiver(app, "status"), msg)
	mustContain(t, "search with a newer file", e.quiver(app, "search", "deploy"), "deploy-check")
	got, _ = e.run(app, "", "doctor")
	mustContain(t, "doctor with a newer file", got, msg)
	out, err := e.run(app, "", "engine", "use", "words")
	if err == nil || !strings.Contains(out, msg) {
		t.Errorf("engine use: %v\n%s", err, out)
	}
	if data, _ := os.ReadFile(file); string(data) != newer {
		t.Errorf("the file was rewritten: %s", data)
	}
}

// The local engine, against the tiny model with random weights that the tests
// generate: quiver is built with the tag tinymodel, which knows the model
// "tiny" and nothing else.
func TestLocalEngine(t *testing.T) {
	dir := t.TempDir()
	tiny := filepath.Join(dir, "quiver-tiny")
	if runtime.GOOS == "windows" {
		tiny += ".exe"
	}
	build := exec.Command("go", "build", "-tags", "tinymodel", "-o", tiny, "../cmd/quiver")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	saved := bin
	bin = tiny
	defer func() { bin = saved }()

	e := newEnv(t)
	app := e.dir("app")
	e.write(filepath.Join(e.qhome, "local"), map[string]string{
		"deploy-check/SKILL.md": skillMD("deploy-check", "Check a deploy.", "Check."),
		"slack-style/SKILL.md":  skillMD("slack-style", "How to write a Slack message.", "Style."),
	})
	embedtest.WriteModel(t, filepath.Join(e.qhome, "cache", "models", "tiny"), 1)
	e.write(filepath.Join(e.qhome, "state"), map[string]string{
		"search.json": `{"version": 1, "engine": "words", "local": {"model": "tiny", "minScore": -1}}`,
	})

	// Under words nothing is embedded and no index is made.
	mustContain(t, "search under words", e.quiver(app, "search", "deploy"), "deploy-check")
	if _, err := os.Stat(filepath.Join(e.qhome, "cache", "embeddings.json")); !os.IsNotExist(err) {
		t.Errorf("an index under words: %v", err)
	}

	mustContain(t, "engine use", e.quiver(app, "engine", "use", "local"), "Search engine: local")
	mustContain(t, "engine", e.quiver(app, "engine"), "Search engine: local")
	mustContain(t, "engine reindex", e.quiver(app, "engine", "reindex"), "Indexed 2 skills with tiny.")
	mustContain(t, "search", e.quiver(app, "search", "deploy"), "deploy-check")
	data, err := os.ReadFile(filepath.Join(e.qhome, "cache", "embeddings.json"))
	if err != nil || !strings.Contains(string(data), `"model": "tiny"`) || strings.Count(string(data), `"id":`) != 2 {
		t.Errorf("index %s, %v", data, err)
	}
	// The block of the file stays when the engine is chosen again.
	e.quiver(app, "engine", "use", "local")
	if data, _ := os.ReadFile(filepath.Join(e.qhome, "state", "search.json")); !strings.Contains(string(data), `"minScore": -1`) {
		t.Errorf("the local block was dropped: %s", data)
	}
	got, _ := e.run(app, "", "doctor")
	mustContain(t, "doctor", got, "search engine: local, model tiny", "2 of 2 skills indexed")

	// The model file goes: search runs as words, with one line, and doctor
	// names the missing file.
	if err := os.Remove(filepath.Join(e.qhome, "cache", "models", "tiny", "model.safetensors")); err != nil {
		t.Fatal(err)
	}
	out, err := e.run(app, "", "search", "deploy")
	if err != nil || !strings.Contains(out, "deploy-check") || strings.Count(out, "searching with words") != 1 {
		t.Errorf("search without the model file: %v\n%s", err, out)
	}
	got, _ = e.run(app, "", "doctor")
	mustContain(t, "doctor without the model file", got, "model.safetensors is missing")
}

// The setup through the binary against a fake model host: the question, the
// download, the checks, the record, and the undo. The binary is built with the
// tag tinymodel, which can trust the host of the test.
func TestEngineSetupThroughTheBinary(t *testing.T) {
	dir := t.TempDir()
	tiny := filepath.Join(dir, "quiver-tiny")
	if runtime.GOOS == "windows" {
		tiny += ".exe"
	}
	build := exec.Command("go", "build", "-tags", "tinymodel", "-o", tiny, "../cmd/quiver")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	saved := bin
	bin = tiny
	defer func() { bin = saved }()

	host := embedtest.NewHost(t, 1)
	pin := strings.Join([]string{host.URL, strconv.Itoa(len(host.Weights)), embedtest.Hash(host.Weights), strconv.Itoa(len(host.Tokenizer)), embedtest.Hash(host.Tokenizer)}, " ")
	t.Setenv("QUIVER_TINY_MODEL_PIN", pin)

	e := newEnv(t)
	app := e.dir("app")
	e.write(filepath.Join(e.qhome, "local"), map[string]string{
		"deploy-check/SKILL.md": skillMD("deploy-check", "Check a deploy.", "Check."),
		"slack-style/SKILL.md":  skillMD("slack-style", "How to write a Slack message.", "Style."),
	})
	e.write(filepath.Join(e.qhome, "state"), map[string]string{
		"search.json": `{"version": 1, "engine": "words", "local": {"model": "tiny", "minScore": -1}}`,
	})

	// Without a terminal and without --yes nothing changes.
	out, err := e.run(app, "y\n", "engine", "setup")
	if err != nil || !strings.Contains(out, "Nothing changed") || host.Requests() != 0 {
		t.Fatalf("setup without a terminal: %v, %d requests\n%s", err, host.Requests(), out)
	}

	out, err = e.run(app, "", "engine", "setup", "--yes")
	if err != nil {
		t.Fatalf("setup: %v\n%s", err, out)
	}
	mustContain(t, "setup", out, "Search by meaning is on.", "Downloading model.safetensors: 100%", "Indexing skills for the search: 2 of 2")
	if host.Requests() != 2 {
		t.Errorf("%d requests", host.Requests())
	}
	mustContain(t, "engine", e.quiver(app, "engine"), "Search engine: local")
	mustContain(t, "search", e.quiver(app, "search", "deploy"), "deploy-check")
	if host.Requests() != 2 {
		t.Errorf("a search used the network: %d requests", host.Requests())
	}
	settings, _ := os.ReadFile(filepath.Join(e.qhome, "state", "search.json"))
	mustContain(t, "settings", string(settings), `"installed"`, embedtest.Hash(host.Weights))
	got, _ := e.run(app, "", "doctor")
	mustContain(t, "doctor", got, "ok     search engine: local, model tiny")

	// Undo: the engine goes back to words, the model stays until it is asked for.
	mustContain(t, "remove", e.quiver(app, "engine", "remove", "--yes"), "Deleted the search index.", "Search engine: words", "model.safetensors")
	if _, err := os.Stat(filepath.Join(e.qhome, "cache", "models", "tiny", "model.safetensors")); !os.IsNotExist(err) {
		t.Errorf("the model was not deleted: %v", err)
	}
}

// The session-start hook through the binary: a home with a Searchable skill
// gets the cue, a home with only an unassigned skill gets nothing, and what
// the user writes with quiver new is Searchable.
func TestHookPassesTheSearchCue(t *testing.T) {
	e := newEnv(t)
	app := e.dir("app")
	e.write(filepath.Join(e.qhome, "local"), map[string]string{
		"loose/SKILL.md": skillMD("loose", "Assigned nowhere.", "Loose."),
	})
	input := `{"cwd": "` + filepath.ToSlash(app) + `", "hook_event_name": "SessionStart"}`
	hook := func() string {
		out, err := e.run(app, input, "hook", "session-start", "claude-code")
		if err != nil {
			t.Fatalf("hook: %v\n%s", err, out)
		}
		return out
	}

	if out := hook(); strings.TrimSpace(out) != "" {
		t.Errorf("an unassigned skill gave output:\n%s", out)
	}

	mustContain(t, "new", e.quiver(app, "new", "style-guide", "--global", "--description", "House style."),
		"Assigned globally as Searchable: style-guide", "quiver add style-guide moves it to another level.")
	out := hook()
	mustContain(t, "hook", out, "quiver-search", "call search_skills before you start", "No skills are listed in this folder")
	mustNotContain(t, "hook", out, "style-guide", "loose")

	e.quiver(app, "add", "loose", "--global")
	out = hook()
	mustContain(t, "hook with a listed skill", out, "loose: Assigned nowhere.", "quiver-search")
	mustNotContain(t, "hook with a listed skill", out, "No skills are listed in this folder")

	e.quiver(app, "add", "style-guide", "--global")
	if out := hook(); strings.Contains(out, "quiver-search") {
		t.Errorf("a cue without a Searchable skill:\n%s", out)
	}
}

// quiver status through the binary: what the listing costs, the longest
// entries with -v, and the hint for a library that search does not find.
func TestStatusCatalogCost(t *testing.T) {
	e := newEnv(t)
	app := e.dir("app")
	e.write(filepath.Join(e.qhome, "local"), map[string]string{
		"deploy-check/SKILL.md": skillMD("deploy-check", "Check a deploy before it goes out.", "Check."),
		"slack-style/SKILL.md":  skillMD("slack-style", "How to write a Slack message.", "Style."),
		"notes/SKILL.md":        skillMD("notes", "Keep notes.", "Notes."),
	})

	out := e.quiver(app, "status")
	mustContain(t, "status", out, "Session start: 0 skills listed, 0 characters", "0 Searchable",
		"Search finds only Searchable skills: 3 skills of the library are at no level here", "quiver add <id> --searchable")

	e.quiver(app, "add", "deploy-check", "slack-style", "--global")
	e.quiver(app, "add", "notes", "--searchable", "--global")
	out = e.quiver(app, "status")
	mustContain(t, "status", out, "Session start: 2 skills listed,", "an estimate", "1 Searchable, which add nothing to the listing.")
	mustNotContain(t, "status", out, "Search finds only", "Longest listed entries")

	out = e.quiver(app, "status", "-v")
	mustContain(t, "status -v", out, "Longest listed entries", "quiver add deploy-check --searchable --global", "quiver add slack-style --searchable --global")

	got := e.quiver(app, "status", "--json")
	mustContain(t, "status --json", got, `"listed": 2`, `"listed_chars":`, `"listed_tokens_estimate":`, `"always_on": 0`, `"searchable": 1`)
	mustNotContain(t, "status --json", got, "Search finds only")
}
