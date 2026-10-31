// Package registry reads and writes ~/.quiver/registry.json: which skills are
// global and which projects get which skills. The folders where each project
// was seen differ between machines, so they are kept apart, in
// ~/.quiver/state/projects.json, and registry.json can be synced as it is.
package registry

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/kheif/quiver/internal/fsutil"
)

// CurrentVersion is the newest registry.json version this binary understands.
// Version 1 kept each project's paths in registry.json.
const CurrentVersion = 2

// StateFile holds each project's paths on this machine, relative to the
// Quiver home.
const StateFile = "state/projects.json"

// stateVersion is the newest StateFile version this binary understands.
const stateVersion = 1

// Registry is the content of registry.json.
type Registry struct {
	Version  int                 `json:"version"`
	Global   []string            `json:"global"`
	AlwaysOn []string            `json:"alwaysOn"`
	Projects map[string]*Project `json:"projects"`
}

// Project is one project's entry, keyed by the ID in its .quiver.json.
type Project struct {
	Name string `json:"name"`
	// Paths lists the folders where the project was seen on this machine.
	// It is stored in StateFile, not in registry.json.
	Paths    []string `json:"-"`
	Skills   []string `json:"skills"`
	AlwaysOn []string `json:"alwaysOn,omitempty"`
}

// Default returns the registry used when registry.json does not exist.
func Default() *Registry {
	return &Registry{
		Version:  CurrentVersion,
		Global:   []string{},
		AlwaysOn: []string{},
		Projects: map[string]*Project{},
	}
}

// Path returns the location of registry.json.
func Path() (string, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "registry.json"), nil
}

// state is the content of StateFile.
type state struct {
	Version int                 `json:"version"`
	Paths   map[string][]string `json:"paths"`
}

// v1Paths reads the paths a version 1 registry.json kept per project.
type v1Paths struct {
	Projects map[string]*struct {
		Paths []string `json:"paths"`
	} `json:"projects"`
}

// StatePath returns the location of StateFile.
func StatePath() (string, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, StateFile), nil
}

// Load reads registry.json and the paths in StateFile. A missing file gives
// the defaults. A version 1 registry.json still holds the paths; they are
// used, and move to StateFile on the next write.
func Load() (*Registry, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	r := Default()
	if err := fsutil.ReadJSON(path, r); err != nil {
		if !errors.Is(err, fsutil.ErrNotExist) {
			return nil, err
		}
		r = Default()
	} else if err := r.check(path); err != nil {
		return nil, err
	}
	paths, err := loadPaths()
	if err != nil {
		return nil, err
	}
	if r.Version == 1 {
		var old v1Paths
		if err := fsutil.ReadJSON(path, &old); err != nil {
			return nil, err
		}
		for id, p := range old.Projects {
			if p != nil {
				paths[id] = append(paths[id], p.Paths...)
			}
		}
	}
	for id, p := range r.Projects {
		p.Paths = paths[id]
	}
	r.normalize()
	return r, nil
}

func loadPaths() (map[string][]string, error) {
	path, err := StatePath()
	if err != nil {
		return nil, err
	}
	var s state
	if err := fsutil.ReadJSON(path, &s); err != nil {
		if errors.Is(err, fsutil.ErrNotExist) {
			return map[string][]string{}, nil
		}
		return nil, err
	}
	if s.Version > stateVersion {
		return nil, fmt.Errorf("%s has version %d, but this quiver supports only up to %d: update quiver", path, s.Version, stateVersion)
	}
	if s.Paths == nil {
		s.Paths = map[string][]string{}
	}
	return s.Paths, nil
}

// Update loads the registry, applies fn and saves the result, all under one
// lock, so concurrent changes are not lost. If fn fails nothing is written.
// fn must not call another locking function (WithLock is not re-entrant).
func Update(fn func(*Registry) error) error {
	return fsutil.WithLock(func() error {
		r, err := Load()
		if err != nil {
			return err
		}
		if err := fn(r); err != nil {
			return err
		}
		return Write(r)
	})
}

// Write saves r without taking the lock. Call it only inside
// fsutil.WithLock, for example when one change touches several files.
// The paths go to StateFile first, so a version 2 registry.json never
// exists without them.
func Write(r *Registry) error {
	path, err := Path()
	if err != nil {
		return err
	}
	statePath, err := StatePath()
	if err != nil {
		return err
	}
	r.normalize()
	s := state{Version: stateVersion, Paths: map[string][]string{}}
	for id, p := range r.Projects {
		if len(p.Paths) > 0 {
			s.Paths[id] = p.Paths
		}
	}
	if err := fsutil.WriteJSONAtomic(statePath, s); err != nil {
		return err
	}
	r.Version = CurrentVersion
	return fsutil.WriteJSONAtomic(path, r)
}

func (r *Registry) check(path string) error {
	if r.Version > CurrentVersion {
		return fmt.Errorf("%s has version %d, but this quiver supports only up to %d: update quiver", path, r.Version, CurrentVersion)
	}
	if r.Version < 1 {
		return fmt.Errorf("%s has invalid version %d", path, r.Version)
	}
	for id, p := range r.Projects {
		if p == nil {
			return fmt.Errorf("%s: project %q is null", path, id)
		}
	}
	return nil
}

// normalize replaces nil lists with empty ones and sorts every list, so the
// file is stable and diffs stay small.
func (r *Registry) normalize() {
	r.Global = sortedSet(r.Global)
	r.AlwaysOn = sortedSet(r.AlwaysOn)
	if r.Projects == nil {
		r.Projects = map[string]*Project{}
	}
	for _, p := range r.Projects {
		p.Paths = sortedSet(p.Paths)
		p.Skills = sortedSet(p.Skills)
		if len(p.AlwaysOn) > 0 {
			p.AlwaysOn = sortedSet(p.AlwaysOn)
		}
	}
}
