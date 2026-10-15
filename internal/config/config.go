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
}

// Source is a remote git repository that skills are read from.
type Source struct {
	Repo string `json:"repo"`
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
	return fsutil.WithLock(func() error { return write(c) })
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
		return write(c)
	})
}

func write(c *Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	path, err := Path()
	if err != nil {
		return err
	}
	return fsutil.WriteJSONAtomic(path, c)
}
