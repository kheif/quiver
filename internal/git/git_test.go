package git

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newRepo returns an empty repository with no git identity, so tests do not
// depend on the user's git config.
func newRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("LC_ALL", "C")
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatal(err)
	}
	return dir
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCommitAndHead(t *testing.T) {
	dir := newRepo(t)
	if HasCommit(dir) {
		t.Fatal("new repo has a commit")
	}
	write(t, dir, "a.md", "a")
	if err := Commit(dir, "first"); err != nil {
		t.Fatal(err)
	}
	first, err := Head(dir)
	if err != nil || len(first) != 40 {
		t.Fatalf("head %q, %v", first, err)
	}
	if at, err := HeadTime(dir); err != nil || time.Since(at) > time.Minute {
		t.Errorf("head time %v, %v", at, err)
	}
	author, _ := run(dir, "log", "-1", "--format=%an <%ae>")
	if author != "Quiver <quiver@localhost>" {
		t.Errorf("author %q", author)
	}

	// No change: no new commit.
	if err := Commit(dir, "empty"); err != nil {
		t.Fatal(err)
	}
	if h, _ := Head(dir); h != first {
		t.Error("empty commit made")
	}
}

func TestCommitOnlyGivenPaths(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "a/SKILL.md", "a")
	write(t, dir, "b/SKILL.md", "b")
	if err := Commit(dir, "add a", "a"); err != nil {
		t.Fatal(err)
	}
	files, _ := run(dir, "ls-files")
	if files != "a/SKILL.md" {
		t.Errorf("committed %q", files)
	}

	// A deleted path is committed as a deletion.
	if err := os.RemoveAll(filepath.Join(dir, "a")); err != nil {
		t.Fatal(err)
	}
	if err := Commit(dir, "remove a", "a"); err != nil {
		t.Fatal(err)
	}
	if files, _ := run(dir, "ls-files"); files != "" {
		t.Errorf("still tracked %q", files)
	}
}

func TestErrorIncludesStderr(t *testing.T) {
	t.Setenv("LC_ALL", "C")
	_, err := Head(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("got %v", err)
	}
}

func TestMissingGit(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := Head(t.TempDir()); !errors.Is(err, ErrNotInstalled) {
		t.Errorf("got %v", err)
	}
}
