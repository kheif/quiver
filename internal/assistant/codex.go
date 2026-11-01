package assistant

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// ErrNoCodex means the codex command is not on PATH.
var ErrNoCodex = errors.New("the codex command was not found")

// Codex keeps its MCP servers in ~/.codex/config.toml (or $CODEX_HOME), which
// the Codex CLI and app share. Quiver changes it only through the codex
// command, which keeps the file's comments and other settings.

// HasCodex reports whether the codex command is on PATH.
func HasCodex() bool {
	_, err := exec.LookPath("codex")
	return err == nil
}

// CodexAddCommand is the command that registers bin with Codex, quoted for
// a POSIX shell.
func CodexAddCommand(bin string) string {
	return "codex mcp add " + ServerName + " -- " + ShellQuote(bin) + " mcp"
}

// CodexRemoveCommand is the command that removes the registration.
func CodexRemoveCommand() string {
	return "codex mcp remove " + ServerName
}

// codexEntry is the part of "codex mcp get --json" that Quiver reads.
type codexEntry struct {
	Transport struct {
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env"`
	} `json:"transport"`
}

// CodexGet returns the Codex registration, or nil if there is none. Scope
// is "user".
func CodexGet() (*Registration, error) {
	e, err := codexGet()
	if err != nil || e == nil {
		return nil, err
	}
	return &Registration{Scope: "user", Command: e.Transport.Command, Args: strings.Join(e.Transport.Args, " ")}, nil
}

func codexGet() (*codexEntry, error) {
	out, err := codex("mcp", "get", ServerName, "--json")
	if err != nil {
		if strings.Contains(out, "No MCP server named") {
			return nil, nil
		}
		return nil, err
	}
	// Warnings may come before the JSON.
	if i := strings.Index(out, "{"); i > 0 {
		out = out[i:]
	}
	var e codexEntry
	if err := json.Unmarshal([]byte(out), &e); err != nil {
		return nil, fmt.Errorf("codex mcp get: %w", err)
	}
	return &e, nil
}

// CodexInstall registers bin with Codex, or points an existing registration
// to bin. codex mcp add replaces the whole entry, so the entry's env is
// passed again.
func CodexInstall(bin string) (Result, error) {
	e, err := codexGet()
	if err != nil {
		return Unchanged, err
	}
	res := Installed
	args := []string{"mcp", "add"}
	if e != nil {
		if e.Transport.Command == bin && slices.Equal(e.Transport.Args, []string{"mcp"}) {
			return Unchanged, nil
		}
		res = Updated
		keys := make([]string, 0, len(e.Transport.Env))
		for k := range e.Transport.Env {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			args = append(args, "--env", k+"="+e.Transport.Env[k])
		}
	}
	if _, err := codex(append(args, ServerName, "--", bin, "mcp")...); err != nil {
		return Unchanged, err
	}
	return res, nil
}

// CodexUninstall removes the registration.
func CodexUninstall() (Result, error) {
	out, err := codex("mcp", "remove", ServerName)
	if err != nil {
		return Unchanged, err
	}
	if strings.Contains(out, "No MCP server named") {
		return Unchanged, nil
	}
	return Removed, nil
}

// codex runs the codex command and returns its combined output.
func codex(args ...string) (string, error) {
	path, err := exec.LookPath("codex")
	if err != nil {
		return "", ErrNoCodex
	}
	out, err := exec.Command(path, args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("codex %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
