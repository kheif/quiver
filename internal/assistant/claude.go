// Package assistant registers Quiver's MCP server with assistants, so the
// user never edits MCP config by hand.
package assistant

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ServerName is the name Quiver's server is registered under.
const ServerName = "quiver"

// ErrNoClaude means the claude command is not on PATH. The caller prints
// the command for the user to run instead.
var ErrNoClaude = errors.New("the claude command was not found")

// Result says what Install or Uninstall changed.
type Result int

const (
	Unchanged Result = iota // already installed with this binary, or not installed
	Installed
	Updated // the registration pointed to another binary
	Removed
)

// Registration is a "claude mcp get" entry.
type Registration struct {
	Scope   string // such as "User config (available in all your projects)"
	Command string
	Args    string
}

// User reports whether the registration is in the user scope, the one
// Install writes.
func (r *Registration) User() bool {
	return strings.HasPrefix(r.Scope, "User")
}

// HasClaude reports whether the claude command is on PATH.
func HasClaude() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// AddCommand is the command that registers bin with Claude Code, quoted for
// a POSIX shell.
func AddCommand(bin string) string {
	return "claude mcp add --scope user " + ServerName + " -- " + ShellQuote(bin) + " mcp"
}

// RemoveCommand is the command that removes the registration.
func RemoveCommand() string {
	return "claude mcp remove " + ServerName + " --scope user"
}

// ConfigSnippet is the mcpServers entry for assistants configured by JSON.
func ConfigSnippet(bin string) string {
	b, _ := json.MarshalIndent(map[string]any{
		"mcpServers": map[string]any{
			ServerName: serverEntry{bin, []string{"mcp"}},
		},
	}, "", "  ")
	return string(b)
}

// Get returns the registration Claude Code uses in dir, or nil if there is
// none. A local or project registration hides the user one there.
func Get(dir string) (*Registration, error) {
	out, err := claude(dir, "mcp", "get", ServerName)
	if err != nil {
		if strings.Contains(out, "No MCP server named") {
			return nil, nil
		}
		return nil, err
	}
	r := &Registration{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		key, value, ok := strings.Cut(strings.TrimSpace(sc.Text()), ": ")
		if !ok {
			continue
		}
		switch key {
		case "Scope":
			r.Scope = value
		case "Command":
			r.Command = value
		case "Args":
			r.Args = value
		}
	}
	return r, nil
}

// Install registers bin in the user scope of Claude Code. An existing user
// registration that runs another binary is replaced. hidden is a local or
// project registration that runs something else and wins over the user one
// in the current folder, or nil.
func Install(bin string) (res Result, hidden *Registration, err error) {
	res, err = installUser(bin)
	if err != nil {
		return res, nil, err
	}
	cwd, err := Get("")
	if err != nil {
		return res, nil, err
	}
	if cwd != nil && !cwd.User() && (cwd.Command != bin || cwd.Args != "mcp") {
		hidden = cwd
	}
	return res, hidden, nil
}

func installUser(bin string) (Result, error) {
	// No local or project registration applies in a new empty folder, so
	// Get sees the user one there.
	empty, err := os.MkdirTemp("", "quiver-")
	if err != nil {
		return Unchanged, err
	}
	defer func() { _ = os.Remove(empty) }()
	reg, err := Get(empty)
	if err != nil {
		return Unchanged, err
	}
	if reg != nil && reg.Command == bin && reg.Args == "mcp" {
		return Unchanged, nil
	}
	if reg != nil {
		if _, err := removeUser(); err != nil {
			return Unchanged, err
		}
	}
	if _, err := claude("", "mcp", "add", "--scope", "user", ServerName, "--", bin, "mcp"); err != nil {
		return Unchanged, err
	}
	if reg != nil {
		return Updated, nil
	}
	return Installed, nil
}

// Uninstall removes the user registration.
func Uninstall() (Result, error) {
	removed, err := removeUser()
	if err != nil || !removed {
		return Unchanged, err
	}
	return Removed, nil
}

// removeUser removes the user registration. It reports whether there was
// one.
func removeUser() (bool, error) {
	out, err := claude("", "mcp", "remove", ServerName, "--scope", "user")
	if err != nil {
		if strings.Contains(out, "No MCP server named") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// claude runs the claude command in dir ("" for the current folder) and
// returns its combined output.
func claude(dir string, args ...string) (string, error) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return "", ErrNoClaude
	}
	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("claude %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// ShellQuote quotes s for a POSIX shell when it has characters the shell
// would interpret.
func ShellQuote(s string) string {
	if s != "" && strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/._-+@:=,") == "" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
