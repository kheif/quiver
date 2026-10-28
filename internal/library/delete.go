package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/registry"
)

// SkillInUseError is returned by DeleteLocal when the skill is assigned and
// force is false.
type SkillInUseError struct {
	ID string
	Usage
}

func (e *SkillInUseError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "skill %s is assigned:", e.ID)
	if len(e.Global) > 0 {
		b.WriteString("\n  global")
	}
	for _, p := range e.Projects {
		fmt.Fprintf(&b, "\n  project %s", registry.ProjectLabel(p.Name, p.Paths))
	}
	b.WriteString("\nrun again with --force to unassign and delete it")
	return b.String()
}

// skillUsage lists where id is assigned.
func skillUsage(r *registry.Registry, id string) Usage {
	var u Usage
	for _, g := range r.Global {
		if g == id {
			u.Global = []string{id}
		}
	}
	for _, pid := range r.ProjectsUsing(id) {
		p := r.Projects[pid]
		u.Projects = append(u.Projects, ProjectUse{ID: pid, Name: p.Name, Paths: p.Paths, Skills: []string{id}})
	}
	sort.Slice(u.Projects, func(i, j int) bool { return u.Projects[i].Name < u.Projects[j].Name })
	return u
}

// DeleteLocal removes the local skill id from the library and commits the
// removal in local/ as "delete <id>". If the skill is assigned, it returns
// a *SkillInUseError, unless force is set: then it unassigns it everywhere
// and returns where it was assigned. Source skills cannot be deleted.
func DeleteLocal(id string, force bool) (Usage, error) {
	if id == "" || strings.HasPrefix(id, ".") || strings.Contains(id, `\`) {
		return Usage{}, fmt.Errorf("invalid skill ID %q", id)
	}
	if name, _, ok := strings.Cut(id, "/"); ok {
		return Usage{}, fmt.Errorf("%s is a skill of source %s and cannot be deleted on its own; quiver source remove %s removes the whole source", id, name, name)
	}
	dir, err := LocalDir()
	if err != nil {
		return Usage{}, err
	}
	target := filepath.Join(dir, id)
	var u Usage
	err = fsutil.WithLock(func() error {
		info, err := os.Lstat(target)
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("local skill %q not found; see quiver list --source local", id)
		} else if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a skill folder", target)
		}
		r, err := registry.Load()
		if err != nil {
			return err
		}
		u = skillUsage(r, id)
		if !u.Empty() && !force {
			return &SkillInUseError{ID: id, Usage: u}
		}
		r.UnassignGlobal(u.Global...)
		for _, p := range u.Projects {
			if _, err := r.Unassign(p.ID, id); err != nil {
				return err
			}
		}
		// A folder that was never committed has nothing to commit away.
		tracked := git.Tracked(dir, id)
		// Move the folder aside first, so a failed registry write can put it back.
		trash, err := os.MkdirTemp(dir, ".delete-"+id+"-")
		if err != nil {
			return err
		}
		if err := os.Rename(target, filepath.Join(trash, id)); err != nil {
			_ = os.RemoveAll(trash)
			return err
		}
		if !u.Empty() {
			if err := registry.Write(r); err != nil {
				return restore(trash, target, err)
			}
		}
		if err := os.RemoveAll(trash); err != nil {
			return err
		}
		if !tracked {
			return nil
		}
		if err := git.Commit(dir, "delete "+id, id); err != nil {
			return fmt.Errorf("skill %s was deleted, but the commit in %s failed: %w", id, dir, err)
		}
		return nil
	})
	return u, err
}
