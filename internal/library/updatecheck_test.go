package library

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/git"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// remoteWithClone makes a remote with one commit and a clone of it as the
// source "acme". commit adds a commit to the remote.
func remoteWithClone(t *testing.T) (commit func(), clone string) {
	t.Helper()
	t.Setenv("QUIVER_HOME", t.TempDir())
	work := t.TempDir()
	if err := git.Init(work); err != nil {
		t.Fatal(err)
	}
	n := 0
	commit = func() {
		n++
		p := filepath.Join(work, "pdf", "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte{byte('0' + n)}, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := git.Commit(work, "change"); err != nil {
			t.Fatal(err)
		}
		gitIn(t, work, "push", "--quiet", "origin", "HEAD")
	}
	bare := filepath.Join(t.TempDir(), "remote.git")
	gitIn(t, "", "init", "--quiet", "--bare", bare)
	gitIn(t, work, "remote", "add", "origin", bare)
	commit()
	cfg := config.Default()
	cfg.Sources["acme"] = config.Source{Repo: "file://" + bare}
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	dir, _ := SourcesDir()
	clone = filepath.Join(dir, "acme")
	if err := git.Clone("file://"+bare, clone); err != nil {
		t.Fatal(err)
	}
	return commit, clone
}

func TestHasUpdateSeesANewCommitOnlyAfterTheInterval(t *testing.T) {
	commit, clone := remoteWithClone(t)
	now := time.Now()
	if got, err := HasUpdate("acme", now); err != nil || got {
		t.Fatalf("first check = %v, %v; want no update", got, err)
	}
	commit()
	if got, err := HasUpdate("acme", now.Add(time.Hour)); err != nil || got {
		t.Fatalf("check inside the interval = %v, %v; want the cached answer, no update", got, err)
	}
	if got, err := HasUpdate("acme", now.Add(7*time.Hour)); err != nil || !got {
		t.Fatalf("check after the interval = %v, %v; want an update", got, err)
	}
	// Inside the new interval the answer stays, so the notice does not vanish.
	if got, _ := HasUpdate("acme", now.Add(8*time.Hour)); !got {
		t.Fatal("the update notice vanished inside the interval")
	}
	// Once the clone has taken the commit, the cached answer says up to date.
	if err := git.Fetch(clone); err != nil {
		t.Fatal(err)
	}
	to, _ := git.ResolveCommit(clone, "@{u}")
	if err := git.FastForward(clone, to); err != nil {
		t.Fatal(err)
	}
	if got, _ := HasUpdate("acme", now.Add(9*time.Hour)); got {
		t.Fatal("still an update after the clone took it")
	}
}

func TestHasUpdateHonoursTheConfiguredInterval(t *testing.T) {
	commit, _ := remoteWithClone(t)
	if err := config.Update(func(c *config.Config) error { c.UpdateCheckInterval = "30m"; return nil }); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if _, err := HasUpdate("acme", now); err != nil {
		t.Fatal(err)
	}
	commit()
	if got, _ := HasUpdate("acme", now.Add(20*time.Minute)); got {
		t.Fatal("asked the remote before 30m passed")
	}
	if got, _ := HasUpdate("acme", now.Add(31*time.Minute)); !got {
		t.Fatal("did not ask the remote after 30m")
	}
}

func TestHasUpdateFailureIsReturnedAndNotRecorded(t *testing.T) {
	_, clone := remoteWithClone(t)
	gitIn(t, clone, "remote", "set-url", "origin", "file://"+filepath.Join(t.TempDir(), "gone.git"))
	if _, err := HasUpdate("acme", time.Now()); err == nil {
		t.Fatal("an unreachable remote gave no error")
	}
	home := os.Getenv("QUIVER_HOME")
	if _, err := os.Stat(filepath.Join(home, "state.json")); !os.IsNotExist(err) {
		t.Fatalf("a failed check was recorded: %v", err)
	}
}

func TestHasUpdateOnAMissingClone(t *testing.T) {
	fixture(t)
	remoteFixture(t)
	if _, err := HasUpdate("gone", time.Now()); err == nil {
		t.Fatal("a source that is not downloaded gave no error")
	}
}
