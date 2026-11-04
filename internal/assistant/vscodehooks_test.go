package assistant

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// vscodeHome points os.UserHomeDir at a temp folder so the hook file is
// isolated, and returns its path.
func vscodeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir reads USERPROFILE on Windows
	path, err := VSCodeHooksPath()
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInstallVSCodeHooks(t *testing.T) {
	path := vscodeHome(t)
	res, err := InstallVSCodeHooks("/opt/bin/quiver", 1)
	if err != nil || res != Installed {
		t.Fatalf("install: %v %v", res, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("hook file: %v", err)
	}
	var m map[string][]struct {
		Type       string
		Command    string
		TimeoutSec int
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("hook JSON: %v\n%s", err, data)
	}
	ss := m["SessionStart"]
	if len(ss) != 1 || ss[0].Type != "command" || ss[0].Command != "/opt/bin/quiver hook session-start vscode" || ss[0].TimeoutSec != 10 {
		t.Errorf("SessionStart: %+v", ss)
	}
	sub := m["SubagentStart"]
	if len(sub) != 1 || sub[0].Command != "/opt/bin/quiver hook subagent-start vscode" || sub[0].TimeoutSec != 10 {
		t.Errorf("SubagentStart: %+v", sub)
	}

	// Writing the same content again changes nothing.
	if res, err := InstallVSCodeHooks("/opt/bin/quiver", 1); err != nil || res != Unchanged {
		t.Errorf("second install: %v %v", res, err)
	}
	// A new binary updates the file.
	if res, err := InstallVSCodeHooks("/new/quiver", 1); err != nil || res != Updated {
		t.Errorf("update: %v %v", res, err)
	}

	present, current, err := VSCodeHookInstalled("/new/quiver")
	if err != nil || !present || !current {
		t.Errorf("installed(/new/quiver): %v %v %v", present, current, err)
	}
	present, current, err = VSCodeHookInstalled("/opt/bin/quiver")
	if err != nil || !present || current {
		t.Errorf("installed(/opt/bin/quiver): %v %v %v", present, current, err)
	}

	if res, err := UninstallVSCodeHooks(); err != nil || res != Removed {
		t.Errorf("uninstall: %v %v", res, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file still there: %v", err)
	}
	if res, err := UninstallVSCodeHooks(); err != nil || res != Unchanged {
		t.Errorf("second uninstall: %v %v", res, err)
	}
	present, _, err = VSCodeHookInstalled("/new/quiver")
	if err != nil || present {
		t.Errorf("installed after uninstall: %v %v", present, err)
	}
}

// Install creates the hooks folder when it does not exist.
func TestInstallVSCodeHooksCreatesFolder(t *testing.T) {
	path := vscodeHome(t)
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("folder already there: %v", err)
	}
	if _, err := InstallVSCodeHooks("/opt/bin/quiver", 1); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not written: %v", err)
	}
}
