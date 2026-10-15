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
