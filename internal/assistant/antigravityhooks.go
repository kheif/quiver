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

// antigravityHookValue is the value of the "quiver" key: a SessionStart hook
// that runs want.
func antigravityHookValue(want string) json.RawMessage {
	entry := encodeObject([]member{
		{"type", mustMarshal("command")},
		{"command", mustMarshal(want)},
		{"timeout", json.RawMessage("10")},
	})
	return encodeObject([]member{
		{"SessionStart", encodeArray([]json.RawMessage{entry})},
	})
}

// InstallAntigravityHooks adds or points Quiver's "quiver" hook in
// Antigravity's hooks file to bin. Every other named hook and key is kept.
func InstallAntigravityHooks(bin string) (Result, error) {
	path, err := AntigravityHooksPath()
	if err != nil {
		return Unchanged, err
	}
	want := antigravityHookValue(antigravityHookTarget.command(bin, antigravitySessionStart))
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

// AntigravityHookInstalled reports whether the hooks file has Quiver's
// "quiver" SessionStart hook, and whether it runs bin. An unreadable or
// invalid file is an error.
func AntigravityHookInstalled(bin string) (present, current bool, err error) {
	path, err := AntigravityHooksPath()
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
	i := index(top, "quiver")
	if i < 0 {
		return false, false, nil
	}
	quiver, err := parseObject(top[i].value)
	if err != nil {
		return true, false, nil
	}
	entries, err := parseArray(quiver, "SessionStart")
	if err != nil {
		return true, false, nil
	}
	want := antigravityHookTarget.command(bin, antigravitySessionStart)
	for _, entry := range entries {
		em, err := parseObject(entry)
		if err != nil {
			continue
		}
		if cmd, ok := stringMember(em, "command"); ok && antigravityHookTarget.isQuiverHook(cmd, antigravitySessionStart.subcommand) {
			return true, cmd == want, nil
		}
	}
	return true, false, nil
}
