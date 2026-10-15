package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestQuiverHome(t *testing.T) {
	home, _ := os.UserHomeDir()

	t.Setenv("QUIVER_HOME", "")
	got, err := QuiverHome()
	if err != nil || got != filepath.Join(home, ".quiver") {
		t.Errorf("default: got %q, %v", got, err)
	}

	t.Setenv("QUIVER_HOME", "/tmp/qh")
	if got, _ := QuiverHome(); got != "/tmp/qh" {
		t.Errorf("override: got %q", got)
	}

	t.Setenv("QUIVER_HOME", "~/qh")
	if got, _ := QuiverHome(); got != filepath.Join(home, "qh") {
		t.Errorf("override with ~: got %q", got)
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	cases := map[string]string{
		"~":        home,
		"~/a/b":    filepath.Join(home, "a/b"),
		"/abs":     "/abs",
		"rel/~":    "rel/~",
		"~other/x": "~other/x",
		"":         "",
	}
	for in, want := range cases {
		got, err := ExpandHome(in)
		if err != nil || got != want {
			t.Errorf("ExpandHome(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}
