#!/bin/bash
# usage: antigravity-hooks-run.sh <dir> [prompts|forms|size|subagents|global]...
# Measures hooks and rules files in the Antigravity CLI (agy -p). <dir> is an
# absolute path to a new folder; every check gets its own workspace there, with
# the server in .agents/mcp_config.json and the hooks in .agents/hooks.json,
# the workspace's customization root, so ~/.gemini is not changed. The global
# check needs $AGY_HOME, a throwaway home folder where `HOME=$AGY_HOME agy` was
# signed in once: it writes that home's ~/.gemini/config/hooks.json, GEMINI.md
# and AGENTS.md and ~/.gemini/GEMINI.md, one at a time. agy keeps each
# session under <home>/.gemini/antigravity-cli/brain. Summarize with
# antigravity-hooks-analyze.py <dir>. Needs $QUIVER and an agy login. See
# docs/spike-hooks.md.
set -u
SPIKE=$(cd "$(dirname "$0")" && pwd)
dir=$1
shift
RULE=$SPIKE/always/reply-style/SKILL.md
CAT='Suggest three names for a cat.'
mkdir -p "$dir/bin" "$dir/bodies"

# inject.py <form> <file>: logs stdin to hook.log in the session's workspace
# and prints the file as an injected step: system (systemMessage), user
# (userMessage), ephemeral (ephemeralMessage) or text (plain stdout).
cat > "$dir/bin/inject.py" <<'EOF'
#!/usr/bin/env python3
import json, os, sys
data = sys.stdin.read()
ws = (json.loads(data).get("workspacePaths") or [os.getcwd()])[0]
open(os.path.join(ws, "hook.log"), "a").write(data.strip() + "\n")
form, text = sys.argv[1], open(sys.argv[2]).read()
step = {"system": {"systemMessage": {"systemMessage": text}}, "user": {"userMessage": text},
        "ephemeral": {"ephemeralMessage": text}}.get(form)
print(text if step is None else json.dumps({"injectSteps": [step]}))
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
INJECT=$dir/bin/inject.py

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

# workspace <name> <quiver home>: a folder with the server in .agents.
workspace() {
  local w=$dir/$1
  mkdir -p "$w/.agents" "$w/out"
  printf '{"mcpServers":{"quiver":{"command":"%s","args":["mcp"],"env":{"QUIVER_HOME":"%s"}}}}\n' "$QUIVER" "$2" > "$w/.agents/mcp_config.json"
}

# hook <file> <event> <command>: a hooks.json with one named hook.
hook() {
  printf '{"quiver":{"%s":[{"type":"command","command":"%s"}]}}\n' "$2" "$3" > "$1"
}

# ask <name> <id> <prompt>: one headless session in that workspace, with
# $AGY_HOME as the home when it is set. Headless agy ends a session without
# an answer when a tool needs a permission it cannot ask for, and the model
# often lists Quiver's folder first, so ls and pwd are allowed.
ask() {
  local w=$dir/$1 t0=$SECONDS
  (cd "$w" && HOME=${AGY_HOME:-$HOME} agy -p "$3" --output-format stream-json \
    --settings '{"permissions":{"allow":["mcp(quiver/*)","command(ls)","command(pwd)"]}}' < /dev/null > "$w/out/$2.jsonl" 2>&1)
  echo "$2 $((SECONDS - t0))" >> "$w/seconds.log"
}
prompts10() {
  local id prompt
  while IFS='|' read -r id prompt; do ask "$1" "$id" "$prompt"; done < "$SPIKE/prompts.txt"
}

# prompts: the 10 prompts with a SessionStart hook; reply-style is not
# assigned, so only the hook brings it.
check_prompts() {
  workspace prompts "$(home plain)"
  hook "$dir/prompts/.agents/hooks.json" SessionStart "$INJECT system '$RULE'"
  prompts10 prompts
}

