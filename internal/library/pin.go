package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/skill"
)

// PinsFolder holds the checked out commits of pinned skills, inside
// sources/, as <source>@<commit>. Its name starts with a dot, so it is
// never taken for a source.
const PinsFolder = ".pins"

// PinDir returns the folder that holds commit of source for pins.
func PinDir(source, commit string) (string, error) {
	dir, err := SourcesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, PinsFolder, source+"@"+commit), nil
}

// Pinned is the result of PinSkill.
type Pinned struct {
	ID     string
	Commit string
	// Previous is the commit the skill was pinned to before, if any.
	Previous string
}

// PinSkill keeps the source skill id at commit rev of its source, HEAD
// when rev is empty, wherever it is assigned. The commit is checked out
// with git worktree into sources/.pins/<source>@<commit>; older commits
// are fetched first when the clone does not have them. "quiver update"
// leaves the skill alone until UnpinSkill.
func PinSkill(id, rev string) (Pinned, error) {
	l, err := Load()
	if err != nil {
		return Pinned{}, err
	}
	s, err := l.Get(id)
	if err != nil {
		return Pinned{}, err
	}
	if s.Source == LocalSource {
		return Pinned{}, fmt.Errorf("%s is a local skill; only skills from sources can be pinned", id)
	}
	cfg, err := config.Load()
	if err != nil {
		return Pinned{}, err
	}
	sources, err := SourcesDir()
	if err != nil {
		return Pinned{}, err
	}
	repo := filepath.Join(sources, s.Source)
	old := cfg.Pins[id]
	dir, err := skillFolder(cfg, repo, s, id)
	if err != nil {
		return Pinned{}, err
	}
	if rev == "" {
		rev = "HEAD"
	}
	commit, err := git.ResolveCommit(repo, rev)
	if err != nil {
		if ferr := git.FetchHistory(repo); ferr != nil {
			return Pinned{}, fmt.Errorf("commit %s is not in source %s, and fetching its history failed: %w", rev, s.Source, ferr)
		}
		if commit, err = git.ResolveCommit(repo, rev); err != nil {
			return Pinned{}, fmt.Errorf("commit %s is not in source %s", rev, s.Source)
		}
	}
	if err := checkOut(repo, s.Source, commit); err != nil {
		return Pinned{}, err
	}
	at, err := PinDir(s.Source, commit)
	if err != nil {
		return Pinned{}, err
	}
	if _, err := os.Stat(filepath.Join(at, filepath.FromSlash(dir), skill.FileName)); err != nil {
		return Pinned{}, errors.Join(
			fmt.Errorf("%s is not in %s at commit %s", dir, s.Source, short(commit)),
			RemoveUnusedPins(),
		)
	}
	err = config.Update(func(c *config.Config) error {
		if _, ok := c.Sources[s.Source]; !ok {
			return fmt.Errorf("source %q not found", s.Source)
		}
		if c.Pins == nil {
			c.Pins = map[string]config.Pin{}
		}
		c.Pins[id] = config.Pin{Commit: commit, Dir: dir}
		return nil
	})
	if err != nil {
		return Pinned{}, errors.Join(err, RemoveUnusedPins())
	}
	res := Pinned{ID: id, Commit: commit}
	if old.Commit != commit {
		res.Previous = old.Commit
	}
	return res, RemoveUnusedPins()
}

// UnpinSkill serves id from its live source again, and deletes the pin
// folder if no other pin uses it. It returns the commit it was pinned to.
func UnpinSkill(id string) (string, error) {
	var commit string
	err := config.Update(func(c *config.Config) error {
		p, ok := c.Pins[id]
		if !ok {
			return fmt.Errorf("%s is not pinned", id)
		}
		commit = p.Commit
		delete(c.Pins, id)
		return nil
	})
	if err != nil {
		return "", err
	}
	return commit, RemoveUnusedPins()
}

// skillFolder returns the folder of the source skill in its repo: the one
// it is pinned to, the one saved with its ID, or where it was loaded from.
func skillFolder(cfg *config.Config, repo string, s *skill.Skill, id string) (string, error) {
	if p, ok := cfg.Pins[id]; ok {
		return p.Dir, nil
	}
	if dir, ok := cfg.Sources[s.Source].IDs[strings.TrimPrefix(id, s.Source+"/")]; ok {
		return dir, nil
	}
	rel, err := filepath.Rel(repo, s.Dir)
	if err != nil || !filepath.IsLocal(rel) {
		return "", fmt.Errorf("cannot find the folder of %s in source %s", id, s.Source)
	}
	return filepath.ToSlash(rel), nil
}

// checkOut makes sure sources/.pins/<source>@<commit> holds commit.
func checkOut(repo, source, commit string) error {
	at, err := PinDir(source, commit)
	if err != nil {
		return err
	}
	if _, err := os.Stat(at); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
		return err
	}
	if _, err := git.ResolveCommit(repo, commit); err != nil {
		if err := git.FetchHistory(repo); err != nil {
			return fmt.Errorf("commit %s of source %s is not downloaded, and fetching it failed: %w", short(commit), source, err)
		}
	}
	return git.AddWorktree(repo, at, commit)
}

// RestorePins checks out the pinned commits of source whose folders are
// missing, for example after the config came from another machine. It
// returns the IDs it could not restore, with the reason.
func RestorePins(source string) ([]string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	sources, err := SourcesDir()
	if err != nil {
		return nil, err
	}
	var failed []string
	for _, id := range pinIDs(cfg, source) {
		if err := checkOut(filepath.Join(sources, source), source, cfg.Pins[id].Commit); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", id, err))
		}
	}
	return failed, nil
}

// pinIDs returns the pinned IDs of source, sorted; every pin when source
// is "".
func pinIDs(cfg *config.Config, source string) []string {
	var ids []string
	for id := range cfg.Pins {
		if source == "" || strings.HasPrefix(id, source+"/") {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// UnusedPins returns the folders in sources/.pins that no pin uses.
func UnusedPins() ([]string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	sources, err := SourcesDir()
	if err != nil {
		return nil, err
	}
	used := map[string]bool{}
	for id, p := range cfg.Pins {
		source, _, _ := strings.Cut(id, "/")
		used[source+"@"+p.Commit] = true
	}
	entries, err := os.ReadDir(filepath.Join(sources, PinsFolder))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !used[e.Name()] {
			out = append(out, filepath.Join(sources, PinsFolder, e.Name()))
		}
	}
	return out, nil
}

// RemoveUnusedPins deletes the folders UnusedPins returns.
func RemoveUnusedPins() error {
	unused, err := UnusedPins()
	if err != nil {
		return err
	}
	sources, err := SourcesDir()
	if err != nil {
		return err
	}
	var errs []error
	for _, at := range unused {
		source, _, _ := strings.Cut(filepath.Base(at), "@")
		repo := filepath.Join(sources, source)
		if _, err := os.Stat(repo); err != nil {
			errs = append(errs, os.RemoveAll(at))
			continue
		}
		errs = append(errs, git.RemoveWorktree(repo, at))
	}
	return errors.Join(errs...)
}

// short is the abbreviated commit hash Quiver prints.
func short(commit string) string {
	return commit[:min(7, len(commit))]
}
