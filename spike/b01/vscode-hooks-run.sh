#!/bin/bash
# usage: vscode-hooks-run.sh <dir> [mcp|prompts|both|forms|size|failures|subagents|user-hooks|user-mcp|instructions|copilot]...
# Measures hooks, MCP and instructions files in VS Code Copilot Chat (agent
# mode). <dir> is an absolute path to a new folder; every check gets its own
# workspace there. VS Code runs as a separate instance with its own profile in
# $VSC/data (default /tmp/qvsc: the path must be short, since VS Code puts its
# IPC socket there), so your own VS Code profile is not changed. Sign in to
# GitHub Copilot in that instance once before the first run. The instance
# loads the extension in vscode-driver, which sends each prompt in a new chat;
# answers are read from the chat session files with vscode-session.py. The
# profile approves every tool call, so Copilot runs its commands in the spike
# workspaces without asking, and it turns off personal skills and hooks, which
# VS Code reads from your home folder whatever the profile. COPILOT_HOME is
# $VSC/copilot-home, where the Copilot harness keeps its sessions and reads
# user hooks, so your ~/.copilot is not used either; user-hooks puts a hook
# file in ~/.quiver-spike-hooks for one run and removes it.
# Summarize with vscode-hooks-analyze.py <dir>. Needs $QUIVER. See
# docs/spike-hooks.md.
set -u
SPIKE=$(cd "$(dirname "$0")" && pwd)
dir=$1
shift
VSC=${VSC:-/tmp/qvsc}
RULE=$SPIKE/always/reply-style/SKILL.md
CAT='Suggest three names for a cat.'
mkdir -p "$dir/bin" "$dir/bodies" "$VSC/data/User" "$VSC/ext"
mkdir "$VSC/lock" 2> /dev/null || { echo "another run uses $VSC (remove $VSC/lock if not)" >&2; exit 1; }
trap 'rmdir "$VSC/lock"' EXIT

# context.py <event> <file> [form]: logs stdin to the workspace's hook.log and
# prints the file for the model. form is nested (hookSpecificOutput, the
# default), top (additionalContext at the top level), dual (both) or text
# (plain stdout).
cat > "$dir/bin/context.py" <<'EOF'
#!/usr/bin/env python3
import json, os, sys
data = sys.stdin.read()
open(os.path.join(os.getcwd(), "hook.log"), "a").write(data.strip() + "\n")
event, text = sys.argv[1], open(sys.argv[2]).read()
form = sys.argv[3] if len(sys.argv) > 3 else "nested"
if form == "text":
    print(text)
elif form == "top":
    print(json.dumps({"additionalContext": text}))
elif form == "dual":
    print(json.dumps({"additionalContext": text, "hookSpecificOutput": {"hookEventName": event, "additionalContext": text}}))
else:
    print(json.dumps({"hookSpecificOutput": {"hookEventName": event, "additionalContext": text}}))
EOF
# body.py <chars> <tag>: <chars> characters between and including a
# START-TOKEN-<tag> and an END-TOKEN-<tag> line, MID-TOKEN-<tag> in the middle.
cat > "$dir/bin/body.py" <<'EOF'
#!/usr/bin/env python3
import sys
n, tag = int(sys.argv[1]), sys.argv[2]
head, mid, tail = f"START-TOKEN-{tag}\n", f"MID-TOKEN-{tag}\n", f"END-TOKEN-{tag}\n"
rest, lines = n - len(head) - len(mid) - len(tail), []
while rest > 0:
    k = min(79, rest - 1)
    lines.append("x" * k + "\n")
    rest -= k + 1
