package assistant

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// commandsOf returns the commands of the hook entries of the given JSON
// values, in order, for flat entries ({"command": ...}) and for groups
// ({"hooks": [{"command": ...}]}).
func commandsOf(t *testing.T, list []any) []string {
	t.Helper()
	var out []string
	for _, v := range list {
		m, _ := v.(map[string]any)
		if cmd, ok := m["command"].(string); ok {
			out = append(out, cmd)
		}
		inner, _ := m["hooks"].([]any)
		out = append(out, commandsOf(t, inner)...)
	}
	return out
}

func wantParts(bin, sub, id string, n int) []string {
	t := hookTarget{id: id}
	var out []string
	for part := 1; part <= n; part++ {
		out = append(out, t.command(bin, hookEvent{subcommand: sub}, part))
	}
	return out
}

func TestHookPartCommand(t *testing.T) {
	tg := hookTarget{id: "claude-code"}
	// Part 1 has no number, so an earlier Quiver's hook is already part 1.
	if got, want := tg.command("/opt/bin/quiver", hookEvent{subcommand: "session-start"}, 1), "/opt/bin/quiver hook session-start claude-code"; got != want {
		t.Errorf("part 1: %q", got)
	}
	if got, want := tg.command("/opt/bin/quiver", hookEvent{subcommand: "session-start"}, 3), "/opt/bin/quiver hook session-start claude-code 3"; got != want {
		t.Errorf("part 3: %q", got)
	}
	for cmd, want := range map[string]int{
		"/opt/bin/quiver hook session-start claude-code":                  1,
		"/opt/bin/quiver hook session-start claude-code 2":                2,
		"/opt/bin/quiver hook session-start claude-code 17":               17,
		"'/Applications/My Apps/quiver' hook session-start claude-code 4": 4,
		// A quieting suffix does not hide the hook, or install would add a second.
		"/opt/bin/quiver hook session-start claude-code 2>/dev/null || true": 1,
		"/opt/bin/quiver hook session-start claude-code 3 2>/dev/null":       3,
		"/opt/bin/quiver hook session-start claude-code >> /tmp/q.log":       1,
	} {
		if part, ok := tg.hookPart(cmd, "session-start"); !ok || part != want {
			t.Errorf("%q: part %d, %v", cmd, part, ok)
		}
	}
	// Not Quiver's: another program, another assistant, another event, a part
	// that is not a number from 2 on.
	for _, cmd := range []string{
		"/opt/bin/other hook session-start claude-code",
		"/opt/bin/quiver hook session-start codex 2",
		"/opt/bin/quiver hook subagent-start claude-code",
		"/opt/bin/quiver hook session-start claude-code 02",
		"/opt/bin/quiver hook session-start claude-code x",
		"/opt/bin/quiver hook session-start claude-code 2 3",
	} {
		if part, ok := tg.hookPart(cmd, "session-start"); ok {
			t.Errorf("%q counted as part %d", cmd, part)
		}
	}
}

