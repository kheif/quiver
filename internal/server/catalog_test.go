package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/skill"
)

const header = "Skills (reusable instructions for specific tasks). Before starting a task, check whether one fits. If it does, call the get_skill tool of this server with its id and follow it.\n"

func skills(descs ...string) []servedSkill {
	var out []servedSkill
	for i, d := range descs {
		out = append(out, servedSkill{Skill: &skill.Skill{ID: string(rune('a' + i)), Description: d}})
	}
	return out
}

func TestInstructions(t *testing.T) {
	w, _ := skillsEnv(t)
	got := connect(t, New(Options{Version: "dev", Log: discard, Where: w})).InitializeResult().Instructions
	want := header + "- db-migration: Rules for database migrations.\n- pdf: Read, create and edit PDF files.\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}

	w.Project = nil
	got = connect(t, New(Options{Version: "dev", Log: discard, Where: w})).InitializeResult().Instructions
	if want := header + "- pdf: Read, create and edit PDF files.\n"; got != want {
		t.Errorf("no project: got %q", got)
	}
}

func TestInstructionsNoSkills(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	if got := connect(t, New(Options{Version: "dev", Log: discard})).InitializeResult().Instructions; got != "" {
		t.Errorf("got %q", got)
	}
}

func TestCatalogShortensDescriptions(t *testing.T) {
	long := strings.Repeat("é", 200)
	got := catalog(skills("Multi\nline   text.", long), 10000)
	want := header + "- a: Multi line text.\n- b: " + strings.Repeat("é", 149) + "…\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestCatalogBudget(t *testing.T) {
	check := func(skills []servedSkill, budget int, want string) {
		t.Helper()
		if got := catalog(skills, budget); got != header+want {
			t.Errorf("budget %d: got %q\nwant %q", budget, got, header+want)
		}
	}
	// Descriptions are dropped from the end.
	two := skills("A.", "B.")
	check(two, len(header+"- a: A.\n- b: B.\n"), "- a: A.\n- b: B.\n")
	check(two, len(header+"- a: A.\n- b: B.\n")-1, "- a: A.\n- b\n")
	check(two, len(header+"- a\n- b\n"), "- a\n- b\n")

	// When the IDs do not fit either, the list ends with a pointer.
	var three []servedSkill
	for _, c := range "abc" {
		three = append(three, servedSkill{Skill: &skill.Skill{ID: strings.Repeat(string(c), 40)}})
	}
	a, b := "- "+strings.Repeat("a", 40)+"\n", "- "+strings.Repeat("b", 40)+"\n"
	check(three, len(header+a+b+"…and 1 more: call list_skills\n"), a+b+"…and 1 more: call list_skills\n")
	check(three, len(header+a+"…and 2 more: call list_skills\n"), a+"…and 2 more: call list_skills\n")
	check(three, len(header+a+"…and 2 more: call list_skills\n")-1, "…and 3 more: call list_skills\n")

	if got := catalog(nil, 3800); got != "" {
		t.Errorf("no skills: %q", got)
	}
}

func TestCatalogManySkills(t *testing.T) {
	var many []servedSkill
	for i := range 500 {
		many = append(many, servedSkill{Skill: &skill.Skill{ID: fmt.Sprintf("source/skill-%03d", i), Description: strings.Repeat("x", 300)}})
	}
	got := catalog(many, maxInstructions)
	if len(got) > 3800 {
		t.Errorf("%d bytes", len(got))
	}
	listed := strings.Count(got, "\n- ")
	if listed == 0 || !strings.HasSuffix(got, fmt.Sprintf("\n…and %d more: call list_skills\n", 500-listed)) {
		t.Errorf("%d listed, ends %q", listed, got[max(0, len(got)-60):])
	}
	if strings.Contains(got, "xxx") {
		t.Error("descriptions kept")
	}
}
