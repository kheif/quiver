package git

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// The helpers below serve quiver sync, which keeps the Quiver home itself
// in a repository with one remote, origin.

// InitBranch creates a repository in dir whose first branch is branch.
func InitBranch(dir, branch string) error {
	if err := Init(dir); err != nil {
		return err
	}
	_, err := run(dir, "symbolic-ref", "HEAD", "refs/heads/"+branch)
	return err
}

// Branch returns the name of the branch checked out in dir.
func Branch(dir string) (string, error) {
	return run(dir, "symbolic-ref", "--quiet", "--short", "HEAD")
}

// AddOrigin sets url as the remote origin of dir.
func AddOrigin(dir, url string) error {
	_, err := run(dir, "remote", "add", "origin", url)
	return err
}

// OriginURL returns the url of the remote origin of dir.
func OriginURL(dir string) (string, error) {
	return run(dir, "remote", "get-url", "origin")
}

// RemoteBranch returns the branch of the repository at url that a clone
// would check out: the one its HEAD names, else main, else the first one.
// It returns "" when the repository has no branches yet.
func RemoteBranch(url string) (string, error) {
	out, err := runNetwork("", "ls-remote", "--symref", "--", url)
	if err != nil {
		return "", err
	}
	var head string
	var branches []string
	for _, line := range strings.Split(out, "\n") {
		if ref, ok := strings.CutPrefix(line, "ref: refs/heads/"); ok {
			head, _, _ = strings.Cut(ref, "\t")
		} else if _, ref, ok := strings.Cut(line, "\trefs/heads/"); ok {
			branches = append(branches, ref)
		}
	}
	for _, want := range []string{head, "main"} {
		if want != "" && slices.Contains(branches, want) {
			return want, nil
		}
	}
	if len(branches) > 0 {
		return branches[0], nil
	}
	return "", nil
}

// FetchOrigin downloads new commits from origin.
func FetchOrigin(dir string) error {
	_, err := runNetwork(dir, "fetch", "--quiet", "origin")
	return err
}

// Push sends branch to origin and makes it the upstream of the branch.
func Push(dir, branch string) error {
	_, err := runNetwork(dir, "push", "--quiet", "--set-upstream", "origin", "HEAD:refs/heads/"+branch)
	return err
}

// Count returns the number of commits reachable from b but not from a.
func Count(dir, a, b string) (int, error) {
	out, err := run(dir, "rev-list", "--count", a+".."+b)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}

// CommitAll stages every change in dir and commits it with parents, the
// way Commit does but with the parents given. It moves the checked-out
// branch to the new commit, even when nothing changed.
func CommitAll(dir, msg string, parents ...string) error {
	if _, err := run(dir, "add", "--all"); err != nil {
		return err
	}
	tree, err := run(dir, "write-tree")
	if err != nil {
		return err
	}
	args := []string{"commit-tree", tree, "-m", msg}
	for _, p := range parents {
		args = append(args, "-p", p)
	}
	commit, err := run(dir, committing(dir, args...)...)
	if err != nil {
		return err
	}
	_, err = run(dir, "update-ref", "HEAD", commit)
	return err
}

// FetchHead downloads HEAD of the repository at path into dir and returns
// its commit.
func FetchHead(dir, path string) (string, error) {
	if _, err := run(dir, "fetch", "--quiet", "--no-tags", path, "HEAD"); err != nil {
		return "", err
	}
	return ResolveCommit(dir, "FETCH_HEAD")
}

// Rebase replays the local commits of dir on top of onto. When it stops on
// a conflict, the error is ErrConflict.
func Rebase(dir, onto string) error {
	return conflictOr(dir, committing(dir, "rebase", "--quiet", onto)...)
}

// ContinueRebase goes on after the conflicts are resolved and staged. A
// commit that became empty is dropped.
func ContinueRebase(dir string) error {
	if _, err := run(dir, "diff", "--cached", "--quiet"); err == nil {
		return conflictOr(dir, committing(dir, "-c", "core.editor=true", "rebase", "--skip")...)
	}
	return conflictOr(dir, committing(dir, "-c", "core.editor=true", "rebase", "--continue")...)
}

// AbortRebase puts dir back as it was before Rebase.
func AbortRebase(dir string) error {
	_, err := run(dir, "rebase", "--abort")
	return err
}

// Rebasing reports whether a rebase stopped in dir and was not finished.
func Rebasing(dir string) bool {
	gitDir, err := run(dir, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return false
	}
	for _, d := range []string{"rebase-merge", "rebase-apply"} {
		if _, err := os.Stat(filepath.Join(gitDir, d)); err == nil {
			return true
		}
	}
	return false
}

// MergeUnrelated merges commit, whose history is unrelated to HEAD, into
// dir. When it stops on a conflict, the error is ErrConflict; finish it
// with CommitMerge or undo it with AbortMerge.
func MergeUnrelated(dir, commit, msg string) error {
	return conflictOr(dir, committing(dir, "merge", "--quiet", "--no-edit", "--allow-unrelated-histories", "-m", msg, commit)...)
}

// CommitMerge commits a merge whose conflicts are resolved and staged.
func CommitMerge(dir string) error {
	_, err := run(dir, committing(dir, "commit", "--quiet", "--no-verify", "--no-edit")...)
	return err
}

// AbortMerge puts dir back as it was before MergeUnrelated.
func AbortMerge(dir string) error {
	_, err := run(dir, "merge", "--abort")
	return err
}

// ErrConflict is returned when a rebase or merge stops on conflicting
// changes.
var ErrConflict = errors.New("conflicting changes")

// conflictOr runs args in dir and returns ErrConflict when files are left
// unmerged.
func conflictOr(dir string, args ...string) error {
	_, err := run(dir, args...)
	if err != nil {
		if files, _ := Unmerged(dir); len(files) > 0 {
			return ErrConflict
		}
	}
	return err
}

// Unmerged returns the files of dir with unresolved conflicts, relative to
// the top of the repository.
func Unmerged(dir string) ([]string, error) {
	out, err := run(dir, "diff", "--name-only", "--diff-filter=U")
	if err != nil || out == "" {
		return nil, err
	}
	return strings.Split(out, "\n"), nil
}

// Stage returns the version of path in a conflict: 1 for the common base,
// 2 for ours and 3 for theirs. It returns nil when that side has no such
// file.
func Stage(dir string, n int, path string) ([]byte, error) {
	spec := ":" + strconv.Itoa(n) + ":" + path
	if _, err := run(dir, "cat-file", "-e", spec); err != nil {
		return nil, nil
	}
	out, err := run(dir, "show", spec)
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

// Add stages paths in dir.
func Add(dir string, paths ...string) error {
	_, err := run(dir, append([]string{"add", "--"}, paths...)...)
	return err
}

// Show returns the content of path at rev, or nil when rev has no such file.
func Show(dir, rev, path string) ([]byte, error) {
	spec := rev + ":" + path
	if _, err := run(dir, "cat-file", "-e", spec); err != nil {
		return nil, nil
	}
	out, err := run(dir, "show", spec)
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

// MergeBase returns the best common ancestor of two commits, or an error
// when their histories are unrelated.
func MergeBase(dir, a, b string) (string, error) {
	return run(dir, "merge-base", a, b)
}
