package cli

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/assistant/assistanttest"
	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/setup"
)

// registered makes the fake claude run /opt/bin/quiver, which is this
// quiver.
func registered(t *testing.T) {
	t.Helper()
	assistanttest.Install(t).SetUser(t, "/opt/bin/quiver", "mcp")
	fakeExecutable(t, "/opt/bin/quiver")
}

func wantLines(t *testing.T, out string, lines ...string) {
	t.Helper()
	for _, l := range lines {
		if !strings.Contains(out, l) {
			t.Errorf("output missing %q:\n%s", l, out)
		}
	}
}

func TestDoctorClean(t *testing.T) {
	sourceEnv(t)
	if _, err := setup.Init(); err != nil {
		t.Fatal(err)
	}
	registered(t)
	out, err := runIn(t, "", "doctor")
	if err != nil || !strings.HasSuffix(out, "\nEverything is fine.\n") || strings.Contains(out, "warn") {
		t.Errorf("%v\n%s", err, out)
	}
}

// doctorEnv is listEnv with one problem for each check from 2 to 7 and 9. The
// broken local skill is already in listEnv.
func doctorEnv(t *testing.T) fixture {
	t.Helper()
	f := listEnv(t)
	assistanttest.Install(t) // claude without a registration
	// 7: pdf-copy is also named pdf, and both are global.
	writeFiles(t, filepath.Join(f.home, "local"), map[string]string{
		"pdf-copy/SKILL.md": "---\nname: pdf\ndescription: Another PDF skill.\n---\n",
	})
	// 5: lost is configured but not downloaded, and a source add was cut off.
	err := config.Update(func(c *config.Config) error {
		c.Sources["lost"] = config.Source{Repo: "https://example.com/lost.git"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(f.home, "sources", ".add-x-123", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	// 4: gone and acme/old are in no library; lost/x is in a missing source.
	err = registry.Update(func(r *registry.Registry) error {
		r.AssignGlobal("gone", "pdf-copy")
		if err := r.Assign("app-000001", "acme/old"); err != nil {
			return err
		}
		return r.Assign("api-000002", "lost/x")
	})
	if err != nil {
		t.Fatal(err)
	}
	// 6: api's folder is gone. 2: listEnv wrote local/ without committing.
	if err := os.RemoveAll(f.api); err != nil {
		t.Fatal(err)
	}
	return f
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for p, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDoctorFindsAndFixes(t *testing.T) {
	f := doctorEnv(t)
	out, err := runIn(t, "", "doctor")
	if err == nil {
		t.Errorf("no error:\n%s", out)
	}
	wantLines(t, out,
		"ok     Quiver home layout",
		"ok     config.json",
		"ok     registry.json",
		"warn   local/ has uncommitted changes",
		"pdf-copy/SKILL.md",
		"error  library skills that cannot be used\n       skipped broken:",
		"error  assigned skills that are not in the library, so they are not served\n"+
			"       global: gone\n"+
			"       project api: lost/x (source lost is not downloaded)\n"+
			"       project app: acme/old\n",
		"error  sources that are not downloaded\n       lost (https://example.com/lost.git): remove it and add it again\n",
		"warn   folders left over from an interrupted source add or remove\n       "+filepath.Join(f.home, "sources", ".add-x-123"),
		"warn   projects with no folder on this machine\n       api [api-000002]: "+f.api,
		`every folder: pdf, pdf-copy share the name "pdf"`,
		"error  not registered with Claude Code; run: quiver mcp install",
	)
	if strings.Contains(out, "fixed") {
		t.Errorf("fixed without --fix:\n%s", out)
	}

	out, _ = runIn(t, "y\n", "doctor", "--fix")
	wantLines(t, out,
		"fixed: committed them",
		"Unassign 2 missing skills? [y/N] ",
		"fixed: unassigned 2 skills",
		"fixed: deleted them",
	)
	if status, err := git.Status(filepath.Join(f.home, "local")); err != nil || status != "" {
		t.Errorf("local status %q, %v", status, err)
	}
	if _, err := os.Stat(filepath.Join(f.home, "sources", ".add-x-123")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("leftover: %v", err)
	}
	r := loadRegistry(t)
	if slices.Contains(r.Global, "gone") || slices.Contains(r.Projects["app-000001"].Skills, "acme/old") {
		t.Errorf("not unassigned: %v, %v", r.Global, r.Projects["app-000001"].Skills)
	}
	// Adding the source again is the fix for lost/x, not unassigning it.
	if !slices.Contains(r.Projects["api-000002"].Skills, "lost/x") {
		t.Errorf("lost/x unassigned: %v", r.Projects["api-000002"].Skills)
	}

	// Only what --fix cannot fix is left.
	out, _ = runIn(t, "", "doctor")
	for _, gone := range []string{"uncommitted", "left over", "global: gone", "acme/old"} {
		if strings.Contains(out, gone) {
			t.Errorf("still reported %q:\n%s", gone, out)
		}
	}
	wantLines(t, out, "lost/x (source lost is not downloaded)", "skipped broken:", "not registered with Claude Code")
}

func TestDoctorFixAsksBeforeUnassigning(t *testing.T) {
	doctorEnv(t)
	out, _ := runIn(t, "n\n", "doctor", "--fix")
	wantLines(t, out, "Not unassigned.")
	if r := loadRegistry(t); !slices.Contains(r.Global, "gone") {
		t.Errorf("unassigned after no: %v", r.Global)
	}
}

func TestDoctorBrokenHome(t *testing.T) {
	home := sourceEnv(t)
	if _, err := setup.Init(); err != nil {
		t.Fatal(err)
	}
	registered(t)
	if err := os.RemoveAll(filepath.Join(home, "local", ".git")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "registry.json"), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runIn(t, "", "doctor")
	if err == nil {
		t.Error("no error")
	}
	wantLines(t, out,
		"error  Quiver home layout\n       "+filepath.Join(home, "local", ".git")+" is missing",
		"error  registry.json",
		"error  local/ is not a git repository",
		"-      assigned skills are in the library (skipped: the registry or the library could not be loaded)",
	)
	// doctor reports the home as it is; it does not repair it on its own.
	if _, err := os.Stat(filepath.Join(home, "local", ".git")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("local/.git created without --fix: %v", err)
	}
	out, _ = runIn(t, "", "doctor", "--fix")
	wantLines(t, out, "fixed: created the missing parts", "ok     local/ is committed", "error  registry.json")
}

func TestDoctorClaudeRunsOtherQuiver(t *testing.T) {
	sourceEnv(t)
	if _, err := setup.Init(); err != nil {
		t.Fatal(err)
	}
	assistanttest.Install(t).SetUser(t, "/old/quiver", "mcp")
	fakeExecutable(t, "/opt/bin/quiver")
	out, err := runIn(t, "", "doctor")
	if err != nil {
		t.Errorf("a warning is not an error: %v", err)
	}
	wantLines(t, out, "warn   Claude Code runs another quiver\n       it runs /old/quiver mcp, but this is /opt/bin/quiver; run: quiver mcp install", "1 warning, no errors.")
}

// nativeEnv is listEnv with copies of served skills in Claude Code: pdf
// in ~/.claude/skills, tdd committed in app's .claude/skills and review in
// an enabled plugin.
func nativeEnv(t *testing.T) (fixture, string) {
	t.Helper()
	f := listEnv(t)
	claude := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", claude)
	plugin := filepath.Join(claude, "plugins", "cache", "acme")
	writeFiles(t, claude, map[string]string{
		"skills/pdf/SKILL.md":   "---\nname: pdf\ndescription: My own PDF skill.\n---\n",
		"skills/other/SKILL.md": "---\nname: other\ndescription: Not served.\n---\n",
		"settings.json":         `{"enabledPlugins": {"acme@market": true}}`,
		"plugins/installed_plugins.json": `{"version": 2, "plugins": {"acme@market": [{"scope": "user", "installPath": "` +
			filepath.ToSlash(plugin) + `"}]}}`,
		"plugins/cache/acme/skills/review/SKILL.md": "---\nname: review\ndescription: Review.\n---\n",
	})
	writeFiles(t, f.app, map[string]string{
		".claude/skills/tdd/SKILL.md": "---\nname: tdd\ndescription: Tests first.\n---\n",
	})
	gitIn(t, f.app, "init", "--quiet")
	gitIn(t, f.app, "add", ".claude")
	t.Chdir(f.app)
	return f, claude
}

func TestDoctorNativeCopies(t *testing.T) {
	f, claude := nativeEnv(t)
	registered(t)
	out, _ := runIn(t, "", "doctor")
	wantLines(t, out,
		"warn   served skills that are also installed in Claude Code",
		"pdf: "+filepath.Join(claude, "skills", "pdf")+" (same name)",
		"acme/review: plugin acme@market (same name); disable the plugin in Claude Code with /plugin",
		filepath.Join("app", ".claude", "skills", "tdd")+" (same name); it is committed in git, delete it there if no one needs it")
	if strings.Contains(out, "other") {
		t.Errorf("unserved skill reported:\n%s", out)
	}

	// --fix deletes only the copy that is neither committed nor in a plugin.
	out, _ = runIn(t, "y\n", "doctor", "--fix")
	wantLines(t, out, "Delete 1 copy? [y/N] ", "fixed: deleted 1 copy")
	if _, err := os.Stat(filepath.Join(claude, "skills", "pdf")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("pdf copy kept: %v", err)
	}
	for _, kept := range []string{filepath.Join(claude, "plugins", "cache", "acme", "skills", "review"), filepath.Join(f.app, ".claude", "skills", "tdd")} {
		if _, err := os.Stat(kept); err != nil {
			t.Errorf("%s: %v", kept, err)
		}
	}
	out, _ = runIn(t, "", "doctor")
	if strings.Contains(out, "skills/pdf") {
		t.Errorf("pdf still reported:\n%s", out)
	}
}

func TestDoctorNativeFixAll(t *testing.T) {
	f := listEnv(t)
	claude := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", claude)
	writeFiles(t, claude, map[string]string{"skills/pdf/SKILL.md": "---\nname: pdf\ndescription: Copy.\n---\n"})
	registered(t)
	t.Chdir(f.app)
	out, _ := runIn(t, "n\n", "doctor", "--fix")
	wantLines(t, out, "Delete 1 copy? [y/N]        Not deleted.")
	out, _ = runIn(t, "y\n", "doctor", "--fix")
	wantLines(t, out, "fixed: deleted 1 copy")
	out, _ = runIn(t, "", "doctor")
	wantLines(t, out, "ok     no served skill is also installed in Claude Code")
}
