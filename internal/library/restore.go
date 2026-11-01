package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/git"
)

// CloneMissing clones every source in the config that is not downloaded,
// such as one added on another machine and synced, and checks out pinned
// commits that are missing. It returns the sources it cloned, and the
// sources and pins it could not restore, with the reason.
func CloneMissing() (cloned, failed []string, err error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	dir, err := SourcesDir()
	if err != nil {
		return nil, nil, err
	}
	names := make([]string, 0, len(cfg.Sources))
	for name := range cfg.Sources {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		target := filepath.Join(dir, name)
		if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
			if err := cloneInto(cfg.Sources[name].Repo, dir, name); err != nil {
				failed = append(failed, fmt.Sprintf("%s: %v", name, err))
				continue
			}
			cloned = append(cloned, name)
		}
		pins, err := RestorePins(name)
		if err != nil {
			return cloned, failed, err
		}
		failed = append(failed, pins...)
	}
	return cloned, failed, nil
}

// cloneInto clones url into dir/name through a temp folder, so an
// interrupted clone never looks like a downloaded source.
func cloneInto(url, dir, name string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(dir, ".add-"+name+"-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	repo := filepath.Join(tmp, name)
	if err := git.Clone(url, repo); err != nil {
		return err
	}
	target := filepath.Join(dir, name)
	return fsutil.WithLock(func() error {
		if _, err := os.Stat(target); err == nil {
			return nil // cloned meanwhile
		}
		return os.Rename(repo, target)
	})
}
