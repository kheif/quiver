package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/registry"
)

// ProjectEnv names the folder to serve instead of the working directory.
const ProjectEnv = "QUIVER_PROJECT"

// Where is what the server serves. It is found once, at startup, because
// the instructions are built before the first request.
type Where struct {
	// Dir is $QUIVER_PROJECT or the working directory. When there is no
	// project yet, creating a skill registers this folder.
	Dir string
	// From is ProjectEnv or "working directory".
	From string
	// Project is nil when Dir is not inside a project. Then only global
	// skills are served.
	Project *project.Project
}

// ProjectID returns the served project's ID, or "" for none.
func (w Where) ProjectID() string {
	if w.Project == nil {
		return ""
	}
	return w.Project.ID
}

// Resolve finds the project to serve. It does not use MCP roots: Claude
// Code starts the server in the project folder, and roots cannot be asked
// for while a request is served (see docs/spike-b01-results.md). On error,
// Project is nil, so the server still serves the global skills.
func Resolve() (Where, error) {
	w := Where{From: ProjectEnv}
	dir := os.Getenv(ProjectEnv)
	if dir == "" {
		w.From = "working directory"
		var err error
		if dir, err = os.Getwd(); err != nil {
			return w, err
		}
	}
	dir, err := fsutil.ExpandHome(dir)
	if err != nil {
		return w, err
	}
	if w.Dir, err = filepath.Abs(dir); err != nil {
		return w, err
	}
	// Find would walk up from a missing folder to a parent project.
	info, err := os.Stat(w.Dir)
	if err != nil {
		return w, err
	}
	if !info.IsDir() {
		return w, fmt.Errorf("%s is not a folder", w.Dir)
	}
	p, err := project.Find(w.Dir)
	if errors.Is(err, project.ErrNotRegistered) {
		return w, nil
	}
	if err != nil {
		return w, err
	}
	w.Project = p
	return w, nil
}

// errUnchanged stops registry.Update from writing when nothing changed.
var errUnchanged = errors.New("unchanged")

// recordPath records the project's folder in the registry, so the path of
// a moved project is updated. It writes only when the paths change, and
// leaves out projects the registry does not know: they have no skills here.
func recordPath(p *project.Project) (*registry.PossibleCopy, error) {
	var copied *registry.PossibleCopy
	err := registry.Update(func(r *registry.Registry) error {
		rp, ok := r.Projects[p.ID]
		if !ok {
			return errUnchanged
		}
		before := slices.Clone(rp.Paths)
		copied = r.EnsureProject(p.ID, p.Root)
		if slices.Equal(before, rp.Paths) {
			return errUnchanged
		}
		return nil
	})
	if errors.Is(err, errUnchanged) {
		err = nil
	}
	return copied, err
}
