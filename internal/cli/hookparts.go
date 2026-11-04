package cli

import (
	"fmt"
	"io"

	"github.com/kheif/quiver/internal/assistant"
	"github.com/kheif/quiver/internal/server"
)

// hookLimits is how much text one hook command passes to each assistant. The
// text is split into as many hook commands as it takes, so no limit leaves a
// skill out (docs/spike-hooks.md):
//   - Claude Code passes 10,000 characters per command and replaces more with
//     a file path and a preview.
//   - Codex counts bytes; Quiver sets the hook's additionalContextLimit to 0,
//     and 50,000 bytes arrived whole.
//   - VS Code and Cursor passed 200,000 characters from one command.
//   - Antigravity drops everything after about 48,900 bytes.
var hookLimits = map[string]server.Limit{
	claudeCode:    {Max: 10000},
	codexID:       {Max: 40000, Bytes: true},
	"vscode":      {Max: 100000},
	"cursor":      {Max: 100000},
	"antigravity": {Max: 40000, Bytes: true},
}

// hookParts returns how many session-start hook commands of Quiver the
// assistant has installed.
func hookParts(id string) (int, error) {
	switch id {
	case claudeCode:
		return assistant.ClaudeHookParts()
	case codexID:
		return assistant.CodexHookParts()
	case "cursor":
		return assistant.CursorHookParts()
	case "antigravity":
		return assistant.AntigravityHookParts()
	case "vscode":
		return assistant.VSCodeHookParts()
	}
	return 0, fmt.Errorf("%s has no session-start hook", id)
}

// neededHookParts returns how many hook commands the assistant needs so that
// every folder gets all of its text. When the library or registry cannot be
// read, it keeps the installed number, or 1.
func neededHookParts(id string) int {
	n, err := server.PartsNeeded(hookLimits[id])
	if err != nil {
		have, _ := hookParts(id)
		return max(have, 1)
	}
	return n
}

// hookBin is the program the assistant's hooks run: Claude Code's is started
// as its install found it, the others' is the binary itself.
func hookBin(id string) (string, error) {
	if id == claudeCode {
		bin, _, err := command()
		return bin, err
	}
	return appCommand()
}

// hookInstalled reports whether the assistant has Quiver's session-start hook,
// and whether every command runs this binary.
func hookInstalled(id string) (present, current bool, err error) {
	bin, err := hookBin(id)
	if err != nil {
		return false, false, err
	}
	switch id {
	case claudeCode:
		return assistant.ClaudeHookInstalled(bin)
	case "cursor":
		return assistant.CursorHookInstalled(bin)
	case "antigravity":
		return assistant.AntigravityHookInstalled(bin)
	case "vscode":
		return assistant.VSCodeHookInstalled(bin)
	case codexID:
		return codexHookInstalled(bin)
	}
	return false, false, fmt.Errorf("%s has no session-start hook", id)
}

// codexHookInstalled is hookInstalled for Codex.
func codexHookInstalled(bin string) (present, current bool, err error) {
	n, err := assistant.CodexHookParts()
	if err != nil || n == 0 {
		return false, false, err
	}
	return true, assistant.CodexHooksCurrent(bin), nil
}

// syncHooks gives every assistant that has Quiver's hooks, running this
// binary, as many hook commands as its text takes now: a skill made always-on,
// a longer skill or a new project can need another command. It writes a line
// to out for each assistant it changes. Assistants that have no Quiver hook,
// or one that runs another quiver, are left alone; doctor reports those.
func syncHooks(out io.Writer) {
	for _, id := range []string{claudeCode, codexID, "cursor", "antigravity", "vscode"} {
		present, current, err := hookInstalled(id)
		if err != nil || !present || !current {
			continue
		}
		want := neededHookParts(id)
		have, err := hookParts(id)
		if err != nil || have == want {
			continue
		}
		bin, err := hookBin(id)
		if err != nil {
			continue
		}
		res, err := installHooks(id, bin, want)
		if err != nil || res == assistant.Unchanged {
			continue
		}
		if id == codexID {
			_, _ = assistant.CodexTrustHooks()
		}
		fmt.Fprintf(out, "Set the %s session-start hooks to %s, as many as your skills' text needs.\n", assistantName(id), plural(want, "command"))
	}
}

// installHooks writes the assistant's hooks, parts commands each.
func installHooks(id, bin string, parts int) (assistant.Result, error) {
	switch id {
	case claudeCode:
		return assistant.InstallClaudeHooks(bin, parts)
	case codexID:
		return assistant.InstallCodexHooks(bin, parts)
	case "cursor":
		return assistant.InstallCursorHooks(bin, parts)
	case "antigravity":
		return assistant.InstallAntigravityHooks(bin, parts)
	case "vscode":
		return assistant.InstallVSCodeHooks(bin, parts)
	}
	return assistant.Unchanged, fmt.Errorf("%s has no session-start hook", id)
}

// assistantName is the name of the assistant for messages.
func assistantName(id string) string {
	switch id {
	case claudeCode:
		return "Claude Code"
	case codexID:
		return "Codex"
	case "cursor":
		return "Cursor"
	case "antigravity":
		return "Antigravity"
	case "vscode":
		return "VS Code"
	}
	return id
}

// actualDelivery is how always-on skills reach the assistant now. An
// assistant that has a hook level gets it only when its config holds Quiver's
// session-start hook: an install that could not write the file, such as one
// that holds comments, leaves the assistant on the skill list alone. missing
// reports that case.
func (t mcpTarget) actualDelivery() (d assistant.Delivery, missing bool) {
	d = t.delivery()
	if !d.HasHook() {
		return d, false
	}
	if present, _, err := hookInstalled(t.id); err != nil || !present {
		return assistant.DeliveryInstructions, true
	}
	return d, false
}
