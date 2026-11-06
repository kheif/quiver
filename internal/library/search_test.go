package library

import (
	"slices"
	"testing"

	"github.com/kheif/quiver/internal/skill"
)

func TestSearch(t *testing.T) {
	l := &Library{skills: map[string]*skill.Skill{}}
	for id, s := range map[string][2]string{
		"pdf":             {"pdf", "Read PDF files."},
		"anthropic/pdf":   {"pdf", "Fill in PDF forms."},
		"pdf-merge":       {"pdf-merge", "Merge files."},
		"docs/to-pdf":     {"to-pdf", "Convert documents."},
		"acme/pdfx":       {"export", "Export reports."}, // the ID contains pdf
		"notes":           {"notes", "Keep notes, also as PDF."},
		"review":          {"review", "Review pull requests."},
		"Upper/Case-Name": {"Case-Name", "Mixed case."},
	} {
		l.skills[id] = &skill.Skill{ID: id, Name: s[0], Description: s[1]}
	}
	ids := func(ms []Match) []string {
		var out []string
		for _, m := range ms {
			out = append(out, m.ID)
		}
		return out
	}
	got := l.Search(" PDF ")
	want := []string{"anthropic/pdf", "pdf", "pdf-merge", "acme/pdfx", "docs/to-pdf", "notes"}
	if !slices.Equal(ids(got), want) {
		t.Errorf("got %v\nwant %v", ids(got), want)
	}
	ranks := []int{RankExactName, RankExactName, RankNamePrefix, RankNameContains, RankNameContains, RankDescription}
	for i, m := range got {
		if m.Rank != ranks[i] {
			t.Errorf("%s: rank %d, want %d", m.ID, m.Rank, ranks[i])
		}
	}
	if got := ids(l.Search("case-name")); !slices.Equal(got, []string{"Upper/Case-Name"}) {
		t.Errorf("case: %v", got)
	}
	if got := l.Search("  "); got != nil {
		t.Errorf("empty: %v", ids(got))
	}
	if got := l.Search("nothing"); len(got) != 0 {
		t.Errorf("no match: %v", ids(got))
	}
}

func TestSearchWordsInAnyOrder(t *testing.T) {
	l := &Library{skills: map[string]*skill.Skill{}}
	for id, s := range map[string][2]string{
		"team/pull-request":        {"pull-request", "Open a pull request."},
		"team/review-pull-request": {"review-pull-request", "Review a change."},
		"sql-optimizer":            {"sql-optimizer", "Finds slow SQL queries. Use when a database query is slow."},
		"notes":                    {"notes", "Keep notes."},
	} {
		l.skills[id] = &skill.Skill{ID: id, Name: s[0], Description: s[1]}
	}
	ids := func(q string) []string {
		var out []string
		for _, m := range l.Search(q) {
			out = append(out, m.ID)
		}
		return out
	}
	for q, want := range map[string][]string{
		// A space finds a hyphen, and the whole text still ranks first.
		"pull request": {"team/pull-request", "team/review-pull-request"},
		// Words in any order match a name or ID.
		"request pull": {"team/pull-request", "team/review-pull-request"},
		// Words spread over the description.
		"database slow":    {"sql-optimizer"},
		"my database slow": nil,
		"slow sql":         {"sql-optimizer"},
		"sql notes":        nil,
	} {
		if got := ids(q); !slices.Equal(got, want) {
			t.Errorf("%q: got %v, want %v", q, got, want)
		}
	}
}
