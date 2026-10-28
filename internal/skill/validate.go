package skill

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Limits from the Agent Skills format. Breaking them is only a warning,
// so existing skills keep working.
const (
	maxNameLen        = 64
	maxDescriptionLen = 1024
)

// ValidateName checks the name of a skill about to be created. It is
// stricter than Validate: a name longer than the limit is an error here,
// because a new skill has no reason to break it.
func ValidateName(name string) error {
	if !nameRE.MatchString(name) {
		return fmt.Errorf("invalid name %q: use lowercase letters, digits and dashes, starting with a letter or digit", name)
	}
	if len(name) > maxNameLen {
		return fmt.Errorf("name %q is longer than %d characters", name, maxNameLen)
	}
	return nil
}

// Validate checks a loaded skill. Problems that make the skill unusable are
// returned as an error; soft issues are returned as warnings. Every message
// starts with the path of SKILL.md.
func Validate(s *Skill) (warnings []string, err error) {
	path := s.Path()
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s: "+format, append([]any{path}, args...)...))
	}
	warn := func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf("%s: "+format, append([]any{path}, args...)...))
	}

	switch v, ok := s.Frontmatter["name"]; {
	case !ok || v == nil:
		fail("name is missing")
	case s.Name == "":
		fail("name must be a non-empty string")
	case !nameRE.MatchString(s.Name):
		fail("invalid name %q: use lowercase letters, digits and dashes, starting with a letter or digit", s.Name)
	default:
		if len(s.Name) > maxNameLen {
			warn("name is longer than %d characters", maxNameLen)
		}
		if folder := filepath.Base(s.Dir); folder != s.Name {
			warn("folder name %q differs from name %q", folder, s.Name)
		}
	}

	switch v, ok := s.Frontmatter["description"]; {
	case !ok || v == nil:
		fail("description is missing")
	case strings.TrimSpace(s.Description) == "":
		fail("description must be a non-empty string")
	case len(s.Description) > maxDescriptionLen:
		warn("description is longer than %d characters", maxDescriptionLen)
	}

	return warnings, errors.Join(errs...)
}
