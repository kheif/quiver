package library

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/kheif/quiver/internal/skill"
)

func TestNativeCopies(t *testing.T) {
	claude := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", claude)
	project := t.TempDir()
	lib := t.TempDir()
	writeTree(t, lib, map[string]string{
		"pdf/SKILL.md":    skillMD("pdf", "PDF."),
		"review/SKILL.md": skillMD("review", "Review."),
		"deploy/SKILL.md": skillMD("deploy", "Deploy."),
	})
	plugins := filepath.Join(claude, "plugins")
	writeTree(t, claude, map[string]string{
		// Same name, other content.
		"skills/pdf/SKILL.md": skillMD("pdf", "Another PDF skill."),
		// Same content under another folder name.
		"skills/my-review/SKILL.md": skillMD("review", "Review."),
		"skills/other/SKILL.md":     skillMD("other", "Not served."),
		"settings.json":             `{"enabledPlugins": {"tools@market": true, "off@market": true, "here@market": true}}`,
		"plugins/installed_plugins.json": `{"version": 2, "plugins": {
			"tools@market": [{"scope": "user", "installPath": "` + filepath.ToSlash(filepath.Join(plugins, "tools")) + `"}],
			"off@market": [{"scope": "user", "installPath": "` + filepath.ToSlash(filepath.Join(plugins, "off")) + `"}],
			"here@market": [{"scope": "project", "projectPath": "/somewhere/else", "installPath": "` + filepath.ToSlash(filepath.Join(plugins, "here")) + `"}]
		}}`,
		"plugins/tools/skills/deploy/SKILL.md": skillMD("deploy", "Deploy."),
		"plugins/off/skills/pdf/SKILL.md":      skillMD("pdf", "PDF."),
		"plugins/here/skills/pdf/SKILL.md":     skillMD("pdf", "PDF."),
	})
	writeTree(t, project, map[string]string{
		".claude/skills/pdf/SKILL.md": skillMD("pdf", "PDF."),
		// The project turns off a plugin the user turned on.
		".claude/settings.local.json": `{"enabledPlugins": {"off@market": false}}`,
	})
	if out, err := exec.Command("git", "-C", project, "init", "--quiet").CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if out, err := exec.Command("git", "-C", project, "add", ".claude/skills").CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	var served []*skill.Skill
	for _, name := range []string{"pdf", "review", "deploy"} {
		s, err := skill.Load(filepath.Join(lib, name))
		if err != nil {
			t.Fatal(err)
		}
		s.ID = "acme/" + name
		served = append(served, s)
	}
	got, err := NativeCopies(served, project)
	if err != nil {
		t.Fatal(err)
	}
	want := []NativeCopy{
		{Native{Name: "deploy", Dir: filepath.Join(plugins, "tools", "skills", "deploy"), Plugin: "tools@market"}, "acme/deploy", true, true},
		{Native{Name: "review", Dir: filepath.Join(claude, "skills", "my-review")}, "acme/review", true, true},
		{Native{Name: "pdf", Dir: filepath.Join(claude, "skills", "pdf")}, "acme/pdf", true, false},
		{Native{Name: "pdf", Dir: filepath.Join(project, ".claude", "skills", "pdf"), Tracked: true}, "acme/pdf", true, true},
	}
	byDir := map[string]NativeCopy{}
	for _, c := range got {
		byDir[c.Dir] = c
	}
	if len(got) != len(want) {
		t.Errorf("got %d copies: %+v", len(got), got)
	}
	for _, w := range want {
		if c, ok := byDir[w.Dir]; !ok || c != w {
			t.Errorf("%s:\n got %+v\nwant %+v", w.Dir, c, w)
		}
	}
	// Without a project, the user's skills and plugins count, and off is on.
	if got, err := NativeCopies(served, ""); err != nil || len(got) != 4 {
		t.Errorf("no project: %+v, %v", got, err)
	}
}

func TestNativeSkillsNoClaude(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(t.TempDir(), "missing"))
	if got, err := NativeSkills(""); err != nil || len(got) != 0 {
		t.Errorf("got %v, %v", got, err)
	}
}
