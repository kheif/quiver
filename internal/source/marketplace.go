package source

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kheif/quiver/internal/skill"
)

// MarketplaceFile is the Claude Code plugin marketplace manifest.
const MarketplaceFile = ".claude-plugin/marketplace.json"

type marketplace struct {
	Metadata struct {
		PluginRoot string `json:"pluginRoot"`
	} `json:"metadata"`
	Plugins []plugin `json:"plugins"`
}

type plugin struct {
	Name string `json:"name"`
	// Source is a relative path, or an object for a plugin in another repo.
	Source json.RawMessage `json:"source"`
	// Skills is a path or a list of paths, relative to the plugin folder.
	Skills json.RawMessage `json:"skills"`
}

// readMarketplace parses root's marketplace.json. ok is false when the file
// does not exist.
func readMarketplace(root string) (m *marketplace, ok bool, err error) {
	data, err := os.ReadFile(filepath.Join(root, MarketplaceFile))
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	m = &marketplace{}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, true, fmt.Errorf("%s: %w", MarketplaceFile, err)
	}
	return m, true, nil
}

// skillDirs returns the skill folders of every plugin, relative to root,
// with the plugin each one belongs to. A folder listed twice is kept once.
func (m *marketplace) skillDirs(root string, d *Discovery) []Found {
	var out []Found
	seen := map[string]bool{}
	for _, p := range m.Plugins {
		pluginDir, ok := m.pluginDir(root, p, d)
		if !ok {
			continue
		}
		for _, dir := range pluginSkills(root, pluginDir, p, d) {
			if !seen[dir] {
				seen[dir] = true
				out = append(out, Found{Dir: dir, Plugin: p.Name})
			}
		}
	}
	return out
}

// pluginDir resolves a plugin's source folder relative to root.
func (m *marketplace) pluginDir(root string, p plugin, d *Discovery) (string, bool) {
	var src string
	if err := json.Unmarshal(p.Source, &src); err != nil {
		d.warn("plugin %q: source is in another repository; skipped", p.Name)
		return "", false
	}
	if m.Metadata.PluginRoot != "" && !strings.HasPrefix(src, "./") {
		src = filepath.Join(m.Metadata.PluginRoot, src)
	}
	dir, ok := inside(root, src)
	if !ok {
		d.warn("plugin %q: source %q is outside the repository; skipped", p.Name, src)
	}
	return dir, ok
}

// pluginSkills returns the skill folders of one plugin. Listed paths may be
// a skill folder or a folder of skills. Without a list, it uses skills/*.
func pluginSkills(root, pluginDir string, p plugin, d *Discovery) []string {
	paths, err := stringList(p.Skills)
	if err != nil {
		d.warn("plugin %q: invalid skills field; skipped", p.Name)
		return nil
	}
	if paths == nil {
		return skillsIn(root, filepath.Join(pluginDir, "skills"))
	}
	var out []string
	for _, rel := range paths {
		dir, ok := inside(root, filepath.Join(pluginDir, rel))
		switch {
		case !ok:
			d.warn("plugin %q: skill path %q is outside the repository; skipped", p.Name, rel)
		case isFile(filepath.Join(root, dir, skill.FileName)):
			out = append(out, filepath.ToSlash(dir))
		default:
			found := skillsIn(root, dir)
			if len(found) == 0 {
				d.warn("plugin %q: no skill at %q", p.Name, rel)
			}
			out = append(out, found...)
		}
	}
	return out
}

// skillsIn returns dir/*/ folders that hold a SKILL.md, relative to root.
func skillsIn(root, dir string) []string {
	entries, err := os.ReadDir(filepath.Join(root, dir))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		sub, ok := inside(root, filepath.Join(dir, e.Name()))
		if ok && isFile(filepath.Join(root, sub, skill.FileName)) {
			out = append(out, filepath.ToSlash(sub))
		}
	}
	return out
}

// inside cleans rel and reports whether it stays inside root, also after
// following symlinks. A path that does not exist is only checked as text;
// later lookups find nothing there.
func inside(root, rel string) (string, bool) {
	rel = filepath.Clean(rel)
	if filepath.IsAbs(rel) || escapes(rel) {
		return "", false
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", false
	}
	real, err := filepath.EvalSymlinks(filepath.Join(root, rel))
	if os.IsNotExist(err) {
		return rel, true
	}
	if err != nil {
		return "", false
	}
	r, err := filepath.Rel(realRoot, real)
	if err != nil || escapes(r) {
		return "", false
	}
	return rel, true
}

func escapes(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// stringList decodes a JSON string or list of strings. It returns nil when
// the field is missing.
func stringList(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return []string{one}, nil
	}
	var list []string
	err := json.Unmarshal(raw, &list)
	return list, err
}
