package source

import (
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
	dirs, err := walkSkills(repoDir)
	if err != nil {
		return nil, err
	}
	for _, dir := range dirs {
		d.Skills = append(d.Skills, Found{ID: skillName(repoDir, dir), Dir: dir})
	}
	sort.Slice(d.Skills, func(i, j int) bool { return d.Skills[i].ID < d.Skills[j].ID })
	return d, nil
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
