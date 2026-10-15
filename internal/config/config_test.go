package config

import (
	"testing"
)

func TestLoadMissingGivesDefaults(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Version != CurrentVersion || c.Sources == nil || len(c.Sources) != 0 {
		t.Errorf("got %+v", c)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	c := Default()
	c.Sources["anthropic"] = Source{Repo: "https://github.com/anthropics/skills.git"}
	if err := Save(c); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Sources["anthropic"].Repo != "https://github.com/anthropics/skills.git" {
		t.Errorf("got %+v", got)
	}
}

func TestUpdate(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	err := Update(func(c *Config) error {
		c.Sources["team"] = Source{Repo: "git@github.com:my-org/skills.git"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := Load()
	if _, ok := got.Sources["team"]; !ok {
		t.Errorf("got %+v", got)
	}
}
