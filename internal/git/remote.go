package git

import (
	"errors"
	"os/exec"
)

// Clone makes a shallow clone (--depth 1) of url into dir. "--" stops a url
// that starts with "-" from being read as an option.
func Clone(url, dir string) error {
	_, err := run("", "clone", "--quiet", "--depth", "1", "--", url, dir)
	return err
}

// Fetch downloads new commits from origin without changing the files.
func Fetch(dir string) error {
	_, err := run(dir, "fetch", "--quiet")
	return err
}

// Pull fast-forwards dir to origin. It fails instead of merging when the
// history has diverged.
func Pull(dir string) error {
	_, err := run(dir, "pull", "--quiet", "--ff-only")
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
	if _, err := run(dir, "fetch", "--quiet", "--depth", "1"); err != nil {
		return err
	}
	_, err := run(dir, "reset", "--quiet", "--hard", "@{u}")
	return err
}
