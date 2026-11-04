package assistant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kheif/quiver/internal/fsutil"
)

// VS Code's hook file is one Quiver owns whole: ~/.copilot/hooks/quiver.json,
// which VS Code's Local and Copilot harnesses both read, and the Copilot CLI
// too (docs/spike-hooks.md). It holds a SessionStart and a SubagentStart entry
// in the flat form, {"type": "command", "command": "<quiver> hook
// session-start vscode", "timeoutSec": 10}. Because the file is Quiver's own,
// it is written whole with no merging; uninstall removes it.

// vscodeHookTarget builds and recognizes Quiver's VS Code hook command.
var vscodeHookTarget = hookTarget{id: "vscode"}

// vscodeHookEvents are the two events in the file: the session start, which
// carries the always-on text, and the sub-agent start, which the spike found a
// sub-agent needs to follow the rule too.
var vscodeHookEvents = []struct {
	key   string
	event hookEvent
}{
	{"SessionStart", hookEvent{subcommand: "session-start"}},
	{"SubagentStart", hookEvent{subcommand: "subagent-start"}},
}

// VSCodeHooksPath is the hook file VS Code reads: ~/.copilot/hooks/quiver.json.
func VSCodeHooksPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".copilot", "hooks", "quiver.json"), nil
}

// vscodeHooksContent is the whole file: SessionStart and SubagentStart, each a
// one-element array of a flat command entry that runs bin.
func vscodeHooksContent(bin string) ([]byte, error) {
	var top []member
	for _, e := range vscodeHookEvents {
		entry := encodeObject([]member{
			{"type", mustMarshal("command")},
			{"command", mustMarshal(vscodeHookTarget.command(bin, e.event))},
			{"timeoutSec", json.RawMessage("10")},
		})
		top = append(top, member{e.key, encodeArray([]json.RawMessage{entry})})
	}
	var out bytes.Buffer
	if err := json.Indent(&out, encodeObject(top), "", "  "); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

// InstallVSCodeHooks writes Quiver's hook file for bin, creating its folder.
// The file is Quiver's own, so it is written whole; Unchanged when it already
// holds the same content.
func InstallVSCodeHooks(bin string) (Result, error) {
	path, err := VSCodeHooksPath()
	if err != nil {
		return Unchanged, err
	}
	want, err := vscodeHooksContent(bin)
	if err != nil {
		return Unchanged, err
	}
	old, err := os.ReadFile(path)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return Unchanged, err
	}
	if exists && bytes.Equal(old, want) {
		return Unchanged, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Unchanged, err
	}
	if err := fsutil.WriteFileAtomic(path, want, 0o644); err != nil {
		return Unchanged, err
	}
	if exists {
		return Updated, nil
	}
	return Installed, nil
}

// UninstallVSCodeHooks removes Quiver's hook file.
func UninstallVSCodeHooks() (Result, error) {
	path, err := VSCodeHooksPath()
	if err != nil {
		return Unchanged, err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return Unchanged, nil
	}
	if err != nil {
		return Unchanged, err
	}
	return Removed, nil
}

// VSCodeHookInstalled reports whether the hook file has Quiver's SessionStart
// hook, and whether it runs bin. An unreadable or invalid file is an error.
func VSCodeHookInstalled(bin string) (present, current bool, err error) {
	path, err := VSCodeHooksPath()
	if err != nil {
		return false, false, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	top, err := parseObject(data)
	if err != nil {
		return false, false, fmt.Errorf("%s is not valid JSON", path)
	}
	entries, err := parseArray(top, "SessionStart")
	if err != nil || entries == nil {
		return false, false, nil
	}
	want := vscodeHookTarget.command(bin, hookEvent{subcommand: "session-start"})
	for _, entry := range entries {
		em, err := parseObject(entry)
		if err != nil {
			continue
		}
		if cmd, ok := stringMember(em, "command"); ok && vscodeHookTarget.isQuiverHook(cmd, "session-start") {
			return true, cmd == want, nil
		}
	}
	return false, false, nil
}
