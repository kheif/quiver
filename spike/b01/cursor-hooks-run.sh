#!/bin/bash
# usage: cursor-hooks-run.sh <dir> [forms|prompts]...
# Measures hooks in the Cursor CLI (cursor-agent -p). <dir> is an absolute
# path to a new folder; every check gets its own workspace there, with the
# server in .cursor/mcp.json and the hooks in .cursor/hooks.json or
# .claude/settings.json, so your own ~/.cursor is not changed. Summarize with
# cursor-hooks-analyze.py <dir>. Needs $QUIVER and a cursor-agent login. See
# docs/spike-hooks.md.
set -u
SPIKE=$(cd "$(dirname "$0")" && pwd)
dir=$1
shift
RULE=$SPIKE/always/reply-style/SKILL.md
CAT='Suggest three names for a cat.'
mkdir -p "$dir/bin"

# context.py <form> <file>: logs stdin to hook.log in the working folder and
# prints the file for the model: cursor ({"additional_context": ...}) or
# claude (hookSpecificOutput.additionalContext of SessionStart).
cat > "$dir/bin/context.py" <<'EOF'
#!/usr/bin/env python3
import json, os, sys
data = sys.stdin.read()
open(os.path.join(os.getcwd(), "hook.log"), "a").write(data.strip() + "\n")
form, text = sys.argv[1], open(sys.argv[2]).read()
if form == "claude":
    print(json.dumps({"hookSpecificOutput": {"hookEventName": "SessionStart", "additionalContext": text}}))
else:
    print(json.dumps({"additional_context": text}))
EOF
chmod +x "$dir/bin/context.py"
CTX=$dir/bin/context.py

# workspace <name>: a folder with the server and a Quiver home where the
# fitting skills are assigned and reply-style is not.
workspace() {
  local w=$dir/$1 h=$dir/home
  if [ ! -d "$h" ]; then
    QUIVER_HOME=$h "$QUIVER" init > /dev/null
    cp -R "$SPIKE"/skills/* "$SPIKE"/always/* "$h/local/"
    QUIVER_HOME=$h "$QUIVER" add sql-migration-review release-notes k8s-debug --global > /dev/null
  fi
  mkdir -p "$w/.cursor" "$w/out"
  printf '{"mcpServers":{"quiver":{"command":"%s","args":["mcp"],"env":{"QUIVER_HOME":"%s"}}}}\n' "$QUIVER" "$h" > "$w/.cursor/mcp.json"
  echo '{"permissions":{"allow":["Mcp(quiver:*)"],"deny":[]}}' > "$w/.cursor/cli.json"
}

# cursor_hook <name> <event> <form>: a .cursor/hooks.json with one hook.
cursor_hook() {
  printf '{"version":1,"hooks":{"%s":[{"command":"%s %s %s"}]}}\n' "$2" "$CTX" "$3" "'$RULE'" > "$dir/$1/.cursor/hooks.json"
}

# ask <name> <id> <prompt>: one headless session in that workspace.
ask() {
  local w=$dir/$1 t0=$SECONDS
  (cd "$w" && cursor-agent -p "$3" --model auto --output-format stream-json --approve-mcps --trust \
    < /dev/null > "$w/out/$2.jsonl" 2>&1)
  echo "$2 $((SECONDS - t0))" >> "$w/seconds.log"
}

# forms: which hook files and events add text, two sessions each.
check_forms() {
  local w
  workspace session-start
  cursor_hook session-start sessionStart cursor
  workspace before-submit
  cursor_hook before-submit beforeSubmitPrompt cursor
  workspace claude
  mkdir -p "$dir/claude/.claude"
  printf '{"hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"%s claude %s"}]}]}}\n' "$CTX" "'$RULE'" > "$dir/claude/.claude/settings.json"
  workspace none
  for w in session-start before-submit claude none; do
    ask $w cat-1 "$CAT"
    ask $w cat-2 "$CAT"
  done
}

# prompts: the 10 prompts with a sessionStart hook.
check_prompts() {
  local id prompt
  workspace prompts
  cursor_hook prompts sessionStart cursor
  while IFS='|' read -r id prompt; do ask prompts "$id" "$prompt"; done < "$SPIKE/prompts.txt"
}

[ $# -gt 0 ] || set -- forms prompts
for c in "$@"; do
  (check_"$c")
  echo "done $c"
done
