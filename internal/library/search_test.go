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
