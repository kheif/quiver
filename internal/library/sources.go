package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/registry"
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
		c.Sources[s.Name] = config.Source{Repo: s.URL, IDs: s.Discovery.Stabilize(nil)}
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

// SourceInfo describes one configured source for "quiver source list".
type SourceInfo struct {
	Name   string
	Repo   string
	Skills int
	// Commit is the short HEAD hash and Updated its commit time. Both are
	// empty when the folder is missing or not a git repository.
	Commit  string
	Updated time.Time
}

// Sources returns every configured source, sorted by name. Skills counts
// the valid skills in l.
func (l *Library) Sources() ([]SourceInfo, error) {
	c, err := config.Load()
	if err != nil {
		return nil, err
	}
	dir, err := SourcesDir()
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, s := range l.skills {
		counts[s.Source]++
	}
	out := make([]SourceInfo, 0, len(c.Sources))
	for name, src := range c.Sources {
		info := SourceInfo{Name: name, Repo: src.Repo, Skills: counts[name]}
		repo := filepath.Join(dir, name)
		if head, err := git.Head(repo); err == nil {
			info.Commit = head[:min(7, len(head))]
			info.Updated, _ = git.HeadTime(repo)
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Usage lists where the skills of one source are assigned.
type Usage struct {
	Global   []string
	Projects []ProjectUse
}

// ProjectUse is the source's skills assigned to one project.
type ProjectUse struct {
	ID     string
	Name   string
	Paths  []string
	Skills []string
}

// Empty reports whether no skill of the source is assigned.
func (u Usage) Empty() bool { return len(u.Global) == 0 && len(u.Projects) == 0 }

// InUseError is returned by RemoveSource when skills of the source are
// assigned and force is false.
type InUseError struct {
	Source string
	Usage
}

func (e *InUseError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "skills from source %q are assigned:", e.Source)
	if len(e.Global) > 0 {
		fmt.Fprintf(&b, "\n  global: %s", strings.Join(e.Global, ", "))
	}
	for _, p := range e.Projects {
		fmt.Fprintf(&b, "\n  project %s: %s", registry.ProjectLabel(p.Name, p.Paths), strings.Join(p.Skills, ", "))
	}
	b.WriteString("\nrun again with --force to unassign them")
	return b.String()
}

// usage collects the assigned IDs that start with "<name>/". IDs that no
// longer exist in the source count too.
func usage(r *registry.Registry, name string) Usage {
	prefix := name + "/"
	from := func(ids []string) []string {
		var out []string
		for _, id := range ids {
			if strings.HasPrefix(id, prefix) {
				out = append(out, id)
			}
		}
		return out
	}
	u := Usage{Global: from(r.Global)}
	for id, p := range r.Projects {
		if skills := from(p.Skills); len(skills) > 0 {
			u.Projects = append(u.Projects, ProjectUse{ID: id, Name: p.Name, Paths: p.Paths, Skills: skills})
		}
	}
	sort.Slice(u.Projects, func(i, j int) bool { return u.Projects[i].Name < u.Projects[j].Name })
	return u
}

// RemoveSource deletes a source and its folder. If its skills are assigned,
// it returns an *InUseError, unless force is set: then it unassigns them and
// returns what it unassigned.
func RemoveSource(name string, force bool) (Usage, error) {
	dir, err := SourcesDir()
	if err != nil {
		return Usage{}, err
	}
	var u Usage
	var trash string
	err = fsutil.WithLock(func() error {
		c, err := config.Load()
		if err != nil {
			return err
		}
		if _, ok := c.Sources[name]; !ok {
			return fmt.Errorf("source %q not found", name)
		}
		r, err := registry.Load()
		if err != nil {
			return err
		}
		u = usage(r, name)
		if !u.Empty() && !force {
			return &InUseError{Source: name, Usage: u}
		}
		r.UnassignGlobal(u.Global...)
		for _, p := range u.Projects {
			if _, err := r.Unassign(p.ID, p.Skills...); err != nil {
				return err
			}
		}
		// Move the folder aside first, so a failed write can put it back.
		target := filepath.Join(dir, name)
		if _, err := os.Stat(target); err == nil {
			if trash, err = os.MkdirTemp(dir, ".remove-"+name+"-"); err != nil {
				return err
			}
			if err := os.Rename(target, filepath.Join(trash, name)); err != nil {
				return err
			}
		}
		if !u.Empty() {
			if err := registry.Write(r); err != nil {
				return restore(trash, target, err)
			}
		}
		delete(c.Sources, name)
		for _, id := range pinIDs(c, name) {
			delete(c.Pins, id)
		}
		if err := config.Write(c); err != nil {
			return restore(trash, target, err)
		}
		return nil
	})
	if err != nil {
		return u, err
	}
	// The worktrees of the source's pins belong to the deleted clone.
	pins, _ := filepath.Glob(filepath.Join(dir, PinsFolder, name+"@*"))
	errs := []error{}
	for _, p := range pins {
		errs = append(errs, os.RemoveAll(p))
	}
	if trash != "" {
		errs = append(errs, os.RemoveAll(trash))
	}
	return u, errors.Join(errs...)
}

// restore moves a source folder back from trash after a failed removal.
func restore(trash, target string, cause error) error {
	if trash != "" {
		_ = os.Rename(filepath.Join(trash, filepath.Base(target)), target)
		_ = os.RemoveAll(trash)
	}
	return cause
}
