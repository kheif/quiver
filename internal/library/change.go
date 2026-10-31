package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/skill"
)

// ErrNothingToChange is returned by UpdateLocal when no change is given.
var ErrNothingToChange = errors.New("nothing to change: give a description, a body or files")

// UpdateLocal changes the local skill id: its description or body when not
// nil, and the extra files, which map slash paths inside the skill folder
// to their new content; an empty content deletes the file. Other
// frontmatter fields are kept. An invalid result is undone. The change is
// committed in local/ as "update <name> (via MCP)". It returns the
// validation warnings.
func UpdateLocal(id string, description, body *string, files map[string]string) (warnings []string, err error) {
	if description == nil && body == nil && len(files) == 0 {
		return nil, ErrNothingToChange
	}
	if err := checkFiles(files); err != nil {
		return nil, err
	}
	local, err := LocalDir()
	if err != nil {
		return nil, err
	}
	if !filepath.IsLocal(id) || strings.ContainsAny(id, `/\`) {
		return nil, fmt.Errorf("no local skill %q", id)
	}
	dir := filepath.Join(local, id)
	err = fsutil.WithLock(func() error {
		if info, err := os.Stat(filepath.Join(dir, skill.FileName)); err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("no local skill %q", id)
		}
		changes := map[string][]byte{}
		for p, content := range files {
			if content == "" {
				if _, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(p))); err != nil {
					return fmt.Errorf("cannot delete %s: it is not in skill %s", p, id)
				}
				changes[p] = nil
			} else {
				changes[p] = []byte(content)
			}
		}
		if description != nil || body != nil {
			old, err := os.ReadFile(filepath.Join(dir, skill.FileName))
			if err != nil {
				return err
			}
			if changes[skill.FileName], err = skill.Rewrite(old, description, body); err != nil {
				return fmt.Errorf("%s: %w", skill.FileName, err)
			}
		}
		undo, err := apply(dir, changes)
		if err != nil {
			return errors.Join(err, undo())
		}
		s, err := skill.Load(dir)
		if err == nil {
			warnings, err = skill.Validate(s)
		}
		if err != nil {
			return errors.Join(fmt.Errorf("the change was undone: %w", err), undo())
		}
		if err := git.Commit(local, "update "+s.Name+" (via MCP)", id); err != nil {
			return fmt.Errorf("skill %s was changed, but the commit in %s failed: %w", id, local, err)
		}
		return nil
	})
	return warnings, err
}

// apply writes changes into dir (nil content deletes the file) and returns
// a function that puts back what was there before.
func apply(dir string, changes map[string][]byte) (undo func() error, err error) {
	type saved struct {
		data []byte
		perm os.FileMode
		ok   bool
	}
	before := map[string]saved{}
	var created []string
	undo = func() error {
		var errs []error
		for path, s := range before {
			if s.ok {
				errs = append(errs, fsutil.WriteFileAtomic(path, s.data, s.perm))
			} else if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				errs = append(errs, err)
			}
		}
		for i := len(created) - 1; i >= 0; i-- {
			_ = os.Remove(created[i]) // only empty folders go
		}
		return errors.Join(errs...)
	}
	for p, data := range changes {
		path := filepath.Join(dir, filepath.FromSlash(p))
		s := saved{perm: 0o644}
		if info, err := os.Lstat(path); err == nil {
			if !info.Mode().IsRegular() {
				return undo, fmt.Errorf("%s is not a regular file", p)
			}
			if s.data, err = os.ReadFile(path); err != nil {
				return undo, err
			}
			s.perm, s.ok = info.Mode().Perm(), true
		}
		before[path] = s
		if data == nil {
			if err := os.Remove(path); err != nil {
				return undo, err
			}
			removeEmptyParents(dir, filepath.Dir(path))
			continue
		}
		for d := filepath.Dir(path); d != dir; d = filepath.Dir(d) {
			if _, err := os.Lstat(d); errors.Is(err, os.ErrNotExist) {
				created = append([]string{d}, created...)
			}
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return undo, err
		}
		if err := fsutil.WriteFileAtomic(path, data, s.perm); err != nil {
			return undo, err
		}
	}
	return undo, nil
}

// removeEmptyParents removes d and its parents up to root while they are
// empty.
func removeEmptyParents(root, d string) {
	for d != root && strings.HasPrefix(d, root) {
		if os.Remove(d) != nil {
			return
		}
		d = filepath.Dir(d)
	}
}
