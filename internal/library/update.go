package library

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/source"
)

// Update is what "quiver update" changed in one source. Skill IDs carry the
// "<source>/" prefix.
type Update struct {
	Source string
	// From and To are HEAD before and after the pull. They are equal when
	// the source was up to date.
	From, To string
	// Reset is set when a rewritten history was replaced by the remote one.
	Reset   bool
	Added   []string
	Changed []string
	Removed []string
}

// ErrDiverged is returned by UpdateSource when the remote history was
// rewritten (for example force-pushed), so a pull cannot fast-forward.
var ErrDiverged = errors.New("the remote history was rewritten, so it cannot be fast-forwarded")

// UpdateSource fast-forwards a source to its remote and compares its skills
// before and after. If the remote history was rewritten, it returns
// ErrDiverged, unless reset is set: then it replaces the local history with
// the remote one.
func UpdateSource(name string, reset bool) (*Update, error) {
	dir, err := SourcesDir()
	if err != nil {
		return nil, err
	}
	repo := filepath.Join(dir, name)
	if _, err := os.Stat(repo); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("source %s is not downloaded; add it again", name)
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	saved := cfg.Sources[name].IDs
	before, err := discover(repo)
	if err != nil {
		return nil, err
	}
	// A source added before IDs were saved starts from the current ones.
	ids := before.Stabilize(saved)
	from, err := git.Head(repo)
	if err != nil {
		return nil, err
	}
	u := &Update{Source: name, From: from}
	if err := git.Pull(repo); err != nil {
		// A failed pull has still fetched, so @{u} is the new remote head.
		if ok, aerr := git.IsAncestor(repo, "HEAD", "@{u}"); aerr != nil || ok {
			return nil, err
		}
		if !reset {
			return nil, ErrDiverged
		}
		if err := git.ResetToUpstream(repo); err != nil {
			return nil, err
		}
		u.Reset = true
	}
	if u.To, err = git.Head(repo); err != nil {
		return nil, err
	}
	if u.From == u.To {
		return u, saveIDs(name, saved, ids)
	}
	after, err := discover(repo)
	if err != nil {
		return nil, err
	}
	ids = after.Stabilize(ids)
	paths, err := git.DiffNames(repo, u.From, u.To)
	if err != nil {
		return nil, err
	}
	u.compare(before, after, paths)
	return u, saveIDs(name, saved, ids)
}

// saveIDs records the source's skill IDs in the config when they changed.
func saveIDs(name string, saved, ids map[string]string) error {
	if maps.Equal(saved, ids) {
		return nil
	}
	return config.Update(func(c *config.Config) error {
		if src, ok := c.Sources[name]; ok {
			src.IDs = ids
			c.Sources[name] = src
		}
		return nil
	})
}

// compare sorts skills into added, changed and removed. A skill changed when
// a changed path is inside its folder, before or after the update.
func (u *Update) compare(before, after *source.Discovery, paths []string) {
	old := map[string]source.Found{}
	for _, f := range before.Skills {
		old[f.ID] = f
	}
	touched := func(dir string) bool {
		for _, p := range paths {
			if dir == "." || p == dir || strings.HasPrefix(p, dir+"/") {
				return true
			}
		}
		return false
	}
	for _, f := range after.Skills {
		id := u.Source + "/" + f.ID
		prev, ok := old[f.ID]
		delete(old, f.ID)
		switch {
		case !ok:
			u.Added = append(u.Added, id)
		case touched(f.Dir) || touched(prev.Dir):
			u.Changed = append(u.Changed, id)
		}
	}
	for _, f := range before.Skills {
		if _, ok := old[f.ID]; ok {
			u.Removed = append(u.Removed, u.Source+"/"+f.ID)
		}
	}
}
