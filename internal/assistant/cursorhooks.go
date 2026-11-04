package assistant

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Cursor's hooks file differs from Claude Code's and Codex's: it is
// {"version": 1, "hooks": {"sessionStart": [...]}}, and each sessionStart
// entry is a flat {command, timeout} object with no matcher and no nested
// hooks array (docs/spike-hooks.md). Cursor's subagentStart can only allow or
// deny, so it cannot add always-on text; Quiver installs no sub-agent hook
// there (TASK52.4).

// cursorHookEvent is the one event Quiver installs in Cursor's hooks file.
const cursorHookEvent = "sessionStart"

// cursorHookTarget recognizes and builds Quiver's Cursor hook command. It
// reuses hookTarget only for command() and isQuiverHook; Cursor's file shape
// is handled here.
var cursorHookTarget = hookTarget{id: "cursor"}

// cursorSessionStart is the hook event whose command Quiver installs.
var cursorSessionStart = hookEvent{subcommand: "session-start"}

// CursorHooksPath is the user's Cursor hooks file: ~/.cursor/hooks.json. The
// Cursor app and the CLI share it.
func CursorHooksPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cursor", "hooks.json"), nil
}

// InstallCursorHooks adds Quiver's sessionStart hook to Cursor's hooks file,
// which runs bin, or points an existing one to bin. version 1 and every other
// hook, event and key are kept.
func InstallCursorHooks(bin string) (Result, error) {
	path, err := CursorHooksPath()
	if err != nil {
		return Unchanged, err
	}
	want := cursorHookTarget.command(bin, cursorSessionStart)
	return editJSONFile(path, func(top []member) ([]member, Result, error) {
		hooks, err := nestedObject(top, "hooks")
		if err != nil {
			return top, Unchanged, err
		}
		entries, err := parseArray(hooks, cursorHookEvent)
		if err != nil {
			return top, Unchanged, err
		}
		res := Unchanged
		found := false
		for i, entry := range entries {
			em, err := parseObject(entry)
			if err != nil {
				continue
			}
			cmd, ok := stringMember(em, "command")
			if !ok || !cursorHookTarget.isQuiverHook(cmd, cursorSessionStart.subcommand) {
				continue
			}
			found = true
			if cmd == want && timeoutOf(em) == hookTimeout {
				return top, Unchanged, nil
			}
			em = set(em, "command", mustMarshal(want))
			em = set(em, "timeout", json.RawMessage("10"))
			entries[i] = encodeObject(em)
			res = Updated
			break
		}
		if !found {
			entry := []member{
				{"command", mustMarshal(want)},
				{"timeout", json.RawMessage("10")},
			}
			entries = append(entries, encodeObject(entry))
			res = Installed
		}
		hooks = set(hooks, cursorHookEvent, encodeArray(entries))
		top = set(top, "hooks", encodeObject(hooks))
		if index(top, "version") < 0 {
			top = append([]member{{"version", json.RawMessage("1")}}, top...)
		}
		return top, res, nil
	})
}

// UninstallCursorHooks removes only Quiver's sessionStart hook from Cursor's
// hooks file, and the sessionStart and hooks keys they leave empty. version
// and the user's own hooks stay.
func UninstallCursorHooks() (Result, error) {
	path, err := CursorHooksPath()
	if err != nil {
		return Unchanged, err
	}
	return editJSONFile(path, func(top []member) ([]member, Result, error) {
		hi := index(top, "hooks")
		if hi < 0 {
			return top, Unchanged, nil
		}
		hooks, err := parseObject(top[hi].value)
		if err != nil {
			return top, Unchanged, nil // not an object: leave it alone
		}
		entries, err := parseArray(hooks, cursorHookEvent)
		if err != nil {
			return top, Unchanged, nil
		}
		kept := entries[:0:0]
		removed := false
		for _, entry := range entries {
			if em, err := parseObject(entry); err == nil {
				if cmd, ok := stringMember(em, "command"); ok && cursorHookTarget.isQuiverHook(cmd, cursorSessionStart.subcommand) {
					removed = true
					continue
				}
			}
			kept = append(kept, entry)
		}
		if !removed {
			return top, Unchanged, nil
		}
		if len(kept) == 0 {
			hooks = slicesDelete(hooks, index(hooks, cursorHookEvent))
		} else {
			hooks = set(hooks, cursorHookEvent, encodeArray(kept))
		}
		if len(hooks) == 0 {
			top = slicesDelete(top, hi)
		} else {
			top = set(top, "hooks", encodeObject(hooks))
		}
		return top, Removed, nil
	})
}
