package cli

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// importProject makes a project web-000003 with skills in .claude/skills:
// pdf (a different skill than the library's pdf), review (new) and notes
// (the same as the library's notes).
func importProject(t *testing.T, f fixture) string {
	t.Helper()
	dir := projectDir(t, "web", "web-000003")
	for name, content := range map[string]string{
		"pdf":    "---\nname: pdf\ndescription: Merge PDF files.\n---\nUse qpdf.\n",
		"review": "---\nname: review\ndescription: Review pull requests.\n---\nBe kind.\n",
		"notes":  "---\nname: notes\ndescription: Keep notes.\n---\n",
	} {
		skill := filepath.Join(dir, ".claude", "skills", name)
		if err := os.MkdirAll(skill, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestImport(t *testing.T) {
	f := listEnv(t)
	dir := importProject(t, f)
	t.Chdir(dir)
	// notes: project, keep. pdf: project, taken, renamed to pdf-web, keep.
	// review: global, delete.
	out, err := runIn(t, "\nn\np\npdf-web\nn\ng\ny\n", "import")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, want := range []string{
		"Found 3 skills in ",
		"notes is already in the library with the same content; assigned it to this project.",
		"the library already has a different skill named pdf.\nNew name (empty to skip): Imported pdf-web into the library and assigned it to this project.",
		"Imported review into the library and assigned it to every folder.",
		"Delete the original folder " + filepath.Join(".claude", "skills", "review") + "? [y/N] Deleted.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	// Nothing in the library was overwritten.
	if data, _ := os.ReadFile(filepath.Join(f.home, "local", "pdf", "SKILL.md")); !strings.Contains(string(data), "Fill in and read PDF forms.") {
		t.Errorf("library pdf changed:\n%s", data)
	}
	if data, _ := os.ReadFile(filepath.Join(f.home, "local", "pdf-web", "SKILL.md")); !strings.HasPrefix(string(data), "---\nname: pdf-web\ndescription: Merge PDF files.") {
		t.Errorf("pdf-web:\n%s", data)
	}
	r := loadRegistry(t)
	if got := r.Projects["web-000003"].Skills; !slices.Equal(got, []string{"notes", "pdf-web"}) {
		t.Errorf("web skills = %v", got)
	}
	if !slices.Contains(r.Global, "review") {
		t.Errorf("global = %v", r.Global)
	}
	// Only review's original was deleted.
	for name, kept := range map[string]bool{"pdf": true, "notes": true, "review": false} {
		_, err := os.Stat(filepath.Join(dir, ".claude", "skills", name))
		if kept && err != nil || !kept && !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s original: %v", name, err)
		}
	}
	if got := commits(t, f.home); !slices.Contains(got, "import pdf-web from web") || !slices.Contains(got, "import review from web") {
		t.Errorf("commits = %v", got)
	}
}

func TestImportYes(t *testing.T) {
	f := listEnv(t)
	dir := importProject(t, f)
	out, err := runIn(t, "", "import", "--yes", "--global", dir)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "Skipped: the library already has a different skill named pdf.") || strings.Contains(out, "[y/N]") {
		t.Errorf("output:\n%s", out)
	}
	r := loadRegistry(t)
	for _, id := range []string{"notes", "review"} {
		if !slices.Contains(r.Global, id) {
			t.Errorf("%s not global: %v", id, r.Global)
		}
	}
	if _, ok := r.Projects["web-000003"]; ok {
		t.Error("project registered for global imports")
	}
	for _, name := range []string{"pdf", "notes", "review"} {
		if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", name)); err != nil {
			t.Errorf("%s original: %v", name, err)
		}
	}
}

func TestImportNothing(t *testing.T) {
	listEnv(t)
	dir := t.TempDir()
	out, err := runIn(t, "", "import", dir)
	if err != nil || !strings.Contains(out, "No skills found in ") || !strings.Contains(out, "--deep") {
		t.Errorf("got %q, %v", out, err)
	}
}

// Skipping at the first question imports nothing and asks nothing more.
func TestImportSkip(t *testing.T) {
	f := listEnv(t)
	dir := importProject(t, f)
	out, err := runIn(t, "s\ns\ns\n", "import", dir)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if n := strings.Count(out, "Skipped."); n != 3 {
		t.Errorf("%d skipped:\n%s", n, out)
	}
	if _, err := os.Stat(filepath.Join(f.home, "local", "review")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("review imported: %v", err)
	}
}
