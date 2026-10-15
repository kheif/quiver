package registry

import (
	"fmt"
	"path/filepath"
	"strings"

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
// Old paths that still hold it are kept, and a PossibleCopy is returned.
func (r *Registry) EnsureProject(id, root string) *PossibleCopy {
	root = filepath.Clean(root)
	p, ok := r.Projects[id]
	if !ok {
		r.Projects[id] = &Project{Name: filepath.Base(root), Paths: []string{root}, Skills: []string{}}
		return nil
	}
	paths := []string{root}
	for _, old := range p.Paths {
		if samePath(old, root) {
			continue
		}
		if found, err := project.Find(old); err == nil && found.ID == id && samePath(found.Root, old) {
			paths = append(paths, old)
		}
	}
	p.Paths = sortedSet(paths)
	if len(p.Paths) > 1 {
		return &PossibleCopy{ID: id, Paths: p.Paths}
	}
	return nil
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
