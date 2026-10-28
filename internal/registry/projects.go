package registry

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/project"
)

// PossibleCopy warns that one project ID lives in more than one folder,
// probably because the folder was copied. `quiver projects split` fixes it.
type PossibleCopy struct {
	ID    string
	Paths []string
}

func (w *PossibleCopy) Error() string {
	return fmt.Sprintf("project %s is in more than one folder (%s); if one is a copy, run `quiver projects split` inside it",
		w.ID, strings.Join(w.Paths, ", "))
}

// EnsureProject adds the project, or records root as its current folder.
// Old paths that no longer hold the same ID are dropped (the folder moved).
// Old paths that still hold it are kept. If one of them is not a git
// worktree of the same repository as root, a PossibleCopy is returned:
// worktrees share the committed .quiver.json, so they are one project.
func (r *Registry) EnsureProject(id, root string) *PossibleCopy {
	root = filepath.Clean(root)
	p, ok := r.Projects[id]
	if !ok {
		r.Projects[id] = &Project{Name: filepath.Base(root), Paths: []string{root}, Skills: []string{}}
		return nil
	}
	paths := []string{root}
	copied := false
	for _, old := range p.Paths {
		if samePath(old, root) {
			continue
		}
		if found, err := project.Find(old); err == nil && found.ID == id && samePath(found.Root, old) {
			paths = append(paths, old)
			copied = copied || !SameRepository(old, root)
		}
	}
	p.Paths = sortedSet(paths)
	if copied {
		return &PossibleCopy{ID: id, Paths: p.Paths}
	}
	return nil
}

// SameRepository reports whether a and b are worktrees of one git
// repository. Folders outside git are never the same repository.
func SameRepository(a, b string) bool {
	ca, err := git.CommonDir(a)
	if err != nil {
		return false
	}
	cb, err := git.CommonDir(b)
	return err == nil && ca == cb
}

// RenameProject changes the project's display name.
func (r *Registry) RenameProject(id, name string) error {
	p, ok := r.Projects[id]
	if !ok {
		return fmt.Errorf("unknown project %q", id)
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("project name must not be empty")
	}
	p.Name = name
	return nil
}

// ForgetProject removes the project and its assignments. Its .quiver.json
// is left alone.
func (r *Registry) ForgetProject(id string) error {
	if _, ok := r.Projects[id]; !ok {
		return fmt.Errorf("unknown project %q", id)
	}
	delete(r.Projects, id)
	return nil
}

// samePath reports whether a and b are the same folder, resolving symlinks
// (on macOS /var is a link to /private/var).
func samePath(a, b string) bool {
	return resolve(a) == resolve(b)
}

func resolve(p string) string {
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return filepath.Clean(p)
}

// RegisterProject finds the project that dir is in, or registers dir as a
// new project, and records it in the registry. Both happen under the
// registry lock, so two processes cannot write different IDs into the same
// new folder. created reports a new .quiver.json.
func RegisterProject(dir string) (p *project.Project, created bool, copied *PossibleCopy, err error) {
	err = Update(func(r *Registry) error {
		var err error
		p, err = project.Find(dir)
		if errors.Is(err, project.ErrNotRegistered) {
			created = true
			p, err = project.Register(dir)
		}
		if err != nil {
			return err
		}
		copied = r.EnsureProject(p.ID, p.Root)
		return nil
	})
	return p, created, copied, err
}

// ProjectLabel is how commands name a project: "<name> (<paths>)".
func ProjectLabel(name string, paths []string) string {
	return fmt.Sprintf("%s (%s)", name, strings.Join(paths, ", "))
}

// Lookup returns the ID of the project with ID or name ref. An ID wins
// over a name; a name shared by several projects is an error.
func (r *Registry) Lookup(ref string) (string, error) {
	if _, ok := r.Projects[ref]; ok {
		return ref, nil
	}
	var ids []string
	for id, p := range r.Projects {
		if p.Name == ref {
			ids = append(ids, id)
		}
	}
	switch len(ids) {
	case 0:
		return "", fmt.Errorf("unknown project %q; see quiver projects", ref)
	case 1:
		return ids[0], nil
	}
	sort.Strings(ids)
	return "", fmt.Errorf("several projects are named %q (%s); use the ID", ref, strings.Join(ids, ", "))
}

// ErrNotCopied is returned by SplitProject when no other folder has the ID.
var ErrNotCopied = errors.New("no other folder has this project's ID, so there is nothing to split")

// ErrWorktree is returned by SplitProject in a git worktree whose other
// worktrees have the same ID. They share the committed .quiver.json, so a
// split would change it for every branch.
var ErrWorktree = errors.New("the other folders with this project's ID are worktrees of the same git repository; they are one project and share its skills, so there is nothing to split")

// SplitProject gives the project that dir is in a new ID, for a folder that
// was copied with its .quiver.json. The new project starts with the old
// one's skills and always-on marks; after that the two are independent.
func SplitProject(dir string) (oldID string, p *project.Project, err error) {
	err = fsutil.WithLock(func() error {
		r, err := Load()
		if err != nil {
			return err
		}
		p, err = project.Find(dir)
		if err != nil {
			return err
		}
		oldID = p.ID
		old, ok := r.Projects[oldID]
		if !ok {
			return fmt.Errorf("project %s is not in the registry; nothing to split", oldID)
		}
		if r.EnsureProject(oldID, p.Root) == nil {
			for _, other := range old.Paths {
				if !samePath(other, p.Root) && SameRepository(other, p.Root) {
					return ErrWorktree
				}
			}
			return ErrNotCopied
		}
		newID := project.NewID(filepath.Base(p.Root))
		r.Projects[newID] = &Project{
			Name:     filepath.Base(p.Root),
			Paths:    []string{p.Root},
			Skills:   slices.Clone(old.Skills),
			AlwaysOn: slices.Clone(old.AlwaysOn),
		}
		old.Paths = slices.DeleteFunc(old.Paths, func(path string) bool { return samePath(path, p.Root) })
		if err := project.SetID(p.Root, newID); err != nil {
			return err
		}
		if err := Write(r); err != nil {
			// Put the old ID back, so the folder matches the registry.
			return errors.Join(err, project.SetID(p.Root, oldID))
		}
		p.ID = newID
		return nil
	})
	return oldID, p, err
}
