package skill

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Files lists the skill's extra files as slash-separated paths relative to
// its folder, sorted. SKILL.md, hidden files and hidden folders (such as
// .git) are skipped.
func Files(s *Skill) ([]string, error) {
	var files []string
	err := filepath.WalkDir(s.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == s.Dir {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(s.Dir, path)
		if err != nil {
			return err
		}
		if rel == FileName {
			return nil
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}
