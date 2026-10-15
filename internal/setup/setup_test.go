package setup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func commitCount(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-list", "--count", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestInitCreatesHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "q")
	t.Setenv("QUIVER_HOME", home)

	created, err := Init()
	if err != nil {
		t.Fatal(err)
	}
	if len(created) == 0 || created[0] != home {
		t.Errorf("created = %v", created)
	}
	for _, p := range []string{"local", "sources", "config.json", "registry.json", "local/README.md"} {
		if _, err := os.Stat(filepath.Join(home, p)); err != nil {
			t.Error(err)
		}
	}
	if n := commitCount(t, filepath.Join(home, "local")); n != "1" {
		t.Errorf("local has %s commits, want 1", n)
	}
}

func TestInitIsIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QUIVER_HOME", home)
	if _, err := Init(); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, "config.json")
	if err := os.WriteFile(cfg, []byte(`{"version": 1, "sources": {"team": {"repo": "x"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	created, err := Init()
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Errorf("second run created %v", created)
	}
	if data, _ := os.ReadFile(cfg); !strings.Contains(string(data), "team") {
		t.Error("config.json was overwritten")
	}
	if n := commitCount(t, filepath.Join(home, "local")); n != "1" {
		t.Errorf("local has %s commits, want 1", n)
	}
}

func TestInitFillsInMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QUIVER_HOME", home)
	if _, err := Init(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(home, "registry.json")); err != nil {
		t.Fatal(err)
	}
	created, err := Init()
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 1 || created[0] != filepath.Join(home, "registry.json") {
		t.Errorf("created = %v", created)
	}
}

func TestEnsureInit(t *testing.T) {
	home := filepath.Join(t.TempDir(), "q")
	t.Setenv("QUIVER_HOME", home)
	if err := EnsureInit(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, "local", ".git")); err != nil {
		t.Error(err)
	}
}
