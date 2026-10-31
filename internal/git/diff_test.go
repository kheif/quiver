package git

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiffAfterFastForward(t *testing.T) {
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
	if err := Fetch(dir); err != nil {
		t.Fatal(err)
	}
	if err := FastForward(dir, "@{u}"); err != nil {
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

func TestDiffFiles(t *testing.T) {
	work, url := newRemote(t)
	dir := filepath.Join(t.TempDir(), "clone")
	if err := Clone(url, dir); err != nil {
		t.Fatal(err)
	}
	from, _ := Head(dir)
	write(t, work, "pdf/SKILL.md", "v2\nmore\n")
	write(t, work, "pdf/run.sh", "echo\n")
	write(t, work, "pdf/logo.png", "\x00\x01")
	write(t, work, "docx/SKILL.md", "other")
	if _, err := run(work, "add", "-A"); err != nil {
		t.Fatal(err)
	}
	// git update-index sets the bit on Windows too; Commit would add the
	// file again without it.
	if _, err := run(work, "update-index", "--chmod=+x", "pdf/run.sh"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(work, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--quiet", "-m", "v2"); err != nil {
		t.Fatal(err)
	}
	push(t, work)
	if err := Fetch(dir); err != nil {
		t.Fatal(err)
	}
	to, err := ResolveCommit(dir, "@{u}")
	if err != nil {
		t.Fatal(err)
	}
	files, err := DiffFiles(dir, from, to, "pdf")
	if err != nil {
		t.Fatal(err)
	}
	want := []FileChange{
		{Path: "pdf/SKILL.md", Status: "M", Mode: "100644", Added: 2, Deleted: 1},
		{Path: "pdf/logo.png", Status: "A", Mode: "100644", Binary: true},
		{Path: "pdf/run.sh", Status: "A", Mode: "100755", Added: 1},
	}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("got %+v", files)
	}
}
