package library

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/source"
)

// countScans replaces scan for the test and returns the call counter.
func countScans(t *testing.T) *int {
	t.Helper()
	n := 0
	t.Cleanup(func() { scan = source.Discover })
	scan = func(dir string) (*source.Discovery, error) {
		n++
		return source.Discover(dir)
	}
	return &n
}

func gitRepo(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	if err := git.Init(dir); err != nil {
		t.Fatal(err)
	}
	if err := git.Commit(dir, "init"); err != nil {
		t.Fatal(err)
	}
}

func loadCount(t *testing.T) int {
	t.Helper()
	l, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return len(l.All())
}

func TestIndexCache(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	repo := remoteFixture(t)
	gitRepo(t, repo)
	scans := countScans(t)

	for range 2 {
		if n := loadCount(t); n != 2 {
			t.Fatalf("got %d skills", n)
		}
	}
	if *scans != 1 {
		t.Fatalf("second load scanned again: %d scans", *scans)
	}
	if out, err := exec.Command("git", "-C", repo, "status", "--porcelain").Output(); err != nil || len(out) > 0 {
		t.Fatalf("index dirties the tree: %q, %v", out, err)
	}

	// A new commit changes HEAD, so the next load scans again.
	p := filepath.Join(repo, "new", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("---\nname: new\ndescription: New.\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := git.Commit(repo, "add new"); err != nil {
		t.Fatal(err)
	}
	if n := loadCount(t); n != 3 || *scans != 2 {
		t.Errorf("got %d skills, %d scans", n, *scans)
	}
}

func TestIndexBrokenCacheRescans(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	repo := remoteFixture(t)
	gitRepo(t, repo)
	if err := os.WriteFile(filepath.Join(repo, ".git", IndexFile), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	scans := countScans(t)
	if n := loadCount(t); n != 2 || *scans != 1 {
		t.Errorf("got %d skills, %d scans", n, *scans)
	}
}

func TestIndexNotGitScansEveryTime(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	remoteFixture(t)
	scans := countScans(t)
	loadCount(t)
	loadCount(t)
	if *scans != 2 {
		t.Errorf("got %d scans", *scans)
	}
}
