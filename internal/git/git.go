// Package git runs the system git binary for Quiver's own repositories:
// local/ and the cloned sources.
package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ErrNotInstalled is returned when no git binary is on PATH.
var ErrNotInstalled = errors.New("git is not installed or not on PATH: install git and try again")

// run runs git with args in dir and returns trimmed stdout. Prompts are
// disabled, so missing credentials fail instead of hanging. Errors include
// the command and git's stderr.
func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", ErrNotInstalled
		}
		msg := strings.TrimSpace(stderr.String())
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Head returns the commit hash HEAD points to.
func Head(dir string) (string, error) {
	return run(dir, "rev-parse", "HEAD")
}

// HeadTime returns the commit time of HEAD.
func HeadTime(dir string) (time.Time, error) {
	out, err := run(dir, "log", "-1", "--format=%cI")
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, out)
}

// HasCommit reports whether dir is a repository with at least one commit.
func HasCommit(dir string) bool {
	_, err := run(dir, "rev-parse", "--verify", "--quiet", "HEAD")
	return err == nil
}

// Init creates a repository in dir.
func Init(dir string) error {
	_, err := run(dir, "init", "--quiet")
	return err
}

// Commit stages paths (or every change when paths is empty) and commits
// them. Deleted paths are staged too. It does nothing when there is no
// change. If the user has no git identity, the commit is made as "Quiver".
func Commit(dir, msg string, paths ...string) error {
	add := []string{"add", "--all", "--"}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	if _, err := run(dir, append(add, paths...)...); err != nil {
		return err
	}
	if _, err := run(dir, "diff", "--cached", "--quiet"); err == nil {
		return nil // nothing staged
	}
	var pre []string
	if email, _ := run(dir, "config", "user.email"); email == "" {
		pre = []string{"-c", "user.name=Quiver", "-c", "user.email=quiver@localhost"}
	}
	args := append(pre, "-c", "commit.gpgsign=false", "commit", "--quiet", "--no-verify", "-m", msg)
	_, err := run(dir, args...)
	return err
}

// Status returns the uncommitted changes in dir, one "XY path" line each
// (git status --short), or "" when there are none.
func Status(dir string) (string, error) {
	out, err := run(dir, "status", "--short", "--untracked-files=all")
	if err != nil {
		return "", err
	}
	return out, nil
}
