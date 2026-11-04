package assistant

import (
	"encoding/json"
	"fmt"
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

// InstallCursorHooks adds Quiver's sessionStart hooks to Cursor's hooks file,
// which run bin, or points existing ones to bin: parts commands, as many as
// the text takes (see server.PartsNeeded), and none beyond them. version 1 and
// every other hook, event and key are kept.
func InstallCursorHooks(bin string, parts int) (Result, error) {
	path, err := CursorHooksPath()
	if err != nil {
		return Unchanged, err
	}
	parts = max(parts, 1)
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
		have := map[int]bool{}
		kept := entries[:0:0]
		for _, entry := range entries {
			em, err := parseObject(entry)
			if err != nil {
				kept = append(kept, entry)
				continue
			}
			cmd, ok := stringMember(em, "command")
			part, quiver := 0, false
			if ok {
				part, quiver = cursorHookTarget.hookPart(cmd, cursorSessionStart.subcommand)
			}
			switch {
			case !quiver:
				kept = append(kept, entry)
			case part > parts || have[part]:
				res = stronger(res, Updated) // beyond the parts needed, or a twin
			default:
				have[part] = true
				want := cursorHookTarget.command(bin, cursorSessionStart, part)
				if cmd != want || timeoutOf(em) != hookTimeout {
					em = set(em, "command", mustMarshal(want))
					em = set(em, "timeout", json.RawMessage("10"))
					entry = encodeObject(em)
					res = stronger(res, Updated)
				}
				kept = append(kept, entry)
			}
		}
		for part := 1; part <= parts; part++ {
			if have[part] {
				continue
			}
			kept = append(kept, encodeObject([]member{
				{"command", mustMarshal(cursorHookTarget.command(bin, cursorSessionStart, part))},
				{"timeout", json.RawMessage("10")},
			}))
			if len(have) == 0 && part == 1 {
				res = stronger(res, Installed)
			} else {
				res = stronger(res, Updated)
			}
		}
		if res == Unchanged {
			return top, Unchanged, nil
		}
		hooks = set(hooks, cursorHookEvent, encodeArray(kept))
		top = set(top, "hooks", encodeObject(hooks))
		if index(top, "version") < 0 {
			top = append([]member{{"version", json.RawMessage("1")}}, top...)
		}
		return top, res, nil
	})
}

// CursorHookParts returns how many sessionStart hook commands of Quiver
// Cursor's hooks file has.
func CursorHookParts() (int, error) {
	path, err := CursorHooksPath()
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	top, err := parseObject(data)
	if err != nil {
		return 0, fmt.Errorf("%s is not valid JSON", path)
	}
	hooks, err := nestedObject(top, "hooks")
	if err != nil {
		return 0, err
	}
	entries, _ := parseArray(hooks, cursorHookEvent)
	return len(cursorHookTarget.quiverCommands(entries, cursorSessionStart.subcommand)), nil
}

// CursorHookInstalled reports whether Cursor's hooks file has Quiver's
// sessionStart hook, and whether every one runs bin.
func CursorHookInstalled(bin string) (present, current bool, err error) {
	path, err := CursorHooksPath()
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
	hooks, err := nestedObject(top, "hooks")
	if err != nil {
		return false, false, err
	}
	entries, _ := parseArray(hooks, cursorHookEvent)
	cmds := cursorHookTarget.quiverCommands(entries, cursorSessionStart.subcommand)
	if len(cmds) == 0 {
		return false, false, nil
	}
	return true, cursorHookTarget.allCurrent(cmds, bin, cursorSessionStart.subcommand), nil
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
