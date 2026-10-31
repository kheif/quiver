// Package config reads and writes ~/.quiver/config.json.
package config

import (
	"errors"
	"path/filepath"

	"github.com/kheif/quiver/internal/fsutil"
)

// CurrentVersion is the newest config.json version this binary understands.
const CurrentVersion = 1

// Config is the content of config.json.
type Config struct {
	Version int               `json:"version"`
	Sources map[string]Source `json:"sources"`
	// Pins maps the IDs of pinned source skills, such as acme/pdf, to the
	// commit they are served from.
	Pins map[string]Pin `json:"pins,omitempty"`
}

// Pin keeps a source skill at one commit of its source.
type Pin struct {
	Commit string `json:"commit"`
	// Dir is the skill's folder in the repo at that commit.
	Dir string `json:"dir"`
}

// Source is a remote git repository that skills are read from.
type Source struct {
	Repo string `json:"repo"`
	// IDs maps each skill ID given out in the source, without the
	// "<source>/" prefix, to the skill's folder in the repo. An ID never
	// changes once given, so assignments survive an update that adds a
	// skill with the same name. IDs of skills removed upstream are kept.
	IDs map[string]string `json:"ids,omitempty"`
}

// Default returns the config used when config.json does not exist.
func Default() *Config {
	return &Config{Version: CurrentVersion, Sources: map[string]Source{}}
}

// Path returns the location of config.json.
func Path() (string, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "config.json"), nil
}

// Load reads and validates config.json. A missing file gives the defaults.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	c := Default()
	if err := fsutil.ReadJSON(path, c); err != nil {
		if errors.Is(err, fsutil.ErrNotExist) {
			return Default(), nil
		}
		return nil, err
	}
	if c.Sources == nil {
		c.Sources = map[string]Source{}
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Save writes c to config.json atomically, under the Quiver lock.
func Save(c *Config) error {
	return fsutil.WithLock(func() error { return Write(c) })
}

// Update loads the config, applies fn and saves the result, all under one
// lock, so concurrent updates are not lost.
func Update(fn func(*Config) error) error {
	return fsutil.WithLock(func() error {
		c, err := Load()
		if err != nil {
			return err
		}
		if err := fn(c); err != nil {
			return err
		}
		return Write(c)
	})
}

// Write validates and saves c without taking the lock. Call it only inside
// fsutil.WithLock, for example when one change touches several files.
func Write(c *Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	path, err := Path()
	if err != nil {
		return err
	}
	return fsutil.WriteJSONAtomic(path, c)
}
