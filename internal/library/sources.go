package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/source"
)

// SourcesDir returns ~/.quiver/sources, where sources are cloned.
func SourcesDir() (string, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "sources"), nil
}

// Staged is a source cloned into a temporary folder. It is not in the
// config until Keep is called.
type Staged struct {
	Name      string
	URL       string
	Discovery *source.Discovery
	tmp       string
}

// StageSource clones url and runs discovery, without adding the source yet,
// so the caller can show the result first. Call Keep or Discard after.
func StageSource(name, url string) (*Staged, error) {
	if err := config.ValidateSourceName(name); err != nil {
		return nil, err
	}
	if err := checkNameFree(name); err != nil {
		return nil, err
	}
	dir, err := SourcesDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp(dir, ".add-"+name+"-")
	if err != nil {
		return nil, err
	}
	s := &Staged{Name: name, URL: url, tmp: tmp}
	// Clone into tmp/<name>, so a skill at the repo root is named after the
	// source.
	if err := git.Clone(url, s.repo()); err != nil {
		_ = s.Discard()
		return nil, err
	}
	if s.Discovery, err = source.Discover(s.repo()); err != nil {
		_ = s.Discard()
		return nil, err
	}
	return s, nil
}

func (s *Staged) repo() string { return filepath.Join(s.tmp, s.Name) }

// Discard deletes the clone.
func (s *Staged) Discard() error { return os.RemoveAll(s.tmp) }

// Keep moves the clone to sources/<name> and saves the source in the config.
func (s *Staged) Keep() error {
	dir, err := SourcesDir()
	if err != nil {
		return err
	}
	target := filepath.Join(dir, s.Name)
	moved := false
	err = config.Update(func(c *config.Config) error {
		if err := checkFree(c, target, s.Name); err != nil {
			return err
		}
		if err := os.Rename(s.repo(), target); err != nil {
			return err
		}
		moved = true
		c.Sources[s.Name] = config.Source{Repo: s.URL}
		return nil
	})
	if err != nil && moved {
		_ = os.Rename(target, s.repo())
	}
	if err != nil {
		return err
	}
	return s.Discard()
}

// checkNameFree fails early, before a slow clone, when the name is taken.
func checkNameFree(name string) error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	dir, err := SourcesDir()
	if err != nil {
		return err
	}
	return checkFree(c, filepath.Join(dir, name), name)
}

func checkFree(c *config.Config, target, name string) error {
	if _, ok := c.Sources[name]; ok {
		return fmt.Errorf("source %q already exists: pick another name with --name", name)
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("folder %s already exists: remove it or pick another name with --name", target)
	}
	return nil
}
