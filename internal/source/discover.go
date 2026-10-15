package source

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kheif/quiver/internal/skill"
)

// Found is one skill folder inside a source repository.
type Found struct {
	// ID is unique inside the source, without the "<source>/" prefix.
	ID string
	// Dir is the skill folder relative to the repo root, slash-separated.
	Dir string
	// Plugin is the marketplace plugin that lists the skill, if any.
	Plugin string
}

// Discovery is the result of scanning one repository.
type Discovery struct {
	Skills []Found
	// Warnings lists things that were skipped. They never stop a scan.
	Warnings []string
}

// Discover returns the skill folders in repoDir, sorted by ID.
func Discover(repoDir string) (*Discovery, error) {
	d := &Discovery{}
	found, err := d.find(repoDir)
	if err != nil {
		return nil, err
	}
	for _, f := range found {
		f.ID = skillName(repoDir, f.Dir)
		d.Skills = append(d.Skills, f)
	}
	d.qualifyCollisions()
	sort.Slice(d.Skills, func(i, j int) bool { return d.Skills[i].ID < d.Skills[j].ID })
	return d, nil
}

// find uses the marketplace manifest when there is one, and otherwise walks
// the repository. A broken manifest falls back to the walk.
func (d *Discovery) find(repoDir string) ([]Found, error) {
	m, ok, err := readMarketplace(repoDir)
	if ok && err == nil {
		return m.skillDirs(repoDir, d), nil
	}
	if ok {
		d.warn("%v; searching the whole repository instead", err)
	} else if err != nil {
		return nil, err
	}
	dirs, err := walkSkills(repoDir)
	found := make([]Found, len(dirs))
	for i, dir := range dirs {
		found[i] = Found{Dir: dir}
	}
	return found, err
}

// qualifyCollisions gives skills that share a name the ID <plugin>/<name>,
// or their folder path when that is not unique either.
func (d *Discovery) qualifyCollisions() {
	var warnings []string
	byID := map[string][]int{}
	for i, f := range d.Skills {
		byID[f.ID] = append(byID[f.ID], i)
	}
	for name, idx := range byID {
		if len(idx) < 2 {
			continue
		}
		ids := map[string]int{}
		for _, i := range idx {
			if f := d.Skills[i]; f.Plugin != "" {
				ids[f.Plugin+"/"+f.ID]++
			}
		}
		var used []string
		for _, i := range idx {
			f := &d.Skills[i]
			if id := f.Plugin + "/" + f.ID; f.Plugin != "" && ids[id] == 1 {
				f.ID = id
			} else {
				f.ID = f.Dir
			}
			used = append(used, f.ID)
		}
		sort.Strings(used)
		warnings = append(warnings, fmt.Sprintf("several skills are named %q; using IDs %s", name, strings.Join(used, ", ")))
	}
	sort.Strings(warnings)
	d.Warnings = append(d.Warnings, warnings...)
}

func (d *Discovery) warn(format string, args ...any) {
	d.Warnings = append(d.Warnings, fmt.Sprintf(format, args...))
}

// walkSkills returns every folder under root that holds a SKILL.md, relative
// to root. It skips hidden folders and node_modules, and does not look inside
// a skill once it is found. Symlinks are not followed.
func walkSkills(root string) ([]string, error) {
	var dirs []string
	err := filepath.WalkDir(root, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !e.IsDir() {
			return nil
		}
		if p != root && (strings.HasPrefix(e.Name(), ".") || e.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if !isFile(filepath.Join(p, skill.FileName)) {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		dirs = append(dirs, filepath.ToSlash(rel))
		return filepath.SkipDir
	})
	return dirs, err
}

// skillName is the folder name of a skill. A skill at the repo root is named
// after the repo folder, which is the source name.
func skillName(repoDir, dir string) string {
	if dir == "." {
		abs, err := filepath.Abs(repoDir)
		if err == nil {
			return filepath.Base(abs)
		}
	}
	return path.Base(dir)
}

func isFile(p string) bool {
	info, err := os.Lstat(p)
	return err == nil && info.Mode().IsRegular()
}
