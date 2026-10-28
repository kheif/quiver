package cli

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/setup"
)

var updateGolden = flag.Bool("update", false, "rewrite the golden files in testdata")

// testdata is absolute, because tests change the working directory.
var testdata, _ = filepath.Abs("testdata")

// golden compares got with testdata/<name>.golden, or rewrites the file
// with -update.
func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join(testdata, name+".golden")
	if *updateGolden {
		if err := os.MkdirAll(testdata, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v; run go test -update", err)
	}
	if got != string(want) {
		t.Errorf("%s differs from %s:\n--- got\n%s--- want\n%s", name, path, got, want)
	}
}

// fixture is a Quiver home with local skills, one source and two projects.
type fixture struct {
	home     string
	app, api string // registered project folders
	tmp      string // parent of every temp folder, for stable output
}

// listEnv builds the fixture:
//   - local skills pdf (global, always-on), tdd (projects app and api),
//     notes (unassigned) and broken (skipped with a warning)
//   - source acme with review (global and app) and deploy (unassigned)
func listEnv(t *testing.T) fixture {
	t.Helper()
	f := fixture{home: sourceEnv(t)}
	// home is <test temp root>/001/q.
	f.tmp = filepath.Dir(filepath.Dir(f.home))
	if err := setup.EnsureInit(); err != nil {
		t.Fatal(err)
	}
	local := map[string]string{
		"pdf":    "---\nname: pdf\ndescription: Fill in and read PDF forms.\n---\n",
		"tdd":    "---\nname: tdd\ndescription: |\n  Write the test first.\n  Then write the code that makes it pass, and refactor while the tests stay green.\n---\n",
		"notes":  "---\nname: notes\ndescription: Keep notes.\n---\n",
		"broken": "no frontmatter\n",
	}
	for folder, content := range local {
		dir := filepath.Join(f.home, "local", folder)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	repo := skillRepo(t, "acme", "review", "deploy")
	f.app = projectDir(t, "app", "app-000001")
	f.api = projectDir(t, "api", "api-000002")
	for _, args := range [][]string{
		{"source", "add", repo, "--name", "acme"},
		{"add", "pdf", "acme/review", "--global"},
		{"add", "pdf", "--global", "--always"},
		{"add", "tdd", "acme/review", "--project", f.app},
		{"add", "tdd", "--project", f.api},
	} {
		if out, err := runIn(t, "", args...); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	return f
}

// projectDir creates a folder registered with a fixed ID.
func projectDir(t *testing.T, name, id string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := project.SetID(dir, id); err != nil {
		t.Fatal(err)
	}
	return dir
}

// run runs quiver and replaces temp paths, so the output can be compared
// with a golden file.
func (f fixture) run(t *testing.T, args ...string) string {
	t.Helper()
	out, err := runIn(t, "", args...)
	if err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
	if real, err := filepath.EvalSymlinks(f.tmp); err == nil {
		out = strings.ReplaceAll(out, real, "$TMP")
	}
	return strings.ReplaceAll(out, f.tmp, "$TMP")
}

func TestListGolden(t *testing.T) {
	f := listEnv(t)
	for name, args := range map[string][]string{
		"list":            {"list"},
		"list-source":     {"list", "--source", "acme"},
		"list-local":      {"list", "--source", "local"},
		"list-unassigned": {"list", "--unassigned"},
		"list-json":       {"list", "--json"},
	} {
		t.Run(name, func(t *testing.T) {
			golden(t, name, f.run(t, args...))
		})
	}
}

func TestListTerminal(t *testing.T) {
	f := listEnv(t)
	t.Setenv("NO_COLOR", "")
	orig := terminal
	terminal = func(io.Writer) (int, bool) { return 60, true }
	t.Cleanup(func() { terminal = orig })
	golden(t, "list-terminal", f.run(t, "list"))

	t.Setenv("NO_COLOR", "1")
	if out := f.run(t, "list"); strings.Contains(out, "\x1b") {
		t.Errorf("NO_COLOR output has escape codes:\n%s", out)
	}
}

func TestListUnknownSource(t *testing.T) {
	listEnv(t)
	_, err := runIn(t, "", "list", "--source", "nope")
	if err == nil || !strings.Contains(err.Error(), `unknown source "nope"`) {
		t.Errorf("err = %v", err)
	}
}

func TestListEmpty(t *testing.T) {
	sourceEnv(t)
	out, err := runIn(t, "", "list")
	if err != nil {
		t.Fatal(err)
	}
	if want := "No skills. Add a source with: quiver source add <repo>\n"; out != want {
		t.Errorf("out = %q", out)
	}
	out, err = runIn(t, "", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if want := "{\n  \"skills\": [],\n  \"warnings\": []\n}\n"; out != want {
		t.Errorf("json = %q", out)
	}
}

func TestCut(t *testing.T) {
	for _, c := range []struct {
		in   string
		n    int
		want string
	}{
		{"short", 10, "short"},
		{"exactly 10", 10, "exactly 10"},
		{"one two three", 8, "one two…"},
		{"line one\n  line two", 0, "line one line two"},
		{"ünïcödé text", 5, "ünïc…"},
	} {
		if got := cut(c.in, c.n); got != c.want {
			t.Errorf("cut(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}
