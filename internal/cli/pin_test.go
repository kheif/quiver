package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/config"
	"github.com/kheif/quiver/internal/library"
)

func revParse(t *testing.T, dir, rev string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-parse", rev).Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func description(t *testing.T, id string) string {
	t.Helper()
	l, err := library.Load()
	if err != nil {
		t.Fatal(err)
	}
	s, err := l.Get(id)
	if err != nil {
		return "(" + err.Error() + ")"
	}
	return s.Description
}

func TestPinAndUnpin(t *testing.T) {
	home := sourceEnv(t)
	bare := bareSource(t, "acme", "pdf", "review")
	if _, err := runIn(t, "", "add", "acme/pdf", "acme/review", "--global"); err != nil {
		t.Fatal(err)
	}
	commit := revParse(t, filepath.Join(home, "sources", "acme"), "HEAD")
	out, err := runIn(t, "", "pin", "acme/pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Pinned acme/pdf to "+commit[:7]+".") {
		t.Errorf("pin:\n%s", out)
	}
	if out, _ := runIn(t, "", "pin", "acme/review"); !strings.Contains(out, "Pinned acme/review") {
		t.Errorf("pin review:\n%s", out)
	}
	at := filepath.Join(home, "sources", library.PinsFolder, "acme@"+commit)
	if _, err := os.Stat(filepath.Join(at, "skills", "pdf", "SKILL.md")); err != nil {
		t.Fatal(err)
	}

	push(t, bare, map[string]string{
		"skills/pdf/SKILL.md": "---\nname: pdf\ndescription: Upstream.\n---\n",
		"skills/review":       "",
	}, false)
	out, err = runIn(t, "", "update")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"pinned   acme/pdf (changed upstream; kept at its pin until quiver unpin acme/pdf)", "pinned   acme/review"} {
		if !strings.Contains(out, want) {
			t.Errorf("update output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "changed  acme/pdf") || strings.Contains(out, "removed") || strings.Contains(out, "warning") {
		t.Errorf("update output:\n%s", out)
	}
	// Pinned skills keep their content, also one removed upstream.
	if got := description(t, "acme/pdf"); got != "Test skill." {
		t.Errorf("pinned pdf = %q", got)
	}
	if got := description(t, "acme/review"); got != "Test skill." {
		t.Errorf("pinned review = %q", got)
	}
	if out, _ := runIn(t, "", "show", "acme/pdf"); !strings.Contains(out, "acme (pinned at "+commit[:7]+")") {
		t.Errorf("show:\n%s", out)
	}

	out, err = runIn(t, "", "unpin", "acme/pdf")
	if err != nil || !strings.Contains(out, "Unpinned acme/pdf (was at "+commit[:7]+")") {
		t.Fatalf("unpin: %v\n%s", err, out)
	}
	if got := description(t, "acme/pdf"); got != "Upstream." {
		t.Errorf("after unpin = %q", got)
	}
	// review still uses the pin folder.
	if _, err := os.Stat(at); err != nil {
		t.Errorf("pin folder of review deleted: %v", err)
	}
	if _, err := runIn(t, "", "unpin", "acme/review"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(at); !os.IsNotExist(err) {
		t.Errorf("unused pin folder kept: %v", err)
	}
	if !strings.Contains(description(t, "acme/review"), "not found") {
		t.Error("review served after unpin")
	}
	if _, err := runIn(t, "", "unpin", "acme/review"); err == nil || !strings.Contains(err.Error(), "acme/review is not pinned") {
		t.Errorf("second unpin: %v", err)
	}
}

// A shallow clone fetches the history it needs for an older commit.
func TestPinOlderCommit(t *testing.T) {
	sourceEnv(t)
	repo := skillRepo(t, "acme", "pdf")
	first := revParse(t, repo, "HEAD")
	if err := os.WriteFile(filepath.Join(repo, "skills", "pdf", "SKILL.md"), []byte("---\nname: pdf\ndescription: Second.\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "--quiet", "-am", "second")
	gitIn(t, repo, "tag", "v2")
	if _, err := runIn(t, "", "source", "add", "file:///"+strings.TrimPrefix(filepath.ToSlash(repo), "/"), "--name", "acme"); err != nil {
		t.Fatal(err)
	}
	if got := revParse(t, filepath.Join(os.Getenv("QUIVER_HOME"), "sources", "acme"), "--is-shallow-repository"); got != "true" {
		t.Fatalf("clone is not shallow: %s", got)
	}
	if out, err := runIn(t, "", "pin", "acme/pdf", first[:10]); err != nil || !strings.Contains(out, "Pinned acme/pdf to "+first[:7]) {
		t.Fatalf("pin: %v\n%s", err, out)
	}
	if got := description(t, "acme/pdf"); got != "Test skill." {
		t.Errorf("pinned = %q", got)
	}
	if out, err := runIn(t, "", "pin", "acme/pdf", "v2"); err != nil || !strings.Contains(out, "(was "+first[:7]+")") {
		t.Fatalf("re-pin: %v\n%s", err, out)
	}
	if got := description(t, "acme/pdf"); got != "Second." {
		t.Errorf("re-pinned = %q", got)
	}
	for _, c := range []struct{ args []string }{
		{[]string{"pin", "acme/pdf", "nope"}},
		{[]string{"pin", "acme/pdf", "--", "--output=x"}},
	} {
		if _, err := runIn(t, "", c.args...); err == nil || !strings.Contains(err.Error(), "is not in source acme") {
			t.Errorf("%v: %v", c.args, err)
		}
	}
}

// A pin from another machine's config is not served until quiver update
// checks it out; doctor reports it, and unused pin folders.
func TestPinRestoreAndDoctor(t *testing.T) {
	home := sourceEnv(t)
	bareSource(t, "acme", "pdf")
	if _, err := runIn(t, "", "pin", "acme/pdf"); err != nil {
		t.Fatal(err)
	}
	pins := filepath.Join(home, "sources", library.PinsFolder)
	gitIn(t, filepath.Join(home, "sources", "acme"), "worktree", "remove", "--force", filepath.Join(pins, "acme@"+revParse(t, filepath.Join(home, "sources", "acme"), "HEAD")))
	if !strings.Contains(description(t, "acme/pdf"), "not found") {
		t.Error("served without its pin folder")
	}
	if out, _ := runIn(t, "", "doctor"); !strings.Contains(out, "pinned skills that are not checked out") {
		t.Errorf("doctor:\n%s", out)
	}
	if _, err := runIn(t, "", "update"); err != nil {
		t.Fatal(err)
	}
	if got := description(t, "acme/pdf"); got != "Test skill." {
		t.Errorf("after update = %q", got)
	}

	stray := filepath.Join(pins, "acme@"+strings.Repeat("0", 40))
	if err := os.MkdirAll(stray, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, _ := runIn(t, "", "doctor", "--fix"); !strings.Contains(out, "pin folders that no pin uses") || !strings.Contains(out, "fixed: deleted them") {
		t.Errorf("doctor --fix:\n%s", out)
	}
	if _, err := os.Stat(stray); !os.IsNotExist(err) {
		t.Errorf("stray pin folder kept: %v", err)
	}
}

func TestPinErrorsAndSourceRemove(t *testing.T) {
	home := sourceEnv(t)
	bareSource(t, "acme", "pdf")
	if out, err := runIn(t, "", "new", "notes", "--description", "Notes.", "--none"); err != nil {
		t.Fatalf("new: %v\n%s", err, out)
	}
	if _, err := runIn(t, "", "pin", "notes"); err == nil || !strings.Contains(err.Error(), "only skills from sources can be pinned") {
		t.Errorf("local: %v", err)
	}
	if _, err := runIn(t, "", "pin", "acme/pdf"); err != nil {
		t.Fatal(err)
	}
	if _, err := runIn(t, "", "source", "remove", "acme"); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load()
	if err != nil || len(c.Pins) != 0 {
		t.Errorf("pins after source remove: %v, %v", c.Pins, err)
	}
	if entries, _ := os.ReadDir(filepath.Join(home, "sources", library.PinsFolder)); len(entries) != 0 {
		t.Errorf("pin folders after source remove: %v", entries)
	}
}
