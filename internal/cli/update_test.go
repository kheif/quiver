package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

// bareSource adds a source named name from a bare repository and returns
// the path of the bare repository.
func bareSource(t *testing.T, name string, skills ...string) string {
	t.Helper()
	bare := filepath.Join(t.TempDir(), name+".git")
	gitIn(t, ".", "clone", "--quiet", "--bare", skillRepo(t, name, skills...), bare)
	if _, err := runIn(t, "", "source", "add", bare); err != nil {
		t.Fatal(err)
	}
	return bare
}

// push clones bare, writes files (an empty content deletes the file),
// commits and pushes. extra is passed to git commit, and force pushes with
// --force.
func push(t *testing.T, bare string, files map[string]string, force bool, extra ...string) {
	t.Helper()
	work := filepath.Join(t.TempDir(), "work")
	gitIn(t, ".", "clone", "--quiet", bare, work)
	for p, content := range files {
		full := filepath.Join(work, filepath.FromSlash(p))
		if content == "" {
			gitIn(t, work, "rm", "--quiet", "-r", p)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitIn(t, work, "add", "-A")
	gitIn(t, work, append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "commit", "--quiet", "-m", "change"}, extra...)...)
	args := []string{"push", "--quiet", "origin", "HEAD"}
	if force {
		args = append(args, "--force")
	}
	gitIn(t, work, args...)
}

func TestUpdate(t *testing.T) {
	sourceEnv(t)
	bare := bareSource(t, "acme", "pdf", "review", "old")
	err := registry.Update(func(r *registry.Registry) error {
		_ = r.EnsureProject("p1", "/work/web")
		r.Projects["p1"].Name = "web"
		r.AssignGlobal("acme/old", "acme/pdf")
		return r.Assign("p1", "acme/old")
	})
	if err != nil {
		t.Fatal(err)
	}
	push(t, bare, map[string]string{
		"skills/pdf/SKILL.md": "---\nname: pdf\ndescription: Updated.\n---\n",
		"skills/new/SKILL.md": "---\nname: new\ndescription: New.\n---\n",
		"skills/old":          "",
		"README.md":           "# changed",
	}, false)

	out, err := runIn(t, "y\n", "update")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"added    acme/new", "changed  acme/pdf (assigned: global)\n           M SKILL.md  +1 -1\n",
		"removed  acme/old (assigned: global, project web)",
		"warning: acme/old will be removed upstream but stays assigned (global, project web)",
		"Apply the changes to acme? [y/N] acme: updated to "} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Count(out, "warning") != 1 {
		t.Errorf("want one warning:\n%s", out)
	}
	if r := loadRegistry(t); !slices.Contains(r.Global, "acme/old") || !slices.Contains(r.Projects["p1"].Skills, "acme/old") {
		t.Errorf("removed skill was unassigned: %+v", r)
	}
	if strings.Contains(out, "acme/review") {
		t.Errorf("unchanged skill listed:\n%s", out)
	}
	l, err := library.Load()
	if err != nil {
		t.Fatal(err)
	}
	if s, err := l.Get("acme/pdf"); err != nil || s.Description != "Updated." {
		t.Errorf("after update: %+v, %v", s, err)
	}

	out, err = runIn(t, "", "update", "acme")
	if err != nil || !strings.Contains(out, "acme: up to date") {
		t.Errorf("second update: %v\n%s", err, out)
	}
}

