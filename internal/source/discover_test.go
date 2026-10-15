package source

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func discover(t *testing.T, dir string) *Discovery {
	t.Helper()
	d, err := Discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDiscoverPlain(t *testing.T) {
	d := discover(t, "testdata/plain")
	want := []Found{{ID: "alpha", Dir: "alpha"}, {ID: "beta", Dir: "beta"}}
	if !reflect.DeepEqual(d.Skills, want) {
		t.Errorf("got %+v", d.Skills)
	}
}

func TestDiscoverNested(t *testing.T) {
	d := discover(t, "testdata/nested")
	want := []Found{{ID: "lint", Dir: "tools/lint"}, {ID: "writer", Dir: "docs/guides/writer"}}
	if !reflect.DeepEqual(d.Skills, want) {
		t.Errorf("got %+v", d.Skills)
	}
}

func TestDiscoverRootSkill(t *testing.T) {
	root := filepath.Join(t.TempDir(), "solo")
	writeSkill(t, root)
	writeSkill(t, filepath.Join(root, ".git", "x"))
	d := discover(t, root)
	if want := []Found{{ID: "solo", Dir: "."}}; !reflect.DeepEqual(d.Skills, want) {
		t.Errorf("got %+v", d.Skills)
	}
}

func writeSkill(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: " + filepath.Base(dir) + "\ndescription: Test.\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
