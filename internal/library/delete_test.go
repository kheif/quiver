package library

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/registry"
)

func TestDeleteLocal(t *testing.T) {
	local := createEnv(t)
	if _, err := CreateLocal("draft", "A draft.", "Body.", nil, ScopeNone, ""); err != nil {
		t.Fatal(err)
	}
	u, err := DeleteLocal("draft", false)
	if err != nil || !u.Empty() {
		t.Fatalf("delete: %+v, %v", u, err)
	}
	if _, err := os.Stat(filepath.Join(local, "draft")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("folder left: %v", err)
	}
	if got := gitOut(t, local, "log", "-1", "--format=%s"); got != "delete draft" {
		t.Errorf("commit = %q", got)
	}
	if got := gitOut(t, local, "status", "--short"); got != "" {
		t.Errorf("uncommitted: %q", got)
	}
	if _, err := DeleteLocal("draft", false); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("twice: %v", err)
	}
}

func TestDeleteLocalAssigned(t *testing.T) {
	local := createEnv(t)
	if _, err := CreateLocal("pdf", "PDF.", "Body.", nil, ScopeProject, "app-000001"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Update(func(r *registry.Registry) error { r.AssignGlobal("pdf"); return r.MarkAlwaysOnGlobal("pdf") }); err != nil {
		t.Fatal(err)
	}
	_, err := DeleteLocal("pdf", false)
	var inUse *SkillInUseError
	if !errors.As(err, &inUse) || !strings.Contains(err.Error(), "global\n  project app (") {
		t.Fatalf("assigned: %v", err)
	}
	if _, err := os.Stat(filepath.Join(local, "pdf")); err != nil {
		t.Errorf("deleted without --force: %v", err)
	}
	u, err := DeleteLocal("pdf", true)
	if err != nil || len(u.Global) != 1 || len(u.Projects) != 1 {
		t.Fatalf("force: %+v, %v", u, err)
	}
	r, _ := registry.Load()
	if len(r.Global) != 0 || len(r.AlwaysOn) != 0 || len(r.Projects["app-000001"].Skills) != 0 {
		t.Errorf("still assigned: %+v %+v", r, r.Projects["app-000001"])
	}
}

func TestDeleteLocalRejects(t *testing.T) {
	local := createEnv(t)
	for id, want := range map[string]string{
		"acme/review": "quiver source remove acme",
		"../x":        "invalid skill ID",
		".git":        "invalid skill ID",
		"":            "invalid skill ID",
	} {
		if _, err := DeleteLocal(id, true); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v", id, err)
		}
	}
	// A skill folder that was never committed is deleted too.
	if err := os.MkdirAll(filepath.Join(local, "loose"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(local, "loose", "SKILL.md"), []byte("---\nname: loose\ndescription: d\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := DeleteLocal("loose", false); err != nil {
		t.Errorf("untracked: %v", err)
	}
}
