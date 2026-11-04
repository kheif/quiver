package server

import (
	"errors"
	"fmt"
	"net/url"
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
	// Dir is $QUIVER_PROJECT or the working directory (for a hook, the
	// session's folder). When there is no project yet, creating a skill
	// registers this folder.
	Dir string
	// From is ProjectEnv, "working directory", or what ResolveIn was given.
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

// Resolve finds the project to serve at startup, from $QUIVER_PROJECT or the
// working directory. Clients that start the server in the project folder, such
// as Claude Code, are served from here. VS Code starts the server in the home
// folder and sends the workspace as an MCP root once the session is
// initialized; when $QUIVER_PROJECT is unset and the working directory is in
// no project, ResolveRoots then serves the project of the first root that is in
// one (docs/spike-hooks.md). On error, Project is nil, so the server still
// serves the global skills.
func Resolve() (Where, error) {
	if os.Getenv(ProjectEnv) != "" {
		return ResolveIn("", "")
	}
	dir, err := os.Getwd()
	if err != nil {
		return Where{From: "working directory"}, err
	}
	return ResolveIn(dir, "working directory")
}

// ResolveIn finds the project for a session in dir, as Resolve does for
// the working directory: $QUIVER_PROJECT wins when it is set. from says
// where dir came from, for Where.From.
func ResolveIn(dir, from string) (Where, error) {
	w := Where{From: ProjectEnv}
	if env := os.Getenv(ProjectEnv); env != "" {
		dir = env
	} else {
		w.From = from
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

// ResolveRoots returns the Where of the first root URI that is inside a
// project, for a client that starts the server outside the project and sends
// the workspace as a root (VS Code). Roots that are not file URIs, are missing,
// or are in no project are skipped. ok is false when no root is in a project,
// so the caller keeps serving the global skills. $QUIVER_PROJECT still wins and
// is applied by the caller before the roots are read.
func ResolveRoots(uris []string) (Where, bool) {
	for _, uri := range uris {
		dir, err := fileURIPath(uri)
		if err != nil {
			continue
		}
		w, err := ResolveIn(dir, "MCP root")
		if err == nil && w.Project != nil {
			return w, true
		}
	}
	return Where{}, false
}

// fileURIPath returns the local path of a file:// URI. A URI without the
// file scheme is an error.
func fileURIPath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	if u.Scheme != "file" {
		return "", fmt.Errorf("not a file URI: %q", uri)
	}
	p := u.Path
	// A Windows file URI is file:///C:/dir, so url.Path is /C:/dir; drop the
	// leading slash before the drive letter.
	if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
		p = p[1:]
	}
	return filepath.FromSlash(p), nil
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
