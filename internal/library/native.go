package library

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/kheif/quiver/internal/git"
	"github.com/kheif/quiver/internal/skill"
)

// Native is a skill installed for Claude Code itself, outside Quiver.
type Native struct {
	Name    string // from the frontmatter, or the folder name
	Dir     string
	Plugin  string // the enabled plugin it comes from, such as caveman@caveman
	Tracked bool   // committed in the git repository it is in
}

// NativeCopy is a native skill that is also served by Quiver.
type NativeCopy struct {
	Native
	ID       string // the served skill
	SameName bool   // has the name of the served skill
	SameText bool   // has the files and content of the served skill
}

// ClaudeDir returns $CLAUDE_CONFIG_DIR if it is set, otherwise ~/.claude.
func ClaudeDir() (string, error) {
	if d := os.Getenv("CLAUDE_CONFIG_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude"), nil
}

// NativeSkills returns the skills Claude Code loads by itself: those in
// ~/.claude/skills, in projectRoot/.claude/skills (none when projectRoot
// is "") and in the skills folder of each enabled plugin. Folders that
// cannot be read are left out.
func NativeSkills(projectRoot string) ([]Native, error) {
	claude, err := ClaudeDir()
	if err != nil {
		return nil, err
	}
	var out []Native
	add := func(dir, plugin string) {
		matches, _ := filepath.Glob(filepath.Join(dir, "*", skill.FileName))
		for _, m := range matches {
			d := filepath.Dir(m)
			n := Native{Name: filepath.Base(d), Dir: d, Plugin: plugin}
			if s, err := skill.Load(d); err == nil && s.Name != "" {
				n.Name = s.Name
			}
			if plugin == "" {
				n.Tracked = git.Tracked(d, ".")
			}
			out = append(out, n)
		}
	}
	add(filepath.Join(claude, "skills"), "")
	if projectRoot != "" && !samePath(filepath.Join(projectRoot, ".claude"), claude) {
		add(filepath.Join(projectRoot, ".claude", "skills"), "")
	}
	plugins, err := enabledPlugins(claude, projectRoot)
	if err != nil {
		return nil, err
	}
	for _, p := range plugins {
		add(filepath.Join(p.dir, "skills"), p.name)
	}
	return out, nil
}

// NativeCopies returns the native skills that have the name or the content
// of a served skill, sorted by folder.
func NativeCopies(served []*skill.Skill, projectRoot string) ([]NativeCopy, error) {
	natives, err := NativeSkills(projectRoot)
	if err != nil {
		return nil, err
	}
	var out []NativeCopy
	for _, n := range natives {
		for _, s := range served {
			c := NativeCopy{Native: n, ID: s.ID, SameName: n.Name == s.Name}
			if same, err := sameTree(n.Dir, s.Dir); err == nil {
				c.SameText = same
			}
			if c.SameName || c.SameText {
				out = append(out, c)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Dir < out[j].Dir })
	return out, nil
}

type plugin struct{ name, dir string }

// enabledPlugins returns the installed plugins that Claude Code's settings
// enable, for the user and for the project at projectRoot. A later
// settings file wins, as in Claude Code: user, project, local project.
func enabledPlugins(claude, projectRoot string) ([]plugin, error) {
	files := []string{filepath.Join(claude, "settings.json")}
	if projectRoot != "" {
		files = append(files,
			filepath.Join(projectRoot, ".claude", "settings.json"),
			filepath.Join(projectRoot, ".claude", "settings.local.json"))
	}
	enabled := map[string]bool{}
	for _, f := range files {
		var s struct {
			EnabledPlugins map[string]bool `json:"enabledPlugins"`
		}
		if err := readJSON(f, &s); err != nil {
			continue // a settings file Claude Code cannot read enables nothing either
		}
		for name, on := range s.EnabledPlugins {
			enabled[name] = on
		}
	}
	var installed struct {
		Plugins map[string][]struct {
			Scope       string `json:"scope"`
			InstallPath string `json:"installPath"`
			ProjectPath string `json:"projectPath"`
		} `json:"plugins"`
	}
	if err := readJSON(filepath.Join(claude, "plugins", "installed_plugins.json"), &installed); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []plugin
	for name, installs := range installed.Plugins {
		if !enabled[name] {
			continue
		}
		for _, in := range installs {
			if in.ProjectPath == "" || projectRoot != "" && samePath(in.ProjectPath, projectRoot) {
				out = append(out, plugin{name, in.InstallPath})
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
