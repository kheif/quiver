package assistant

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// hookTimeout is each hook's timeout in seconds. The default is 600, and a
// slow hook delays the first reply by its whole run time (docs/spike-hooks.md).
const hookTimeout = 10

// hookEvent is one hook Quiver installs: its event key in the file, the hook
// subcommand it runs, and the matcher of its group ("" for no matcher).
type hookEvent struct{ event, subcommand, matcher string }

// quiverHookEvents are the two hooks every assistant with a session-start hook
// gets. The file keys (SessionStart, SubagentStart) are the same for Claude
// Code and Codex.
var quiverHookEvents = []hookEvent{
	{"SessionStart", "session-start", "startup|clear|compact"},
	{"SubagentStart", "subagent-start", ""},
}

// hookTarget is an assistant's hook file and the fields its entries carry.
type hookTarget struct {
	path string // the hooks file
	id   string // the assistant id in the hook command and recognition
	// extra are members added to each new hook entry, after type, command
	// and timeout (such as Codex's additionalContextLimit and statusMessage).
	extra []member
}

// ClaudeSettingsPath is the user's Claude Code settings file:
// $CLAUDE_CONFIG_DIR/settings.json, or ~/.claude/settings.json.
func ClaudeSettingsPath() (string, error) {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "settings.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// claudeHookTarget is the Claude Code hook file; its entries carry only the
// timeout.
func claudeHookTarget() (hookTarget, error) {
	path, err := ClaudeSettingsPath()
	return hookTarget{path: path, id: "claude-code"}, err
}

// CodexHooksPath is Codex's hooks file: $CODEX_HOME/hooks.json, or
// ~/.codex/hooks.json. The Codex CLI and app share it.
func CodexHooksPath() (string, error) {
	if dir := os.Getenv("CODEX_HOME"); dir != "" {
		return filepath.Join(dir, "hooks.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "hooks.json"), nil
}

// codexHookTarget is Codex's hook file. Its entries carry the timeout,
// additionalContextLimit 0 (so Codex never cuts the block) and a
// statusMessage for Codex's review list.
func codexHookTarget() (hookTarget, error) {
	path, err := CodexHooksPath()
	return hookTarget{path: path, id: "codex", extra: []member{
		{"additionalContextLimit", json.RawMessage("0")},
		{"statusMessage", mustMarshal("Quiver always-on skills")},
	}}, err
}

// InstallCodexHooks adds Quiver's hooks to Codex's hooks file, which run bin.
// After it, the caller trusts the hooks through codex app-server.
func InstallCodexHooks(bin string) (Result, error) {
	t, err := codexHookTarget()
	if err != nil {
		return Unchanged, err
	}
	return t.install(bin)
}

// UninstallCodexHooks removes only Quiver's hooks from Codex's hooks file.
func UninstallCodexHooks() (Result, error) {
	t, err := codexHookTarget()
	if err != nil {
		return Unchanged, err
	}
	return t.uninstall()
}

// InstallClaudeHooks adds Quiver's SessionStart and SubagentStart hooks to the
// Claude Code settings file, which run bin, or points existing ones to bin.
func InstallClaudeHooks(bin string) (Result, error) {
	t, err := claudeHookTarget()
	if err != nil {
		return Unchanged, err
	}
	return t.install(bin)
}

// UninstallClaudeHooks removes only Quiver's hooks from the Claude Code
// settings file.
func UninstallClaudeHooks() (Result, error) {
	t, err := claudeHookTarget()
	if err != nil {
		return Unchanged, err
	}
	return t.uninstall()
}

// install adds or updates Quiver's hooks in the target's file, which run bin.
// Every other key, group and hook is kept.
func (t hookTarget) install(bin string) (Result, error) {
	return editJSONFile(t.path, func(top []member) ([]member, Result, error) {
		hooks, err := nestedObject(top, "hooks")
		if err != nil {
			return top, Unchanged, err
		}
		res := Unchanged
		for _, h := range quiverHookEvents {
			var r Result
			hooks, r, err = t.installHook(hooks, h, bin)
			if err != nil {
				return top, Unchanged, err
			}
			res = stronger(res, r)
		}
		if res == Unchanged {
			return top, Unchanged, nil
		}
		return set(top, "hooks", encodeObject(hooks)), res, nil
	})
}

// uninstall removes only Quiver's hooks from the target's file, and a group or
// event left empty by the removal.
func (t hookTarget) uninstall() (Result, error) {
	return editJSONFile(t.path, func(top []member) ([]member, Result, error) {
		i := index(top, "hooks")
		if i < 0 {
			return top, Unchanged, nil
		}
		hooks, err := parseObject(top[i].value)
		if err != nil {
			return top, Unchanged, nil // not an object: leave it alone
		}
		res := Unchanged
		for _, h := range quiverHookEvents {
			var r Result
			hooks, r = t.uninstallHook(hooks, h)
			res = stronger(res, r)
		}
		if res == Unchanged {
			return top, Unchanged, nil
		}
		if len(hooks) == 0 {
			return slicesDelete(top, i), Removed, nil
		}
		return set(top, "hooks", encodeObject(hooks)), res, nil
	})
}

// command is the command string of the hook that runs bin for h.
func (t hookTarget) command(bin string, h hookEvent) string {
	return ShellQuote(bin) + " hook " + h.subcommand + " " + t.id
}

// installHook adds or updates Quiver's hook for h in the hooks object. groups
// of the event keep their order; other groups and hooks are kept.
func (t hookTarget) installHook(hooks []member, h hookEvent, bin string) ([]member, Result, error) {
	groups, err := parseArray(hooks, h.event)
	if err != nil {
		return hooks, Unchanged, err
	}
	want := t.command(bin, h)
	for gi, group := range groups {
		entries, gm, err := groupEntries(group)
		if err != nil {
			continue // not a group we understand: leave it alone
		}
		for ei, entry := range entries {
			em, err := parseObject(entry)
			if err != nil {
				continue
			}
			cmd, ok := stringMember(em, "command")
			if !ok || !t.isQuiverHook(cmd, h.subcommand) {
				continue
			}
			if cmd == want && timeoutOf(em) == hookTimeout {
				return hooks, Unchanged, nil
			}
			em = set(em, "command", mustMarshal(want))
			em = set(em, "timeout", json.RawMessage("10"))
			entries[ei] = encodeObject(em)
			gm = set(gm, "hooks", encodeArray(entries))
			groups[gi] = encodeObject(gm)
			return set(hooks, h.event, encodeArray(groups)), Updated, nil
		}
	}
	groups = append(groups, t.newGroup(want, h))
	return set(hooks, h.event, encodeArray(groups)), Installed, nil
}

// uninstallHook removes Quiver's hook for h, and a group or event left empty.
func (t hookTarget) uninstallHook(hooks []member, h hookEvent) ([]member, Result) {
	ei := index(hooks, h.event)
	if ei < 0 {
		return hooks, Unchanged
	}
	groups, err := parseArray(hooks, h.event)
	if err != nil {
		return hooks, Unchanged
	}
	for gi := 0; gi < len(groups); gi++ {
		entries, gm, err := groupEntries(groups[gi])
		if err != nil {
			continue
		}
		kept := entries[:0:0]
		removed := false
		for _, entry := range entries {
			em, err := parseObject(entry)
			if err == nil {
				if cmd, ok := stringMember(em, "command"); ok && t.isQuiverHook(cmd, h.subcommand) {
					removed = true
					continue
				}
			}
			kept = append(kept, entry)
		}
		if !removed {
			continue
		}
		if len(kept) == 0 {
			groups = slicesDeleteAt(groups, gi)
		} else {
			gm = set(gm, "hooks", encodeArray(kept))
			groups[gi] = encodeObject(gm)
		}
		if len(groups) == 0 {
			return slicesDelete(hooks, ei), Removed
		}
		return set(hooks, h.event, encodeArray(groups)), Removed
	}
	return hooks, Unchanged
}

// newGroup is the group that runs want for h: a matcher when h has one, then a
// single command hook with the timeout and the target's extra fields.
func (t hookTarget) newGroup(want string, h hookEvent) json.RawMessage {
	entry := []member{
		{"type", mustMarshal("command")},
		{"command", mustMarshal(want)},
		{"timeout", json.RawMessage("10")},
	}
	entry = append(entry, t.extra...)
	var group []member
	if h.matcher != "" {
		group = append(group, member{"matcher", mustMarshal(h.matcher)})
	}
	group = append(group, member{"hooks", encodeArray([]json.RawMessage{encodeObject(entry)})})
	return encodeObject(group)
}

// isQuiverHook reports whether command runs Quiver's hook for subcommand and
// the target's assistant: its arguments end with "hook <subcommand> <id>" and
// its program is named quiver or quiver.exe.
func (t hookTarget) isQuiverHook(command, subcommand string) bool {
	if !strings.HasSuffix(command, " hook "+subcommand+" "+t.id) {
		return false
	}
	base := filepath.Base(firstToken(command))
	return base == "quiver" || base == "quiver.exe"
}

// firstToken returns the first shell word of s, unquoting a single-quoted
// first word.
func firstToken(s string) string {
	s = strings.TrimLeft(s, " ")
	if strings.HasPrefix(s, "'") {
		if end := strings.IndexByte(s[1:], '\''); end >= 0 {
			return strings.ReplaceAll(s[1:1+end], `'\''`, "'")
		}
	}
	if i := strings.IndexByte(s, ' '); i >= 0 {
		return s[:i]
	}
	return s
}

// nestedObject returns the members of the object at key, or nil when the key
// is absent. A value that is not an object is an error.
func nestedObject(members []member, key string) ([]member, error) {
	i := index(members, key)
	if i < 0 {
		return nil, nil
	}
	obj, err := parseObject(members[i].value)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// parseArray returns the elements of the array at key in members, or nil when
// the key is absent.
func parseArray(members []member, key string) ([]json.RawMessage, error) {
	i := index(members, key)
	if i < 0 {
		return nil, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(members[i].value, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// groupEntries returns the hooks array of a group and the group's members.
func groupEntries(group json.RawMessage) ([]json.RawMessage, []member, error) {
	gm, err := parseObject(group)
	if err != nil {
		return nil, nil, err
	}
	entries, err := parseArray(gm, "hooks")
	if err != nil {
		return nil, nil, err
	}
	return entries, gm, nil
}

// stringMember returns the value of a string member.
func stringMember(members []member, key string) (string, bool) {
	i := index(members, key)
	if i < 0 {
		return "", false
	}
	var s string
	if err := json.Unmarshal(members[i].value, &s); err != nil {
		return "", false
	}
	return s, true
}

// timeoutOf returns the numeric timeout of a hook entry, or -1.
func timeoutOf(members []member) int {
	i := index(members, "timeout")
	if i < 0 {
		return -1
	}
	var n int
	if err := json.Unmarshal(members[i].value, &n); err != nil {
		return -1
	}
	return n
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// encodeArray encodes raw elements as a compact JSON array.
func encodeArray(items []json.RawMessage) json.RawMessage {
	var b bytes.Buffer
	b.WriteByte('[')
	for i, it := range items {
		if i > 0 {
			b.WriteByte(',')
		}
		_ = json.Compact(&b, it)
	}
	b.WriteByte(']')
	return b.Bytes()
}

// stronger returns the result that reports more: Installed or Removed over
// Updated over Unchanged.
func stronger(a, b Result) Result {
	rank := map[Result]int{Unchanged: 0, Updated: 1, Installed: 2, Removed: 2}
	if rank[b] > rank[a] {
		return b
	}
	return a
}

// slicesDelete removes the member at i.
func slicesDelete(members []member, i int) []member {
	return append(members[:i:i], members[i+1:]...)
}

// slicesDeleteAt removes the element at i.
func slicesDeleteAt(items []json.RawMessage, i int) []json.RawMessage {
	return append(items[:i:i], items[i+1:]...)
}