// A second skill named pdf does not take the ID of the first, also for a
// source added before IDs were saved.
func TestUpdateKeepsSkillIDs(t *testing.T) {
	for _, saved := range []bool{true, false} {
		t.Run(fmt.Sprintf("saved=%v", saved), func(t *testing.T) {
			sourceEnv(t)
			bare := bareSource(t, "src", "pdf")
			if !saved {
				err := config.Update(func(c *config.Config) error {
					s := c.Sources["src"]
					s.IDs = nil
					c.Sources["src"] = s
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := runIn(t, "", "add", "src/pdf", "--global"); err != nil {
				t.Fatal(err)
			}
			push(t, bare, map[string]string{"extra/pdf/SKILL.md": "---\nname: pdf\ndescription: Other.\n---\n"}, false)

			out, err := runIn(t, "", "update")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, "added    src/extra/pdf") || strings.Contains(out, "removed") || strings.Contains(out, "warning") {
				t.Errorf("output:\n%s", out)
			}
			l, err := library.Load()
			if err != nil {
				t.Fatal(err)
			}
			if s, err := l.Get("src/pdf"); err != nil || s.Description != "Test skill." {
				t.Errorf("src/pdf: %+v, %v", s, err)
			}
			if s, err := l.Get("src/extra/pdf"); err != nil || s.Description != "Other." {
				t.Errorf("src/extra/pdf: %+v, %v", s, err)
			}
			c, err := config.Load()
			if err != nil {
				t.Fatal(err)
			}
			want := map[string]string{"pdf": "skills/pdf", "extra/pdf": "extra/pdf"}
			if got := c.Sources["src"].IDs; !reflect.DeepEqual(got, want) {
				t.Errorf("ids = %v", got)
			}
		})
	}
}

func TestUpdateContinuesAfterFailure(t *testing.T) {
	sourceEnv(t)
	broken := bareSource(t, "aaa", "pdf")
	ok := bareSource(t, "bbb", "pdf")
	// A rewritten history cannot be fast-forwarded.
	push(t, broken, map[string]string{"README.md": "# rewritten"}, true, "--amend")
	push(t, ok, map[string]string{"skills/new/SKILL.md": "---\nname: new\ndescription: New.\n---\n"}, false)

	out, err := runIn(t, "", "update")
	if err == nil || !strings.Contains(err.Error(), "could not update aaa") {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(out, "aaa: failed:") || !strings.Contains(out, "quiver update aaa --reset") || !strings.Contains(out, "added    bbb/new") {
		t.Errorf("output:\n%s", out)
	}

	out, err = runIn(t, "", "update", "aaa", "--reset")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "(reset to the rewritten remote history)") {
		t.Errorf("output:\n%s", out)
	}
	if out, _ := runIn(t, "", "update", "aaa"); !strings.Contains(out, "aaa: up to date") {
		t.Errorf("after reset:\n%s", out)
	}
}

func TestUpdateUnknownSource(t *testing.T) {
	sourceEnv(t)
	if _, err := runIn(t, "", "update", "nope"); err == nil || !strings.Contains(err.Error(), `source "nope" not found`) {
		t.Errorf("got %v", err)
	}
}

func TestUpdatePreviewAsksForScripts(t *testing.T) {
	sourceEnv(t)
	bare := bareSource(t, "acme", "pdf", "review", "spare")
	if _, err := runIn(t, "", "add", "acme/pdf", "acme/review", "--global"); err != nil {
		t.Fatal(err)
	}
	if _, err := runIn(t, "", "add", "acme/pdf", "--global", "--always"); err != nil {
		t.Fatal(err)
	}
	// pdf gets a script and an executable; spare is not assigned.
	work := filepath.Join(t.TempDir(), "work")
	gitIn(t, ".", "clone", "--quiet", bare, work)
	writeFiles(t, work, map[string]string{
		"skills/pdf/scripts/fill.py": "print('fill')\n",
		"skills/pdf/tool":            "#!/bin/sh\necho hi\n",
		"skills/pdf/notes.md":        "Notes.\n",
		"skills/spare/SKILL.md":      "---\nname: spare\ndescription: Changed.\n---\n",
	})
	if err := os.Chmod(filepath.Join(work, "skills", "pdf", "tool"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, work, "add", "-A")
	gitIn(t, work, "update-index", "--chmod=+x", "skills/pdf/tool")
	gitIn(t, work, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "--quiet", "-m", "script")
	gitIn(t, work, "push", "--quiet", "origin", "HEAD")

	// No answer: nothing is applied.
	out, err := runIn(t, "", "update")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"  changed  acme/pdf (assigned: global, always-on)\n",
		"           A notes.md         +1 -0\n",
		"           A scripts/fill.py  +1 -0  needs review\n",
		"           A tool             +2 -0  needs review\n",
		"  changed  acme/spare\n",
		"Apply the changes to acme? [y/N] \nacme: not updated; run quiver update acme --yes",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "review (assigned") {
		t.Errorf("unchanged skill listed:\n%s", out)
	}
	l, err := library.Load()
	if err != nil {
		t.Fatal(err)
	}
	if s, _ := l.Get("acme/spare"); s == nil || s.Description != "Test skill." {
		t.Errorf("declined update was applied: %+v", s)
	}

	// --diff shows the patch of the assigned skill only.
	out, err = runIn(t, "n\n", "update", "--diff")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "+print('fill')") || strings.Contains(out, "+description: Changed.") || !strings.Contains(out, "acme: not updated") {
		t.Errorf("--diff:\n%s", out)
	}

	out, err = runIn(t, "", "update", "--yes")
	if err != nil || strings.Contains(out, "Apply the changes") || !strings.Contains(out, "acme: updated to ") {
		t.Fatalf("--yes: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("QUIVER_HOME"), "sources", "acme", "skills", "pdf", "scripts", "fill.py")); err != nil {
		t.Errorf("not applied: %v", err)
	}
	if out, _ := runIn(t, "", "update"); !strings.Contains(out, "acme: up to date") {
		t.Errorf("after --yes:\n%s", out)
	}
}

// A change to skills that are not assigned is applied without a question.
func TestUpdateUnassignedNoQuestion(t *testing.T) {
	sourceEnv(t)
	bare := bareSource(t, "acme", "pdf")
	push(t, bare, map[string]string{"skills/pdf/run.sh": "echo\n"}, false)
	out, err := runIn(t, "", "update")
	if err != nil || strings.Contains(out, "Apply the changes") || !strings.Contains(out, "  changed  acme/pdf\n") {
		t.Fatalf("%v\n%s", err, out)
	}
	if out, _ := runIn(t, "", "update"); !strings.Contains(out, "acme: up to date") {
		t.Errorf("not applied:\n%s", out)
	}
}
