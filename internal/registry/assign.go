package registry

import (
	"fmt"
	"sort"
)

// Assign gives the project the skills. The project must already be known
// (see EnsureProject).
func (r *Registry) Assign(projectID string, ids ...string) error {
	p, ok := r.Projects[projectID]
	if !ok {
		return fmt.Errorf("unknown project %q", projectID)
	}
	p.Skills = sortedSet(append(p.Skills, ids...))
	return nil
}

// Unassign takes the skills away from the project, including their
// always-on mark. It returns the IDs that were not assigned.
func (r *Registry) Unassign(projectID string, ids ...string) (notAssigned []string, err error) {
	p, ok := r.Projects[projectID]
	if !ok {
		return nil, fmt.Errorf("unknown project %q", projectID)
	}
	notAssigned = missing(p.Skills, ids...)
	p.Skills = without(p.Skills, ids...)
	p.AlwaysOn = without(p.AlwaysOn, ids...)
	return notAssigned, nil
}

// AssignGlobal makes the skills available in every folder.
func (r *Registry) AssignGlobal(ids ...string) {
	r.Global = sortedSet(append(r.Global, ids...))
}

// UnassignGlobal removes global skills, including their always-on mark. It
// returns the IDs that were not global.
func (r *Registry) UnassignGlobal(ids ...string) (notAssigned []string) {
	notAssigned = missing(r.Global, ids...)
	r.Global = without(r.Global, ids...)
	r.AlwaysOn = without(r.AlwaysOn, ids...)
	return notAssigned
}

// Served returns the skills served in the project: the global skills plus
// the project's own, sorted and without duplicates. An unknown or empty
// projectID gets only the global skills.
func (r *Registry) Served(projectID string) []string {
	ids := append([]string{}, r.Global...)
	if p, ok := r.Projects[projectID]; ok {
		ids = append(ids, p.Skills...)
	}
	return sortedSet(ids)
}

// ProjectsUsing returns the IDs of the projects that have the skill
// assigned, sorted. Global use is not included; check r.Global for that.
func (r *Registry) ProjectsUsing(skillID string) []string {
	var out []string
	for id, p := range r.Projects {
		if len(missing(p.Skills, skillID)) == 0 {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
