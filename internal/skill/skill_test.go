package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSkill creates dir/<name>/SKILL.md with content and returns the folder.
func writeSkill(t *testing.T, name, content string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadValid(t *testing.T) {
	dir := writeSkill(t, "pdf", "---\nname: pdf\ndescription: Work with PDFs.\nlicense: MIT\nmetadata:\n  version: 2\n---\n# PDF\n\nBody text.\n")
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "pdf" || s.Description != "Work with PDFs." {
		t.Errorf("got name %q description %q", s.Name, s.Description)
	}
	if s.Body != "# PDF\n\nBody text.\n" {
		t.Errorf("body = %q", s.Body)
	}
	if s.Frontmatter["license"] != "MIT" {
		t.Errorf("unknown field not kept: %v", s.Frontmatter)
	}
	if m, ok := s.Frontmatter["metadata"].(map[string]any); !ok || m["version"] != 2 {
		t.Errorf("nested field not kept: %v", s.Frontmatter["metadata"])
	}
}

func TestLoadCRLFAndEmptyBody(t *testing.T) {
	dir := writeSkill(t, "x", "---\r\nname: x\r\ndescription: d\r\n---")
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "x" || s.Body != "" {
		t.Errorf("got %+v", s)
	}
}

func TestLoadErrors(t *testing.T) {
	cases := map[string]string{
		"no frontmatter": "# Just markdown\n",
		"no closing":     "---\nname: x\n",
		"bad yaml":       "---\nname: [x\n---\n",
	}
	for label, content := range cases {
		t.Run(label, func(t *testing.T) {
			dir := writeSkill(t, "x", content)
			_, err := Load(dir)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), filepath.Join(dir, FileName)) {
				t.Errorf("error has no path: %v", err)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), filepath.Join(dir, FileName)) {
		t.Errorf("got %v", err)
	}
}
