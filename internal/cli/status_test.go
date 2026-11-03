package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/assistant"
	"github.com/kheif/quiver/internal/assistant/assistanttest"
	"github.com/kheif/quiver/internal/server"
)

func TestStatusGolden(t *testing.T) {
	f := listEnv(t)
	sub := filepath.Join(f.app, "src")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "other")
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(server.ProjectEnv, "")
	for _, c := range []struct {
		name, dir, env string
	}{
		{"status-project", sub, ""},
		{"status-unregistered", other, ""},
		{"status-env", other, f.api},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Chdir(c.dir)
			t.Setenv(server.ProjectEnv, c.env)
			t.Setenv("PATH", t.TempDir()) // no assistant installed, so the golden is stable
			golden(t, c.name, f.run(t, "status"))
		})
	}
}

func TestStatusPossibleCopy(t *testing.T) {
	f := listEnv(t)
	cp := projectDir(t, "app-copy", "app-000001")
	t.Chdir(cp)
	t.Setenv("PATH", t.TempDir()) // no assistant installed, so the golden is stable
	golden(t, "status-copy", f.run(t, "status"))
	// status only reports; the registry keeps the one known path.
	if p := loadRegistry(t).Projects["app-000001"]; len(p.Paths) != 1 {
		t.Errorf("paths = %v", p.Paths)
	}
}

// status lists each installed assistant and how always-on skills reach it.
func TestStatusAssistants(t *testing.T) {
	f := listEnv(t)
	assistanttest.Install(t) // claude-code: a hook
	for _, id := range []string{"cursor", "claude-desktop"} {
		a := jsonAssistant(t, id)
		if err := os.MkdirAll(a.Dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(f.app)
	out := f.run(t, "status")
	wantLines(t, out,
		"Assistants (how always-on skills reach them):",
		"  Claude Code: hook",
		"  Cursor: instructions",
		"  Claude Desktop: on request",
	)
}

// jsonAssistant returns the JSON assistant with the given id.
func jsonAssistant(t *testing.T, id string) *assistant.JSONAssistant {
	t.Helper()
	all, err := assistant.JSONAssistants()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range all {
		if a.ID == id {
			return a
		}
	}
	t.Fatalf("no assistant %q", id)
	return nil
}

func TestStatusMissingSkill(t *testing.T) {
	f := listEnv(t)
	if err := os.RemoveAll(filepath.Join(f.home, "local", "tdd")); err != nil {
		t.Fatal(err)
	}
	t.Chdir(f.app)
	out := f.run(t, "status")
	if want := "warning: tdd is assigned but not in the library, so it is not served"; !strings.Contains(out, want) {
		t.Errorf("output missing %q:\n%s", want, out)
	}
}

func TestStatusNativeCopies(t *testing.T) {
	_, claude := nativeEnv(t)
	out, err := runIn(t, "", "status")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	wantLines(t, out,
		"warning: pdf is also installed in Claude Code, so it may be loaded twice: "+filepath.Join(claude, "skills", "pdf")+" (same name)\n",
		"warning: acme/review is also installed in Claude Code, so it may be loaded twice: plugin acme@market (same name)\n",
		"warning: tdd is also installed in Claude Code",
		"Run quiver doctor --fix to delete the copies.\n")
}

func TestStatusJSON(t *testing.T) {
	f := listEnv(t)
	t.Setenv(server.ProjectEnv, "")
	t.Chdir(f.app)
	t.Setenv("PATH", t.TempDir()) // no assistant installed, so the list is empty
	var got statusJSON
	if err := json.Unmarshal([]byte(f.run(t, "status", "--json")), &got); err != nil {
		t.Fatal(err)
	}
	want := statusJSON{
		Folder:  f.app,
		Project: &statusProject{ID: "app-000001", Name: "app", Root: f.app},
		Skills: []servedSkill{
			{ID: "acme/review", Scope: "global", InLibrary: true},
			{ID: "pdf", Scope: "global", AlwaysOn: true, InLibrary: true},
			{ID: "tdd", Scope: "project", InLibrary: true},
		},
		Assistants: []statusAssistant{},
		Warnings:   []string{},
	}
	// f.run writes temp paths as <tmp>.
	want.Folder, want.Project.Root = got.Folder, got.Project.Root
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}

	// Outside a project, project is null and a missing skill is shown.
	if err := os.RemoveAll(filepath.Join(f.home, "local", "pdf")); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	out := f.run(t, "status", "--json")
	if !strings.Contains(out, `"project": null`) || !strings.Contains(out, `"id": "pdf",
      "scope": "global",
      "always_on": true,
      "in_library": false`) {
		t.Errorf("outside a project:\n%s", out)
	}
}