# forms: the other injected step types, plain stdout, PreInvocation, a failing
# hook, and no hook, two sessions each.
check_forms() {
  local h w
  h=$(home plain)
  for w in user ephemeral text; do
    workspace form-$w "$h"
    hook "$dir/form-$w/.agents/hooks.json" SessionStart "$INJECT $w '$RULE'"
  done
  workspace form-pre-invocation "$h"
  hook "$dir/form-pre-invocation/.agents/hooks.json" PreInvocation "$INJECT system '$RULE'"
  workspace form-exit1 "$h"
  hook "$dir/form-exit1/.agents/hooks.json" SessionStart "$INJECT system '$RULE'; exit 1"
  workspace form-none "$h"
  for w in user ephemeral text pre-invocation exit1 none; do
    ask form-$w cat-1 "$CAT"
    ask form-$w cat-2 "$CAT"
  done
}

# size: which part of a long hook text reaches the model?
check_size() {
  local h n
  h=$(home plain)
  for n in 20000 50000 200000; do
    "$dir/bin/body.py" $n $n > "$dir/bodies/$n.txt"
    workspace size-$n "$h"
    hook "$dir/size-$n/.agents/hooks.json" SessionStart "$INJECT system '$dir/bodies/$n.txt'"
    ask size-$n cat-1 "$CAT"
  done
}

# subagents: does a sub-agent get the SessionStart text?
check_subagents() {
  workspace sub "$(home plain)"
  hook "$dir/sub/.agents/hooks.json" SessionStart "$INJECT system '$RULE'"
  local p='Use the invoke_subagent tool to start one subagent with exactly this task, adding nothing to it: "Suggest three names for a cat." Then print its reply verbatim.'
  ask sub sub-1 "$p"
  ask sub sub-2 "$p"
}

# global: files in the global customization root ($AGY_HOME/.gemini/config)
# and ~/.gemini/GEMINI.md, one at a time.
check_global() {
  [ -n "${AGY_HOME:-}" ] || { echo "global needs AGY_HOME" >&2; return; }
  local g=$AGY_HOME/.gemini/config h
  local pointer='Before your first reply, load the tools of the quiver MCP server and follow its instructions.'
  h=$(home always always)
  # A hook in the global hooks.json.
  workspace global-hook "$(home plain)"
  hook "$g/hooks.json" SessionStart "$INJECT system '$RULE'"
  ask global-hook cat-1 "$CAT"
  ask global-hook cat-2 "$CAT"
  rm "$g/hooks.json"
  # The pointer in the global GEMINI.md, the 10 prompts.
  workspace global-gemini "$h"
  echo "$pointer" > "$g/GEMINI.md"
  prompts10 global-gemini
  rm "$g/GEMINI.md"
  # The same in the global AGENTS.md, and in ~/.gemini/GEMINI.md.
  workspace global-agents "$h"
  echo "$pointer" > "$g/AGENTS.md"
  ask global-agents cat-1 "$CAT"
  ask global-agents cat-2 "$CAT"
  rm "$g/AGENTS.md"
  workspace gemini-home "$h"
  echo "$pointer" > "$AGY_HOME/.gemini/GEMINI.md"
  ask gemini-home cat-1 "$CAT"
  ask gemini-home cat-2 "$CAT"
  rm "$AGY_HOME/.gemini/GEMINI.md"
  # A global rule and a project AGENTS.md with another rule: added or replaced?
  workspace both-rules "$(home plain)"
  echo 'Start every reply with the line [global-rule].' > "$g/GEMINI.md"
  echo 'End every reply with the line [project-rule].' > "$dir/both-rules/AGENTS.md"
  ask both-rules cat-1 "$CAT"
  ask both-rules cat-2 "$CAT"
  rm "$g/GEMINI.md"
}

[ $# -gt 0 ] || set -- prompts forms size subagents global
for c in "$@"; do
  (check_"$c")
  echo "done $c"
done
