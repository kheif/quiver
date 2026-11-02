#!/bin/bash
# usage: claude-hooks-run.sh <dir> [prompts|size|failures|sources|subagents|interactive]...
# Measures SessionStart hooks in Claude Code. <dir> is an absolute path to a
# new folder. Each run gets its own folder there, with the hook in that
# folder's .claude/settings.json, and reads no other settings, so your own
# config is not changed. Without a check name it runs all of them. Summarize
# with claude-hooks-analyze.py <dir>. Needs $QUIVER, a claude login, and
# expect for the interactive check. See docs/spike-hooks.md.
set -u
SPIKE=$(cd "$(dirname "$0")" && pwd)
dir=$1
shift
RULE=$SPIKE/always/reply-style/SKILL.md
HAIKU=(--model haiku --disallowedTools Read,Bash,Grep,Glob,Agent)
mkdir -p "$dir/bin" "$dir/bodies" "$dir/with space"

# body.py <chars> <tag> [fill]: exactly <chars> characters between and
# including a START-TOKEN-<tag> line and an END-TOKEN-<tag> line.
cat > "$dir/bin/body.py" <<'EOF'
#!/usr/bin/env python3
import sys
n, tag = int(sys.argv[1]), sys.argv[2]
fill = sys.argv[3] if len(sys.argv) > 3 else "x"
head, tail = f"START-TOKEN-{tag}\n", f"END-TOKEN-{tag}\n"
rest, lines = n - len(head) - len(tail), []
while rest > 0:
    k = min(79, rest - 1)
    lines.append(fill * k + "\n")
    rest -= k + 1
