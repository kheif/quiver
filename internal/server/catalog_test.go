package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/registry"
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
	want := header + "- db-migration: Rules for database migrations.\n- pdf: Read, create and edit PDF files.\n\n" + createHint
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}

	w.Project = nil
	got = connect(t, New(Options{Version: "dev", Log: discard, Where: w})).InitializeResult().Instructions
	if want := header + "- pdf: Read, create and edit PDF files.\n\n" + createHint; got != want {
		t.Errorf("no project: got %q", got)
	}
}

func TestInstructionsAlwaysOn(t *testing.T) {
	w, _ := skillsEnv(t)
	err := registry.Update(func(r *registry.Registry) error {
		if err := r.MarkAlwaysOnGlobal("pdf"); err != nil {
			return err
		}
		// gone is assigned but not in the library, so it is left out.
		return r.MarkAlwaysOn(w.ProjectID(), "db-migration", "gone")
	})
	if err != nil {
		t.Fatal(err)
	}
	const rule = ". Before your first reply in this conversation, call the get_skill tool of this server for each always-on skill and follow it for the whole conversation.\n\n"
	catalog := header + "- db-migration: Rules for database migrations.\n- pdf: Read, create and edit PDF files.\n\n" + createHint
	got := connect(t, New(Options{Version: "dev", Log: discard, Where: w})).InitializeResult().Instructions
	if want := "Always-on skills: db-migration, pdf" + rule + catalog; got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}

	w.Project = nil
	got = connect(t, New(Options{Version: "dev", Log: discard, Where: w})).InitializeResult().Instructions
	if want := "Always-on skills: pdf" + rule + header + "- pdf: Read, create and edit PDF files.\n\n" + createHint; got != want {
		t.Errorf("no project: got %q", got)
	}
}

// Too many skills for the instructions: the always-on pointer stays first,
// then the pointer to list_skills replaces the list.
func TestInstructionsTooManySkills(t *testing.T) {
	var many []servedSkill
	for i := range 500 {
		many = append(many, servedSkill{Skill: &skill.Skill{ID: fmt.Sprintf("source/skill-%03d", i), Description: strings.Repeat("x", 300)}, AlwaysOn: i < 3})
	}
	got := buildInstructions(many)
	want := "Always-on skills: source/skill-000, source/skill-001, source/skill-002. " +
		"Before your first reply in this conversation, call the get_skill tool of this server for each always-on skill and follow it for the whole conversation.\n\n" +
		listPointer(500) + "\n" + createHint
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	if len(got) > maxInstructions {
		t.Errorf("%d bytes", len(got))
	}
}

func TestInstructionsNoSkills(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	if got := connect(t, New(Options{Version: "dev", Log: discard})).InitializeResult().Instructions; got != createHint {
		t.Errorf("got %q", got)
	}
}

func TestCatalogShortensDescriptions(t *testing.T) {
	long := strings.Repeat("é", maxDescription+1)
	got := catalog(skills("Multi\nline   text.", long), 100000)
	want := header + "- a: Multi line text.\n- b: " + strings.Repeat("é", maxDescription-1) + "…\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// The whole list or none: a list that does not fit becomes the pointer.
func TestCatalogBudget(t *testing.T) {
	two := skills("A.", "B.")
	list := header + "- a: A.\n- b: B.\n"
	if got := catalog(two, len(list)); got != list {
		t.Errorf("fits: got %q", got)
	}
	if got := catalog(two, len(list)-1); got != listPointer(2) {
		t.Errorf("one byte over: got %q", got)
	}
	if !strings.Contains(listPointer(2), "this folder has 2, too many to list here. Before your first reply in this conversation, call the list_skills tool of this server") {
		t.Errorf("pointer %q", listPointer(2))
	}
	if got := catalog(nil, 3800); got != "" {
		t.Errorf("no skills: %q", got)
	}
}
