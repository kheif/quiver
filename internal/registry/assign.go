package registry

import (
	"fmt"
	"slices"
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
	p.Searchable = searchableOnly(p.Searchable, p.Skills)
	return nil
}

// AssignSearchable makes the skills Searchable in the project. A skill that
// the project served before moves: it loses its place in Skills and its
// always-on mark.
func (r *Registry) AssignSearchable(projectID string, ids ...string) error {
	p, ok := r.Projects[projectID]
	if !ok {
		return fmt.Errorf("unknown project %q", projectID)
	}
	p.Skills = without(p.Skills, ids...)
	p.AlwaysOn = without(p.AlwaysOn, ids...)
	p.Searchable = searchableOnly(append(p.Searchable, ids...), p.Skills)
	return nil
}

// AssignSearchableGlobal makes the skills Searchable in every folder. A
// skill that was global moves, like in AssignSearchable.
func (r *Registry) AssignSearchableGlobal(ids ...string) {
	r.Global = without(r.Global, ids...)
	r.AlwaysOn = without(r.AlwaysOn, ids...)
	r.Searchable = searchableOnly(append(r.Searchable, ids...), r.Global)
}

// Unassign takes the skills away from the project, including their
// always-on mark and their Searchable level. It returns the IDs that were
// neither served nor Searchable in the project.
func (r *Registry) Unassign(projectID string, ids ...string) (notAssigned []string, err error) {
	p, ok := r.Projects[projectID]
	if !ok {
		return nil, fmt.Errorf("unknown project %q", projectID)
	}
	notAssigned = missing(append(slices.Clone(p.Skills), p.Searchable...), ids...)
	p.Skills = without(p.Skills, ids...)
	p.AlwaysOn = without(p.AlwaysOn, ids...)
	p.Searchable = searchableOnly(p.Searchable, append(slices.Clone(p.Skills), ids...))
	return notAssigned, nil
}

// AssignGlobal makes the skills available in every folder.
func (r *Registry) AssignGlobal(ids ...string) {
	r.Global = sortedSet(append(r.Global, ids...))
	r.Searchable = searchableOnly(r.Searchable, r.Global)
}

// UnassignGlobal removes global skills, including their always-on mark and
// their Searchable level. It returns the IDs that were neither global nor
// Searchable everywhere.
func (r *Registry) UnassignGlobal(ids ...string) (notAssigned []string) {
	notAssigned = missing(append(slices.Clone(r.Global), r.Searchable...), ids...)
	r.Global = without(r.Global, ids...)
	r.AlwaysOn = without(r.AlwaysOn, ids...)
	r.Searchable = without(r.Searchable, ids...)
	if len(r.Searchable) == 0 {
		r.Searchable = nil
	}
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
	if len(r.Searchable) > 0 {
		r.Searchable = swap(r.Searchable)
	}
	for _, p := range r.Projects {
		p.Skills, p.AlwaysOn = swap(p.Skills), swap(p.AlwaysOn)
		if len(p.Searchable) > 0 {
			p.Searchable = swap(p.Searchable)
		}
	}
	r.normalize()
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

// AlwaysOnInOrder returns the always-on skills served in the project in the
// order a session-start hook prints them: the global ones first, then the
// project's own, each ID once. Within each scope the order is the
// registry's, which keeps every list sorted.
func (r *Registry) AlwaysOnInOrder(projectID string) []string {
	ids := append([]string{}, r.AlwaysOn...)
	if p, ok := r.Projects[projectID]; ok {
		for _, id := range p.AlwaysOn {
			if !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}
	return ids
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

// Level is how a skill is assigned in a folder.
type Level int

const (
	// NotAssigned: the skill is in no list of the global or project scope.
	NotAssigned Level = iota
	// Searchable: not listed at session start; the assistant finds it by
	// searching.
	Searchable
	// OnDemand: listed at session start and loaded when it fits.
	OnDemand
	// AlwaysOn: loaded in full at session start.
	AlwaysOn
)

// LevelIn returns the level of the skill in the project's folder: its most
// served level across the global and the project scope. An unknown or
// empty projectID looks at the global scope only.
func (r *Registry) LevelIn(projectID, skillID string) Level {
	level := scopeLevel(skillID, r.Global, r.AlwaysOn, r.Searchable)
	if p, ok := r.Projects[projectID]; ok {
		level = max(level, scopeLevel(skillID, p.Skills, p.AlwaysOn, p.Searchable))
	}
	return level
}

func scopeLevel(id string, served, always, searchable []string) Level {
	switch {
	case slices.Contains(served, id) && slices.Contains(always, id):
		return AlwaysOn
	case slices.Contains(served, id):
		return OnDemand
	case slices.Contains(searchable, id):
		return Searchable
	}
	return NotAssigned
}

// GlobalLevel is the level of the skill in the global scope alone.
func (r *Registry) GlobalLevel(skillID string) Level {
	return scopeLevel(skillID, r.Global, r.AlwaysOn, r.Searchable)
}

// ProjectLevel is the level of the skill in the project's own scope alone.
// An unknown project gives NotAssigned.
func (r *Registry) ProjectLevel(projectID, skillID string) Level {
	p, ok := r.Projects[projectID]
	if !ok {
		return NotAssigned
	}
	return scopeLevel(skillID, p.Skills, p.AlwaysOn, p.Searchable)
}

// SearchableProjects returns the IDs of the projects in which the skill is
// Searchable, sorted. Global use is not included; check r.Searchable.
func (r *Registry) SearchableProjects(skillID string) []string {
	var out []string
	for id, p := range r.Projects {
		if slices.Contains(p.Searchable, skillID) {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// SearchableServed returns the skills that are Searchable in the project's
// folder: Searchable in either scope and not served in either, sorted.
func (r *Registry) SearchableServed(projectID string) []string {
	ids := append([]string{}, r.Searchable...)
	if p, ok := r.Projects[projectID]; ok {
		ids = append(ids, p.Searchable...)
	}
	return without(sortedSet(ids), r.Served(projectID)...)
}
