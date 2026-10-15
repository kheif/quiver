package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestInitCommand(t *testing.T) {
	home := filepath.Join(t.TempDir(), "q")
	t.Setenv("QUIVER_HOME", home)

	out := run(t, "init")
	if !strings.Contains(out, "created "+home) {
		t.Errorf("first run: %q", out)
	}
	if _, err := os.Stat(filepath.Join(home, "local", ".git")); err != nil {
		t.Error(err)
	}

	if err := os.Remove(filepath.Join(home, "registry.json")); err != nil {
		t.Fatal(err)
	}
	out = run(t, "init")
	if !strings.Contains(out, "created "+filepath.Join(home, "registry.json")) {
		t.Errorf("fill-in run: %q", out)
	}

	out = run(t, "init")
	if !strings.Contains(out, "Quiver home is ready") {
		t.Errorf("third run: %q", out)
	}
}
