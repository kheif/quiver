package cli

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/git"
)

// commitHash is a short commit hash in show's output.
var commitHash = regexp.MustCompile(`\(([0-9a-f]{7})\)`)

// showRun runs show and replaces the commit hash, which changes per run.
func showRun(t *testing.T, f fixture, args ...string) string {
	t.Helper()
	return commitHash.ReplaceAllString(f.run(t, append([]string{"show"}, args...)...), "(<commit>)")
}

// runOK runs quiver and returns its output as it is, for JSON.
func runOK(t *testing.T, args ...string) string {
	t.Helper()
	out, err := runIn(t, "", args...)
	if err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
	return out
}

func TestShowLocal(t *testing.T) {
	f := listEnv(t)
	writeFiles(t, filepath.Join(f.home, "local", "pdf"), map[string]string{
		"references/forms.md": "# Forms\n",
		"scripts/fill.sh":     "#!/bin/sh\n",
	})
	golden(t, "show-local", showRun(t, f, "pdf"))
	golden(t, "show-unassigned", showRun(t, f, "notes"))
}

func TestShowRemote(t *testing.T) {
	f := listEnv(t)
	golden(t, "show-remote", showRun(t, f, "acme/review"))

	var got shownSkill
	if err := json.Unmarshal([]byte(runOK(t, "show", "acme/review", "--json")), &got); err != nil {
		t.Fatal(err)
	}
	head, err := git.Head(filepath.Join(f.home, "sources", "acme"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "acme" || got.Commit != head[:7] || len(got.Assigned) != 2 || got.Assigned[0].Project != "" || got.Assigned[1].Project != "app-000001" {
		t.Errorf("%+v", got)
	}
	if !strings.HasPrefix(got.Text, "---\nname: review\n") || got.Files == nil {
		t.Errorf("text %q, files %v", got.Text, got.Files)
	}
}

func TestShowUnknown(t *testing.T) {
	listEnv(t)
	if _, err := runIn(t, "", "show", "pfd"); err == nil || !strings.Contains(err.Error(), "pdf") {
		t.Errorf("err = %v", err)
	}
}

func TestSearch(t *testing.T) {
	f := listEnv(t)
	t.Chdir(f.app)
	golden(t, "search", f.run(t, "search", "e"))
	var got struct{ Skills []foundSkill }
	if err := json.Unmarshal([]byte(runOK(t, "search", "TDD", "--json")), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Skills) != 1 || got.Skills[0].ID != "tdd" || got.Skills[0].Here != "project" {
		t.Errorf("%+v", got.Skills)
	}
	if out := f.run(t, "search", "nothing", "here"); out != "No skill matches \"nothing here\".\n" {
		t.Errorf("no match: %q", out)
	}
}
