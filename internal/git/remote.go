package git

import (
	"errors"
	"os"
	"os/exec"
)

// Clone makes a shallow clone (--depth 1) of url into dir. "--" stops a url
// that starts with "-" from being read as an option.
func Clone(url, dir string) error {
	_, err := runNetwork("", "clone", "--quiet", "--depth", "1", "--", url, dir)
	return err
}

// Fetch downloads new commits from origin without changing the files.
func Fetch(dir string) error {
	_, err := runNetwork(dir, "fetch", "--quiet")
	return err
}

// Pull fast-forwards dir to origin. It fails instead of merging when the
// history has diverged.
func Pull(dir string) error {
	_, err := runNetwork(dir, "pull", "--quiet", "--ff-only")
	return err
}

// IsAncestor reports whether commit a is an ancestor of commit b. After a
// failed Pull, IsAncestor(dir, "HEAD", "@{u}") is false when the remote
// history was rewritten.
func IsAncestor(dir, a, b string) (bool, error) {
	_, err := run(dir, "merge-base", "--is-ancestor", a, b)
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false, nil
	}
	return err == nil, err
}

// ResetToUpstream fetches the remote and moves the branch and files to it,
// dropping local commits and changes. Old commits stay in the object store
// until git prunes them, so they can still be diffed.
func ResetToUpstream(dir string) error {
	if _, err := runNetwork(dir, "fetch", "--quiet", "--depth", "1"); err != nil {
		return err
	}
	_, err := run(dir, "reset", "--quiet", "--hard", "@{u}")
	return err
}

// ResolveCommit returns the full hash of the commit rev names in dir.
func ResolveCommit(dir, rev string) (string, error) {
	return run(dir, "rev-parse", "--verify", "--quiet", "--end-of-options", rev+"^{commit}")
}

// FetchHistory downloads the whole history of a shallow clone, or new
// commits of a full one, so older commits can be resolved.
func FetchHistory(dir string) error {
	args := []string{"fetch", "--quiet"}
	if shallow, err := run(dir, "rev-parse", "--is-shallow-repository"); err == nil && shallow == "true" {
		args = append(args, "--unshallow")
	}
	_, err := runNetwork(dir, args...)
	return err
}

// AddWorktree checks commit out into path, detached, as a worktree of the
// repository in dir. Its objects stay in dir.
func AddWorktree(dir, path, commit string) error {
	_, err := run(dir, "worktree", "add", "--quiet", "--detach", "--", path, commit)
	return err
}

// RemoveWorktree deletes the worktree at path and forgets it in dir. A
// folder git does not know as a worktree is deleted too.
func RemoveWorktree(dir, path string) error {
	if _, err := run(dir, "worktree", "remove", "--force", path); err == nil {
		return nil
	}
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	_, err := run(dir, "worktree", "prune")
	return err
}
