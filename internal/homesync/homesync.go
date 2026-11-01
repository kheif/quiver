// Package homesync keeps the Quiver home in a git repository with one
// remote, so the library and the assignments can be synced between
// machines (quiver sync).
//
// The repository tracks local/, config.json and registry.json. local/ is
// part of it instead of a submodule: one commit and one push carry a sync,
// and no second remote is needed for the skills. Sources are cloned again
// from config.json, and state/ and logs/ belong to each machine.
package homesync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/setup"
)

// gitignore lists what the repository tracks; everything else is ignored.
const gitignore = `# Written by quiver sync init. Only these are synced: sources are cloned
# again from config.json, and the rest belongs to this machine.
/*
!/.gitignore
!/local/
!/config.json
!/registry.json
`

// localBackup is where sync init keeps local/.git until the home repository
// holds its history, so a failed init can put it back.
const localBackup = ".sync-local-git"

// Result tells what a sync did.
type Result struct {
	Remote string
	Branch string
	// Pulled and Pushed are true when changes came in and went out.
	Pulled, Pushed bool
	// Merged lists the files whose changes on both machines were merged.
	Merged []string
	// Cloned lists the sources that were cloned, and Failed the sources and
	// pins that could not be restored, with the reason.
	Cloned, Failed []string
}

// Init turns the Quiver home into a git repository that syncs with remote,
// and pushes it. When the remote already holds a synced home, from another
// machine, the two are merged first. On failure nothing is changed.
func Init(remote string) (*Result, error) {
	if strings.HasPrefix(remote, "-") {
		return nil, fmt.Errorf("invalid git remote %q", remote)
	}
	home, err := fsutil.QuiverHome()
	if err != nil {
		return nil, err
	}
	if setup.Synced(home) {
		url, _ := git.OriginURL(home)
		return nil, fmt.Errorf("%s already syncs with %s: run quiver sync", home, url)
	}
	branch, err := git.RemoteBranch(remote)
	if err != nil {
		return nil, err
	}
	res := &Result{Remote: remote, Branch: branch}
	if branch == "" {
		res.Branch = "main"
	}
	err = fsutil.WithLock(func() error { return initRepo(home, res, branch != "") })
	if err != nil {
		return nil, err
	}
	return res, restore(res)
}

func initRepo(home string, res *Result, join bool) (err error) {
	if err := pathsOut(); err != nil {
		return err
	}
	if err := git.InitBranch(home, res.Branch); err != nil {
		return err
	}
	local := filepath.Join(home, "local")
	backup := filepath.Join(home, localBackup)
	var start string
	defer func() {
		if err == nil {
			_ = os.RemoveAll(backup)
			return
		}
		if start != "" {
			_ = git.AbortMerge(home)
			_ = git.ResetTo(home, start)
		}
		_ = os.RemoveAll(filepath.Join(home, ".git"))
		_ = os.Remove(filepath.Join(home, ".gitignore"))
		if _, serr := os.Stat(backup); serr == nil {
			_ = os.Rename(backup, filepath.Join(local, ".git"))
		}
	}()
	if err := git.AddOrigin(home, res.Remote); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(home, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		return err
	}
	// The history of local/ becomes a parent of the first commit, so git
	// log still shows it.
	var parents []string
	if _, serr := os.Stat(filepath.Join(local, ".git")); serr == nil {
		if git.HasCommit(local) {
			head, err := git.FetchHead(home, "./local")
			if err != nil {
				return err
			}
			parents = append(parents, head)
		}
		if err := os.Rename(filepath.Join(local, ".git"), backup); err != nil {
			return err
		}
	}
	if err := git.CommitAll(home, "Start syncing the Quiver home", parents...); err != nil {
		return err
	}
	if start, err = git.Head(home); err != nil {
		return err
	}
	if join {
		if err := git.FetchOrigin(home); err != nil {
			return err
		}
		upstream := "origin/" + res.Branch
		res.Pulled = true
		err := git.MergeUnrelated(home, upstream, "Join the synced Quiver home")
		if errors.Is(err, git.ErrConflict) {
			if err := resolve(home, res); err != nil {
				return fmt.Errorf("%w\nquiver sync init changed nothing. Make this machine's copy match the other one, or rename or delete it, then run it again", err)
			}
			err = git.CommitMerge(home)
		}
		if err != nil {
			return err
		}
	}
	res.Pushed = true
	return git.Push(home, res.Branch)
}

