package cli

import (
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/assistant"
	"github.com/kheif/quiver/internal/assistant/assistanttest"
	"github.com/kheif/quiver/internal/registry"
)

// bigSkills makes n local skills of about 4,000 characters each and returns
// their ids. They are not assigned.
func bigSkills(t *testing.T, n int) []string {
	t.Helper()
	var ids []string
	for i := range n {
		id := "big" + string(rune('a'+i))
		writeBody(t, id, strings.Repeat("x", 3990)+"\n")
		ids = append(ids, id)
	}
	return ids
}

func claudeParts(t *testing.T) int {
	t.Helper()
	n, err := assistant.ClaudeHookParts()
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// quiver install gives Claude Code as many hook commands as the text of the
// skills takes.
func TestInstallWritesAsManyHooksAsTheTextTakes(t *testing.T) {
	assignEnv(t, "biga", "bigb", "bigc")
	bigSkills(t, 3)
	if _, err := runIn(t, "", "add", "biga", "bigb", "bigc", "--global", "--always"); err != nil {
		t.Fatal(err)
	}
	assistanttest.Install(t)
	appHome(t)
	fakeExecutable(t, "/opt/bin/quiver")
	if _, err := runIn(t, "", "install", "claude-code"); err != nil {
		t.Fatal(err)
	}
	// Three skills of 4,000 characters do not fit one command of 10,000.
	if n := claudeParts(t); n != 2 {
		t.Errorf("%d hook commands", n)
	}
}

// A command that changes skills gives the assistants as many hook commands as
// the text takes then, in both directions, and leaves a hook that runs another
// quiver alone.
func TestSkillCommandsKeepHooksSized(t *testing.T) {
	assignEnv(t, "biga", "bigb", "bigc", "bigd", "bige")
	bigSkills(t, 5)
	registered(t) // one hook command, running /opt/bin/quiver
	if n := claudeParts(t); n != 1 {
		t.Fatalf("start: %d", n)
	}
	if _, err := runIn(t, "", "add", "biga", "bigb", "bigc", "bigd", "bige", "--global", "--always"); err != nil {
		t.Fatal(err)
	}
	if n := claudeParts(t); n != 3 {
		t.Errorf("after add --always: %d, want 3", n)
	}
	if _, err := runIn(t, "", "remove", "bigc", "bigd", "bige", "--global"); err != nil {
		t.Fatal(err)
	}
	if n := claudeParts(t); n != 1 {
		t.Errorf("after remove: %d, want 1", n)
	}
	// A hook that runs another quiver is not ours to change.
	if _, err := assistant.InstallClaudeHooks("/old/quiver", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := runIn(t, "", "add", "bigc", "bigd", "bige", "--global", "--always"); err != nil {
		t.Fatal(err)
	}
	if n := claudeParts(t); n != 1 {
		t.Errorf("another quiver's hook changed: %d", n)
	}
	// A command that only reads does not touch hooks.
	if _, err := assistant.InstallClaudeHooks("/opt/bin/quiver", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := runIn(t, "", "list"); err != nil {
		t.Fatal(err)
	}
	if n := claudeParts(t); n != 1 {
		t.Errorf("list changed the hooks: %d", n)
	}
}

// A project's own always-on skills count: the hook commands are as many as
// the largest folder needs.
func TestProjectAlwaysOnGrowsHooks(t *testing.T) {
	dir := assignEnv(t, "biga", "bigb", "bigc")
	bigSkills(t, 3)
	registered(t)
	if _, err := runIn(t, "", "add", "biga", "bigb", "bigc", "--project", dir, "--always", "--yes"); err != nil {
		t.Fatal(err)
	}
	if n := claudeParts(t); n != 2 {
		t.Errorf("%d hook commands for a project with 12,000 characters", n)
	}
}

// doctor warns when the hook has fewer commands than the text takes.
func TestDoctorWarnsWhenHooksAreShort(t *testing.T) {
	assignEnv(t, "biga", "bigb", "bigc")
	bigSkills(t, 3)
	registered(t)
	// Marked always-on without a quiver command, so no sync runs.
	err := registry.Update(func(r *registry.Registry) error {
		for _, id := range []string{"biga", "bigb", "bigc"} {
			r.AssignGlobal(id)
			if err := r.MarkAlwaysOnGlobal(id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	out, _ := runIn(t, "", "doctor")
	wantLines(t, out, "warn   the Claude Code session-start hook has 1 command, but the text of your skills takes 2, so the rest does not load; run: quiver install claude-code")
	if _, err := runIn(t, "", "install", "claude-code"); err != nil {
		t.Fatal(err)
	}
	out, _ = runIn(t, "", "doctor")
	if strings.Contains(out, "does not load") {
		t.Errorf("still warns:\n%s", out)
	}
}
