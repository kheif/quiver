package project

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setup makes a fake $HOME and returns a folder inside it.
func setup(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, "code", "app")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindNestedSubfolder(t *testing.T) {
	root := setup(t)
	write(t, filepath.Join(root, FileName), `{"id": "app-k3f9q2"}`)
	sub := filepath.Join(root, "src", "lib")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	p, err := Find(sub)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "app-k3f9q2" || p.Root != root {
		t.Errorf("got %+v", p)
	}
}

func TestFindMissing(t *testing.T) {
	_, err := Find(setup(t))
	if !errors.Is(err, ErrNotRegistered) {
		t.Errorf("got %v", err)
	}
}

func TestFindStopsBeforeHome(t *testing.T) {
	dir := setup(t)
	write(t, filepath.Join(os.Getenv("HOME"), FileName), `{"id": "home-aaaaaa"}`)
	if _, err := Find(dir); !errors.Is(err, ErrNotRegistered) {
		t.Errorf("got %v", err)
	}
}

func TestFindInvalidFile(t *testing.T) {
	cases := map[string]struct{ content, want string }{
		"not json":   {`{id`, "invalid JSON"},
		"not object": {`[]`, "invalid JSON"},
		"null":       {`null`, "invalid JSON"},
		"no id":      {`{}`, "id is missing"},
		"id number":  {`{"id": 3}`, "id must be"},
		"empty id":   {`{"id": ""}`, "id must be"},
		"id slash":   {`{"id": "a/b"}`, "id must be"},
	}
	for label, c := range cases {
		t.Run(label, func(t *testing.T) {
			root := setup(t)
			path := filepath.Join(root, FileName)
			write(t, path, c.content)
			_, err := Find(root)
			if err == nil || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), path) {
				t.Errorf("got %v, want %q with path", err, c.want)
			}
		})
	}
}
