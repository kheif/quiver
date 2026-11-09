package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/git"
)

// updateChecksFile holds, per source, when its remote was last asked and what
// it answered.
const updateChecksFile = "state.json"

type updateCheck struct {
	At     time.Time `json:"at"`
	Remote string    `json:"remote"`
}

type updateState struct {
	UpdateChecks map[string]updateCheck `json:"updateChecks"`
}

// HasUpdate reports whether the remote of a source has a commit that the
// clone does not have. It asks the remote only when the last answer is older
// than the configured interval; in between it repeats the last answer, so a
// notice does not vanish and come back. It downloads nothing and changes no
// source. A failed question is returned and not recorded.
func HasUpdate(name string, now time.Time) (bool, error) {
	dir, err := SourcesDir()
	if err != nil {
		return false, err
	}
	repo := filepath.Join(dir, name)
	if _, err := os.Stat(repo); err != nil {
		return false, fmt.Errorf("source %s is not downloaded", name)
	}
	cfg, err := config.Load()
	if err != nil {
		return false, err
	}
	local, err := git.Head(repo)
	if err != nil {
		return false, err
	}
	state, path, err := loadUpdateState()
	if err != nil {
		return false, err
	}
	last := state.UpdateChecks[name]
	if age := now.Sub(last.At); last.Remote != "" && age >= 0 && age < cfg.CheckInterval() {
		return last.Remote != local, nil
	}
	remote, err := git.RemoteHead(repo)
	if err != nil {
		return false, err
	}
	err = fsutil.WithLock(func() error {
		// Another panel may have written meanwhile: change only this source.
		state, _, err := loadUpdateState()
		if err != nil {
			return err
		}
		state.UpdateChecks[name] = updateCheck{At: now, Remote: remote}
		return fsutil.WriteJSONAtomic(path, state)
	})
	return remote != local, err
}

// loadUpdateState reads state.json. A missing or unreadable file gives an
// empty state, which only makes the next check ask the remote again.
func loadUpdateState() (*updateState, string, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return nil, "", err
	}
	path := filepath.Join(home, updateChecksFile)
	s := &updateState{}
	if err := fsutil.ReadJSON(path, s); err != nil && !errors.Is(err, fsutil.ErrNotExist) {
		s = &updateState{}
	}
	if s.UpdateChecks == nil {
		s.UpdateChecks = map[string]updateCheck{}
	}
	return s, path, nil
}
