package registry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func setHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("QUIVER_HOME", home)
	return home
}

func writeFile(t *testing.T, content string) {
	t.Helper()
	path, _ := Path()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadMissingGivesDefault(t *testing.T) {
	setHome(t)
	r, err := Load()
	if err != nil || !reflect.DeepEqual(r, Default()) {
		t.Errorf("got %+v, %v", r, err)
	}
}

func TestUpdateRoundTrip(t *testing.T) {
	home := setHome(t)
	err := Update(func(r *Registry) error {
		r.Global = []string{"b", "a", "b"}
		r.Projects["app-k3f9q2"] = &Project{Name: "app"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(home, "registry.json"))
	for _, want := range []string{`"version": 2`, `"global": [`, `"skills": []`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("missing %s in\n%s", want, data)
		}
	}
	if strings.Contains(string(data), "null") || strings.Contains(string(data), "paths") {
		t.Errorf("null in\n%s", data)
	}
	r, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Global, []string{"a", "b"}) || r.Projects["app-k3f9q2"].Name != "app" {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateErrorWritesNothing(t *testing.T) {
	setHome(t)
	boom := errors.New("boom")
	err := Update(func(r *Registry) error {
		r.Global = []string{"a"}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("got %v", err)
	}
	path, _ := Path()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file written: %v", err)
	}
}

func TestLoadRejectsBadFile(t *testing.T) {
	cases := map[string]struct{ content, want string }{
		"newer version": {`{"version": 3}`, "update quiver"},
		"zero version":  {`{"version": 0}`, "invalid version 0"},
		"null project":  {`{"version": 1, "projects": {"a": null}}`, `project "a" is null`},
		"bad json":      {`{`, "decode"},
	}
	for label, c := range cases {
		t.Run(label, func(t *testing.T) {
			setHome(t)
			writeFile(t, c.content)
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("got %v, want %q", err, c.want)
			}
		})
	}
}

func TestConcurrentUpdatesKeepEveryChange(t *testing.T) {
	setHome(t)
	const writers = 8
	var wg sync.WaitGroup
	for i := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := Update(func(r *Registry) error {
				r.AssignGlobal(fmt.Sprintf("skill-%d", i))
				return nil
			})
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	r, err := Load()
	if err != nil || len(r.Global) != writers {
		t.Errorf("got %v, %v", r.Global, err)
	}
}

func writeState(t *testing.T, content string) {
	t.Helper()
	path, _ := StatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPathsGoToState(t *testing.T) {
	home := setHome(t)
	err := Update(func(r *Registry) error {
		r.Projects["app-k3f9q2"] = &Project{Name: "app", Paths: []string{"/b", "/a"}}
		r.Projects["tool-8xw2mf"] = &Project{Name: "tool"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(home, StateFile))
	want := `{
  "version": 1,
  "paths": {
    "app-k3f9q2": [
      "/a",
      "/b"
    ]
  }
}
`
	if string(data) != want {
		t.Errorf("state =\n%s\nwant\n%s", data, want)
	}
	r, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Projects["app-k3f9q2"].Paths; !reflect.DeepEqual(got, []string{"/a", "/b"}) {
		t.Errorf("paths = %v", got)
	}
	if got := r.Projects["tool-8xw2mf"].Paths; got == nil || len(got) != 0 {
		t.Errorf("paths = %#v, want empty", got)
	}
}

func TestMigrateVersion1(t *testing.T) {
	home := setHome(t)
	v1 := `{
  "version": 1,
  "global": ["a"],
  "alwaysOn": [],
  "projects": {
    "app-k3f9q2": {"name": "app", "paths": ["/code/app"], "skills": ["db"], "alwaysOn": ["db"]}
  }
}
`
	writeFile(t, v1)
	r, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Projects["app-k3f9q2"].Paths; !reflect.DeepEqual(got, []string{"/code/app"}) {
		t.Errorf("paths = %v", got)
	}
	// Reading alone changes nothing.
	if data, _ := os.ReadFile(filepath.Join(home, "registry.json")); string(data) != v1 {
		t.Errorf("registry.json changed on read:\n%s", data)
	}
	if _, err := os.Stat(filepath.Join(home, StateFile)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("state written on read: %v", err)
	}

	if err := Update(func(*Registry) error { return nil }); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(home, "registry.json"))
	if !strings.Contains(string(data), `"version": 2`) || strings.Contains(string(data), "paths") {
		t.Errorf("registry.json not migrated:\n%s", data)
	}
	r, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	p := r.Projects["app-k3f9q2"]
	if !reflect.DeepEqual(p.Paths, []string{"/code/app"}) || !reflect.DeepEqual(p.Skills, []string{"db"}) || !reflect.DeepEqual(p.AlwaysOn, []string{"db"}) {
		t.Errorf("after migration got %+v", p)
	}
}

// Two machines share one registry.json, each with its own state. Recording
// a project's folder on either machine leaves registry.json as it is.
func TestTwoMachinesShareRegistry(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	t.Setenv("QUIVER_HOME", a)
	err := Update(func(r *Registry) error {
		r.AssignGlobal("g")
		r.Projects["app-k3f9q2"] = &Project{Name: "app", Paths: []string{"/machine-a/app"}, Skills: []string{"db"}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	shared, _ := os.ReadFile(filepath.Join(a, "registry.json"))
	if err := os.WriteFile(filepath.Join(b, "registry.json"), shared, 0o644); err != nil {
		t.Fatal(err)
	}

	for home, path := range map[string]string{b: "/machine-b/app", a: "/machine-a/moved"} {
		t.Setenv("QUIVER_HOME", home)
		err := Update(func(r *Registry) error {
			r.Projects["app-k3f9q2"].Paths = []string{path}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		r, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if got := r.Projects["app-k3f9q2"].Paths; !reflect.DeepEqual(got, []string{path}) {
			t.Errorf("%s: paths = %v", home, got)
		}
		if data, _ := os.ReadFile(filepath.Join(home, "registry.json")); string(data) != string(shared) {
			t.Errorf("%s: registry.json changed:\n%s\nwas\n%s", home, data, shared)
		}
	}
}

func TestStateDropsForgottenProjects(t *testing.T) {
	home := setHome(t)
	writeFile(t, `{"version": 2, "projects": {"app-k3f9q2": {"name": "app"}}}`)
	writeState(t, `{"version": 1, "paths": {"app-k3f9q2": ["/app"], "gone-000001": ["/gone"]}}`)
	if err := Update(func(*Registry) error { return nil }); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(home, StateFile))
	if strings.Contains(string(data), "gone") || !strings.Contains(string(data), "/app") {
		t.Errorf("state =\n%s", data)
	}
}

func TestLoadRejectsBadState(t *testing.T) {
	cases := map[string]struct{ content, want string }{
		"newer version": {`{"version": 2}`, "update quiver"},
		"bad json":      {`{`, "decode"},
	}
	for label, c := range cases {
		t.Run(label, func(t *testing.T) {
			setHome(t)
			writeState(t, c.content)
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("got %v, want %q", err, c.want)
			}
		})
	}
}
