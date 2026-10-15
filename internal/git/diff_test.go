package git

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiffAfterPull(t *testing.T) {
	work, url := newRemote(t)
	dir := filepath.Join(t.TempDir(), "clone")
	if err := Clone(url, dir); err != nil {
		t.Fatal(err)
	}
	from, _ := Head(dir)

	write(t, work, "pdf/SKILL.md", "v2")
	write(t, work, "docx/SKILL.md", "new")
	if err := Commit(work, "v2"); err != nil {
		t.Fatal(err)
	}
	push(t, work)
	if err := Pull(dir); err != nil {
		t.Fatal(err)
	}
	to, _ := Head(dir)

	names, err := DiffNames(dir, from, to)
	if err != nil || !reflect.DeepEqual(names, []string{"docx/SKILL.md", "pdf/SKILL.md"}) {
		t.Errorf("names %v, %v", names, err)
	}

	patch, err := Diff(dir, from, to, "pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(patch, "-v1") || !strings.Contains(patch, "+v2") || strings.Contains(patch, "docx") {
		t.Errorf("patch:\n%s", patch)
	}

	if names, err := DiffNames(dir, to, to); err != nil || names != nil {
		t.Errorf("same commit: %v, %v", names, err)
	}
}