func TestClaudeHookPartsInstallGrowShrinkRemove(t *testing.T) {
	path := hookSettings(t)
	// The user's own hook in the same event stays through all of it.
	if err := os.WriteFile(path, []byte(`{"hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"echo mine"}]}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	const bin = "/opt/bin/quiver"
	if res, err := InstallClaudeHooks(bin, 3); err != nil || res != Installed {
		t.Fatalf("install 3: %v %v", res, err)
	}
	m := readSettings(t, path)
	hooks := m["hooks"].(map[string]any)
	ss := commandsOf(t, hooks["SessionStart"].([]any))
	if want := append([]string{"echo mine"}, wantParts(bin, "session-start", "claude-code", 3)...); !reflect.DeepEqual(ss, want) {
		t.Errorf("SessionStart commands:\n got %q\nwant %q", ss, want)
	}
	if sub := commandsOf(t, hooks["SubagentStart"].([]any)); !reflect.DeepEqual(sub, wantParts(bin, "subagent-start", "claude-code", 3)) {
		t.Errorf("SubagentStart commands: %q", sub)
	}
	if n, err := ClaudeHookParts(); err != nil || n != 3 {
		t.Errorf("ClaudeHookParts = %d, %v", n, err)
	}
	if present, current, err := ClaudeHookInstalled(bin); err != nil || !present || !current {
		t.Errorf("installed: %v %v %v", present, current, err)
	}
	if res, _ := InstallClaudeHooks(bin, 3); res != Unchanged {
		t.Errorf("again: %v", res)
	}
	// One of the parts runs another quiver: not current.
	if res, err := InstallClaudeHooks("/old/quiver", 3); err != nil || res != Updated {
		t.Errorf("move: %v %v", res, err)
	}
	if _, current, _ := ClaudeHookInstalled(bin); current {
		t.Error("current after a move")
	}
	// Fewer parts: the extra commands go, the user's hook stays.
	if res, err := InstallClaudeHooks(bin, 1); err != nil || res != Updated {
		t.Fatalf("shrink: %v %v", res, err)
	}
	m = readSettings(t, path)
	hooks = m["hooks"].(map[string]any)
	if ss := commandsOf(t, hooks["SessionStart"].([]any)); !reflect.DeepEqual(ss, append([]string{"echo mine"}, wantParts(bin, "session-start", "claude-code", 1)...)) {
		t.Errorf("after shrink: %q", ss)
	}
	if res, err := InstallClaudeHooks(bin, 4); err != nil || res != Updated {
		t.Fatalf("grow: %v %v", res, err)
	}
	if res, err := UninstallClaudeHooks(); err != nil || res != Removed {
		t.Fatalf("uninstall: %v %v", res, err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "quiver") || !strings.Contains(string(data), "echo mine") {
		t.Errorf("after uninstall:\n%s", data)
	}
}

func TestCodexHookParts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	const bin = "/opt/bin/quiver"
	if _, err := InstallCodexHooks(bin, 2); err != nil {
		t.Fatal(err)
	}
	if n, err := CodexHookParts(); err != nil || n != 2 {
		t.Errorf("CodexHookParts = %d, %v", n, err)
	}
	if !CodexHooksCurrent(bin) || CodexHooksCurrent("/other/quiver") {
		t.Error("CodexHooksCurrent")
	}
	// Each part carries Codex's own fields: no limit, and a status message.
	data, _ := os.ReadFile(dir + "/hooks.json")
	if strings.Count(string(data), `"additionalContextLimit":0`) != 4 && strings.Count(string(data), `"additionalContextLimit": 0`) != 4 {
		t.Errorf("want 4 hooks with additionalContextLimit 0:\n%s", data)
	}
}

func TestCursorHookParts(t *testing.T) {
	path := cursorHooksHome(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"version":1,"hooks":{"sessionStart":[{"command":"echo mine"}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	const bin = "/opt/bin/quiver"
	if res, err := InstallCursorHooks(bin, 3); err != nil || res != Installed {
		t.Fatalf("install: %v %v", res, err)
	}
	read := func() []string {
		data, _ := os.ReadFile(path)
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatal(err)
		}
		return commandsOf(t, m["hooks"].(map[string]any)["sessionStart"].([]any))
	}
	if got, want := read(), append([]string{"echo mine"}, wantParts(bin, "session-start", "cursor", 3)...); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
	if n, _ := CursorHookParts(); n != 3 {
		t.Errorf("CursorHookParts = %d", n)
	}
	if res, _ := InstallCursorHooks(bin, 3); res != Unchanged {
		t.Errorf("again: %v", res)
	}
	if res, err := InstallCursorHooks(bin, 1); err != nil || res != Updated {
		t.Fatalf("shrink: %v %v", res, err)
	}
	if got, want := read(), append([]string{"echo mine"}, wantParts(bin, "session-start", "cursor", 1)...); !reflect.DeepEqual(got, want) {
		t.Errorf("after shrink: %q", got)
	}
	if p, c, err := CursorHookInstalled(bin); err != nil || !p || !c {
		t.Errorf("installed: %v %v %v", p, c, err)
	}
	if _, err := InstallCursorHooks(bin, 2); err != nil {
		t.Fatal(err)
	}
	if res, _ := UninstallCursorHooks(); res != Removed {
		t.Errorf("uninstall: %v", res)
	}
	if got := read(); !reflect.DeepEqual(got, []string{"echo mine"}) {
		t.Errorf("after uninstall: %q", got)
	}
}

func TestAntigravityHookParts(t *testing.T) {
	antigravityHooksHome(t)
	const bin = "/opt/bin/quiver"
	if res, err := InstallAntigravityHooks(bin, 3); err != nil || res != Installed {
		t.Fatalf("install: %v %v", res, err)
	}
	if n, err := AntigravityHookParts(); err != nil || n != 3 {
		t.Errorf("AntigravityHookParts = %d, %v", n, err)
	}
	if p, c, err := AntigravityHookInstalled(bin); err != nil || !p || !c {
		t.Errorf("installed: %v %v %v", p, c, err)
	}
	if res, _ := InstallAntigravityHooks(bin, 3); res != Unchanged {
		t.Errorf("again: %v", res)
	}
	if res, _ := InstallAntigravityHooks(bin, 1); res != Updated {
		t.Errorf("shrink: %v", res)
	}
	if n, _ := AntigravityHookParts(); n != 1 {
		t.Errorf("after shrink: %d", n)
	}
}

func TestVSCodeHookParts(t *testing.T) {
	vscodeHome(t)
	const bin = "/opt/bin/quiver"
	if res, err := InstallVSCodeHooks(bin, 3); err != nil || res != Installed {
		t.Fatalf("install: %v %v", res, err)
	}
	path, _ := VSCodeHooksPath()
	data, _ := os.ReadFile(path)
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	for ev, sub := range map[string]string{"SessionStart": "session-start", "SubagentStart": "subagent-start"} {
		if got := commandsOf(t, m[ev].([]any)); !reflect.DeepEqual(got, wantParts(bin, sub, "vscode", 3)) {
			t.Errorf("%s: %q", ev, got)
		}
	}
	if n, _ := VSCodeHookParts(); n != 3 {
		t.Errorf("VSCodeHookParts = %d", n)
	}
	if p, c, err := VSCodeHookInstalled(bin); err != nil || !p || !c {
		t.Errorf("installed: %v %v %v", p, c, err)
	}
	if res, _ := InstallVSCodeHooks(bin, 1); res != Updated {
		t.Errorf("shrink: %v", res)
	}
}
