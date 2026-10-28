// Package assistanttest puts a fake claude command on PATH for tests.
package assistanttest

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// script answers "mcp get", "mcp add" and "mcp remove" for the server named
// quiver like Claude Code does. $DIR/user holds the user registration as
// "<command>|<args>"; $DIR/local holds one local registration as
// "<folder>|<command>|<args>", which wins in that folder. Every call is
// appended to $DIR/calls.
const script = `#!/bin/sh
dir="$(dirname "$0")"
echo "$*" >> "$dir/calls"
case "$1 $2" in
"mcp get")
  if [ -f "$dir/local" ]; then
    IFS='|' read -r folder command args < "$dir/local"
    if [ "$folder" = "$(pwd -P)" ]; then
      printf 'quiver:\n  Scope: Local config (private to you in this project)\n  Command: %s\n  Args: %s\n' "$command" "$args"
      exit 0
    fi
  fi
  if [ ! -f "$dir/user" ]; then echo "No MCP server named \"$3\". Configured servers: graft"; exit 1; fi
  IFS='|' read -r command args < "$dir/user"
  printf 'quiver:\n  Scope: User config (available in all your projects)\n  Status: ✔ Connected\n  Type: stdio\n  Command: %s\n  Args: %s\n  Environment:\n' "$command" "$args"
  ;;
"mcp add")
  # mcp add --scope user quiver -- <command> <args>
  if [ -f "$dir/user" ]; then echo "MCP server quiver already exists in user config"; exit 1; fi
  printf '%s|%s\n' "$7" "$8" > "$dir/user"
  echo "Added stdio MCP server quiver"
  ;;
"mcp remove")
  if [ ! -f "$dir/user" ]; then echo "No MCP server named \"$3\" in user scope"; exit 1; fi
  rm "$dir/user"
  echo "Removed MCP server quiver"
  ;;
*) echo "unexpected: $*"; exit 2 ;;
esac
`

// Fake is a fake claude command.
type Fake struct{ dir string }

// Install puts a fake claude command first on PATH, with no registration.
func Install(t *testing.T) *Fake {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake claude command is a shell script")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return &Fake{dir}
}

// SetUser stores a user registration that runs command with args.
func (f *Fake) SetUser(t *testing.T, command, args string) {
	t.Helper()
	f.write(t, "user", command+"|"+args)
}

// SetLocal stores a local registration that wins in the current folder.
func (f *Fake) SetLocal(t *testing.T, command, args string) {
	t.Helper()
	wd, err := os.Getwd()
	if err == nil {
		wd, err = filepath.EvalSymlinks(wd)
	}
	if err != nil {
		t.Fatal(err)
	}
	f.write(t, "local", wd+"|"+command+"|"+args)
}

// User returns the user registration as "<command>|<args>", or "" for none.
func (f *Fake) User() string {
	b, _ := os.ReadFile(filepath.Join(f.dir, "user"))
	return strings.TrimSpace(string(b))
}

// Calls returns the argument lines of every call so far.
func (f *Fake) Calls() []string {
	b, _ := os.ReadFile(filepath.Join(f.dir, "calls"))
	if len(b) == 0 {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

func (f *Fake) write(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.dir, name), []byte(content+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
