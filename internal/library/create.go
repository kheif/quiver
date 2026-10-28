package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/skill"
	"gopkg.in/yaml.v3"
)

// Scope says where CreateLocal assigns a new skill.
type Scope string

const (
	ScopeNone    Scope = ""
	ScopeGlobal  Scope = "global"
	ScopeProject Scope = "project"
)

// ErrSkillExists matches the error CreateLocal returns when the name is
// taken.
var ErrSkillExists = errors.New("skill already exists")

// LocalDir returns ~/.quiver/local, the git repository of local skills.
func LocalDir() (string, error) {
	home, err := fsutil.QuiverHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "local"), nil
}

// CreateLocal writes local/<name>/SKILL.md and the extra files, assigns the
// skill and commits it in local/ as "add <name>". files maps slash paths
// inside the skill folder to their content. With ScopeProject the project
// must already be in the registry (see registry.EnsureProject). Nothing is
// left behind when it fails before the commit. It returns the validation
// warnings of the new skill.
func CreateLocal(name, description, body string, files map[string]string, scope Scope, projectID string) (warnings []string, err error) {
	if err := skill.ValidateName(name); err != nil {
		return nil, err
	}
	if err := checkFiles(files); err != nil {
		return nil, err
	}
	content, err := skillFile(name, description, body)
	if err != nil {
		return nil, err
	}
	dir, err := LocalDir()
	if err != nil {
		return nil, err
	}
	target := filepath.Join(dir, name)
	err = fsutil.WithLock(func() error {
		if _, err := os.Lstat(target); err == nil {
			return fmt.Errorf("%w: %s", ErrSkillExists, target)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		r, err := registry.Load()
		if err != nil {
			return err
		}
		switch scope {
		case ScopeNone:
		case ScopeGlobal:
			r.AssignGlobal(name)
		case ScopeProject:
			if err := r.Assign(projectID, name); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown scope %q", scope)
		}
		// Write into a hidden folder first, which the library skips, so a
		// half-written skill is never loaded.
		tmp, err := os.MkdirTemp(dir, ".new-"+name+"-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		if warnings, err = writeSkill(tmp, content, files); err != nil {
			return err
		}
		if err := os.Rename(tmp, target); err != nil {
			return err
		}
		if scope != ScopeNone {
			if err := registry.Write(r); err != nil {
				return errors.Join(err, os.RemoveAll(target))
			}
		}
		if err := git.Commit(dir, "add "+name, name); err != nil {
			return fmt.Errorf("skill %s was created, but the commit in %s failed: %w", name, dir, err)
		}
		return nil
	})
	return warnings, err
}

// checkFiles rejects extra file paths that could leave the skill folder,
// write into git, or replace SKILL.md.
func checkFiles(files map[string]string) error {
	for p := range files {
		local := filepath.FromSlash(p)
		if !filepath.IsLocal(local) || strings.Contains(p, `\`) {
			return fmt.Errorf("invalid file path %q: use a relative path inside the skill folder, without ..", p)
		}
		parts := strings.Split(filepath.ToSlash(filepath.Clean(local)), "/")
		if slices.Contains(parts, ".git") {
			return fmt.Errorf("invalid file path %q: .git is not allowed", p)
		}
		if len(parts) == 1 && parts[0] == skill.FileName {
			return fmt.Errorf("invalid file path %q: SKILL.md is written from the name, description and body", p)
		}
	}
	return nil
}

// skillFile returns the SKILL.md content. yaml quotes the description where
// it needs to.
func skillFile(name, description, body string) ([]byte, error) {
	front, err := yaml.Marshal(struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}{name, strings.TrimSpace(description)})
	if err != nil {
		return nil, err
	}
	content := "---\n" + string(front) + "---\n"
	if body = strings.TrimSpace(body); body != "" {
		content += "\n" + body + "\n"
	}
	return []byte(content), nil
}

// writeSkill writes the skill into dir and validates it.
func writeSkill(dir string, content []byte, files map[string]string) ([]string, error) {
	if err := os.WriteFile(filepath.Join(dir, skill.FileName), content, 0o644); err != nil {
		return nil, err
	}
	for p, data := range files {
		path := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			return nil, err
		}
	}
	s, err := skill.Load(dir)
	if err != nil {
		return nil, err
	}
	// Validate against the final folder name, so no "folder differs" warning.
	s.Dir = filepath.Join(filepath.Dir(dir), s.Name)
	return skill.Validate(s)
}

// CommitEdit validates the local skill after an edit and commits its folder
// in local/ as "edit <id>". An invalid skill is not committed. It returns
// the validation warnings.
func CommitEdit(id string) (warnings []string, err error) {
	dir, err := LocalDir()
	if err != nil {
		return nil, err
	}
	s, err := skill.Load(filepath.Join(dir, id))
	if err != nil {
		return nil, err
	}
	if warnings, err = skill.Validate(s); err != nil {
		return nil, err
	}
	return warnings, fsutil.WithLock(func() error {
		return git.Commit(dir, "edit "+id, id)
	})
}
