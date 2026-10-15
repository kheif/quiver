package registry

import "sort"

// sortedSet returns the items sorted and without duplicates. It never
// returns nil, so lists are written as [] instead of null.
func sortedSet(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, s := range items {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// without returns items minus every entry in drop.
func without(items []string, drop ...string) []string {
	skip := map[string]bool{}
	for _, d := range drop {
		skip[d] = true
	}
	out := make([]string, 0, len(items))
	for _, s := range items {
		if !skip[s] {
			out = append(out, s)
		}
	}
	return out
}

// missing returns the entries of ids that are not in items.
func missing(items []string, ids ...string) []string {
	have := map[string]bool{}
	for _, s := range items {
		have[s] = true
	}
	var out []string
	for _, id := range ids {
		if !have[id] {
			out = append(out, id)
		}
	}
	return out
}
