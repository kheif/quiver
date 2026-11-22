package library

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/skill"
)

// libraryOf makes a library of the given skills, each as id: {name, description}.
func libraryOf(skills map[string][2]string) *Library {
	l := &Library{skills: map[string]*skill.Skill{}}
	for id, s := range skills {
		l.skills[id] = &skill.Skill{ID: id, Name: s[0], Description: s[1]}
	}
	return l
}

// withFiller adds n skills that match nothing of the tests, so that the
// statistics of a library look like those of a real one.
func withFiller(skills map[string][2]string, n int) map[string][2]string {
	for i := range n {
		skills[fmt.Sprintf("filler/skill-%02d", i)] = [2]string{fmt.Sprintf("skill-%02d", i), fmt.Sprintf("Handles filler task number %d for the team.", i)}
	}
	return skills
}

func ids(ms []Match) []string {
	var out []string
	for _, m := range ms {
		out = append(out, m.ID)
	}
	return out
}

func find(t *testing.T, l *Library, query string) []Match {
	t.Helper()
	got, err := NewWords(l.All()).Search(context.Background(), query, l.All())
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestSearchTiers(t *testing.T) {
	l := libraryOf(map[string][2]string{
		"pdf":             {"pdf", "Read PDF files."},
		"anthropic/pdf":   {"pdf", "Fill in PDF forms."},
		"pdf-merge":       {"pdf-merge", "Merge files."},
		"docs/to-pdf":     {"to-pdf", "Convert documents."},
		"acme/pdfx":       {"export", "Export reports."}, // the ID contains pdf
		"notes":           {"notes", "Keep notes, also as PDF."},
		"review":          {"review", "Review pull requests."},
		"Upper/Case-Name": {"Case-Name", "Mixed case."},
	})
	got := find(t, l, " PDF ")
	// The description of notes mentions PDF once: below the cutoff, so a
	// single word in a description alone does not match.
	want := []string{"anthropic/pdf", "pdf", "pdf-merge", "acme/pdfx", "docs/to-pdf"}
	if !slices.Equal(ids(got), want) {
		t.Errorf("got %v\nwant %v", ids(got), want)
	}
	ranks := []int{RankExactName, RankExactName, RankNamePrefix, RankNameContains, RankNameContains}
	for i, m := range got {
		if m.Rank != ranks[i] {
			t.Errorf("%s: rank %d, want %d", m.ID, m.Rank, ranks[i])
		}
	}
	if got := ids(find(t, l, "case-name")); !slices.Equal(got, []string{"Upper/Case-Name"}) {
		t.Errorf("case: %v", got)
	}
	if got := ids(find(t, l, "CASE NAME")); !slices.Equal(got, []string{"Upper/Case-Name"}) {
		t.Errorf("case and space: %v", got)
	}
	if got := find(t, l, "  "); got != nil {
		t.Errorf("empty: %v", ids(got))
	}
	if got := find(t, l, "nothing"); len(got) != 0 {
		t.Errorf("no match: %v", ids(got))
	}
}

func TestSearchPartialWordOfAName(t *testing.T) {
	l := libraryOf(withFiller(map[string][2]string{
		"acme/pull-request-review": {"pull-request-review", "Review a change."},
	}, 40))
	if got := ids(find(t, l, "reques")); !slices.Equal(got, []string{"acme/pull-request-review"}) {
		t.Errorf("partial word: %v", got)
	}
}

func TestSearchWordsInAnyOrder(t *testing.T) {
	l := libraryOf(withFiller(map[string][2]string{
		"team/pull-request":        {"pull-request", "Open a pull request."},
		"team/review-pull-request": {"review-pull-request", "Review a change."},
		"notes":                    {"notes", "Keep notes."},
	}, 40))
	for q, want := range map[string][]string{
		// A space finds a hyphen, and the whole text still ranks first.
		"pull request": {"team/pull-request", "team/review-pull-request"},
		// Words in any order match a name or ID.
		"request pull": {"team/pull-request", "team/review-pull-request"},
		"notes pull":   nil,
	} {
		if got := ids(find(t, l, q)); !slices.Equal(got, want) {
			t.Errorf("%q: got %v, want %v", q, got, want)
		}
	}
}

// A library with one skill that fits a query by its description alone.
func descriptionLibrary() *Library {
	return libraryOf(withFiller(map[string][2]string{
		"ops/db-tuning":  {"db-tuning", "Finds slow SQL queries and suggests indexes. Use when a database report takes too long."},
		"ops/log-triage": {"log-triage", "Groups error messages by cause when a deploy goes wrong."},
		"notes":          {"notes", "Keep meeting notes."},
	}, 80))
}

func TestSearchRanksByDescription(t *testing.T) {
	l := descriptionLibrary()
	got := find(t, l, "my database report takes too long because of slow sql queries")
	if len(got) != 1 || got[0].ID != "ops/db-tuning" || got[0].Rank != RankDescription || got[0].Score <= descriptionCutoff {
		t.Fatalf("got %+v", got)
	}
}

func TestSearchCutoff(t *testing.T) {
	l := descriptionLibrary()
	// A query that fits nothing returns nothing, even when one word is shared.
	for _, q := range []string{"plan a birthday party for twenty guests", "reports", "slow"} {
		if got := find(t, l, q); len(got) != 0 {
			t.Errorf("%q: %v", q, ids(got))
		}
	}
	// A long sentence does not match everything: words that most skills
	// contain count little, and the cutoff holds the rest back.
	long := strings.Repeat("handles the filler task for the team and the notes ", 8)
	if got := find(t, l, long); len(got) > 3 {
		t.Errorf("a long sentence matches %d skills", len(got))
	}
}

func TestSearchScoreIsTheSameInAnySet(t *testing.T) {
	l := descriptionLibrary()
	all := l.All()
	w := NewWords(all)
	q := "my database report takes too long because of slow sql queries"
	whole, _ := w.Search(context.Background(), q, all)
	var small []*skill.Skill
	for _, s := range all {
		if s.ID == "ops/db-tuning" || s.ID == "notes" {
			small = append(small, s)
		}
	}
	part, _ := w.Search(context.Background(), q, small)
	if len(whole) != 1 || len(part) != 1 || whole[0].Score != part[0].Score {
		t.Errorf("whole %+v, small set %+v", whole, part)
	}
	if got, _ := w.Search(context.Background(), q, nil); len(got) != 0 {
		t.Errorf("empty set: %v", ids(got))
	}
}

func TestSearchTiesBreakByID(t *testing.T) {
	l := libraryOf(withFiller(map[string][2]string{
		"b/twin": {"twin-tool", "Repairs broken sprockets in the assembly line."},
		"a/twin": {"twin-tool", "Repairs broken sprockets in the assembly line."},
	}, 80))
	got := find(t, l, "twin")
	if want := []string{"a/twin", "b/twin"}; !slices.Equal(ids(got), want) {
		t.Errorf("name tier: %v, want %v", ids(got), want)
	}
	got = find(t, l, "repair broken sprockets assembly line")
	if want := []string{"a/twin", "b/twin"}; !slices.Equal(ids(got), want) {
		t.Errorf("score tie: %v, want %v", ids(got), want)
	}
}

func TestSearchFoldsCaseAndTerms(t *testing.T) {
	l := descriptionLibrary()
	a := ids(find(t, l, "My Database Report Takes TOO Long, because of SLOW SQL queries!"))
	b := ids(find(t, l, "my database report takes too long because of slow sql queries"))
	if !slices.Equal(a, b) || len(a) != 1 {
		t.Errorf("case and punctuation: %v and %v", a, b)
	}
	// A plural finds the singular, as the ranking stems.
	if got := ids(find(t, l, "my database reports take too long because of slow sql query")); len(got) != 1 {
		t.Errorf("stemming: %v", got)
	}
}

type brokenEngine struct{}

func (brokenEngine) Search(context.Context, string, []*skill.Skill) ([]Match, error) {
	return nil, errors.New("model file is missing")
}

func TestSearchFallsBackToWords(t *testing.T) {
	engines["broken"] = func(*Library) (Searcher, error) { return brokenEngine{}, nil }
	defer delete(engines, "broken")
	l := libraryOf(map[string][2]string{"pdf": {"pdf", "Read PDF files."}})
	s, err := l.NewSearch("broken")
	if err != nil {
		t.Fatal(err)
	}
	got, note := s.Find(context.Background(), "pdf", l.All())
	if len(got) != 1 || !strings.Contains(note, "broken cannot run (model file is missing); searched with words") {
		t.Errorf("got %v, note %q", ids(got), note)
	}
}

func TestSearchEngineThatCannotBeBuiltFallsBack(t *testing.T) {
	engines["unbuilt"] = func(*Library) (Searcher, error) { return nil, errors.New("no model") }
	defer delete(engines, "unbuilt")
	l := libraryOf(map[string][2]string{"pdf": {"pdf", "Read PDF files."}})
	s, err := l.NewSearch("unbuilt")
	if err != nil {
		t.Fatal(err)
	}
	if s.Engine != EngineWords || !strings.Contains(s.Problem, "unbuilt cannot run (no model); searching with words") {
		t.Errorf("engine %q, problem %q", s.Engine, s.Problem)
	}
	if got, note := s.Find(context.Background(), "pdf", l.All()); len(got) != 1 || note != "" {
		t.Errorf("got %v, note %q", ids(got), note)
	}
}

func TestNewSearchUnknownOverride(t *testing.T) {
	l := libraryOf(nil)
	if _, err := l.NewSearch("vectors"); err == nil || !strings.Contains(err.Error(), `unknown search engine "vectors": use words`) {
		t.Errorf("err = %v", err)
	}
}
