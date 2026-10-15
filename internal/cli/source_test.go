package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/git"
)

// runIn runs quiver with stdin and returns the output and the error.
func runIn(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// sourceEnv sets up an empty Quiver home and a git setup that ignores the
// user's config.
func sourceEnv(t *testing.T) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), "q")
	t.Setenv("QUIVER_HOME", home)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	return home
}

// skillRepo creates a git repository named name with the given skills.
func skillRepo(t *testing.T, name string, skills ...string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := git.Init(dir); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"README.md": "# " + name}
	for _, s := range skills {
		files["skills/"+s+"/SKILL.md"] = "---\nname: " + s + "\ndescription: Test skill.\n---\n"
	}
	for p, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := git.Commit(dir, "init"); err != nil {
		t.Fatal(err)
	}
	return dir
}

func sourceNames(t *testing.T) []string {
	t.Helper()
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for n := range c.Sources {
		names = append(names, n)
	}
	return names
}

// leftovers lists what is in sources/ besides the given folders.
func leftovers(t *testing.T, home string, keep ...string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(home, "sources"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if !strings.Contains(" "+strings.Join(keep, " ")+" ", " "+e.Name()+" ") {
			out = append(out, e.Name())
		}
	}
	return out
}

func TestSourceAdd(t *testing.T) {
	home := sourceEnv(t)
	repo := skillRepo(t, "acme", "pdf", "review")

	out, err := runIn(t, "", "source", "add", repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`Added source "acme" with 2 skills`, "acme/pdf", "acme/review", "may contain scripts"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	if got := sourceNames(t); len(got) != 1 || got[0] != "acme" {
		t.Errorf("config sources %v", got)
	}
	if _, err := os.Stat(filepath.Join(home, "sources", "acme", "skills", "pdf", "SKILL.md")); err != nil {
		t.Error(err)
	}
	if extra := leftovers(t, home, "acme"); len(extra) != 0 {
		t.Errorf("left behind %v", extra)
	}

	_, err = runIn(t, "", "source", "add", repo)
	if err == nil || !strings.Contains(err.Error(), "--name") {
		t.Errorf("duplicate name: %v", err)
	}
	if _, err := runIn(t, "", "source", "add", repo, "--name", "acme-two"); err != nil {
		t.Errorf("--name: %v", err)
	}
	if _, err := runIn(t, "", "source", "add", repo, "--name", "Bad Name"); err == nil {
		t.Error("invalid --name was accepted")
	}
}

func TestSourceAddNoSkills(t *testing.T) {
	home := sourceEnv(t)
	repo := skillRepo(t, "empty")

	out, err := runIn(t, "n\n", "source", "add", repo)
	if err != nil || !strings.Contains(out, "Source not added") {
		t.Fatalf("answer no: %v\n%s", err, out)
	}
	if got := sourceNames(t); len(got) != 0 {
		t.Errorf("config sources %v", got)
	}
	if extra := leftovers(t, home); len(extra) != 0 {
		t.Errorf("left behind %v", extra)
	}

	if _, err := runIn(t, "y\n", "source", "add", repo); err != nil {
		t.Fatal(err)
	}
	if got := sourceNames(t); len(got) != 1 {
		t.Errorf("config sources %v", got)
	}
}
