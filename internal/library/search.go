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

// Search returns the skills whose ID, name or description contain text,
// ignoring case: exact names first, then names that start with text, then
// names or IDs that contain it, then descriptions. Matches of one rank
// are sorted by ID. Empty text matches nothing.
func (l *Library) Search(text string) []Match {
	q := strings.ToLower(strings.TrimSpace(text))
	if q == "" {
		return nil
	}
	var out []Match
	for _, s := range l.skills {
		name, id := strings.ToLower(s.Name), strings.ToLower(s.ID)
		rank := -1
		switch {
		case name == q || id == q:
			rank = RankExactName
		case strings.HasPrefix(name, q):
			rank = RankNamePrefix
		case strings.Contains(name, q) || strings.Contains(id, q):
			rank = RankNameContains
		case strings.Contains(strings.ToLower(s.Description), q):
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
