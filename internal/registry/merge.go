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
// is dropped too. A skill that one machine made Searchable and the other
// served keeps its served level. The result has no paths.
func Merge(base, ours, theirs []byte) (*Registry, error) {
	for _, in := range [][]byte{base, ours, theirs} {
		if err := checkVersion(in); err != nil {
			return nil, err
		}
	}
	// The version follows the content (writeVersion), so a version that
	// differs between the files is not a conflict.
	data, err := jsonmerge.Merge(noVersion(base), noVersion(ours), noVersion(theirs))
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
	// A skill that is both served and Searchable leaves the searchable list
	// first, so the always-on marks below still see it as served.
	r.Searchable = searchableOnly(r.Searchable, r.Global)
	for _, p := range r.Projects {
		p.Searchable = searchableOnly(p.Searchable, p.Skills)
	}
	r.AlwaysOn = keep(r.AlwaysOn, r.Global)
	for _, p := range r.Projects {
		p.AlwaysOn = keep(p.AlwaysOn, p.Skills)
	}
	r.normalize()
	r.Version = r.writeVersion()
	return r, nil
}

// keep returns the entries of items that are also in allowed.
func keep(items, allowed []string) []string {
	return without(items, missing(allowed, items...)...)
}

// noVersion returns the file without its version, or nil for nil. A file
// that is not a JSON object is returned as it is, for jsonmerge to refuse.
func noVersion(data []byte) []byte {
	if data == nil {
		return nil
	}
	var doc map[string]json.RawMessage
	if json.Unmarshal(data, &doc) != nil {
		return data
	}
	delete(doc, "version")
	out, err := json.Marshal(doc)
	if err != nil {
		return data
	}
	return out
}

// checkVersion refuses a file from a newer quiver, so a merge never drops
// fields it does not know.
func checkVersion(data []byte) error {
	var v struct {
		Version int `json:"version"`
	}
	if data == nil || json.Unmarshal(data, &v) != nil {
		return nil
	}
	if v.Version > CurrentVersion {
		return fmt.Errorf("registry.json has version %d, but this quiver supports only up to %d: update quiver", v.Version, CurrentVersion)
	}
	return nil
}
