package fsutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// WithLock runs fn while holding the file lock on <QuiverHome>/.lock.
// Every write to the Quiver home goes through it, because several MCP server
// processes and the CLI can run at the same time.
func WithLock(fn func() error) error {
	home, err := QuiverHome()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return err
	}
	lock := flock.New(filepath.Join(home, ".lock"))
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("lock %s: %w", lock.Path(), err)
	}
	defer func() { _ = lock.Unlock() }()
	return fn()
}
