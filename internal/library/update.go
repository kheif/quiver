package library

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/source"
)

// Update is what "quiver update" changes in one source. FetchSource fills
// it without changing the source; Apply then moves the source to To. Skill
// IDs carry the "<source>/" prefix.
type Update struct {
	Source string
	// From and To are HEAD before and after the update. They are equal when
	// the source is up to date.
	From, To string
	// Reset is set when a rewritten history replaces the local one.
	Reset   bool
	Added   []string
	Changed []string
	Removed []string
	// Pinned lists pinned skills that changed or were removed upstream.
	// They stay at their pinned commit, so they are not in Changed or
	// Removed.
	Pinned []string
	// Files lists the changed files of each added or changed skill, with
	// paths relative to the skill folder.
	Files map[string][]git.FileChange
	// Dirs maps each added, changed or removed skill to its folder in the
	// repo, as Diff takes it.
	Dirs map[string]string
	// PinErrors lists pins whose commit could not be checked out.
	PinErrors []string

	repo  string
	saved map[string]string // the IDs in the config
	ids   map[string]string // the IDs after the update
}

// ErrDiverged is returned by FetchSource when the remote history was
// rewritten (for example force-pushed), so it cannot be fast-forwarded.
var ErrDiverged = errors.New("the remote history was rewritten, so it cannot be fast-forwarded")

// UpdateSource is FetchSource followed by Apply.
func UpdateSource(name string, reset bool) (*Update, error) {
	u, err := FetchSource(name, reset)
	if err != nil {
		return nil, err
	}
	return u, u.Apply()
}

// FetchSource downloads the new commits of a source without applying them,
// and compares its skills now and at the remote head. If the remote history
// was rewritten, it returns ErrDiverged, unless reset is set: then the
// remote history is taken.
func FetchSource(name string, reset bool) (*Update, error) {
	dir, err := SourcesDir()
	if err != nil {
		return nil, err
	}
	repo := filepath.Join(dir, name)
	if _, err := os.Stat(repo); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("source %s is not downloaded; add it again", name)
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	u := &Update{Source: name, repo: repo, saved: cfg.Sources[name].IDs}
	before, err := discover(repo)
	if err != nil {
		return nil, err
	}
	// A source added before IDs were saved starts from the current ones.
	u.ids = before.Stabilize(u.saved)
	if u.From, err = git.Head(repo); err != nil {
		return nil, err
	}
	if err := git.Fetch(repo); err != nil {
		return nil, err
	}
	if u.To, err = git.ResolveCommit(repo, "@{u}"); err != nil {
		return nil, err
	}
	if u.From == u.To {
		return u, nil
	}
	if ok, err := git.IsAncestor(repo, u.From, u.To); err != nil {
		return nil, err
	} else if !ok {
		if !reset {
			return nil, ErrDiverged
		}
		u.Reset = true
	}
	after, err := discoverAt(repo, dir, name, u.To)
	if err != nil {
		return nil, err
	}
	u.ids = after.Stabilize(u.ids)
	paths, err := git.DiffNames(repo, u.From, u.To)
	if err != nil {
		return nil, err
	}
	u.compare(before, after, paths)
	u.keepPinned(cfg)
	return u, u.diffFiles()
}

// discoverAt runs discovery on commit of the source, checked out into a
// temporary worktree named after the source, so a skill at the repo root
// gets the same ID.
func discoverAt(repo, sources, name, commit string) (*source.Discovery, error) {
	tmp, err := os.MkdirTemp(sources, ".preview-"+name+"-")
	if err != nil {
		return nil, err
	}
	at := filepath.Join(tmp, name)
	defer func() {
		_ = git.RemoveWorktree(repo, at)
		_ = os.RemoveAll(tmp)
	}()
	if err := git.AddWorktree(repo, at, commit); err != nil {
		return nil, err
	}
	return scan(at)
}

