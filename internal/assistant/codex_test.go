package assistant

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/assistant/assistanttest"
)

func TestCodexInstall(t *testing.T) {
	f := assistanttest.InstallCodex(t)
	if reg, err := CodexGet(); reg != nil || err != nil {
		t.Fatalf("before: %+v, %v", reg, err)
	}
	if res, err := CodexInstall("/opt/bin/quiver"); err != nil || res != Installed {
		t.Fatalf("install: %v, %v", res, err)
	}
	reg, err := CodexGet()
	if err != nil || reg == nil || reg.Command != "/opt/bin/quiver" || reg.Args != "mcp" {
		t.Fatalf("get: %+v, %v", reg, err)
	}
	if res, err := CodexInstall("/opt/bin/quiver"); err != nil || res != Unchanged {
		t.Errorf("again: %v, %v", res, err)
	}
	if want := []string{"mcp get quiver --json", "mcp get quiver --json", "mcp add quiver -- /opt/bin/quiver mcp", "mcp get quiver --json", "mcp get quiver --json"}; !slices.Equal(f.Calls(), want) {
		t.Errorf("calls %q", f.Calls())
	}
}

// codex mcp add replaces the whole entry, so an update passes its env again.
func TestCodexInstallKeepsEnv(t *testing.T) {
	f := assistanttest.InstallCodex(t)
	f.SetCodex(t, "/old/quiver", "mcp", "QUIVER_HOME=/h")
	if res, err := CodexInstall("/new/quiver"); err != nil || res != Updated {
		t.Fatalf("install: %v, %v", res, err)
	}
	if f.User() != "/new/quiver|mcp|--env QUIVER_HOME=/h" {
		t.Errorf("user %q", f.User())
	}
}

func TestCodexUninstall(t *testing.T) {
	f := assistanttest.InstallCodex(t)
	f.SetCodex(t, "/opt/bin/quiver", "mcp", "")
	if res, err := CodexUninstall(); err != nil || res != Removed {
		t.Fatalf("uninstall: %v, %v", res, err)
	}
	if res, err := CodexUninstall(); err != nil || res != Unchanged {
		t.Errorf("again: %v, %v", res, err)
	}
}

func TestNoCodex(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if HasCodex() {
		t.Error("HasCodex")
	}
	if _, err := CodexInstall("/opt/bin/quiver"); !errors.Is(err, ErrNoCodex) {
		t.Errorf("install: %v", err)
	}
	if _, err := CodexUninstall(); !errors.Is(err, ErrNoCodex) {
		t.Errorf("uninstall: %v", err)
	}
	if got := CodexAddCommand("/opt/my bin/quiver"); got != "codex mcp add quiver -- '/opt/my bin/quiver' mcp" {
		t.Errorf("add command %q", got)
	}
}

// The fake codex copies what the real one does. When codex is installed,
// this checks the real one, in a throwaway CODEX_HOME.
func TestRealCodex(t *testing.T) {
	if !HasCodex() {
		t.Skip("codex is not installed")
	}
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	config := "# kept\nmodel = \"x\"\n\n[mcp_servers.quiver]\ncommand = \"/old/quiver\"\nargs = [\"mcp\"]\nenv = { QUIVER_HOME = \"/h\" }\n"
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	if res, err := CodexInstall("/new/quiver"); err != nil || res != Updated {
		t.Fatalf("install: %v, %v", res, err)
	}
	e, err := codexGet()
	if err != nil || e == nil || e.Transport.Command != "/new/quiver" || e.Transport.Env["QUIVER_HOME"] != "/h" {
		t.Fatalf("get: %+v, %v", e, err)
	}
	if res, err := CodexInstall("/new/quiver"); err != nil || res != Unchanged {
		t.Errorf("again: %v, %v", res, err)
	}
	if res, err := CodexUninstall(); err != nil || res != Removed {
		t.Errorf("uninstall: %v, %v", res, err)
	}
	if res, err := CodexUninstall(); err != nil || res != Unchanged {
		t.Errorf("uninstall again: %v, %v", res, err)
	}
	if reg, err := CodexGet(); reg != nil || err != nil {
		t.Errorf("get after uninstall: %+v, %v", reg, err)
	}
	b, _ := os.ReadFile(filepath.Join(home, "config.toml"))
	if !strings.HasPrefix(string(b), "# kept\nmodel = \"x\"") {
		t.Errorf("config.toml:\n%s", b)
	}
}

// TestRealCodexTrust checks the app-server trust flow against the real codex
// in a throwaway CODEX_HOME, when codex is installed and logged in.
func TestRealCodexTrust(t *testing.T) {
	if !HasCodex() {
		t.Skip("codex is not installed")
	}
	auth, err := os.ReadFile(filepath.Join(realCodexHome(t), "auth.json"))
	if err != nil {
		t.Skip("no codex login to copy")
	}
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "auth.json"), auth, 0o600); err != nil {
		t.Fatal(err)
	}

	if res, err := InstallCodexHooks("/opt/bin/quiver"); err != nil || res != Installed {
		t.Fatalf("install hooks: %v %v", res, err)
	}
	need, err := CodexHooksNeedTrust()
	if err != nil || len(need) != 2 {
		t.Fatalf("before trust, need: %v %v", need, err)
	}
	if n, err := CodexTrustHooks(); err != nil || n != 2 {
		t.Fatalf("trust: %d %v", n, err)
	}
	if need, err := CodexHooksNeedTrust(); err != nil || len(need) != 0 {
		t.Fatalf("after trust, need: %v %v", need, err)
	}

	// Moving the binary changes the hooks, so they need trust again.
	if res, err := InstallCodexHooks("/new/quiver"); err != nil || res != Updated {
		t.Fatalf("update hooks: %v %v", res, err)
	}
	if need, err := CodexHooksNeedTrust(); err != nil || len(need) != 2 {
		t.Fatalf("after move, need: %v %v", need, err)
	}
	if n, err := CodexTrustHooks(); err != nil || n != 2 {
		t.Fatalf("trust after move: %d %v", n, err)
	}
	if need, err := CodexHooksNeedTrust(); err != nil || len(need) != 0 {
		t.Fatalf("after second trust, need: %v %v", need, err)
	}
}

// realCodexHome is the user's real CODEX_HOME, to copy a login from.
func realCodexHome(t *testing.T) string {
	t.Helper()
	if dir := os.Getenv("CODEX_HOME"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(home, ".codex")
}
