package config

import (
	"encoding/json"
	"fmt"

	"github.com/kheif/quiver/internal/jsonmerge"
)

// Merge merges two versions of config.json that changed apart from base,
// such as the files of two synced machines. A nil version stands for a file
// that did not exist. Sources and pins added on either machine are kept;
// one source or pin changed differently on both is a conflict.
func Merge(base, ours, theirs []byte) (*Config, error) {
	data, err := jsonmerge.Merge(base, ours, theirs)
	if err != nil {
		return nil, fmt.Errorf("config.json: %w", err)
	}
	c := Default()
	if data != nil {
		if err := json.Unmarshal(data, c); err != nil {
			return nil, fmt.Errorf("config.json: %w", err)
		}
	}
	if c.Sources == nil {
		c.Sources = map[string]Source{}
	}
	return c, c.Validate()
}
