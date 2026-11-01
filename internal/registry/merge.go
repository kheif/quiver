package registry

import (
	"encoding/json"
	"fmt"

	"github.com/kheif/quiver/internal/jsonmerge"
)

// Merge merges two versions of registry.json that changed apart from base,
// such as the files of two synced machines. A nil version stands for a file
// that did not exist. Assignment lists are merged as sets, so a skill
// assigned on either machine stays assigned and one unassigned on either
// machine is dropped. An always-on mark whose skill is no longer assigned
// is dropped too. The result has no paths.
func Merge(base, ours, theirs []byte) (*Registry, error) {
	data, err := jsonmerge.Merge(base, ours, theirs)
	if err != nil {
		return nil, fmt.Errorf("registry.json: %w", err)
	}
	r := Default()
	if data != nil {
		if err := json.Unmarshal(data, r); err != nil {
			return nil, fmt.Errorf("registry.json: %w", err)
		}
	}
	if err := r.check("registry.json"); err != nil {
		return nil, err
	}
	r.AlwaysOn = keep(r.AlwaysOn, r.Global)
	for _, p := range r.Projects {
		p.AlwaysOn = keep(p.AlwaysOn, p.Skills)
	}
	r.normalize()
	return r, nil
}

// keep returns the entries of items that are also in allowed.
func keep(items, allowed []string) []string {
	return without(items, missing(allowed, items...)...)
}
