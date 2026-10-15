// Package registry reads and writes ~/.quiver/registry.json: which skills are
// global and which projects get which skills.
package registry

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/kheif/quiver/internal/fsutil"
)

// CurrentVersion is the newest registry.json version this binary understands.
const CurrentVersion = 1

// Registry is the content of registry.json.
type Registry struct {
	Version  int                 `json:"version"`
	Global   []string            `json:"global"`
	AlwaysOn []string            `json:"alwaysOn"`
	Projects map[string]*Project `json:"projects"`
}

// Project is one project's entry, keyed by the ID in its .quiver.json.
type Project struct {
	Name     string   `json:"name"`
	Paths    []string `json:"paths"`
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

// Load reads registry.json. A missing file gives the defaults.
func Load() (*Registry, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	r := Default()
	if err := fsutil.ReadJSON(path, r); err != nil {
		if errors.Is(err, fsutil.ErrNotExist) {
			return Default(), nil
		}
		return nil, err
	}
	if err := r.check(path); err != nil {
		return nil, err
	}
	r.normalize()
	return r, nil
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
func Write(r *Registry) error {
	path, err := Path()
	if err != nil {
		return err
	}
	r.normalize()
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
