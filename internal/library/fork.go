package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/skill"
)

// ForkedFromField is the frontmatter field a fork gets: the ID of the
// source skill and the source commit, as <id>@<commit>.
const ForkedFromField = "forkedFrom"

// Forked is the result of ForkRemote.
type Forked struct {
	// Name is the ID of the local copy, and Dir its folder.
	Name, Dir string
	// From is <id>@<commit>, as written to ForkedFromField.
	From string
	// Replaced lists where the source skill was assigned; the copy took
	// its place there.
	Replaced Usage
	Warnings []string
}

// ForkRemote copies the source skill id into local/<name> (name defaults
// to the skill's name), records where it came from in ForkedFromField, and
// puts the copy in the place of the original wherever it is assigned,
// always-on marks included. It commits the copy in local/ as
// "fork <name> from <id>". From then on the copy can be edited, and
// "quiver update" no longer changes what the projects get.
func ForkRemote(id, name string) (Forked, error) {
	l, err := Load()
	if err != nil {
		return Forked{}, err
	}
	s, err := l.Get(id)
	if err != nil {
		return Forked{}, err
	}
	if s.Source == LocalSource {
		return Forked{}, fmt.Errorf("%s is already a local skill; change it with quiver edit %s", id, id)
	}
	if name == "" {
		name = s.Name
	}
	if err := skill.ValidateName(name); err != nil {
		return Forked{}, fmt.Errorf("%w; choose a name with --as", err)
	}
	home, err := fsutil.QuiverHome()
	if err != nil {
		return Forked{}, err
	}
	commit, err := git.Head(filepath.Join(home, "sources", s.Source))
	if err != nil {
		return Forked{}, fmt.Errorf("source %s: %w", s.Source, err)
	}
	dir, err := LocalDir()
	if err != nil {
		return Forked{}, err
	}
	res := Forked{Name: name, Dir: filepath.Join(dir, name), From: id + "@" + commit}
	err = fsutil.WithLock(func() error {
		if _, err := os.Lstat(res.Dir); err == nil {
			return fmt.Errorf("%w: %s; choose another name with --as", ErrSkillExists, res.Dir)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		r, err := registry.Load()
		if err != nil {
			return err
		}
		res.Replaced = skillUsage(r, id)
		r.Replace(id, name)
		tmp, err := os.MkdirTemp(dir, ".new-"+name+"-")
		if err != nil {
			return err
		}
		defer func() { _ = os.RemoveAll(tmp) }()
		if res.Warnings, err = copySkill(s.Dir, tmp, name, s.Name); err != nil {
			return err
		}
		if err := markFork(filepath.Join(tmp, skill.FileName), res.From); err != nil {
			return err
		}
		if err := os.Rename(tmp, res.Dir); err != nil {
			return err
		}
		if !res.Replaced.Empty() {
			if err := registry.Write(r); err != nil {
				return errors.Join(err, os.RemoveAll(res.Dir))
			}
		}
		if err := git.Commit(dir, "fork "+name+" from "+id, name); err != nil {
			return fmt.Errorf("skill %s was forked, but the commit in %s failed: %w", name, dir, err)
		}
		return nil
	})
	return res, err
}

// markFork writes ForkedFromField into the SKILL.md at path.
func markFork(path, from string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if data, err = skill.SetField(data, ForkedFromField, from); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
