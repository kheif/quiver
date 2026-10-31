package library

import (
	"testing"

	"github.com/kheif/quiver/internal/git"
)

func TestNeedsReview(t *testing.T) {
	for path, want := range map[string]bool{
		"scripts/fill.py": true, "run.SH": true, "index.js": true, "setup.ps1": true,
		"SKILL.md": false, "references/notes.md": false, "logo.png": false, "Makefile": false,
	} {
		if got := NeedsReview(git.FileChange{Path: path, Mode: "100644"}); got != want {
			t.Errorf("%s: got %v", path, got)
		}
	}
	if !NeedsReview(git.FileChange{Path: "tool", Mode: "100755"}) {
		t.Error("executable not flagged")
	}
}
