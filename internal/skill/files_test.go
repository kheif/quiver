package skill

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFiles(t *testing.T) {
	dir := writeSkill(t, "pdf", "---\nname: pdf\ndescription: d\n---\n")
	for _, f := range []string{
		"references/forms.md",
		"scripts/fill.py",
		"z.txt",
		".DS_Store",
		".git/config",
		"references/.hidden.md",
	} {
		p := filepath.Join(dir, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Files(s)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"references/forms.md", "scripts/fill.py", "z.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFilesNone(t *testing.T) {
	dir := writeSkill(t, "x", "---\nname: x\ndescription: d\n---\n")
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Files(s)
	if err != nil || len(got) != 0 {
		t.Errorf("got %v, %v", got, err)
	}
}
