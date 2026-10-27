package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Added    []string
	Changed  []string
	Removed  []string
}

// UpdateSource fast-forwards a source to its remote and compares its skills
// before and after.
func UpdateSource(name string) (*Update, error) {
	dir, err := SourcesDir()
	if err != nil {
		return nil, err
	}
	repo := filepath.Join(dir, name)
	if _, err := os.Stat(repo); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("source %s is not downloaded; add it again", name)
	}
	before, err := discover(repo)
	if err != nil {
		return nil, err
	}
	from, err := git.Head(repo)
	if err != nil {
		return nil, err
	}
	if err := git.Pull(repo); err != nil {
		return nil, err
	}
	to, err := git.Head(repo)
	if err != nil {
		return nil, err
	}
	u := &Update{Source: name, From: from, To: to}
	if from == to {
		return u, nil
	}
	after, err := discover(repo)
	if err != nil {
		return nil, err
	}
	paths, err := git.DiffNames(repo, from, to)
	if err != nil {
		return nil, err
	}
	u.compare(before, after, paths)
	return u, nil
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