lines.insert(len(lines) // 2, mid)
sys.stdout.write(head + "".join(lines) + tail)
EOF
chmod +x "$dir"/bin/*
CTX=$dir/bin/context.py

# settings [key=json]...: the instance's user settings, with these added. A
# chat.hookFilesLocations value is added to the default workspace locations.
settings() {
  python3 - "$VSC/data/User/settings.json" "$@" <<'EOF'
import json, sys
hooks = {".github/hooks": True, ".claude/settings.local.json": True, ".claude/settings.json": True,
         "~/.copilot/hooks": False, "~/.claude/settings.json": False}
s = {"workbench.startupEditor": "none", "telemetry.telemetryLevel": "off", "update.mode": "none",
     "extensions.autoUpdate": False, "security.workspace.trust.enabled": False,
     "chat.defaultToCopilotHarness": False, "chat.tools.global.autoApprove": True,
     "chat.useAgentSkills": False, "chat.hookFilesLocations": hooks,
     "chat.instructionsFilesLocations": {".github/instructions": True, "~/.copilot/instructions": False}}
for kv in sys.argv[2:]:
    k, v = kv.split("=", 1)
    s[k] = dict(hooks, **json.loads(v)) if k == "chat.hookFilesLocations" else json.loads(v)
json.dump(s, open(sys.argv[1], "w"), indent=2)
EOF
}

# home <name> [always]: a Quiver home with the fixture skills; the fitting
# ones are assigned, and reply-style too, as always-on, with `always`.
home() {
  local h=$dir/homes/$1
  if [ ! -d "$h" ]; then
    QUIVER_HOME=$h "$QUIVER" init > /dev/null
    cp -R "$SPIKE"/skills/* "$SPIKE"/always/* "$h/local/"
    QUIVER_HOME=$h "$QUIVER" add sql-migration-review release-notes k8s-debug --global > /dev/null
    [ -z "${2:-}" ] || QUIVER_HOME=$h "$QUIVER" add reply-style --global --always > /dev/null
  fi
  echo "$h"
}

# workspace <name> <quiver home>: a folder with the server in .vscode/mcp.json.
workspace() {
  local w=$dir/$1
  mkdir -p "$w/.vscode" "$w/.github/hooks"
  printf '{"servers":{"quiver":{"type":"stdio","command":"%s","args":["mcp"],"env":{"QUIVER_HOME":"%s"}}}}\n' "$QUIVER" "$2" > "$w/.vscode/mcp.json"
}

# hooks <file> <event>=<command>...: a hook file in the flat VS Code form.
hooks() {
  local f=$1
  shift
  python3 - "$@" > "$f" <<'EOF'
import json, sys
h = {}
for a in sys.argv[1:]:
    event, command = a.split("=", 1)
    h.setdefault(event, []).append({"type": "command", "command": command})
print(json.dumps({"hooks": h}, indent=2))
EOF
}

# open_ws <dir>: opens a workspace in the instance and waits for the driver.
open_ws() {
  rm -rf "$1/.queue"
  open -n -a "Visual Studio Code" --env COPILOT_HOME="$VSC/copilot-home" \
    --args --user-data-dir "$VSC/data" --extensions-dir "$VSC/ext" \
    --extensionDevelopmentPath="$SPIKE/vscode-driver" --new-window "$1"
  local i
  for i in $(seq 1 60); do [ -d "$1/.queue" ] && return; sleep 1; done
  echo "$1: the driver did not start" >&2
}
close_ws() {
  touch "$1/.queue/close"
  sleep 3
}
quit_vscode() {
  pkill -f "user-data-dir $VSC/data"
  sleep 3
}

# storage <dir>: the workspace's storage folder in the instance.
storage() {
  grep -l "\"file://$1\"" "$VSC"/data/User/workspaceStorage/*/workspace.json 2>/dev/null | head -1 | xargs dirname 2>/dev/null
}

# ask <dir> <id> <prompt>: sends a prompt in a new chat and waits until a new
# answer is saved, at most 10 minutes. Writes <dir>/asked.log: id, session
# file, seconds. vscode-hooks-analyze.py matches sessions by their prompt.
ask() {
  local w=$1 id=$2 prompt=$3 t0=$SECONDS before s i
  [ -d "$w/.queue" ] || { echo "$id no-driver 0" >> "$w/asked.log"; return; }
  s=$(storage "$w")
  before=$(python3 "$SPIKE/vscode-session.py" --count "$s/chatSessions")
  printf '%s\n' "$prompt" > "$w/.queue/$id.txt"
  for i in $(seq 1 600); do
    s=$(storage "$w")
    if [ -n "$s" ] && python3 "$SPIKE/vscode-session.py" --done "$s/chatSessions" "$before" > "$w/.last" 2> /dev/null; then
      echo "$id $(cat "$w/.last") $((SECONDS - t0))" >> "$w/asked.log"
      return
    fi
    sleep 1
  done
  echo "$id timeout $((SECONDS - t0))" >> "$w/asked.log"
}

