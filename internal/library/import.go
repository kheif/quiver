package library

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/skill"
)

// Found is a skill folder found by FindSkills.
type Found struct {
	Dir         string
	Name        string // from the frontmatter, or the folder name
	Description string
	Err         error // the folder could not be read as a skill
}

// skipDirs are never searched by FindSkills.
var skipDirs = []string{".git", "node_modules"}

// FindSkills returns the skill folders under root: each folder in
// root/.claude/skills with a SKILL.md, or with deep every folder with a
// SKILL.md. skip (the Quiver home) is not searched. The result is sorted by
// folder.
func FindSkills(root string, deep bool, skip string) ([]Found, error) {
	var dirs []string
	if deep {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// An unreadable folder is skipped, not fatal.
				if d != nil && d.IsDir() && path != root {
					return filepath.SkipDir
				}
				return err
			}
			if !d.IsDir() {
				return nil
			}
			if slices.Contains(skipDirs, d.Name()) || skip != "" && samePath(path, skip) {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, skill.FileName)); err == nil {
				dirs = append(dirs, path)
				// Skills do not nest: a SKILL.md lower down is a file of this skill.
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		matches, err := filepath.Glob(filepath.Join(root, ".claude", "skills", "*", skill.FileName))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			dirs = append(dirs, filepath.Dir(m))
		}
	}
	sort.Strings(dirs)
	out := make([]Found, 0, len(dirs))
	for _, dir := range dirs {
		out = append(out, ReadFound(dir, filepath.Base(dir)))
	}
	return out, nil
}

// ReadFound reads the skill folder dir. Its name is the one in the
// frontmatter, or fallback when there is none.
func ReadFound(dir, fallback string) Found {
	f := Found{Dir: dir, Name: fallback}
	s, err := skill.Load(dir)
	if err != nil {
		f.Err = err
		return f
	}
	if s.Name != "" {
		f.Name = s.Name
	}
	f.Description = s.Description
	return f
}

func samePath(a, b string) bool {
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	if err1 != nil || err2 != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return ra == rb
}

// SameAsLocal reports whether the local skill name has the same files and
// content as the folder src. Folders named .git are ignored.
func SameAsLocal(src, name string) (bool, error) {
	dir, err := LocalDir()
	if err != nil {
		return false, err
	}
	return sameTree(src, filepath.Join(dir, name))
}

func sameTree(a, b string) (bool, error) {
	fa, err := treeFiles(a)
	if err != nil {
		return false, err
	}
	fb, err := treeFiles(b)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if !slices.Equal(fa, fb) {
		return false, nil
	}
	for _, rel := range fa {
		da, err := os.ReadFile(filepath.Join(a, rel))
		if err != nil {
			return false, err
		}
		db, err := os.ReadFile(filepath.Join(b, rel))
		if err != nil {
			return false, err
		}
		if !bytes.Equal(da, db) {
			return false, nil
		}
	}
	return true, nil
}

// treeFiles lists the regular files and links under dir, relative and
// sorted, without .git folders.
func treeFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	sort.Strings(files)
	return files, err
}

// Imported is the result of ImportLocal.
type Imported struct {
	// Reused is set when the library already had the skill with the same
	// content, so it was only assigned.
	Reused   bool
	Warnings []string
}

// ImportLocal copies the skill folder src into local/<name>, assigns it and
// commits it in local/ as "import <name> from <from>". If name differs from
// the name in the frontmatter, SKILL.md gets the new name. If local/<name>
// already has the same content, it is only assigned. Any other skill named
// name is an ErrSkillExists. Symbolic links and .git folders are not copied.
// src is never changed.
func ImportLocal(src, name string, scope Scope, projectID, from string) (Imported, error) {
	var res Imported
	if err := skill.ValidateName(name); err != nil {
		return res, err
	}
	s, err := skill.Load(src)
	if err != nil {
		return res, err
	}
	dir, err := LocalDir()
	if err != nil {
		return res, err
	}
	target := filepath.Join(dir, name)
	err = fsutil.WithLock(func() error {
		r, err := registry.Load()
		if err != nil {
			return err
		}
		if err := assign(r, scope, projectID, name); err != nil {
			return err
		}
		if _, err := os.Lstat(target); err == nil {
			same, err := sameTree(src, target)
			if err != nil {
				return err
			}
			if !same || s.Name != name {
				return fmt.Errorf("%w: %s", ErrSkillExists, target)
			}
			res.Reused = true
			return registry.Write(r)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		tmp, err := os.MkdirTemp(dir, ".new-"+name+"-")
		if err != nil {
			return err
		}
		defer func() { _ = os.RemoveAll(tmp) }()
		if res.Warnings, err = copySkill(src, tmp, name, s.Name); err != nil {
			return err
		}
		if err := os.Rename(tmp, target); err != nil {
			return err
		}
		if scope != ScopeNone {
			if err := registry.Write(r); err != nil {
				return errors.Join(err, os.RemoveAll(target))
			}
		}
		if err := git.Commit(dir, "import "+name+" from "+from, name); err != nil {
			return fmt.Errorf("skill %s was imported, but the commit in %s failed: %w", name, dir, err)
		}
		return nil
	})
	return res, err
}

// copySkill copies src into dst, gives SKILL.md the name if it had another
// one, and validates the copy. It returns the validation warnings and a
// warning per skipped link.
func copySkill(src, dst, name, oldName string) (warnings []string, err error) {
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		to := filepath.Join(dst, rel)
		switch {
		case d.IsDir() && d.Name() == ".git":
			return filepath.SkipDir
		case d.IsDir():
			return os.MkdirAll(to, 0o755)
		case !d.Type().IsRegular():
			warnings = append(warnings, fmt.Sprintf("%s: not a regular file, not copied", path))
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if rel == skill.FileName && name != oldName {
			data = setName(data, name)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(to, data, info.Mode().Perm())
	})
	if err != nil {
		return nil, err
	}
	s, err := skill.Load(dst)
	if err != nil {
		return nil, err
	}
	// Validate against the final folder name, so no "folder differs" warning.
	s.Dir = filepath.Join(filepath.Dir(dst), name)
	more, err := skill.Validate(s)
	return append(warnings, more...), err
}

var nameLine = regexp.MustCompile(`(?m)^name:.*$`)

// setName replaces the name line in the frontmatter of a SKILL.md, or adds
// one at its top.
func setName(data []byte, name string) []byte {
	line := []byte("name: " + name)
	rest, ok := bytes.CutPrefix(data, []byte("---\n"))
	if !ok {
		return data
	}
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		return data
	}
	front := rest[:end+1]
	if loc := nameLine.FindIndex(front); loc != nil {
		front = slices.Concat(front[:loc[0]], line, front[loc[1]:])
	} else {
		front = slices.Concat(line, []byte("\n"), front)
	}
	return slices.Concat([]byte("---\n"), front, rest[end+1:])
}

// assign adds name to r for scope.
func assign(r *registry.Registry, scope Scope, projectID, name string) error {
	switch scope {
	case ScopeNone:
	case ScopeGlobal:
		r.AssignGlobal(name)
	case ScopeProject:
		return r.Assign(projectID, name)
	default:
		return fmt.Errorf("unknown scope %q", scope)
	}
	return nil
}
