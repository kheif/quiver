package server

import (
	"cmp"
	"slices"
	"unicode/utf8"

	"github.com/kheif/quiver/internal/library"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/skill"
)

// maxLargest is how many listed skills Cost names as the most expensive.
const maxLargest = 10

// ListedEntry is one listed skill and what its line costs.
type ListedEntry struct {
	ID    string
	Scope string // global or project
	Chars int    // characters of its catalog line
}

// CatalogCost is what a folder's assignments cost at the start of a session.
// It is made from the same builders as the session-start text, so the figures
// cannot drift from what a session receives. The token figures are an
// estimate: the characters divided by four.
type CatalogCost struct {
	// Listed skills are on demand: one catalog line each.
	Listed       int
	ListedChars  int
	ListedTokens int
	// AlwaysOn skills are loaded in full, counted apart, before the split into
	// parts.
	AlwaysOn       int
	AlwaysOnChars  int
	AlwaysOnTokens int
	// Searchable skills add nothing to the session-start text.
	Searchable int
	// Largest are the listed skills with the longest lines, the longest first.
	Largest []ListedEntry
}

// EstimateTokens is the estimate used for every token figure: characters
// divided by four, rounded up.
func EstimateTokens(chars int) int { return (chars + 3) / 4 }

// Cost reports what the folder's assignments add to the session-start text. A
// skill that is assigned but not in the library, or that cannot be read, is
// left out, as BuildSessionContext leaves it out.
func Cost(r *registry.Registry, l *library.Library, projectID string) CatalogCost {
	var c CatalogCost
	c.Searchable = len(r.SearchableServed(projectID))
	cache := textCache{}
	loaded := map[string]bool{}
	for _, id := range r.AlwaysOnInOrder(projectID) {
		sk, err := l.Get(id)
		if err != nil {
			continue
		}
		b := cache.get(sk)
		if b.err != nil {
			continue
		}
		loaded[id] = true
		c.AlwaysOn++
		c.AlwaysOnChars += utf8.RuneCountInString("\n## " + oneLine(id) + "\n" + b.text)
	}
	var listed []*skill.Skill
	for _, id := range r.Served(projectID) {
		if loaded[id] {
			continue
		}
		sk, err := l.Get(id)
		if err != nil {
			continue
		}
		listed = append(listed, sk)
		scope := "project"
		if slices.Contains(r.Global, id) {
			scope = "global"
		}
		c.Largest = append(c.Largest, ListedEntry{ID: id, Scope: scope, Chars: utf8.RuneCountInString(CatalogLine(sk))})
	}
	c.Listed = len(listed)
	c.ListedChars = CatalogChars(listed)
	c.ListedTokens = EstimateTokens(c.ListedChars)
	c.AlwaysOnTokens = EstimateTokens(c.AlwaysOnChars)
	slices.SortFunc(c.Largest, func(a, b ListedEntry) int {
		return cmp.Or(cmp.Compare(b.Chars, a.Chars), cmp.Compare(a.ID, b.ID))
	})
	if len(c.Largest) > maxLargest {
		c.Largest = c.Largest[:maxLargest]
	}
	return c
}
