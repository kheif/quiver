// Package assistanttest puts fake claude and codex commands on PATH for
// tests.
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

// Fake is a fake claude or codex command.
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

// SetCodex stores a registration of the fake codex that runs command with
// args, and env ("K=V" or "").
func (f *Fake) SetCodex(t *testing.T, command, args, env string) {
	t.Helper()
	if env != "" {
		env = "--env " + env
	}
	f.write(t, "user", command+"|"+args+"|"+env)
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

// codexScript answers "mcp get --json", "mcp add" and "mcp remove" for the
// server named quiver like Codex does. $DIR/user holds the registration as
// "<command>|<args>|<env>", env being "--env K=V" options or empty. Every
// call is appended to $DIR/calls.
const codexScript = `#!/bin/sh
dir="$(dirname "$0")"
echo "$*" >> "$dir/calls"
if [ "$1" = "app-server" ]; then exec python3 "$dir/appserver.py"; fi
case "$1 $2" in
"mcp get")
  if [ ! -f "$dir/user" ]; then echo "Error: No MCP server named '$3' found."; exit 1; fi
  IFS='|' read -r command args env < "$dir/user"
  envjson=null
  if [ -n "$env" ]; then envjson="{\"$(echo "$env" | sed 's/^--env //; s/=/\": \"/')\"}"; fi
  printf '{"name":"quiver","transport":{"type":"stdio","command":"%s","args":["%s"],"env":%s}}\n' "$command" "$args" "$envjson"
  ;;
"mcp add")
  # mcp add [--env K=V] quiver -- <command> <args>
  shift 2
  env=
  if [ "$1" = "--env" ]; then env="--env $2"; shift 2; fi
  printf '%s|%s|%s\n' "$3" "$4" "$env" > "$dir/user"
  echo "Added global MCP server 'quiver'."
  ;;
"mcp remove")
  if [ ! -f "$dir/user" ]; then echo "No MCP server named '$3' found."; exit 0; fi
  rm "$dir/user"
  echo "Removed global MCP server 'quiver'."
  ;;
*) echo "unexpected: $*"; exit 2 ;;
esac
`

// codexAppServer is a fake "codex app-server": it answers the hooks/list and
// config/batchWrite requests Quiver sends to trust its hooks. It reads the
// hooks file ($CODEX_HOME/hooks.json, or ~/.codex/hooks.json) and keeps the
// trusted hashes in trusted.json next to it. A hook's hash is the SHA-256 of
// its command, so moving the binary makes the hook "modified".
const codexAppServer = `import hashlib, json, os, sys

home = os.environ.get("CODEX_HOME") or os.path.join(os.path.expanduser("~"), ".codex")
hooks_path = os.path.join(home, "hooks.json")
trusted_path = os.path.join(home, "trusted.json")
LABEL = {"SessionStart": "session_start", "SubagentStart": "subagent_start"}

def load(path, default):
    try:
        return json.load(open(path))
    except FileNotFoundError:
        return default

def listing():
    doc = load(hooks_path, {"hooks": {}})
    trusted = load(trusted_path, {})
    hooks = []
    for event, groups in doc.get("hooks", {}).items():
        for gi, group in enumerate(groups):
            for hi, h in enumerate(group.get("hooks", [])):
                cmd = h.get("command", "")
                key = "%s:%s:%d:%d" % (hooks_path, LABEL.get(event, event), gi, hi)
                cur = "sha256:" + hashlib.sha256(cmd.encode()).hexdigest()
                status = "untrusted"
                if key in trusted:
                    status = "trusted" if trusted[key] == cur else "modified"
                hooks.append({"key": key, "command": cmd, "currentHash": cur, "trustStatus": status})
    return {"data": [{"cwd": os.getcwd(), "hooks": hooks}]}

def batch(params):
    trusted = load(trusted_path, {})
    for edit in params.get("edits", []):
        for key, val in edit.get("value", {}).items():
            trusted[key] = val["trusted_hash"]
    json.dump(trusted, open(trusted_path, "w"))
    return {}

for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    m = json.loads(line)
    method, mid = m.get("method"), m.get("id")
    if mid is None:
        continue
    if method == "initialize":
        result = {}
    elif method == "hooks/list":
        result = listing()
    elif method == "config/batchWrite":
        result = batch(m.get("params", {}))
    else:
        result = {}
    sys.stdout.write(json.dumps({"jsonrpc": "2.0", "id": mid, "result": result}) + "\n")
    sys.stdout.flush()
`

// InstallCodex puts a fake codex command first on PATH, with no
// registration. The fake keeps at most one env variable and answers the
// app-server hook-trust requests.
func InstallCodex(t *testing.T) *Fake {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake codex command is a shell script")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte(codexScript), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "appserver.py"), []byte(codexAppServer), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return &Fake{dir}
}
