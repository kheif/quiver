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

// vscodeHooksContent is the whole file: SessionStart and SubagentStart, each an
// array of parts flat command entries that run bin.
func vscodeHooksContent(bin string, parts int) ([]byte, error) {
	var top []member
	for _, e := range vscodeHookEvents {
		var entries []json.RawMessage
		for part := 1; part <= max(parts, 1); part++ {
			entries = append(entries, encodeObject([]member{
				{"type", mustMarshal("command")},
				{"command", mustMarshal(vscodeHookTarget.command(bin, e.event, part))},
				{"timeoutSec", json.RawMessage("10")},
			}))
		}
		top = append(top, member{e.key, encodeArray(entries)})
	}
	var out bytes.Buffer
	if err := json.Indent(&out, encodeObject(top), "", "  "); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

// InstallVSCodeHooks writes Quiver's hook file for bin, creating its folder,
// with parts commands for each event, as many as the text takes (see
// server.PartsNeeded). The file is Quiver's own, so it is written whole;
// Unchanged when it already holds the same content.
func InstallVSCodeHooks(bin string, parts int) (Result, error) {
	path, err := VSCodeHooksPath()
	if err != nil {
		return Unchanged, err
	}
	want, err := vscodeHooksContent(bin, parts)
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

// vscodeCommands returns the commands of Quiver's SessionStart hooks in the
// hook file. A missing file has none; an unreadable or invalid one is an error.
func vscodeCommands() ([]string, error) {
	path, err := VSCodeHooksPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	top, err := parseObject(data)
	if err != nil {
		return nil, fmt.Errorf("%s is not valid JSON", path)
	}
	entries, err := parseArray(top, "SessionStart")
	if err != nil {
		return nil, nil
	}
	return vscodeHookTarget.quiverCommands(entries, "session-start"), nil
}

// VSCodeHookInstalled reports whether the hook file has Quiver's SessionStart
// hook, and whether every command runs bin. An unreadable or invalid file is
// an error.
func VSCodeHookInstalled(bin string) (present, current bool, err error) {
	cmds, err := vscodeCommands()
	if err != nil || len(cmds) == 0 {
		return false, false, err
	}
	return true, vscodeHookTarget.allCurrent(cmds, bin, "session-start"), nil
}

// VSCodeHookParts returns how many SessionStart hook commands of Quiver the
// hook file has.
func VSCodeHookParts() (int, error) {
	cmds, err := vscodeCommands()
	return len(cmds), err
}
