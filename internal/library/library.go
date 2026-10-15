// Package library collects every skill Quiver knows: the user's local skills
// and, later, skills from remote sources.
package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/skill"
)

// LocalSource is the Source of the user's own skills.
const LocalSource = "local"

// Library is a snapshot of all skills, keyed by skill ID.
type Library struct {
	skills map[string]*skill.Skill
	// Warnings lists skills that were skipped or look wrong. They never stop
	// a load.
	Warnings []string
}

// Load scans every source. Today that is only ~/.quiver/local.
func Load() (*Library, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return nil, err
	}
	l := &Library{skills: map[string]*skill.Skill{}}
	if err := l.addLocal(filepath.Join(home, "local")); err != nil {
		return nil, err
	}
	return l, nil
}

// All returns every skill, sorted by ID.
func (l *Library) All() []*skill.Skill {
	out := make([]*skill.Skill, 0, len(l.skills))
	for _, s := range l.skills {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// addLocal adds every dir/<folder>/SKILL.md. The ID is the folder name, so
// it is unique even when two skills share a name.
func (l *Library) addLocal(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		l.addSkill(filepath.Join(dir, e.Name()), LocalSource, e.Name())
	}
	return nil
}

// addSkill loads and validates one skill folder. Problems become warnings.
func (l *Library) addSkill(dir, source, id string) {
	s, err := skill.Load(dir)
	if err != nil {
		l.Warnings = append(l.Warnings, fmt.Sprintf("skipped %s: %v", id, err))
		return
	}
	warnings, err := skill.Validate(s)
	if err != nil {
		l.Warnings = append(l.Warnings, fmt.Sprintf("skipped %s: %v", id, err))
		return
	}
	l.Warnings = append(l.Warnings, warnings...)
	s.Source, s.ID = source, id
	l.skills[id] = s
}
