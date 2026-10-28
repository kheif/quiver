package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/registry"
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

func TestSourceList(t *testing.T) {
	home := sourceEnv(t)
	if out, _ := runIn(t, "", "source", "list"); !strings.Contains(out, "No sources") {
		t.Errorf("empty list: %q", out)
	}
	repo := skillRepo(t, "acme", "pdf", "review")
	if _, err := runIn(t, "", "source", "add", repo); err != nil {
		t.Fatal(err)
	}
	// A source whose folder is gone shows as missing.
	if err := os.RemoveAll(filepath.Join(home, "sources", "acme")); err != nil {
		t.Fatal(err)
	}
	if _, err := runIn(t, "", "source", "add", repo, "--name", "beta"); err != nil {
		t.Fatal(err)
	}
	out, err := runIn(t, "", "source", "list")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("got\n%s", out)
	}
	if f := strings.Fields(lines[1]); f[0] != "acme" || f[1] != "0" || f[2] != "missing" {
		t.Errorf("acme row %q", lines[1])
	}
	if f := strings.Fields(lines[2]); f[0] != "beta" || f[1] != "2" || len(f[2]) != 7 || f[4] != repo {
		t.Errorf("beta row %q", lines[2])
	}
}

// TestSourceRemoveAssigned is the end-to-end flow: add a source from a
// bare repository, assign its skills, get refused, then force the removal.
func TestSourceRemoveAssigned(t *testing.T) {
	home := sourceEnv(t)
	bare := filepath.Join(t.TempDir(), "acme.git")
	if out, err := exec.Command("git", "clone", "--quiet", "--bare", skillRepo(t, "acme", "pdf", "review"), bare).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if _, err := runIn(t, "", "source", "add", bare); err != nil {
		t.Fatal(err)
	}
	err := registry.Update(func(r *registry.Registry) error {
		_ = r.EnsureProject("p1", "/work/web")
		r.Projects["p1"].Name = "web"
		r.AssignGlobal("acme/pdf", "local-skill")
		return r.Assign("p1", "acme/review")
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = runIn(t, "", "source", "remove", "acme")
	if err == nil {
		t.Fatal("remove of an assigned source succeeded")
	}
	for _, want := range []string{"global: acme/pdf", "project web (" + filepath.Clean("/work/web") + "): acme/review", "--force"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in %v", want, err)
		}
	}
	if got := sourceNames(t); len(got) != 1 {
		t.Errorf("refused remove changed config: %v", got)
	}

	out, err := runIn(t, "", "source", "remove", "acme", "--force")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Unassigned globally: acme/pdf", "Unassigned from web: acme/review", `Removed source "acme"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	r, err := registry.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Global) != 1 || r.Global[0] != "local-skill" || len(r.Projects["p1"].Skills) != 0 {
		t.Errorf("registry after remove: %+v, %+v", r.Global, r.Projects["p1"])
	}
	if got := sourceNames(t); len(got) != 0 {
		t.Errorf("config sources %v", got)
	}
	if extra := leftovers(t, home); len(extra) != 0 {
		t.Errorf("left behind %v", extra)
	}
	if _, err := runIn(t, "", "source", "remove", "acme"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("second remove: %v", err)
	}
}
