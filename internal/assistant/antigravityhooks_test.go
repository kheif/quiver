package assistant

import (
	"os"
	"path/filepath"
	"testing"
)

// antigravityHooksHome points HOME at a temp folder and returns the hooks file
// path.
func antigravityHooksHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // os.UserHomeDir reads USERPROFILE on Windows
	return filepath.Join(dir, ".gemini", "config", "hooks.json")
}

// quiverSessionStart returns the command and timeout of Quiver's SessionStart
// hook in Antigravity's file.
func quiverSessionStart(t *testing.T, m map[string]any) (command string, timeout float64) {
	t.Helper()
	quiver, _ := m["quiver"].(map[string]any)
	list, _ := quiver["SessionStart"].([]any)
	for _, e := range list {
		entry, _ := e.(map[string]any)
		cmd, _ := entry["command"].(string)
		if antigravityHookTarget.isQuiverHook(cmd, "session-start") {
			to, _ := entry["timeout"].(float64)
			return cmd, to
		}
	}
	return "", 0
}

func TestAntigravityHooksMissingFile(t *testing.T) {
	path := antigravityHooksHome(t)
	res, err := InstallAntigravityHooks("/opt/bin/quiver", 1)
	if err != nil || res != Installed {
		t.Fatalf("install: %v %v", res, err)
	}
	m := readSettings(t, path)
	cmd, to := quiverSessionStart(t, m)
	if cmd != "/opt/bin/quiver hook session-start antigravity" || to != hookTimeout {
		t.Errorf("SessionStart: %q timeout %v", cmd, to)
	}
	quiver := m["quiver"].(map[string]any)
	entry := quiver["SessionStart"].([]any)[0].(map[string]any)
	if entry["type"] != "command" {
		t.Errorf("type %v, want command", entry["type"])
	}

	present, current, err := AntigravityHookInstalled("/opt/bin/quiver")
	if err != nil || !present || !current {
		t.Errorf("installed: present %v current %v err %v", present, current, err)
	}

	// A second identical install changes nothing.
	if res, err := InstallAntigravityHooks("/opt/bin/quiver", 1); err != nil || res != Unchanged {
		t.Errorf("second install: %v %v", res, err)
	}
}

func TestAntigravityHooksKeepsOtherHooks(t *testing.T) {
	path := antigravityHooksHome(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const existing = `{
  "mytool": {
    "SessionStart": [
      {"type": "command", "command": "echo hi"}
    ]
  }
}
`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := InstallAntigravityHooks("/opt/bin/quiver", 1)
	if err != nil || res != Installed {
		t.Fatalf("install: %v %v", res, err)
	}
	m := readSettings(t, path)
	if _, ok := m["mytool"]; !ok {
		t.Errorf("lost mytool hook")
	}
	cmd, _ := quiverSessionStart(t, m)
	if cmd != "/opt/bin/quiver hook session-start antigravity" {
		t.Errorf("quiver hook: %q", cmd)
	}
	// The backup holds the original file.
	backup, err := os.ReadFile(path + ".quiver-backup")
	if err != nil || string(backup) != existing {
		t.Errorf("backup: %v\n%s", err, backup)
	}
}

func TestAntigravityHooksUpdateAfterMove(t *testing.T) {
	antigravityHooksHome(t)
	if _, err := InstallAntigravityHooks("/old/quiver", 1); err != nil {
		t.Fatal(err)
	}
	res, err := InstallAntigravityHooks("/new/quiver", 1)
	if err != nil || res != Updated {
		t.Fatalf("move: %v %v", res, err)
	}
	path, _ := AntigravityHooksPath()
	m := readSettings(t, path)
	cmd, _ := quiverSessionStart(t, m)
	if cmd != "/new/quiver hook session-start antigravity" {
		t.Errorf("after move: %q", cmd)
	}
	// Doctor sees the old binary as not current.
	if _, current, _ := AntigravityHookInstalled("/old/quiver"); current {
		t.Errorf("old binary reported current")
	}
}

func TestAntigravityHooksRemove(t *testing.T) {
	path := antigravityHooksHome(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const existing = `{
  "mytool": {
    "SessionStart": [
      {"type": "command", "command": "echo hi"}
    ]
  }
}
`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallAntigravityHooks("/opt/bin/quiver", 1); err != nil {
		t.Fatal(err)
	}
	res, err := UninstallAntigravityHooks()
	if err != nil || res != Removed {
		t.Fatalf("uninstall: %v %v", res, err)
	}
	m := readSettings(t, path)
	if _, has := m["quiver"]; has {
		t.Errorf("quiver hook still there: %v", m["quiver"])
	}
	if _, ok := m["mytool"]; !ok {
		t.Errorf("lost mytool hook on uninstall")
	}
	// Uninstall again changes nothing.
	if res, err := UninstallAntigravityHooks(); err != nil || res != Unchanged {
		t.Errorf("second uninstall: %v %v", res, err)
	}
}

func TestAntigravityHookInstalledMissing(t *testing.T) {
	antigravityHooksHome(t)
	present, current, err := AntigravityHookInstalled("/opt/bin/quiver")
	if err != nil || present || current {
		t.Errorf("missing file: present %v current %v err %v", present, current, err)
	}
}

func TestAntigravityHooksInvalidJSON(t *testing.T) {
	path := antigravityHooksHome(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const broken = `{"quiver": {`
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallAntigravityHooks("/opt/bin/quiver", 1); err == nil {
		t.Errorf("no error on invalid JSON")
	}
	data, _ := os.ReadFile(path)
	if string(data) != broken {
		t.Errorf("file changed:\n%s", data)
	}
	if _, _, err := AntigravityHookInstalled("/opt/bin/quiver"); err == nil {
		t.Errorf("no error reading invalid JSON")
	}
}
