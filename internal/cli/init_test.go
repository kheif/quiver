package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/assistant/assistanttest"
	"github.com/kheif/quiver/internal/project"
)

func run(t *testing.T, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestInitCommand(t *testing.T) {
	home := filepath.Join(t.TempDir(), "q")
	t.Setenv("QUIVER_HOME", home)

	out := run(t, "init")
	if !strings.Contains(out, "created "+home) {
		t.Errorf("first run: %q", out)
	}
	if _, err := os.Stat(filepath.Join(home, "local", ".git")); err != nil {
		t.Error(err)
	}

	if err := os.Remove(filepath.Join(home, "registry.json")); err != nil {
		t.Fatal(err)
	}
	out = run(t, "init")
	if !strings.Contains(out, "created "+filepath.Join(home, "registry.json")) {
		t.Errorf("fill-in run: %q", out)
	}

	out = run(t, "init")
	if !strings.Contains(out, "Quiver home is ready") {
		t.Errorf("third run: %q", out)
	}
}

// initEnv gives guided init a skill in ~/.claude/skills and a folder,
// not registered, with a skill in .claude/skills, and makes it the
// current folder. It returns both skill folders.
func initEnv(t *testing.T) (user, proj string) {
	t.Helper()
	sourceEnv(t)
	fakeTerminal(t)
	user = filepath.Join(os.Getenv("HOME"), ".claude", "skills", "greet")
	writeFiles(t, user, map[string]string{"SKILL.md": "---\nname: greet\ndescription: Say hello.\n---\n"})
	dir := filepath.Join(t.TempDir(), "web")
	proj = filepath.Join(dir, ".claude", "skills", "review")
	writeFiles(t, proj, map[string]string{"SKILL.md": "---\nname: review\ndescription: Review code.\n---\n"})
	t.Chdir(dir)
	return user, proj
}

func TestInitGuided(t *testing.T) {
	user, proj := initEnv(t)
	claude := assistanttest.Install(t)
	fakeExecutable(t, "/opt/bin/quiver")
	// MCP: yes. greet: import, default global, delete. review: import,
	// default project, register, delete.
	out, err := runIn(t, "y\ny\n\n\ny\n\ny\n\n", "init")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	wantLines(t, out,
		"Register the MCP server with your assistants? [Y/n] Registered with Claude Code: /opt/bin/quiver mcp",
		"Found 1 skill in "+filepath.Dir(user)+". Import them into the library? [Y/n] Found 1 skill",
		"Assign to (p)roject, (g)lobal, or (s)kip? [g] Imported greet into the library and assigned it to every folder.",
		"Found 1 skill in "+filepath.Dir(proj)+". Import them into the library? [Y/n] Found 1 skill",
		"See what the assistant gets in this folder with: quiver status",
	)
	if claude.User() != "/opt/bin/quiver|mcp" {
		t.Errorf("claude user %q", claude.User())
	}
	r := loadRegistry(t)
	if !reflect.DeepEqual(r.Global, []string{"greet"}) {
		t.Errorf("global = %v", r.Global)
	}
	p, err := project.Find(".")
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Projects[p.ID]; got == nil || !reflect.DeepEqual(got.Skills, []string{"review"}) {
		t.Errorf("project = %+v", got)
	}
	for _, dir := range []string{user, proj} {
		if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("original %s kept: %v", dir, err)
		}
	}
}

func TestInitGuidedSkipped(t *testing.T) {
	user, proj := initEnv(t)
	claude := assistanttest.Install(t)
	out, err := runIn(t, "n\nn\nn\n", "init")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	wantLines(t, out,
		"Register the MCP server with your assistants? [Y/n] Skipped. Run it later with: quiver mcp install",
		"Found 1 skill in "+filepath.Dir(user)+". Import them into the library? [Y/n] Skipped. Run it later with: quiver import "+filepath.Dir(user),
		"Found 1 skill in "+filepath.Dir(proj)+". Import them into the library? [Y/n] Skipped. Run it later with: quiver import",
	)
	if claude.User() != "" {
		t.Errorf("registered: %q", claude.User())
	}
	if r := loadRegistry(t); len(r.Global) != 0 || len(r.Projects) != 0 {
		t.Errorf("registry changed: %+v", r)
	}
	for _, dir := range []string{user, proj} {
		if _, err := os.Stat(dir); err != nil {
			t.Error(err)
		}
	}
}

// With --yes, outside a terminal and with nothing to import, init asks
// only what applies.
func TestInitPlain(t *testing.T) {
	initEnv(t)
	assistanttest.Install(t)
	if out, err := runIn(t, "", "init", "--yes"); err != nil || strings.Contains(out, "?") {
		t.Errorf("--yes: %v\n%s", err, out)
	}
	interactive = func(io.Reader) bool { return false }
	if out, err := runIn(t, "", "init"); err != nil || out != "Quiver home is ready: "+os.Getenv("QUIVER_HOME")+"\n" {
		t.Errorf("not a terminal: %v\n%s", err, out)
	}

	sourceEnv(t)
	fakeTerminal(t)
	t.Chdir(t.TempDir())
	out, err := runIn(t, "n\n", "init")
	if err != nil || strings.Contains(out, "Import") || !strings.Contains(out, "Register the MCP server") {
		t.Errorf("nothing to import: %v\n%s", err, out)
	}
}
