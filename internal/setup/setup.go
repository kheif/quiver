// Package setup creates the Quiver home (~/.quiver) and fills in missing parts.
package setup

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/registry"
)

const localReadme = `# Quiver skills

Your own skills, managed by Quiver. Each folder with a SKILL.md is one skill.
`

// Init creates the Quiver home and everything in it that is missing. It never
// overwrites existing files. It returns the paths it created.
func Init() ([]string, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return nil, err
	}
	var created []string
	if _, err := os.Stat(home); errors.Is(err, os.ErrNotExist) {
		created = append(created, home)
	}
	err = fsutil.WithLock(func() error {
		for _, dir := range []string{"local", "sources"} {
			c, err := ensureDir(filepath.Join(home, dir))
			if err != nil {
				return err
			}
			created = append(created, c...)
		}
		files := map[string]any{"config.json": config.Default(), "registry.json": registry.Default()}
		for _, name := range []string{"config.json", "registry.json"} {
			c, err := ensureJSON(filepath.Join(home, name), files[name])
			if err != nil {
				return err
			}
			created = append(created, c...)
		}
		c, err := ensureLocalRepo(filepath.Join(home, "local"))
		created = append(created, c...)
		return err
	})
	return created, err
}

// EnsureInit runs Init if any part of the Quiver home is missing, for
// example when the home folder exists but is empty, or an earlier Init
// failed halfway.
func EnsureInit() error {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return err
	}
	for _, p := range []string{"config.json", "registry.json", "sources", filepath.Join("local", ".git")} {
		if _, err := os.Stat(filepath.Join(home, p)); err != nil {
			_, err = Init()
			return err
		}
	}
	return nil
}

func ensureDir(path string) ([]string, error) {
	if _, err := os.Stat(path); err == nil {
		return nil, nil
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, err
	}
	return []string{path}, nil
}

func ensureJSON(path string, v any) ([]string, error) {
	if _, err := os.Stat(path); err == nil {
		return nil, nil
	}
	if err := fsutil.WriteJSONAtomic(path, v); err != nil {
		return nil, err
	}
	return []string{path}, nil
}

// ensureLocalRepo makes local/ a git repository with at least one commit.
func ensureLocalRepo(dir string) ([]string, error) {
	var created []string
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		if err := git(dir, "init", "--quiet"); err != nil {
			return nil, err
		}
		created = append(created, filepath.Join(dir, ".git"))
	}
	if git(dir, "rev-parse", "--verify", "--quiet", "HEAD") == nil {
		return created, nil // already has a commit
	}
	readme := filepath.Join(dir, "README.md")
	if _, err := os.Stat(readme); err != nil {
		if err := os.WriteFile(readme, []byte(localReadme), 0o644); err != nil {
			return created, err
		}
		created = append(created, readme)
	}
	if err := git(dir, "add", "README.md"); err != nil {
		return created, err
	}
	return created, git(dir, "commit", "--quiet", "--no-verify", "-m", "Initialize Quiver skills")
}

// git runs a git command in dir. If the user has no git identity, commits
// are made as "Quiver".
func git(dir string, args ...string) error {
	var pre []string
	if out, _ := exec.Command("git", "config", "user.email").Output(); len(bytes.TrimSpace(out)) == 0 {
		pre = []string{"-c", "user.name=Quiver", "-c", "user.email=quiver@localhost"}
	}
	cmd := exec.Command("git", append(append(pre, "-c", "commit.gpgsign=false"), args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %v: %w: %s", args, err, bytes.TrimSpace(stderr.Bytes()))
	}
	return nil
}
