package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newRemote returns a work repository with one commit and a file:// URL of a
// bare copy of it. Push new work commits with push.
func newRemote(t *testing.T) (work, url string) {
	t.Helper()
	work = newRepo(t)
	write(t, work, "pdf/SKILL.md", "v1")
	if err := Commit(work, "v1"); err != nil {
		t.Fatal(err)
	}
	bare := filepath.Join(t.TempDir(), "remote.git")
	if _, err := run("", "clone", "--quiet", "--bare", work, bare); err != nil {
		t.Fatal(err)
	}
	if _, err := run(work, "remote", "add", "origin", bare); err != nil {
		t.Fatal(err)
	}
	return work, "file://" + bare
}

func push(t *testing.T, work string) {
	t.Helper()
	if _, err := run(work, "push", "--quiet", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
}

func TestCloneIsShallow(t *testing.T) {
	work, url := newRemote(t)
	write(t, work, "pdf/SKILL.md", "v2")
	if err := Commit(work, "v2"); err != nil {
		t.Fatal(err)
	}
	push(t, work)

	dir := filepath.Join(t.TempDir(), "clone")
	if err := Clone(url, dir); err != nil {
		t.Fatal(err)
	}
	if count, _ := run(dir, "rev-list", "--count", "HEAD"); count != "1" {
		t.Errorf("clone has %s commits, want 1", count)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "pdf", "SKILL.md"))
	if string(data) != "v2" {
		t.Errorf("got %q", data)
	}
}

func TestFetchThenFastForward(t *testing.T) {
	work, url := newRemote(t)
	dir := filepath.Join(t.TempDir(), "clone")
	if err := Clone(url, dir); err != nil {
		t.Fatal(err)
	}
	old, _ := Head(dir)

	write(t, work, "pdf/SKILL.md", "v2")
	if err := Commit(work, "v2"); err != nil {
		t.Fatal(err)
	}
	push(t, work)
	want, _ := Head(work)

	if err := Fetch(dir); err != nil {
		t.Fatal(err)
	}
	if h, _ := Head(dir); h != old {
		t.Error("fetch moved HEAD")
	}
	if remote, _ := run(dir, "rev-parse", "FETCH_HEAD"); remote != want {
		t.Errorf("FETCH_HEAD %s, want %s", remote, want)
	}

	if err := FastForward(dir, "@{u}"); err != nil {
		t.Fatal(err)
	}
	if h, _ := Head(dir); h != want {
		t.Errorf("head %s, want %s", h, want)
	}
}

func TestCloneBadURL(t *testing.T) {
	t.Setenv("LC_ALL", "C")
	err := Clone("file:///does/not/exist", filepath.Join(t.TempDir(), "x"))
	if err == nil || !strings.Contains(err.Error(), "does not appear to be a git repository") {
		t.Errorf("got %v", err)
	}
}

func TestResetAfterRewrite(t *testing.T) {
	work, url := newRemote(t)
	dir := filepath.Join(t.TempDir(), "clone")
	if err := Clone(url, dir); err != nil {
		t.Fatal(err)
	}
	old, _ := Head(dir)

	write(t, work, "pdf/SKILL.md", "rewritten")
	if _, err := run(work, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--quiet", "--amend", "-am", "v1b"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(work, "push", "--quiet", "--force", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	want, _ := Head(work)

	if err := Fetch(dir); err != nil {
		t.Fatal(err)
	}
	if err := FastForward(dir, "@{u}"); err == nil {
		t.Fatal("fast-forward to a rewritten history succeeded")
	}
	if ok, err := IsAncestor(dir, "HEAD", "@{u}"); ok || err != nil {
		t.Fatalf("IsAncestor = %v, %v; want false", ok, err)
	}
	if err := ResetTo(dir, "@{u}"); err != nil {
		t.Fatal(err)
	}
	if h, _ := Head(dir); h != want {
		t.Errorf("head %s, want %s", h, want)
	}
	if names, err := DiffNames(dir, old, want); err != nil || len(names) != 1 || names[0] != "pdf/SKILL.md" {
		t.Errorf("diff with the old head: %v, %v", names, err)
	}
	if ok, err := IsAncestor(dir, old, "HEAD"); ok || err != nil {
		t.Errorf("IsAncestor(old) = %v, %v; want false", ok, err)
	}
}
