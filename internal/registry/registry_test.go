package registry

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	for _, want := range []string{`"global": [`, `"skills": []`, `"paths": []`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("missing %s in\n%s", want, data)
		}
	}
	if strings.Contains(string(data), "null") {
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
		"newer version": {`{"version": 2}`, "update quiver"},
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
