package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture builds a local library with valid and invalid skills.
func fixture(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("QUIVER_HOME", home)
	files := map[string]string{
		"pdf/SKILL.md":           "---\nname: pdf\ndescription: Work with PDFs.\n---\nBody",
		"db-migration/SKILL.md":  "---\nname: db-migration\ndescription: Run migrations.\n---\n",
		"renamed/SKILL.md":       "---\nname: other-name\ndescription: Folder differs.\n---\n",
		"broken/SKILL.md":        "no frontmatter",
		"no-desc/SKILL.md":       "---\nname: no-desc\n---\n",
		"empty-folder/notes.txt": "x",
		".git/SKILL.md":          "---\nname: git\ndescription: hidden\n---\n",
		"README.md":              "# Quiver skills",
	}
	for name, content := range files {
		p := filepath.Join(home, "local", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadLocal(t *testing.T) {
	fixture(t)
	l, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, s := range l.All() {
		ids = append(ids, s.ID)
		if s.Source != LocalSource {
			t.Errorf("%s: source %q", s.ID, s.Source)
		}
	}
	if got := strings.Join(ids, ","); got != "db-migration,pdf,renamed" {
		t.Errorf("ids %s", got)
	}
	all := strings.Join(l.Warnings, "\n")
	for _, want := range []string{"skipped broken", "skipped no-desc", "skipped empty-folder", `folder name "renamed" differs`} {
		if !strings.Contains(all, want) {
			t.Errorf("missing warning %q in\n%s", want, all)
		}
	}
	if len(l.Warnings) != 4 {
		t.Errorf("got %d warnings:\n%s", len(l.Warnings), all)
	}
}

func TestLoadNoLocalFolder(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	l, err := Load()
	if err != nil || len(l.All()) != 0 || len(l.Warnings) != 0 {
		t.Errorf("got %+v, %v", l, err)
	}
}
