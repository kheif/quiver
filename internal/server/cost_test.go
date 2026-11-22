package server

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kheif/quiver/internal/registry"
)

func costOf(t *testing.T, projectID string) CatalogCost {
	t.Helper()
	return Cost(mustRegistry(t), mustLibrary(t), projectID)
}

func TestCostOfAnEmptyFolder(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	if c := costOf(t, ""); c.Listed != 0 || c.ListedChars != 0 || c.AlwaysOn != 0 || c.Searchable != 0 || len(c.Largest) != 0 {
		t.Errorf("cost %+v", c)
	}
}

// The figure is the length of the catalog lines that a session receives, and
// the always-on skills are counted apart.
func TestCostEqualsTheLinesOfTheSession(t *testing.T) {
	sessionEnv(t, 6, 2, 80)
	if err := registry.Update(func(r *registry.Registry) error { r.AssignSearchableGlobal("loose"); return nil }); err != nil {
		t.Fatal(err)
	}
	c := costOf(t, "")
	if c.Listed != 4 || c.AlwaysOn != 2 || c.AlwaysOnChars == 0 {
		t.Fatalf("cost %+v", c)
	}
	text := sessionText(t, "")
	_, after, _ := strings.Cut(text, skillsIntro)
	chars, lines := 0, 0
	for _, l := range strings.SplitAfter(after, "\n") {
		if strings.HasPrefix(l, "- ") {
			chars += utf8.RuneCountInString(l)
			lines++
		}
	}
	if lines != c.Listed || chars != c.ListedChars {
		t.Errorf("session has %d lines of %d characters, cost says %d of %d", lines, chars, c.Listed, c.ListedChars)
	}
	if c.ListedTokens != (c.ListedChars+3)/4 || c.AlwaysOnTokens != (c.AlwaysOnChars+3)/4 {
		t.Errorf("tokens %+v", c)
	}
	// The always-on text is the size of the units before the split.
	if c.AlwaysOnChars < 2*80 {
		t.Errorf("always-on size %d", c.AlwaysOnChars)
	}
}

func TestCostCountsGlobalAndProjectAndSearchable(t *testing.T) {
	w, _ := skillsEnv(t)
	if err := registry.Update(func(r *registry.Registry) error { r.AssignSearchableGlobal("draft"); return nil }); err != nil {
		t.Fatal(err)
	}
	// pdf is global, db-migration is the project's, gone is not in the library.
	c := costOf(t, w.ProjectID())
	if c.Listed != 2 || c.Searchable != 1 {
		t.Errorf("cost %+v", c)
	}
	global := costOf(t, "")
	if global.Listed != 1 {
		t.Errorf("global cost %+v", global)
	}
	byID := map[string]string{}
	for _, e := range c.Largest {
		byID[e.ID] = e.Scope
	}
	if byID["pdf"] != "global" || byID["db-migration"] != "project" {
		t.Errorf("largest %+v", c.Largest)
	}
}

func TestCostNamesTheTenLongestFirst(t *testing.T) {
	sessionEnv(t, 14, 0, 10)
	c := costOf(t, "")
	if c.Listed != 14 || len(c.Largest) != 10 {
		t.Fatalf("%d listed, %d largest", c.Listed, len(c.Largest))
	}
	for i := 1; i < len(c.Largest); i++ {
		if c.Largest[i].Chars > c.Largest[i-1].Chars {
			t.Errorf("not longest first: %+v", c.Largest)
		}
	}
}

// A catalog line is made without building the text of the skill, which reads
// its files.
func TestCatalogLineReadsNoFile(t *testing.T) {
	sessionEnv(t, 2, 0, 20)
	cache := textCache{}
	for _, sk := range mustLibrary(t).All() {
		if line := cache.line(sk); !strings.HasPrefix(line, "- "+sk.ID+": ") {
			t.Errorf("line %q", line)
		}
	}
	if len(cache) != 0 {
		t.Errorf("the lines built the text of %d skills", len(cache))
	}
}
