package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/config"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestUpdateSkillBody(t *testing.T) {
	w, local := createEnv(t)
	writeFiles(t, filepath.Join(local, "pdf"), map[string]string{
		"SKILL.md": "---\nname: pdf\ndescription: Read, create and edit PDF files.\nlicense: MIT\n---\nUse pdftk.\n",
	})
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	out, isErr := call(t, cs, "update_skill", map[string]any{"id": "pdf", "body": "Use qpdf.\n\nNever pdftk."})
	if isErr || !strings.Contains(out, "Updated skill pdf") {
		t.Fatalf("got %v: %s", isErr, out)
	}
	want := "---\nname: pdf\ndescription: Read, create and edit PDF files.\nlicense: MIT\n---\n\nUse qpdf.\n\nNever pdftk.\n"
	if got := readFile(t, filepath.Join(local, "pdf", "SKILL.md")); got != want {
		t.Errorf("SKILL.md =\n%s", got)
	}
	if got := lastCommit(t, local); got != "update pdf (via MCP)" {
		t.Errorf("commit = %q", got)
	}
	if out, _ := call(t, cs, "get_skill", map[string]any{"id": "pdf"}); !strings.Contains(out, "Use qpdf.") {
		t.Errorf("get_skill:\n%s", out)
	}

	// A new description keeps the body and the other fields.
	if out, isErr := call(t, cs, "update_skill", map[string]any{"id": "pdf", "description": "Work with PDFs: forms too."}); isErr {
		t.Fatal(out)
	}
	want = "---\nname: pdf\ndescription: 'Work with PDFs: forms too.'\nlicense: MIT\n---\n\nUse qpdf.\n\nNever pdftk.\n"
	if got := readFile(t, filepath.Join(local, "pdf", "SKILL.md")); got != want {
		t.Errorf("SKILL.md =\n%s", got)
	}
}

func TestUpdateSkillFiles(t *testing.T) {
	w, local := createEnv(t)
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	out, isErr := call(t, cs, "update_skill", map[string]any{
		"id": "db-migration",
		"files": map[string]string{
			"references/rollback.md": "Keep a down migration.\n",
			"scripts/run.sh":         "",
		},
	})
	if isErr {
		t.Fatal(out)
	}
	dir := filepath.Join(local, "db-migration")
	if got := readFile(t, filepath.Join(dir, "references", "rollback.md")); got != "Keep a down migration.\n" {
		t.Errorf("rollback.md = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "scripts")); !os.IsNotExist(err) {
		t.Errorf("scripts/ left behind: %v", err)
	}
	if got := readFile(t, filepath.Join(dir, "SKILL.md")); !strings.Contains(got, "Read references/checklist.md first.") {
		t.Errorf("SKILL.md changed:\n%s", got)
	}
	if got := lastCommit(t, local); got != "update db-migration (via MCP)" {
		t.Errorf("commit = %q", got)
	}
	if out, _ := call(t, cs, "get_skill_file", map[string]any{"id": "db-migration", "path": "references/rollback.md"}); !strings.Contains(out, "Keep a down migration.") {
		t.Errorf("get_skill_file:\n%s", out)
	}
}

func TestUpdateSkillErrors(t *testing.T) {
	w, local := createEnv(t)
	home := os.Getenv("QUIVER_HOME")
	cfg := config.Default()
	cfg.Sources["acme"] = config.Source{Repo: "https://example.com/acme.git"}
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, filepath.Join(home, "sources", "acme"), map[string]string{
		"skills/review/SKILL.md": "---\nname: review\ndescription: Review code.\n---\nCheck the tests.\n",
	})
	outside := filepath.Join(filepath.Dir(local), "outside.md")
	cs := connect(t, New(Options{Version: "dev", Log: discard, Where: w}))
	before := lastCommit(t, local)
	for _, c := range []struct {
		args map[string]any
		want string
	}{
		{map[string]any{"id": "acme/review", "body": "x"}, `quiver fork acme/review`},
		{map[string]any{"id": "pdf", "files": map[string]string{"../outside.md": "x"}}, `invalid file path "../outside.md"`},
		{map[string]any{"id": "pdf", "files": map[string]string{"/etc/x": "x"}}, `invalid file path "/etc/x"`},
		{map[string]any{"id": "pdf", "files": map[string]string{"SKILL.md": "x"}}, "SKILL.md is written from"},
		{map[string]any{"id": "pdf", "files": map[string]string{".git/config": "x"}}, ".git is not allowed"},
		{map[string]any{"id": "pdf", "files": map[string]string{"missing.md": ""}}, "cannot delete missing.md"},
		{map[string]any{"id": "../pdf", "body": "x"}, "not found"},
		{map[string]any{"id": "pdf"}, "nothing to change"},
		{map[string]any{"id": "nope", "body": "x"}, "not found"},
		{map[string]any{"id": "pdf", "description": ""}, "the change was undone"},
	} {
		out, isErr := call(t, cs, "update_skill", c.args)
		if !isErr || !strings.Contains(out, c.want) {
			t.Errorf("%v: got %v: %s", c.args, isErr, out)
		}
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Errorf("file written outside the skill: %v", err)
	}
	if got := readFile(t, filepath.Join(local, "pdf", "SKILL.md")); got != "---\nname: pdf\ndescription: Read, create and edit PDF files.\n---\nUse pdftk.\n" {
		t.Errorf("pdf changed:\n%s", got)
	}
	if got := lastCommit(t, local); got != before {
		t.Errorf("commit = %q", got)
	}
}
