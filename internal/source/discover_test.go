package source

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func discover(t *testing.T, dir string) *Discovery {
	t.Helper()
	d, err := Discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDiscoverPlain(t *testing.T) {
	d := discover(t, "testdata/plain")
	want := []Found{{ID: "alpha", Dir: "alpha"}, {ID: "beta", Dir: "beta"}}
	if !reflect.DeepEqual(d.Skills, want) {
		t.Errorf("got %+v", d.Skills)
	}
}

func TestDiscoverNested(t *testing.T) {
	d := discover(t, "testdata/nested")
	want := []Found{{ID: "lint", Dir: "tools/lint"}, {ID: "writer", Dir: "docs/guides/writer"}}
	if !reflect.DeepEqual(d.Skills, want) {
		t.Errorf("got %+v", d.Skills)
	}
}

func TestDiscoverRootSkill(t *testing.T) {
	root := filepath.Join(t.TempDir(), "solo")
	writeSkill(t, root)
	writeSkill(t, filepath.Join(root, ".git", "x"))
	d := discover(t, root)
	if want := []Found{{ID: "solo", Dir: "."}}; !reflect.DeepEqual(d.Skills, want) {
		t.Errorf("got %+v", d.Skills)
	}
}

func writeSkill(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: " + filepath.Base(dir) + "\ndescription: Test.\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// anthropics-skills and caveman are trimmed copies of the real repositories
// (checked 2026-10-15); custom covers pluginRoot, remote and unsafe paths.
func TestDiscoverMarketplace(t *testing.T) {
	cases := map[string][]Found{
		"anthropics-skills": {
			{ID: "pdf", Dir: "skills/pdf", Plugin: "document-skills"},
			{ID: "skill-creator", Dir: "skills/skill-creator", Plugin: "example-skills"},
			{ID: "xlsx", Dir: "skills/xlsx", Plugin: "document-skills"},
		},
		"caveman": {
			{ID: "caveman", Dir: "skills/caveman", Plugin: "caveman"},
			{ID: "caveman-commit", Dir: "skills/caveman-commit", Plugin: "caveman"},
		},
		"custom": {
			{ID: "fmt", Dir: "plugins/formatter/skills/fmt", Plugin: "formatter"},
			{ID: "one", Dir: "bundle/extra/one", Plugin: "bundle"},
			{ID: "two", Dir: "bundle/extra/two", Plugin: "bundle"},
		},
	}
	for repo, want := range cases {
		t.Run(repo, func(t *testing.T) {
			d := discover(t, filepath.Join("testdata", repo))
			if !reflect.DeepEqual(d.Skills, want) {
				t.Errorf("got %+v", d.Skills)
			}
		})
	}
}

func TestDiscoverMarketplaceWarnings(t *testing.T) {
	d := discover(t, "testdata/custom")
	want := []string{
		`plugin "remote": source is in another repository; skipped`,
		`plugin "escape": skill path "../outside" is outside the repository; skipped`,
		`plugin "escape": no skill at "./missing"`,
	}
	if !reflect.DeepEqual(d.Warnings, want) {
		t.Errorf("got %q", d.Warnings)
	}
}

func TestDiscoverSymlinkOutside(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "secret")
	writeSkill(t, outside)
	root := t.TempDir()
	writeManifest(t, root, `{"plugins": [{"name": "p", "source": "./", "skills": ["./link"]}]}`)
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	d := discover(t, root)
	if len(d.Skills) != 0 || len(d.Warnings) != 1 {
		t.Errorf("got %+v, %q", d.Skills, d.Warnings)
	}
}

func TestDiscoverBrokenManifestWalks(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, `{`)
	writeSkill(t, filepath.Join(root, "a"))
	d := discover(t, root)
	if len(d.Skills) != 1 || d.Skills[0].ID != "a" || len(d.Warnings) != 1 {
		t.Errorf("got %+v, %q", d.Skills, d.Warnings)
	}
}

func writeManifest(t *testing.T, root, content string) {
	t.Helper()
	path := filepath.Join(root, MarketplaceFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverCollisions(t *testing.T) {
	cases := map[string]struct {
		skills   []Found
		warnings []string
	}{
		"collide": {
			skills: []Found{
				{ID: "both/a/review", Dir: "both/a/review", Plugin: "both"},
				{ID: "both/b/review", Dir: "both/b/review", Plugin: "both"},
				{ID: "cli/lint", Dir: "cli/skills/lint", Plugin: "cli"},
				{ID: "deploy", Dir: "web/skills/deploy", Plugin: "web"},
				{ID: "web/lint", Dir: "web/skills/lint", Plugin: "web"},
			},
			warnings: []string{
				`several skills are named "lint"; using IDs cli/lint, web/lint`,
				`several skills are named "review"; using IDs both/a/review, both/b/review`,
			},
		},
		"collide-plain": {
			skills: []Found{
				{ID: "backend/test", Dir: "backend/test"},
				{ID: "docs", Dir: "docs"},
				{ID: "frontend/test", Dir: "frontend/test"},
			},
			warnings: []string{`several skills are named "test"; using IDs backend/test, frontend/test`},
		},
	}
	for repo, c := range cases {
		t.Run(repo, func(t *testing.T) {
			d := discover(t, filepath.Join("testdata", repo))
			if !reflect.DeepEqual(d.Skills, c.skills) {
				t.Errorf("got %+v", d.Skills)
			}
			if !reflect.DeepEqual(d.Warnings, c.warnings) {
				t.Errorf("got %q", d.Warnings)
			}
		})
	}
}
