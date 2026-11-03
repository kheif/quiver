package assistant

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// hookSettings points CLAUDE_CONFIG_DIR at a temp folder and returns the
// settings file path.
func hookSettings(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	return filepath.Join(dir, "settings.json")
}

// readSettings decodes the settings file.
func readSettings(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !json.Valid(data) {
		t.Fatalf("not valid JSON:\n%s", data)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// quiverHooks returns the command and timeout of Quiver's hook for the event,
// and the number of groups in that event.
func quiverHooks(t *testing.T, m map[string]any, event, subcommand string) (command string, timeout float64, groups int) {
	t.Helper()
	hooks, _ := m["hooks"].(map[string]any)
	list, _ := hooks[event].([]any)
	groups = len(list)
	for _, g := range list {
		group, _ := g.(map[string]any)
		entries, _ := group["hooks"].([]any)
		for _, e := range entries {
			entry, _ := e.(map[string]any)
			cmd, _ := entry["command"].(string)
			if isQuiverHook(cmd, subcommand) {
				to, _ := entry["timeout"].(float64)
				return cmd, to, groups
			}
		}
	}
	return "", 0, groups
}

func TestClaudeHooksMissingFile(t *testing.T) {
	path := hookSettings(t)
	res, err := InstallClaudeHooks("/opt/bin/quiver")
	if err != nil || res != Installed {
		t.Fatalf("install: %v %v", res, err)
	}
	m := readSettings(t, path)
	cmd, to, groups := quiverHooks(t, m, "SessionStart", "session-start")
	if cmd != "/opt/bin/quiver hook session-start claude-code" || to != claudeHookTimeout || groups != 1 {
		t.Errorf("SessionStart: %q timeout %v groups %d", cmd, to, groups)
	}
	// The SessionStart group carries the matcher.
	hooks := m["hooks"].(map[string]any)
	ss := hooks["SessionStart"].([]any)[0].(map[string]any)
	if ss["matcher"] != "startup|clear|compact" {
		t.Errorf("matcher %v", ss["matcher"])
	}
	// SubagentStart has no matcher.
	sub := hooks["SubagentStart"].([]any)[0].(map[string]any)
	if _, has := sub["matcher"]; has {
		t.Errorf("SubagentStart has a matcher: %v", sub)
	}
	cmd, to, _ = quiverHooks(t, m, "SubagentStart", "subagent-start")
	if cmd != "/opt/bin/quiver hook subagent-start claude-code" || to != claudeHookTimeout {
		t.Errorf("SubagentStart: %q timeout %v", cmd, to)
	}

	// A second identical install changes nothing.
	if res, err := InstallClaudeHooks("/opt/bin/quiver"); err != nil || res != Unchanged {
		t.Errorf("second install: %v %v", res, err)
	}
}

func TestClaudeHooksKeepsOtherSettings(t *testing.T) {
	path := hookSettings(t)
	const existing = `{
  "model": "opus",
  "hooks": {
    "SessionStart": [
      {"matcher": "startup", "hooks": [{"type": "command", "command": "echo hi"}]}
    ],
    "PreToolUse": [
      {"hooks": [{"type": "command", "command": "guard"}]}
    ]
  }
}
`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := InstallClaudeHooks("/opt/bin/quiver")
	if err != nil || res != Installed {
		t.Fatalf("install: %v %v", res, err)
	}
	m := readSettings(t, path)
	if m["model"] != "opus" {
		t.Errorf("lost model: %v", m["model"])
	}
	hooks := m["hooks"].(map[string]any)
	if _, ok := hooks["PreToolUse"]; !ok {
		t.Errorf("lost PreToolUse")
	}
	// The user's own SessionStart group is kept; Quiver's is added.
	if _, _, groups := quiverHooks(t, m, "SessionStart", "session-start"); groups != 2 {
		t.Errorf("SessionStart groups %d, want 2", groups)
	}
	ss := hooks["SessionStart"].([]any)
	first := ss[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)
	if first["command"] != "echo hi" {
		t.Errorf("user hook changed: %v", first)
	}
	// The backup holds the original file.
	backup, err := os.ReadFile(path + ".quiver-backup")
	if err != nil || string(backup) != existing {
		t.Errorf("backup: %v\n%s", err, backup)
	}
}

func TestClaudeHooksUpdateAfterMove(t *testing.T) {
	hookSettings(t)
	if _, err := InstallClaudeHooks("/old/quiver"); err != nil {
		t.Fatal(err)
	}
	res, err := InstallClaudeHooks("/new/quiver")
	if err != nil || res != Updated {
		t.Fatalf("move: %v %v", res, err)
	}
	path, _ := ClaudeSettingsPath()
	m := readSettings(t, path)
	cmd, _, groups := quiverHooks(t, m, "SessionStart", "session-start")
	if cmd != "/new/quiver hook session-start claude-code" || groups != 1 {
		t.Errorf("after move: %q groups %d", cmd, groups)
	}
}

func TestClaudeHooksRemove(t *testing.T) {
	path := hookSettings(t)
	const existing = `{
  "hooks": {
    "SessionStart": [
      {"matcher": "startup", "hooks": [{"type": "command", "command": "echo hi"}]}
    ]
  }
}
`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallClaudeHooks("/opt/bin/quiver"); err != nil {
		t.Fatal(err)
	}
	res, err := UninstallClaudeHooks()
	if err != nil || res != Removed {
		t.Fatalf("uninstall: %v %v", res, err)
	}
	m := readSettings(t, path)
	if cmd, _, _ := quiverHooks(t, m, "SessionStart", "session-start"); cmd != "" {
		t.Errorf("SessionStart hook still there: %q", cmd)
	}
	hooks := m["hooks"].(map[string]any)
	// The user's SessionStart group stays; Quiver's empty SubagentStart event
	// is gone.
	if n := len(hooks["SessionStart"].([]any)); n != 1 {
		t.Errorf("SessionStart groups %d, want 1", n)
	}
	if _, has := hooks["SubagentStart"]; has {
		t.Errorf("SubagentStart not removed: %v", hooks["SubagentStart"])
	}
	first := hooks["SessionStart"].([]any)[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)
	if first["command"] != "echo hi" {
		t.Errorf("user hook changed: %v", first)
	}
	// Uninstall again changes nothing.
	if res, err := UninstallClaudeHooks(); err != nil || res != Unchanged {
		t.Errorf("second uninstall: %v %v", res, err)
	}
}

func TestClaudeHooksInvalidJSON(t *testing.T) {
	path := hookSettings(t)
	const broken = `{"hooks": {`
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallClaudeHooks("/opt/bin/quiver"); err == nil {
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
