package library

import (
	"cmp"
	"slices"
	"strings"

	"github.com/kheif/quiver/internal/skill"
)

// How well a skill matches a search; lower is better.
const (
	RankExactName = iota
	RankNamePrefix
	RankNameContains
	RankDescription
)

// Match is a skill found by Search.
type Match struct {
	*skill.Skill
	Rank int
}

// fold lower-cases s and treats hyphens and underscores like spaces, so
// "pull request" finds pull-request.
func fold(s string) string {
	return strings.NewReplacer("-", " ", "_", " ").Replace(strings.ToLower(s))
}

// Search returns the skills whose ID, name or description contain text,
// ignoring case, hyphens and underscores: exact names first, then names
// that start with text, then names or IDs that contain it, then
// descriptions. When the whole text is not found, a skill still matches if
// every word of it is, in any order: in its name or ID (ranked with the
// names that contain the text), or anywhere in its name, ID and description
// (ranked with the descriptions). Matches of one rank are sorted by ID.
// Empty text matches nothing.
func (l *Library) Search(text string) []Match {
	q := fold(strings.TrimSpace(text))
	if q == "" {
		return nil
	}
	words := strings.Fields(q)
	containsAll := func(s string) bool {
		for _, w := range words {
			if !strings.Contains(s, w) {
				return false
			}
		}
		return true
	}
	var out []Match
	for _, s := range l.skills {
		name, id, desc := fold(s.Name), fold(s.ID), fold(s.Description)
		rank := -1
		switch {
		case name == q || id == q:
			rank = RankExactName
		case strings.HasPrefix(name, q):
			rank = RankNamePrefix
		case strings.Contains(name, q) || strings.Contains(id, q):
			rank = RankNameContains
		case strings.Contains(desc, q):
			rank = RankDescription
		case containsAll(name) || containsAll(id):
			rank = RankNameContains
		case containsAll(name + " " + id + " " + desc):
			rank = RankDescription
		}
		if rank >= 0 {
			out = append(out, Match{s, rank})
		}
	}
	slices.SortFunc(out, func(a, b Match) int {
		return cmp.Or(cmp.Compare(a.Rank, b.Rank), cmp.Compare(a.ID, b.ID))
	})
	return out
}
