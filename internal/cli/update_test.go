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

	out, err := runIn(t, "", "update")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"added    acme/new", "changed  acme/pdf", "removed  acme/old",
		"warning: acme/old was removed upstream but is still assigned (global, project web)"} {
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
