package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSourceName(t *testing.T) {
	for _, ok := range []string{"anthropic", "team", "my-org", "a1", "9x"} {
		if err := ValidateSourceName(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "Team", "-x", "a_b", "a/b", "a b", "local"} {
		if err := ValidateSourceName(bad); err == nil {
			t.Errorf("%q: want error", bad)
		}
	}
}

func TestSaveRejectsInvalidSourceName(t *testing.T) {
	t.Setenv("QUIVER_HOME", t.TempDir())
	c := Default()
	c.Sources["Bad Name"] = Source{Repo: "x"}
	if err := Save(c); err == nil {
		t.Fatal("want error")
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("QUIVER_HOME"), "config.json")); !os.IsNotExist(err) {
		t.Error("invalid config was written")
	}
}

func TestLoadRejectsNewerVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QUIVER_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte(`{"version": 99, "sources": {}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "update quiver") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadRejectsInvalidSourceName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QUIVER_HOME", home)
	data := `{"version": 1, "sources": {"local": {"repo": "x"}}}`
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("want error")
	}
}
