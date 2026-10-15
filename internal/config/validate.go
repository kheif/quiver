package config

import (
	"fmt"
	"regexp"
)

var sourceNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// ValidateSourceName checks that name can be used as a source name.
// "local" is reserved for the user's own skills.
func ValidateSourceName(name string) error {
	if !sourceNameRE.MatchString(name) {
		return fmt.Errorf("invalid source name %q: use lowercase letters, digits and dashes, starting with a letter or digit", name)
	}
	if name == "local" {
		return fmt.Errorf("source name %q is reserved", name)
	}
	return nil
}

// Validate checks the version and every source.
func (c *Config) Validate() error {
	if c.Version > CurrentVersion {
		return fmt.Errorf("config.json has version %d, but this quiver supports only up to %d: update quiver", c.Version, CurrentVersion)
	}
	if c.Version < 1 {
		return fmt.Errorf("config.json has invalid version %d", c.Version)
	}
	for name, s := range c.Sources {
		if err := ValidateSourceName(name); err != nil {
			return err
		}
		if s.Repo == "" {
			return fmt.Errorf("source %q has no repo", name)
		}
	}
	return nil
}
