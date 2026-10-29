// Package git runs the system git binary for Quiver's own repositories:
// local/ and the cloned sources.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ErrNotInstalled is returned when no git binary is on PATH.
var ErrNotInstalled = errors.New("git is not installed or not on PATH: install git and try again")

// How long a git command may run. A stalled network would otherwise make
// clone, fetch and pull wait forever. Variables, so tests can shorten them.
var (
	localTimeout   = time.Minute
	networkTimeout = 5 * time.Minute
)

// run runs a local git command with args in dir and returns trimmed stdout.
func run(dir string, args ...string) (string, error) {
	return runTimeout(localTimeout, dir, args...)
}

// runNetwork is run for commands that talk to a remote.
func runNetwork(dir string, args ...string) (string, error) {
	return runTimeout(networkTimeout, dir, args...)
}

// runTimeout runs git and stops it after timeout. Prompts are disabled, so
// missing credentials fail instead of hanging. Errors include the command
// and git's stderr.
func runTimeout(timeout time.Duration, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	// Helpers such as git-remote-https can keep the output open after git
	// is killed; stop waiting for them.
	cmd.WaitDelay = 5 * time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", ErrNotInstalled
		}
		if ctx.Err() != nil {
			return "", fmt.Errorf("git %s: no answer within %v; check the network and try again", strings.Join(args, " "), timeout)
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

// CommonDir returns the absolute path of the git folder shared by every
// worktree of the repository that dir is in. Two folders are worktrees of
// the same repository when their CommonDir is the same.
func CommonDir(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(out) {
		out = filepath.Join(dir, out)
	}
	if real, err := filepath.EvalSymlinks(out); err == nil {
		out = real
	}
	return filepath.Clean(out), nil
}

// Tracked reports whether git tracks any file under path in dir.
func Tracked(dir, path string) bool {
	out, err := run(dir, "ls-files", "--", path)
	return err == nil && out != ""
}