# ask_copilot <dir> <id> <prompt>: ask for the Copilot harness, whose sessions
# are $VSC/copilot-home/session-state/<id>/events.jsonl.
ask_copilot() {
  local w=$1 id=$2 t0=$SECONDS i
  printf '%s\n' "$3" > "$w/.queue/$id.txt"
  for i in $(seq 1 300); do
    if grep -l "\"cwd\": *\"$w\"" "$VSC"/copilot-home/session-state/*/events.jsonl 2> /dev/null |
      xargs grep -l '"final_answer"' 2> /dev/null | grep -q .; then
      echo "$id copilot $((SECONDS - t0))" >> "$w/asked.log"
      return
    fi
    sleep 2
  done
  echo "$id timeout $((SECONDS - t0))" >> "$w/asked.log"
}

# run <name> [prompts]: opens a workspace prepared by the caller, asks the 10
# prompts or the cat prompt, closes it.
run() {
  local w=$dir/$1 id prompt
  open_ws "$w"
  if [ "${2:-cat}" = prompts ]; then
    while IFS='|' read -r id prompt; do ask "$w" "$id" "$prompt"; done < "$SPIKE/prompts.txt"
  else
    ${ASK:-ask} "$w" cat "$CAT"
  fi
  close_ws "$w"
}

# mcp: the 10 prompts with the server only; reply-style is assigned as
# always-on, so only the server's instructions bring it.
check_mcp() {
  settings
  workspace mcp "$(home always always)"
  run mcp prompts
}

# prompts: the 10 prompts with the hook; reply-style is not assigned.
check_prompts() {
  settings
  workspace prompts "$(home plain)"
  hooks "$dir/prompts/.github/hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE'"
  run prompts prompts
}

# both: the hook, and reply-style also assigned as always-on.
check_both() {
  settings
  workspace both "$(home always always)"
  hooks "$dir/both/.github/hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE'"
  run both prompts
}

# forms: which hook files and output forms reach the model, one chat each.
check_forms() {
  settings
  local h w
  h=$(home plain)
  workspace form-text "$h"
  hooks "$dir/form-text/.github/hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE' text"
  workspace form-top "$h"
  hooks "$dir/form-top/.github/hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE' top"
  workspace form-camel "$h"
  hooks "$dir/form-camel/.github/hooks/quiver.json" "sessionStart=$CTX SessionStart '$RULE'"
  # Claude Code's nested form with a matcher, in .github/hooks.
  workspace form-nested "$h"
  printf '{"hooks":{"SessionStart":[{"matcher":"startup","hooks":[{"type":"command","command":"%s SessionStart %s"}]}]}}\n' "$CTX" "'$RULE'" > "$dir/form-nested/.github/hooks/quiver.json"
  # A workspace .claude/settings.json, read only with chat.useClaudeHooks.
  workspace form-claude "$h"
  mkdir -p "$dir/form-claude/.claude"
  printf '{"hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"%s SessionStart %s"}]}]}}\n' "$CTX" "'$RULE'" > "$dir/form-claude/.claude/settings.json"
  workspace form-claude-on "$h"
  cp -R "$dir/form-claude/.claude" "$dir/form-claude-on/"
  workspace form-dual "$h"
  hooks "$dir/form-dual/.github/hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE' dual"
  for w in form-text form-top form-dual form-camel form-nested form-claude; do run $w; done
  settings chat.useClaudeHooks=true
  run form-claude-on
  # chat.hookFilesLocations takes no absolute path.
  mkdir -p "$dir/user-hooks"
  hooks "$dir/user-hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE'"
  settings "chat.hookFilesLocations={\"$dir/user-hooks\":true}"
  workspace form-user "$h"
  run form-user
  settings
}

# user-hooks: user-level hook files in the Local harness: one in
# $COPILOT_HOME/hooks (does Local follow COPILOT_HOME?), one in a ~/ folder
# named in chat.hookFilesLocations.
check_user_hooks() {
  local h
  h=$(home plain)
  mkdir -p "$VSC/copilot-home/hooks" "$HOME/.quiver-spike-hooks"
  hooks "$VSC/copilot-home/hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE'"
  settings "chat.hookFilesLocations={\"~/.copilot/hooks\":true}"
  workspace user-copilot-home "$h"
  run user-copilot-home
  rm "$VSC/copilot-home/hooks/quiver.json"
  hooks "$HOME/.quiver-spike-hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE'"
  settings 'chat.hookFilesLocations={"~/.quiver-spike-hooks":true}'
  workspace user-home "$h"
  run user-home
  rm -r "$HOME/.quiver-spike-hooks"
  settings
}

# size: which part of a long hook text reaches the model?
check_size() {
  settings
  local n h
  h=$(home plain)
  for n in 20000 50000 200000; do
    "$dir/bin/body.py" $n $n > "$dir/bodies/$n.txt"
    workspace size-$n "$h"
    hooks "$dir/size-$n/.github/hooks/quiver.json" "SessionStart=$CTX SessionStart '$dir/bodies/$n.txt'"
    run size-$n
  done
}

# failures: a hook that is slow, fails or times out.
check_failures() {
  settings
  local h t
  h=$(home plain)
  workspace fail-none "$h"
  workspace fail-sleep5 "$h"
  hooks "$dir/fail-sleep5/.github/hooks/quiver.json" "SessionStart=sleep 5; $CTX SessionStart '$RULE'"
  workspace fail-exit1 "$h"
  hooks "$dir/fail-exit1/.github/hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE'; exit 1"
  workspace fail-exit2 "$h"
  hooks "$dir/fail-exit2/.github/hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE'; echo boom >&2; exit 2"
  workspace fail-timeout "$h"
  printf '{"hooks":{"SessionStart":[{"type":"command","command":"%s SessionStart %s; sleep 8","timeout":2}]}}\n' "$CTX" "'$RULE'" > "$dir/fail-timeout/.github/hooks/quiver.json"
  workspace fail-missing "$h"
  hooks "$dir/fail-missing/.github/hooks/quiver.json" "SessionStart=/nonexistent/quiver hook session-start vscode"
  for t in none sleep5 exit1 exit2 timeout missing; do run fail-$t; done
}

# subagents: does a sub-agent get the text, with and without SubagentStart?
check_subagents() {
  settings
  local h p
  h=$(home plain)
  p='Use the runSubagent tool to start one subagent with exactly this prompt, adding nothing to it: "Suggest three names for a cat." Then print its reply verbatim.'
  workspace sub-only "$h"
  hooks "$dir/sub-only/.github/hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE'"
  workspace sub-with "$h"
  hooks "$dir/sub-with/.github/hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE'" "SubagentStart=$CTX SubagentStart '$RULE'"
  for w in sub-only sub-with; do
    open_ws "$dir/$w"
    ask "$dir/$w" sub-1 "$p"
    ask "$dir/$w" sub-2 "$p"
    close_ws "$dir/$w"
  done
}

# user-mcp: the server in the instance's user mcp.json instead of the
# workspace's: which folder does it start in, with and without a cwd?
check_user_mcp() {
  local h w
  h=$(home always always)
  for w in user-mcp user-mcp-cwd; do
    mkdir -p "$dir/$w"
    quit_vscode
    settings
    if [ $w = user-mcp ]; then
      printf '{"servers":{"quiver":{"type":"stdio","command":"%s","args":["mcp"],"env":{"QUIVER_HOME":"%s"}}}}\n' "$QUIVER" "$h" > "$VSC/data/User/mcp.json"
    else
      printf '{"servers":{"quiver":{"type":"stdio","command":"%s","args":["mcp"],"cwd":"${workspaceFolder}","env":{"QUIVER_HOME":"%s"}}}}\n' "$QUIVER" "$h" > "$VSC/data/User/mcp.json"
    fi
    mkdir -p "$h/logs"
    : > "$h/logs/mcp.log"
    run $w
    cp "$h/logs/mcp.log" "$dir/$w/mcp.log" 2> /dev/null
  done
  rm -f "$VSC/data/User/mcp.json"
}

# instructions: a user-level instructions file that points the model to Quiver.
check_instructions() {
  mkdir -p "$VSC/data/User/prompts"
  cat > "$VSC/data/User/prompts/quiver.instructions.md" <<'EOF'
---
applyTo: "**"
---
Before your first reply, load the tools of the quiver MCP server and follow its instructions.
EOF
  settings
  workspace instructions "$(home always always)"
  run instructions prompts
  rm -f "$VSC/data/User/prompts/quiver.instructions.md"
}

# copilot: the Copilot harness instead of the Local one: a hook file in the
# Local form with nested output, the same with both outputs, the Copilot form
# (version 1, camelCase, top-level output), and a user hook in
# $COPILOT_HOME/hooks with both outputs.
check_copilot() {
  quit_vscode
  settings chat.defaultToCopilotHarness=true
  local h w
  h=$(home plain)
  workspace copilot-local "$h"
  hooks "$dir/copilot-local/.github/hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE'"
  workspace copilot-dual "$h"
  hooks "$dir/copilot-dual/.github/hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE' dual"
  workspace copilot-form "$h"
  printf '{"version":1,"hooks":{"sessionStart":[{"type":"command","command":"%s sessionStart %s top","timeoutSec":10}]}}\n' "$CTX" "'$RULE'" > "$dir/copilot-form/.github/hooks/quiver.json"
  for w in copilot-local copilot-dual copilot-form; do ASK=ask_copilot run $w; done
  mkdir -p "$VSC/copilot-home/hooks"
  hooks "$VSC/copilot-home/hooks/quiver.json" "SessionStart=$CTX SessionStart '$RULE' dual"
  workspace copilot-user "$h"
  ASK=ask_copilot run copilot-user
  rm "$VSC/copilot-home/hooks/quiver.json"
  quit_vscode
  settings
}

[ $# -gt 0 ] || set -- mcp prompts both forms size failures subagents user-hooks user-mcp instructions copilot
for c in "$@"; do
  (check_"${c//-/_}")
  echo "done $c"
done
