package assistant

import (
	"os"
	"path/filepath"
	"testing"
)

// cursorHooksHome points HOME at a temp folder and returns the hooks file path.
func cursorHooksHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // os.UserHomeDir reads USERPROFILE on Windows
	return filepath.Join(dir, ".cursor", "hooks.json")
}

// cursorSessionStartHook returns Quiver's sessionStart command and timeout, and
// the number of sessionStart entries.
func cursorSessionStartHook(t *testing.T, m map[string]any) (command string, timeout float64, entries int) {
	t.Helper()
	hooks, _ := m["hooks"].(map[string]any)
	list, _ := hooks["sessionStart"].([]any)
	entries = len(list)
	for _, e := range list {
		entry, _ := e.(map[string]any)
		cmd, _ := entry["command"].(string)
		if cursorHookTarget.isQuiverHook(cmd, "session-start") {
			to, _ := entry["timeout"].(float64)
			return cmd, to, entries
		}
	}
	return "", 0, entries
}

func TestCursorHooksMissingFile(t *testing.T) {
	path := cursorHooksHome(t)
	res, err := InstallCursorHooks("/opt/bin/quiver")
	if err != nil || res != Installed {
		t.Fatalf("install: %v %v", res, err)
	}
	m := readSettings(t, path)
	if m["version"] != float64(1) {
		t.Errorf("version %v, want 1", m["version"])
	}
	cmd, to, entries := cursorSessionStartHook(t, m)
	if cmd != "/opt/bin/quiver hook session-start cursor" || to != hookTimeout || entries != 1 {
		t.Errorf("sessionStart: %q timeout %v entries %d", cmd, to, entries)
	}
	// No sub-agent hook: Cursor's subagentStart can only allow or deny.
	hooks := m["hooks"].(map[string]any)
	if _, has := hooks["subagentStart"]; has {
		t.Errorf("subagentStart written: %v", hooks["subagentStart"])
	}

	// A second identical install changes nothing.
	if res, err := InstallCursorHooks("/opt/bin/quiver"); err != nil || res != Unchanged {
		t.Errorf("second install: %v %v", res, err)
	}
}

func TestCursorHooksKeepsOtherHooks(t *testing.T) {
	path := cursorHooksHome(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const existing = `{
  "version": 1,
  "hooks": {
    "sessionStart": [
      {"command": "echo hi", "timeout": 5}
    ],
    "beforeShellExecution": [
      {"command": "guard"}
    ]
  }
}
`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := InstallCursorHooks("/opt/bin/quiver")
	if err != nil || res != Installed {
		t.Fatalf("install: %v %v", res, err)
	}
	m := readSettings(t, path)
	hooks := m["hooks"].(map[string]any)
	if _, ok := hooks["beforeShellExecution"]; !ok {
		t.Errorf("lost beforeShellExecution")
	}
	// The user's own sessionStart entry is kept; Quiver's is added.
	cmd, _, entries := cursorSessionStartHook(t, m)
	if cmd != "/opt/bin/quiver hook session-start cursor" || entries != 2 {
		t.Errorf("sessionStart: %q entries %d, want 2", cmd, entries)
	}
	first := hooks["sessionStart"].([]any)[0].(map[string]any)
	if first["command"] != "echo hi" {
		t.Errorf("user hook changed: %v", first)
	}
	// The backup holds the original file.
	backup, err := os.ReadFile(path + ".quiver-backup")
	if err != nil || string(backup) != existing {
		t.Errorf("backup: %v\n%s", err, backup)
	}
}

func TestCursorHooksUpdateAfterMove(t *testing.T) {
	cursorHooksHome(t)
	if _, err := InstallCursorHooks("/old/quiver"); err != nil {
		t.Fatal(err)
	}
	res, err := InstallCursorHooks("/new/quiver")
	if err != nil || res != Updated {
		t.Fatalf("move: %v %v", res, err)
	}
	path, _ := CursorHooksPath()
	m := readSettings(t, path)
	cmd, _, entries := cursorSessionStartHook(t, m)
	if cmd != "/new/quiver hook session-start cursor" || entries != 1 {
		t.Errorf("after move: %q entries %d", cmd, entries)
	}
}

func TestCursorHooksRemove(t *testing.T) {
	path := cursorHooksHome(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const existing = `{
  "version": 1,
  "hooks": {
    "sessionStart": [
      {"command": "echo hi", "timeout": 5}
    ]
  }
}
`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallCursorHooks("/opt/bin/quiver"); err != nil {
		t.Fatal(err)
	}
	res, err := UninstallCursorHooks()
	if err != nil || res != Removed {
		t.Fatalf("uninstall: %v %v", res, err)
	}
	m := readSettings(t, path)
	if cmd, _, _ := cursorSessionStartHook(t, m); cmd != "" {
		t.Errorf("sessionStart hook still there: %q", cmd)
	}
	// The user's own sessionStart entry stays; version stays.
	hooks := m["hooks"].(map[string]any)
	if n := len(hooks["sessionStart"].([]any)); n != 1 {
		t.Errorf("sessionStart entries %d, want 1", n)
	}
	first := hooks["sessionStart"].([]any)[0].(map[string]any)
	if first["command"] != "echo hi" {
		t.Errorf("user hook changed: %v", first)
	}
	if m["version"] != float64(1) {
		t.Errorf("version %v, want 1", m["version"])
	}
	// Uninstall again changes nothing.
	if res, err := UninstallCursorHooks(); err != nil || res != Unchanged {
		t.Errorf("second uninstall: %v %v", res, err)
	}
}

// Removing Quiver's only sessionStart entry drops the empty sessionStart and
// hooks keys, leaving version.
func TestCursorHooksRemoveOnlyEntry(t *testing.T) {
	path := cursorHooksHome(t)
	if _, err := InstallCursorHooks("/opt/bin/quiver"); err != nil {
		t.Fatal(err)
	}
	if res, err := UninstallCursorHooks(); err != nil || res != Removed {
		t.Fatalf("uninstall: %v %v", res, err)
	}
	m := readSettings(t, path)
	if _, has := m["hooks"]; has {
		t.Errorf("empty hooks not removed: %v", m["hooks"])
	}
	if m["version"] != float64(1) {
		t.Errorf("version %v, want 1", m["version"])
	}
}

func TestCursorHooksInvalidJSON(t *testing.T) {
	path := cursorHooksHome(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	const broken = `{"hooks": {`
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallCursorHooks("/opt/bin/quiver"); err == nil {
		t.Errorf("no error on invalid JSON")
	}
	data, _ := os.ReadFile(path)
	if string(data) != broken {
		t.Errorf("file changed:\n%s", data)
	}
	if _, err := os.Stat(path + ".quiver-backup"); err == nil {
		t.Errorf("wrote a backup for an unchanged file")
	}
}
