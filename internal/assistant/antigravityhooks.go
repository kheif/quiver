package assistant

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Antigravity's hooks file is keyed by hook name, each name holding a map of
// events to command entries: {"quiver": {"SessionStart": [{"type": "command",
// "command": "...", "timeout": 10}]}} (docs/spike-hooks.md). Quiver owns the
// "quiver" key and keeps every other named hook. SessionStart also runs for
// each sub-agent, so there is no second entry.

// antigravityHookTarget recognizes and builds Quiver's Antigravity hook
// command.
var antigravityHookTarget = hookTarget{id: "antigravity"}

// antigravitySessionStart is the hook event Quiver installs.
var antigravitySessionStart = hookEvent{subcommand: "session-start"}

// AntigravityHooksPath is the global Antigravity hooks file,
// ~/.gemini/config/hooks.json, which every Antigravity product reads.
func AntigravityHooksPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini", "config", "hooks.json"), nil
}

// antigravityHookValue is the value of the "quiver" key: SessionStart hooks
// that run bin, one for each of parts.
func antigravityHookValue(bin string, parts int) json.RawMessage {
	var entries []json.RawMessage
	for part := 1; part <= max(parts, 1); part++ {
		entries = append(entries, encodeObject([]member{
			{"type", mustMarshal("command")},
			{"command", mustMarshal(antigravityHookTarget.command(bin, antigravitySessionStart, part))},
			{"timeout", json.RawMessage("10")},
		}))
	}
	return encodeObject([]member{
		{"SessionStart", encodeArray(entries)},
	})
}

// InstallAntigravityHooks adds or points Quiver's "quiver" hook in
// Antigravity's hooks file to bin, with parts commands, as many as the text
// takes (see server.PartsNeeded). Every other named hook and key is kept.
func InstallAntigravityHooks(bin string, parts int) (Result, error) {
	path, err := AntigravityHooksPath()
	if err != nil {
		return Unchanged, err
	}
	want := antigravityHookValue(bin, parts)
	return editJSONFile(path, func(top []member) ([]member, Result, error) {
		i := index(top, "quiver")
		if i >= 0 && same(top[i].value, want) {
			return top, Unchanged, nil
		}
		res := Installed
		if i >= 0 {
			res = Updated
		}
		return set(top, "quiver", want), res, nil
	})
}

// UninstallAntigravityHooks removes only Quiver's "quiver" hook from
// Antigravity's hooks file.
func UninstallAntigravityHooks() (Result, error) {
	path, err := AntigravityHooksPath()
	if err != nil {
		return Unchanged, err
	}
	return editJSONFile(path, func(top []member) ([]member, Result, error) {
		i := index(top, "quiver")
		if i < 0 {
			return top, Unchanged, nil
		}
		return slicesDelete(top, i), Removed, nil
	})
}

// antigravityCommands returns the commands of Quiver's SessionStart hooks in
// Antigravity's hooks file, and whether the "quiver" key is there at all. An
// unreadable or invalid file is an error.
func antigravityCommands() (cmds []string, keyed bool, err error) {
	path, err := AntigravityHooksPath()
	if err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	top, err := parseObject(data)
	if err != nil {
		return nil, false, fmt.Errorf("%s is not valid JSON", path)
	}
	i := index(top, "quiver")
	if i < 0 {
		return nil, false, nil
	}
	quiver, err := parseObject(top[i].value)
	if err != nil {
		return nil, true, nil
	}
	entries, err := parseArray(quiver, "SessionStart")
	if err != nil {
		return nil, true, nil
	}
	return antigravityHookTarget.quiverCommands(entries, antigravitySessionStart.subcommand), true, nil
}

// AntigravityHookInstalled reports whether the hooks file has Quiver's
// "quiver" SessionStart hook, and whether every command runs bin. An
// unreadable or invalid file is an error.
func AntigravityHookInstalled(bin string) (present, current bool, err error) {
	cmds, keyed, err := antigravityCommands()
	if err != nil || !keyed {
		return false, false, err
	}
	if len(cmds) == 0 {
		return true, false, nil
	}
	return true, antigravityHookTarget.allCurrent(cmds, bin, antigravitySessionStart.subcommand), nil
}

// AntigravityHookParts returns how many SessionStart hook commands of Quiver
// Antigravity's hooks file has.
func AntigravityHookParts() (int, error) {
	cmds, _, err := antigravityCommands()
	return len(cmds), err
}