// diffFiles fills Files for the added and changed skills.
func (u *Update) diffFiles() error {
	u.Files = map[string][]git.FileChange{}
	for _, id := range slices.Concat(u.Added, u.Changed) {
		dir := u.Dirs[id]
		files, err := git.DiffFiles(u.repo, u.From, u.To, dir)
		if err != nil {
			return err
		}
		for i := range files {
			if dir != "." {
				files[i].Path = strings.TrimPrefix(files[i].Path, dir+"/")
			}
		}
		u.Files[id] = files
	}
	return nil
}

// scriptExts are the file types NeedsReview flags, besides executable files.
var scriptExts = []string{".sh", ".bash", ".zsh", ".fish", ".ps1", ".bat", ".cmd", ".py", ".js", ".mjs", ".cjs", ".ts", ".rb", ".pl", ".php"}

// NeedsReview reports whether a changed file is a script or an executable,
// which an assistant may run, so the change should be read before it is
// applied.
func NeedsReview(f git.FileChange) bool {
	return f.Mode == "100755" || slices.Contains(scriptExts, strings.ToLower(filepath.Ext(f.Path)))
}

// Diff returns the patch of one added or changed skill.
func (u *Update) Diff(id string) (string, error) {
	return git.Diff(u.repo, u.From, u.To, u.Dirs[id])
}

// Apply moves the source to To: a fast-forward, or a reset when the remote
// history was rewritten. Then it saves the skill IDs and checks out pinned
// commits that are missing.
func (u *Update) Apply() error {
	if u.From != u.To {
		apply := git.FastForward
		if u.Reset {
			apply = git.ResetTo
		}
		if err := apply(u.repo, u.To); err != nil {
			return err
		}
	}
	return u.finish(u.Source, u.saved, u.ids)
}

// keepPinned moves pinned skills out of Changed and Removed into Pinned.
func (u *Update) keepPinned(cfg *config.Config) {
	pinned := func(id string) bool {
		if _, ok := cfg.Pins[id]; ok {
			u.Pinned = append(u.Pinned, id)
			return true
		}
		return false
	}
	u.Changed = slices.DeleteFunc(u.Changed, pinned)
	u.Removed = slices.DeleteFunc(u.Removed, pinned)
	slices.Sort(u.Pinned)
}

// finish saves the skill IDs and checks out pinned commits that are
// missing, such as pins in a config from another machine.
func (u *Update) finish(name string, saved, ids map[string]string) error {
	if err := saveIDs(name, saved, ids); err != nil {
		return err
	}
	var err error
	u.PinErrors, err = RestorePins(name)
	return err
}

// saveIDs records the source's skill IDs in the config when they changed.
func saveIDs(name string, saved, ids map[string]string) error {
	if maps.Equal(saved, ids) {
		return nil
	}
	return config.Update(func(c *config.Config) error {
		if src, ok := c.Sources[name]; ok {
			src.IDs = ids
			c.Sources[name] = src
		}
		return nil
	})
}

// compare sorts skills into added, changed and removed. A skill changed when
// a changed path is inside its folder, before or after the update.
func (u *Update) compare(before, after *source.Discovery, paths []string) {
	old := map[string]source.Found{}
	for _, f := range before.Skills {
		old[f.ID] = f
	}
	touched := func(dir string) bool {
		for _, p := range paths {
			if dir == "." || p == dir || strings.HasPrefix(p, dir+"/") {
				return true
			}
		}
		return false
	}
	u.Dirs = map[string]string{}
	for _, f := range after.Skills {
		id := u.Source + "/" + f.ID
		prev, ok := old[f.ID]
		delete(old, f.ID)
		switch {
		case !ok:
			u.Added = append(u.Added, id)
		case touched(f.Dir) || touched(prev.Dir):
			u.Changed = append(u.Changed, id)
		default:
			continue
		}
		u.Dirs[id] = f.Dir
	}
	for _, f := range before.Skills {
		if _, ok := old[f.ID]; ok {
			u.Removed = append(u.Removed, u.Source+"/"+f.ID)
			u.Dirs[u.Source+"/"+f.ID] = f.Dir
		}
	}
}
