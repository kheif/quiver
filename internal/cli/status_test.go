package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/server"
)

func TestStatusGolden(t *testing.T) {
	f := listEnv(t)
	sub := filepath.Join(f.app, "src")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "other")
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(server.ProjectEnv, "")
	for _, c := range []struct {
		name, dir, env string
	}{
		{"status-project", sub, ""},
		{"status-unregistered", other, ""},
		{"status-env", other, f.api},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Chdir(c.dir)
			t.Setenv(server.ProjectEnv, c.env)
			golden(t, c.name, f.run(t, "status"))
		})
	}
}

func TestStatusPossibleCopy(t *testing.T) {
	f := listEnv(t)
	cp := projectDir(t, "app-copy", "app-000001")
	t.Chdir(cp)
	golden(t, "status-copy", f.run(t, "status"))
	// status only reports; the registry keeps the one known path.
	if p := loadRegistry(t).Projects["app-000001"]; len(p.Paths) != 1 {
		t.Errorf("paths = %v", p.Paths)
	}
}

func TestStatusMissingSkill(t *testing.T) {
	f := listEnv(t)
	if err := os.RemoveAll(filepath.Join(f.home, "local", "tdd")); err != nil {
		t.Fatal(err)
	}
	t.Chdir(f.app)
	out := f.run(t, "status")
	if want := "warning: tdd is assigned but not in the library, so it is not served"; !strings.Contains(out, want) {
		t.Errorf("output missing %q:\n%s", want, out)
	}
}
