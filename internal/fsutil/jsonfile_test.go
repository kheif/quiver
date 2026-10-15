package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type doc struct {
	N int `json:"n"`
}

func TestWriteReadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "a.json")
	if err := WriteJSONAtomic(path, doc{N: 1}); err != nil {
		t.Fatal(err)
	}
	var got doc
	if err := ReadJSON(path, &got); err != nil || got.N != 1 {
		t.Fatalf("got %+v, %v", got, err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o644 {
		t.Errorf("mode %v", info.Mode().Perm())
	}
}

func TestReadJSONMissing(t *testing.T) {
	err := ReadJSON(filepath.Join(t.TempDir(), "none.json"), &doc{})
	if !errors.Is(err, ErrNotExist) {
		t.Fatalf("got %v", err)
	}
}

func TestInterruptedWriteKeepsOldFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.json")
	if err := WriteJSONAtomic(path, doc{N: 1}); err != nil {
		t.Fatal(err)
	}

	rename = func(string, string) error { return errors.New("crash") }
	t.Cleanup(func() { rename = os.Rename })

	if err := WriteJSONAtomic(path, doc{N: 2}); err == nil {
		t.Fatal("want error")
	}
	var got doc
	if err := ReadJSON(path, &got); err != nil || got.N != 1 {
		t.Fatalf("old file changed: %+v, %v", got, err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("temp file left behind: %v", entries)
	}
}
