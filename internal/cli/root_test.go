package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "quiver dev\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestVerboseFlag(t *testing.T) {
	t.Cleanup(func() { verbose = false })
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--verbose"})
	cmd.SetOut(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !verbose {
		t.Error("verbose not set")
	}
}

func TestBareQuiver(t *testing.T) {
	home := sourceEnv(t)
	dir := t.TempDir()
	t.Chdir(dir)
	next := func(out string) string {
		_, after, _ := strings.Cut(out, "Next:    ")
		line, _, _ := strings.Cut(after, "\n")
		return line
	}
	// No home yet: it is not created, and init is the next step.
	out, err := runIn(t, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "Folder:  "+dir+" (not a project") || next(out) != "quiver init" || !strings.Contains(out, "Available Commands:") {
		t.Errorf("no home:\n%s", out)
	}
	if _, err := os.Stat(home); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("home created: %v", err)
	}

	assignEnv(t, "pdf", "tdd")
	t.Chdir(dir)
	if out, _ = runIn(t, ""); next(out) != "quiver add <id> (see quiver list)" || !strings.Contains(out, "Served:  0 skills\n") {
		t.Errorf("nothing served:\n%s", out)
	}
	for _, args := range [][]string{{"add", "pdf", "--global", "--always"}, {"add", "tdd"}} {
		if _, err := runIn(t, "", args...); err != nil {
			t.Fatal(err)
		}
	}
	out, _ = runIn(t, "")
	want := "Project: " + filepath.Base(dir) + " (" + dir + ")\nServed:  2 skills (1 global, 1 project, 1 always-on)\nNext:    quiver status\n\nUsage:"
	if !strings.HasPrefix(out, want) {
		t.Errorf("got:\n%s\nwant prefix:\n%s", out, want)
	}
	if _, err := runIn(t, "", "nope"); err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("unknown command: %v", err)
	}
}

func TestBareQuiverEmptyLibrary(t *testing.T) {
	sourceEnv(t)
	t.Chdir(t.TempDir())
	if _, err := runIn(t, "", "init"); err != nil {
		t.Fatal(err)
	}
	if out, _ := runIn(t, ""); !strings.Contains(out, "Next:    quiver new <name> to write a skill, or quiver source add <repo>\n") {
		t.Errorf("got:\n%s", out)
	}
}