sys.stdout.write(head + "".join(lines) + tail)
EOF
# context.py <event> <file>: prints a file as additionalContext.
cat > "$dir/bin/context.py" <<'EOF'
#!/usr/bin/env python3
import json, sys
print(json.dumps({"hookSpecificOutput": {"hookEventName": sys.argv[1], "additionalContext": open(sys.argv[2]).read()}}))
EOF
# token.sh: prints a token that names the source of this firing.
cat > "$dir/bin/token.sh" <<'EOF'
#!/bin/bash
s=$(cat); echo "$s" >> "$HOOK_LOG"
echo "CONTEXT-TOKEN-$(echo "$s" | python3 -c 'import json,sys;print(json.load(sys.stdin)["source"])')"
EOF
chmod +x "$dir"/bin/*
# A hook logs its stdin to the run's hook.log, then does the rest.
log='cat >> "$HOOK_LOG"; echo >> "$HOOK_LOG"; '

# settings <event> <matcher or -> <hook>...: prints a settings file. A hook
# that starts with { is a JSON entry, anything else is a command.
settings() {
  python3 - "$@" <<'EOF'
import json, sys
event, matcher, hooks = sys.argv[1], sys.argv[2], sys.argv[3:]
entry = {"hooks": [json.loads(h) if h.startswith("{") else {"type": "command", "command": h} for h in hooks]}
if matcher != "-":
    entry["matcher"] = matcher
print(json.dumps({"hooks": {event: [entry]}}))
EOF
}

# run <name> <settings> <prompt> [claude option]...: one session in <dir>/<name>.
run() {
  local name=$1 set=$2 prompt=$3
  shift 3
  local d=$dir/$name
  mkdir -p "$d/.claude"
  echo "$set" > "$d/.claude/settings.json"
  local t0=$SECONDS
  (cd "$d" && HOOK_LOG=$d/hook.log claude -p "$prompt" --strict-mcp-config --setting-sources project \
    --output-format stream-json --verbose "$@" < /dev/null > out.jsonl 2> err.txt)
  echo "$((SECONDS - t0))" > "$d/seconds.txt"
}

# prompts: the 10 prompts against the real server. The hook delivers
# reply-style; `both` also assigns it as always-on, so the server's pointer is
# active too.
prompts_run() {
  local name=$1 hook=$2 always=$3
  local d=$dir/$name
  mkdir -p "$d/proj/.claude" "$d/out"
  export QUIVER_HOME=$d/home
  "$QUIVER" init > /dev/null
  cp -R "$SPIKE"/skills/* "$SPIKE"/always/* "$QUIVER_HOME/local/"
  (cd "$d/proj" && "$QUIVER" add sql-migration-review release-notes k8s-debug > /dev/null)
  [ -z "$always" ] || (cd "$d/proj" && "$QUIVER" add reply-style --always > /dev/null)
  settings SessionStart - "$hook" > "$d/proj/.claude/settings.json"
  printf '{"mcpServers":{"quiver":{"command":"%s","args":["mcp"],"env":{"QUIVER_HOME":"%s"}}}}' "$QUIVER" "$QUIVER_HOME" > "$d/mcp.json"
  while IFS='|' read -r id prompt; do
    (cd "$d/proj" && HOOK_LOG=$d/hook.log claude -p "$prompt" --mcp-config "$d/mcp.json" --strict-mcp-config \
      --setting-sources project --allowedTools "mcp__quiver__*" --output-format stream-json --verbose \
      < /dev/null > "$d/out/$id.jsonl" 2>&1) &
  done < "$SPIKE/prompts.txt"
}
check_prompts() {
  prompts_run prompts-text "${log}cat '$RULE'" ""
  prompts_run prompts-json "${log}'$dir/bin/context.py' SessionStart '$RULE'" ""
  prompts_run prompts-both "${log}cat '$RULE'" always
  wait
}

# size: does the end of a long text still reach the model?
check_size() {
  local q='Without using any tool: quote every line in your context that begins with START-TOKEN or END-TOKEN, one per line. If there are none, write NONE.'
  local b=$dir/bodies
  for n in 5000 9900 10100 20000 50000; do
    "$dir/bin/body.py" $n $n > "$b/$n.txt"
    run size-text-$n "$(settings SessionStart - "cat '$b/$n.txt'")" "$q" "${HAIKU[@]}" &
    run size-json-$n "$(settings SessionStart - "'$dir/bin/context.py' SessionStart '$b/$n.txt'")" "$q" "${HAIKU[@]}" &
  done
  for t in a1 a2 a3; do "$dir/bin/body.py" 9000 $t > "$b/$t.txt"; done
  run size-two "$(settings SessionStart - "cat '$b/a1.txt'" "cat '$b/a2.txt'")" "$q" "${HAIKU[@]}" &
  run size-three "$(settings SessionStart - "cat '$b/a1.txt'" "cat '$b/a2.txt'" "cat '$b/a3.txt'")" "$q" "${HAIKU[@]}" &
  "$dir/bin/body.py" 7000 tr ş > "$b/tr.txt"
  run size-multibyte "$(settings SessionStart - "cat '$b/tr.txt'")" "$q" "${HAIKU[@]}" &
  # Over the limit, with tools: does the model open the saved file? The rule
  # is either in the preview or after it.
  python3 -c "import sys; f='x'*79+'\n'; r=open(sys.argv[1]).read(); open(sys.argv[2]+'/rule-end.txt','w').write(f*250+r); open(sys.argv[2]+'/rule-start.txt','w').write(r+f*250)" "$RULE" "$b"
  for i in 1 2 3; do
    run big-rule-end-$i "$(settings SessionStart - "cat '$b/rule-end.txt'")" "Suggest three names for a cat." &
    run big-rule-start-$i "$(settings SessionStart - "cat '$b/rule-start.txt'")" "Suggest three names for a cat." &
  done
  wait
}

# failures: a hook that is slow, fails or cannot start.
check_failures() {
  local q='Without using any tool: quote the line in your context that contains START-TOKEN. If there is none, write NONE.'
  local b=$dir/bodies/fail.txt
  "$dir/bin/body.py" 5000 fail > "$b"
  run fail-none '{}' "$q" "${HAIKU[@]}" &
  run fail-sleep5 "$(settings SessionStart - "sleep 5; cat '$b'")" "$q" "${HAIKU[@]}" &
  run fail-sleep20 "$(settings SessionStart - "sleep 20; cat '$b'")" "$q" "${HAIKU[@]}" &
  run fail-timeout "$(settings SessionStart - '{"type":"command","command":"cat '"'$b'"'; sleep 8","timeout":2}')" "$q" "${HAIKU[@]}" &
  run fail-exit1 "$(settings SessionStart - "cat '$b'; exit 1")" "$q" "${HAIKU[@]}" &
  run fail-exit2 "$(settings SessionStart - "cat '$b'; echo boom >&2; exit 2")" "$q" "${HAIKU[@]}" &
  run fail-missing "$(settings SessionStart - "/nonexistent/quiver hook session-start claude-code")" "$q" "${HAIKU[@]}" &
  run fail-wrong-event "$(settings SessionStart - "echo '{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"START-TOKEN-wrong\"}}'")" "$q" "${HAIKU[@]}" &
  # A binary in a folder with a space in its name.
  printf '#!/bin/sh\ncat "%s"\n' "$b" > "$dir/with space/h.sh"
  chmod +x "$dir/with space/h.sh"
  local h=$dir/with\ space/h.sh
  run path-exec "$(settings SessionStart - '{"type":"command","command":"'"$h"'","args":[]}')" "$q" "${HAIKU[@]}" &
  run path-quoted "$(settings SessionStart - "'$h'")" "$q" "${HAIKU[@]}" &
  run path-unquoted "$(settings SessionStart - "$h")" "$q" "${HAIKU[@]}" &
  wait
}

# sources: headless chains with a resume, and /compact without a new session.
chain() {
  local name=$1 set=$2 middle=$3
  local d=$dir/$name
  mkdir -p "$d/.claude"
  echo "$set" > "$d/.claude/settings.json"
  local c=(--strict-mcp-config --setting-sources project --output-format stream-json --verbose)
  cd "$d" || return
  export HOOK_LOG=$d/hook.log
  claude -p "Say OK." "${c[@]}" < /dev/null > 1.jsonl 2>&1
  local id
  id=$(python3 -c "import json;print(next(json.loads(l)['session_id'] for l in open('1.jsonl') if l.startswith('{')))")
  [ -z "$middle" ] || claude -p "$middle" --resume "$id" "${c[@]}" < /dev/null > 2.jsonl 2>&1
  claude -p "Suggest three names for a cat." --resume "$id" "${c[@]}" < /dev/null > 3.jsonl 2>&1
}
check_sources() {
  local all startup token
  all=$(settings SessionStart - "${log}cat '$RULE'")
  startup=$(settings SessionStart startup "${log}cat '$RULE'")
  token=$(settings SessionStart - "'$dir/bin/token.sh'")
  for i in 1 2 3; do
    chain chain-resume-all-$i "$all" "" &
    chain chain-resume-startup-$i "$startup" "" &
    chain chain-compact-all-$i "$all" "/compact" &
    chain chain-compact-startup-$i "$startup" "/compact" &
    chain chain-resume-tokens-$i "$token" 'Without using any tool: quote every line in your context that begins with CONTEXT-TOKEN, one per line.' &
  done
  wait
}

# subagents: does a sub-agent get the text, with and without SubagentStart?
check_subagents() {
  local p='Use the Agent tool with subagent_type general-purpose and exactly this prompt, adding nothing to it: "Suggest three names for a cat." Then print its reply verbatim.'
  local only with
  only=$(settings SessionStart - "${log}cat '$RULE'")
  with=$(python3 -c "
import json, sys
only, sub = json.loads(sys.argv[1]), sys.argv[2]
only['hooks']['SubagentStart'] = [{'hooks': [{'type': 'command', 'command': sub}]}]
print(json.dumps(only))" "$only" "${log}'$dir/bin/context.py' SubagentStart '$RULE'")
  for i in 1 2 3; do
    run sub-only-$i "$only" "$p" --allowedTools Agent &
    run sub-with-$i "$with" "$p" --allowedTools Agent &
  done
  wait
}

# interactive: a real terminal session through /clear and /compact, once per
# matcher, in a new folder so the trust dialog appears.
check_interactive() {
  for v in "all:-" "startup:startup" "scc:startup|clear|compact"; do
    local d=$dir/tui-${v%%:*}
    mkdir -p "$d/.claude"
    settings SessionStart "${v#*:}" "${log}cat '$RULE'" > "$d/.claude/settings.json"
    expect "$SPIKE/claude-hooks-tui.exp" "$d" > /dev/null 2>&1 &
  done
  wait
}

[ $# -gt 0 ] || set -- prompts size failures sources subagents interactive
for c in "$@"; do
  (check_"$c")
  echo "done $c"
done
