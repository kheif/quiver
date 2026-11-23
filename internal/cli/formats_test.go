package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/setup"
)

// goldenHome copies the home that a release's binary made (see
// testdata/formats/make-home.sh) into a fresh Quiver home, makes the
// repository of local/ that the script cannot commit, and gives the source
// a real repository to clone.
func goldenHome(t *testing.T, version string) string {
	t.Helper()
	home := sourceEnv(t)
	golden := filepath.Join("..", "..", "testdata", "formats", version)
	err := filepath.WalkDir(golden, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(golden, path)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(home, rel), 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(home, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Init(); err != nil {
		t.Fatal(err)
	}
	if err := git.Commit(filepath.Join(home, "local"), "golden home"); err != nil {
		t.Fatal(err)
	}
	// The script's source has its skills in the repository root.
	src := filepath.Join(t.TempDir(), "acme")
	for _, name := range []string{"one", "two"} {
		writeFiles(t, src, map[string]string{name + "/SKILL.md": "---\nname: " + name + "\ndescription: Skill of the source.\n---\n"})
	}
	if err := git.Init(src); err != nil {
		t.Fatal(err)
	}
	if err := git.Commit(src, "skills"); err != nil {
		t.Fatal(err)
	}
	err = config.Update(func(c *config.Config) error {
		s := c.Sources["acme"]
		s.Repo = src
		c.Sources["acme"] = s
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, failed, err := library.CloneMissing(); err != nil || len(failed) > 0 {
		t.Fatalf("cloning the source: %v %v", failed, err)
	}
	return home
}

func readHomeFile(t *testing.T, home, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(home, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestGoldenHomeV150(t *testing.T) {
	home := goldenHome(t, "v1.5.0")
	registered(t)
	regBefore := readHomeFile(t, home, "registry.json")
	cfgBefore := readHomeFile(t, home, "config.json")

	// The current code reads the home as it is.
	r, err := registry.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Global, []string{"acme/one", "notes"}) || !reflect.DeepEqual(r.AlwaysOn, []string{"notes"}) || len(r.Searchable) != 0 {
		t.Errorf("global %v, always-on %v, searchable %v", r.Global, r.AlwaysOn, r.Searchable)
	}
	if len(r.Projects) != 1 {
		t.Fatalf("projects: %+v", r.Projects)
	}
	for _, p := range r.Projects {
		if !reflect.DeepEqual(p.Skills, []string{"acme/two", "deploy"}) || !reflect.DeepEqual(p.AlwaysOn, []string{"deploy"}) {
			t.Errorf("project %+v", p)
		}
	}
	if out := quiver(t, "doctor"); !strings.HasSuffix(out, "\nEverything is fine.\n") || strings.Contains(out, "warn") {
		t.Errorf("doctor:\n%s", out)
	}
	if got := readHomeFile(t, home, "registry.json"); got != regBefore {
		t.Errorf("reading changed registry.json:\n%s", got)
	}
	if got := readHomeFile(t, home, "config.json"); got != cfgBefore {
		t.Errorf("reading changed config.json:\n%s", got)
	}

	// A write without a Searchable skill keeps version 2 and the content.
	quiver(t, "add", "acme/two", "--global")
	data := readHomeFile(t, home, "registry.json")
	if !strings.Contains(data, `"version": 2`) || strings.Contains(data, "searchable") {
		t.Errorf("registry.json after a write:\n%s", data)
	}

	// The first Searchable skill upgrades the file to version 3; the rest is unchanged.
	err = registry.Update(func(r *registry.Registry) error {
		r.AssignSearchableGlobal("acme/one")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	data = readHomeFile(t, home, "registry.json")
	if !strings.Contains(data, `"version": 3`) || !strings.Contains(data, `"searchable": [`) {
		t.Errorf("registry.json after the first Searchable skill:\n%s", data)
	}
	r, err = registry.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Global, []string{"acme/two", "notes"}) || !reflect.DeepEqual(r.Searchable, []string{"acme/one"}) {
		t.Errorf("global %v, searchable %v", r.Global, r.Searchable)
	}
	if got := readHomeFile(t, home, "config.json"); got != cfgBefore {
		t.Errorf("config.json changed:\n%s", got)
	}
	if out := quiver(t, "doctor"); !strings.HasSuffix(out, "\nEverything is fine.\n") || strings.Contains(out, "warn") {
		t.Errorf("doctor after the upgrade:\n%s", out)
	}

	// No search settings file: the engine is words, and a search, a status and a
	// doctor create neither the index nor a model folder.
	if _, err := os.Stat(filepath.Join(home, "state", "search.json")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("search.json exists after the upgrade: %v", err)
	}
	if out := quiver(t, "engine"); out != "Search engine: words\n" {
		t.Errorf("engine: %q", out)
	}
	quiver(t, "search", "notes")
	quiver(t, "status")
	for _, p := range []string{"embeddings.json", "models"} {
		if _, err := os.Stat(filepath.Join(home, "cache", p)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("cache/%s exists although the local engine was never used: %v", p, err)
		}
	}
}

// The home that the v1.6.0 binary made: a registry of version 3 with a
// Searchable skill. The current code reads it as it is, and a write keeps it.
func TestGoldenHomeV160(t *testing.T) {
	home := goldenHome(t, "v1.6.0")
	registered(t)
	regBefore := readHomeFile(t, home, "registry.json")
	cfgBefore := readHomeFile(t, home, "config.json")

	r, err := registry.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Global, []string{"acme/one", "notes"}) || !reflect.DeepEqual(r.AlwaysOn, []string{"notes"}) || !reflect.DeepEqual(r.Searchable, []string{"journal"}) {
		t.Errorf("global %v, always-on %v, searchable %v", r.Global, r.AlwaysOn, r.Searchable)
	}
	if out := quiver(t, "doctor"); !strings.HasSuffix(out, "\nEverything is fine.\n") || strings.Contains(out, "warn") {
		t.Errorf("doctor:\n%s", out)
	}
	if got := readHomeFile(t, home, "registry.json"); got != regBefore {
		t.Errorf("reading changed registry.json:\n%s", got)
	}
	if got := readHomeFile(t, home, "config.json"); got != cfgBefore {
		t.Errorf("reading changed config.json:\n%s", got)
	}
	if out := quiver(t, "engine"); out != "Search engine: words\n" {
		t.Errorf("engine: %q", out)
	}

	// A write keeps version 3 and the Searchable skill.
	quiver(t, "add", "acme/two", "--global")
	data := readHomeFile(t, home, "registry.json")
	if !strings.Contains(data, `"version": 3`) || !strings.Contains(data, `"journal"`) {
		t.Errorf("registry.json after a write:\n%s", data)
	}
	if out := quiver(t, "status"); !strings.Contains(out, "1 Searchable") {
		t.Errorf("status:\n%s", out)
	}
}
