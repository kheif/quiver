// Package fsutil holds file helpers shared by all packages: paths, atomic writes and locking.
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// QuiverHome returns $QUIVER_HOME if it is set, otherwise ~/.quiver.
func QuiverHome() (string, error) {
	if h := os.Getenv("QUIVER_HOME"); h != "" {
		return ExpandHome(h)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".quiver"), nil
}

// ExpandHome replaces a leading ~ in path with the user's home directory.
// Paths like ~other are returned unchanged.
func ExpandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, "~"+string(filepath.Separator)) {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, path[1:]), nil
}
