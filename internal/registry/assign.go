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

// MarkAlwaysOn marks the project's skills always-on. Each skill must
// already be assigned to the project.
func (r *Registry) MarkAlwaysOn(projectID string, ids ...string) error {
	p, ok := r.Projects[projectID]
	if !ok {
		return fmt.Errorf("unknown project %q", projectID)
	}
	if m := missing(p.Skills, ids...); len(m) > 0 {
		return fmt.Errorf("not assigned to project %q: %v", projectID, m)
	}
	p.AlwaysOn = sortedSet(append(p.AlwaysOn, ids...))
	return nil
}

// UnmarkAlwaysOn removes the always-on mark from the project's skills and
// keeps them assigned. It returns the IDs that were not marked.
func (r *Registry) UnmarkAlwaysOn(projectID string, ids ...string) (notMarked []string, err error) {
	p, ok := r.Projects[projectID]
	if !ok {
		return nil, fmt.Errorf("unknown project %q", projectID)
	}
	notMarked = missing(p.AlwaysOn, ids...)
	p.AlwaysOn = without(p.AlwaysOn, ids...)
	return notMarked, nil
}

// MarkAlwaysOnGlobal marks global skills always-on. Each skill must already
// be global.
func (r *Registry) MarkAlwaysOnGlobal(ids ...string) error {
	if m := missing(r.Global, ids...); len(m) > 0 {
		return fmt.Errorf("not global: %v", m)
	}
	r.AlwaysOn = sortedSet(append(r.AlwaysOn, ids...))
	return nil
}

// UnmarkAlwaysOnGlobal removes the always-on mark from global skills and
// keeps them global. It returns the IDs that were not marked.
func (r *Registry) UnmarkAlwaysOnGlobal(ids ...string) (notMarked []string) {
	notMarked = missing(r.AlwaysOn, ids...)
	r.AlwaysOn = without(r.AlwaysOn, ids...)
	return notMarked
}

// Replace puts newID in the place of oldID wherever oldID is assigned or
// marked always-on: globally and in every project.
func (r *Registry) Replace(oldID, newID string) {
	swap := func(ids []string) []string {
		if len(missing(ids, oldID)) > 0 {
			return ids
		}
		return sortedSet(append(without(ids, oldID), newID))
	}
	r.Global, r.AlwaysOn = swap(r.Global), swap(r.AlwaysOn)
	for _, p := range r.Projects {
		p.Skills, p.AlwaysOn = swap(p.Skills), swap(p.AlwaysOn)
	}
}

// AlwaysOnServed returns the always-on skills served in the project: the
// global ones plus the project's own, sorted and without duplicates.
func (r *Registry) AlwaysOnServed(projectID string) []string {
	ids := append([]string{}, r.AlwaysOn...)
	if p, ok := r.Projects[projectID]; ok {
		ids = append(ids, p.AlwaysOn...)
	}
	return sortedSet(ids)
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
