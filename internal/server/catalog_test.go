package server

import (
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/registry"
)

func TestInstructions(t *testing.T) {
	w, _ := skillsEnv(t)
	want := usage + createHint
	for _, project := range []bool{true, false} {
		if !project {
			w.Project = nil
		}
		got := connect(t, New(Options{Version: "dev", Log: discard, Where: w})).InitializeResult().Instructions
		if got != want {
			t.Errorf("project %v: got %q\nwant %q", project, got, want)
		}
	}
	// The catalog is the hook's: the instructions name no skill, and stay far
	// below the 4096 characters Claude Code keeps.
	if strings.Contains(want, "pdf") || len(want) > 1500 {
		t.Errorf("instructions are %d bytes: %q", len(want), want)
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
	const rule = ". If the conversation already has a <quiver-always-on> block, these skills are loaded: follow them. Otherwise call the get_skill tool of this server for each before your first reply and follow it for the whole conversation.\n\n"
	got := connect(t, New(Options{Version: "dev", Log: discard, Where: w})).InitializeResult().Instructions
	if want := usage + "Always-on skills: db-migration, pdf" + rule + createHint; got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}

	w.Project = nil
	got = connect(t, New(Options{Version: "dev", Log: discard, Where: w})).InitializeResult().Instructions
	if want := usage + "Always-on skills: pdf" + rule + createHint; got != want {
		t.Errorf("no project: got %q", got)
	}
}

func TestInstructionsNoSkills(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	if got := connect(t, New(Options{Version: "dev", Log: discard})).InitializeResult().Instructions; got != usage+createHint {
		t.Errorf("got %q", got)
	}
}

func TestShortenDescriptions(t *testing.T) {
	long := strings.Repeat("é", maxDescription+1)
	if got, want := shorten("Multi\nline   text."), "Multi line text."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := shorten(long), strings.Repeat("é", maxDescription-1)+"…"; got != want {
		t.Errorf("long: got %d bytes", len(got))
	}
}
