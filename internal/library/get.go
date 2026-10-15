package library

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/kheif/quiver/internal/skill"
)

// ErrSkillNotFound matches the error Get returns for an unknown ID.
var ErrSkillNotFound = errors.New("skill not found")

// NotFoundError is returned by Get. It matches ErrSkillNotFound.
type NotFoundError struct {
	ID          string
	Suggestions []string
}

func (e *NotFoundError) Error() string {
	msg := fmt.Sprintf("skill %q not found", e.ID)
	if len(e.Suggestions) > 0 {
		msg += "; did you mean " + strings.Join(e.Suggestions, ", ") + "?"
	}
	return msg
}

func (e *NotFoundError) Is(target error) bool { return target == ErrSkillNotFound }

// maxSuggestions caps the "did you mean" list.
const maxSuggestions = 3

// Get returns the skill with the ID, or a *NotFoundError with close IDs.
func (l *Library) Get(id string) (*skill.Skill, error) {
	if s, ok := l.skills[id]; ok {
		return s, nil
	}
	return nil, &NotFoundError{ID: id, Suggestions: l.suggest(id)}
}

// suggest returns up to maxSuggestions IDs close to id, closest first. An ID
// is close when its whole text or its last part (after "/") is within a
// small edit distance, so "pdf" finds "anthropic/pdf".
func (l *Library) suggest(id string) []string {
	type match struct {
		id   string
		dist int
	}
	limit := max(2, len(id)/5)
	var matches []match
	for known := range l.skills {
		d := distance(id, known)
		if i := strings.LastIndex(known, "/"); i >= 0 {
			d = min(d, distance(id, known[i+1:]))
		}
		if d <= limit {
			matches = append(matches, match{known, d})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].dist != matches[j].dist {
			return matches[i].dist < matches[j].dist
		}
		return matches[i].id < matches[j].id
	})
	var out []string
	for i := 0; i < len(matches) && i < maxSuggestions; i++ {
		out = append(out, matches[i].id)
	}
	return out
}

// distance is the Levenshtein edit distance between a and b.
func distance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}