// Sync commits the changes on this machine, pulls with rebase and pushes.
// Changes to registry.json and config.json on both machines are merged; any
// other conflict stops the sync with nothing pulled or pushed. Afterwards
// sources that are in the config but not downloaded are cloned.
func Sync() (*Result, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return nil, err
	}
	if !setup.Synced(home) {
		return nil, fmt.Errorf("%s is not synced yet: run quiver sync init <git-remote>", home)
	}
	res := &Result{}
	if res.Remote, err = git.OriginURL(home); err != nil {
		return nil, err
	}
	if res.Branch, err = git.Branch(home); err != nil {
		return nil, fmt.Errorf("the repository in %s is not on a branch: check it out with git, then run quiver sync again", home)
	}
	if err := fsutil.WithLock(func() error { return syncRepo(home, res) }); err != nil {
		return nil, err
	}
	return res, restore(res)
}

func syncRepo(home string, res *Result) error {
	if git.Rebasing(home) {
		return fmt.Errorf("a rebase in %s is not finished: finish it with git rebase --continue or undo it with git rebase --abort, then run quiver sync again", home)
	}
	if err := pathsOut(); err != nil {
		return err
	}
	status, err := git.Status(home)
	if err != nil {
		return err
	}
	if status != "" {
		host, _ := os.Hostname()
		if err := git.Commit(home, strings.TrimSpace("Sync from "+host)); err != nil {
			return err
		}
	}
	if err := git.FetchOrigin(home); err != nil {
		return err
	}
	upstream := "origin/" + res.Branch
	if _, err := git.ResolveCommit(home, upstream); err != nil {
		res.Pushed = true // the remote has no such branch yet
		return git.Push(home, res.Branch)
	}
	behind, err := git.Count(home, "HEAD", upstream)
	if err != nil {
		return err
	}
	if res.Pulled = behind > 0; res.Pulled {
		if err := rebase(home, upstream, res); err != nil {
			return err
		}
	}
	ahead, err := git.Count(home, upstream, "HEAD")
	if res.Pushed = ahead > 0; err != nil || !res.Pushed {
		return err
	}
	if err := git.Push(home, res.Branch); err != nil {
		return fmt.Errorf("%w\nThe remote may have changed meanwhile: run quiver sync again", err)
	}
	return nil
}

// rebase replays the local commits on upstream and merges the JSON files
// that conflict. On any other conflict it puts everything back.
func rebase(home, upstream string, res *Result) error {
	err := git.Rebase(home, upstream)
	for errors.Is(err, git.ErrConflict) {
		if err := resolve(home, res); err != nil {
			_ = git.AbortRebase(home)
			return fmt.Errorf("%w\nNothing was pulled or pushed. Resolve it with git in %s: run git pull --rebase, fix the files, git add them and run git rebase --continue; then run quiver sync again", err, home)
		}
		err = git.ContinueRebase(home)
	}
	if err != nil && git.Rebasing(home) {
		_ = git.AbortRebase(home)
	}
	return err
}

// resolve merges the conflicting registry.json and config.json and stages
// them. It fails, resolving nothing, when any other file conflicts.
func resolve(home string, res *Result) error {
	files, err := git.Unmerged(home)
	if err != nil {
		return err
	}
	var other []string
	for _, f := range files {
		if f != "registry.json" && f != "config.json" {
			other = append(other, f)
		}
	}
	if len(other) > 0 {
		return fmt.Errorf("these files changed on both machines and cannot be merged automatically:\n  %s", strings.Join(other, "\n  "))
	}
	for _, f := range files {
		var stages [3][]byte
		for i := range stages {
			if stages[i], err = git.Stage(home, i+1, f); err != nil {
				return err
			}
		}
		var merged any
		if f == "registry.json" {
			merged, err = registry.Merge(stages[0], stages[1], stages[2])
		} else {
			merged, err = config.Merge(stages[0], stages[1], stages[2])
		}
		if err != nil {
			return err
		}
		if err := fsutil.WriteJSONAtomic(filepath.Join(home, f), merged); err != nil {
			return err
		}
		if err := git.Add(home, f); err != nil {
			return err
		}
		if !slices.Contains(res.Merged, f) {
			res.Merged = append(res.Merged, f)
		}
	}
	return nil
}

// pathsOut rewrites a version 1 registry.json, which still holds the
// folders of this machine's projects, so they move to state/ before the
// file is committed. Call it inside fsutil.WithLock.
func pathsOut() error {
	r, err := registry.Load()
	if err != nil || r.Version != 1 {
		return err
	}
	return registry.Write(r)
}

// restore clones the sources and checks out the pins that are missing.
func restore(res *Result) error {
	var err error
	res.Cloned, res.Failed, err = library.CloneMissing()
	return err
}
