#!/bin/bash
# usage: codex-hooks-run.sh <dir> [trust|prompts|size|failures|sources|subagents|interactive|project]...
# Measures SessionStart hooks in Codex. <dir> is an absolute path, without
# spaces, to a new folder. CODEX_HOME must name a throwaway Codex home with its
# own `codex login`: the script writes hooks.json and hook trust there and never
# touches ~/.codex. Every hook command is <dir>/bin/slot.sh <slot>, which runs
# the session folder's hooks/<slot> if there is one, so one set of hooks,
# trusted once, serves every run. The trust check trusts them in the TUI, so
# run it first. Without a check name it runs all of them. Summarize with
# codex-hooks-analyze.py <dir>. Needs $QUIVER, and pyte in $PYTHON (default
# python3) for the TUI checks. See docs/spike-hooks.md.
set -u
SPIKE=$(cd "$(dirname "$0")" && pwd)
dir=$1
shift
case ${CODEX_HOME:-} in
  "" | "$HOME/.codex" | "$HOME/.codex/")
    echo "set CODEX_HOME to a throwaway Codex home" >&2
    exit 1
    ;;
esac
RULE=$SPIKE/always/reply-style/SKILL.md
PY=${PYTHON:-python3}
SANDBOX=(-c 'sandbox_mode="read-only"')
EXEC=(codex exec --json --skip-git-repo-check "${SANDBOX[@]}")
LOW=(-c 'model_reasoning_effort="low"')
CAT='Suggest three names for a cat.'
mkdir -p "$dir/bin" "$dir/bodies" "$dir/with space"

# slot.sh <slot>: runs the session folder's hooks/<slot>; the hook's cwd is the
# session's cwd.
cat > "$dir/bin/slot.sh" <<'EOF'
#!/bin/sh
if [ -x "./hooks/$1" ]; then exec "./hooks/$1"; fi
cat > /dev/null
EOF
cp "$dir/bin/slot.sh" "$dir/with space/slot.sh"
# body.py <chars> <tag> [fill]: exactly <chars> characters between and
# including a START-TOKEN-<tag> line and an END-TOKEN-<tag> line, with a
# MID-TOKEN-<tag> line in the middle.
cat > "$dir/bin/body.py" <<'EOF'
#!/usr/bin/env python3
import sys
n, tag = int(sys.argv[1]), sys.argv[2]
fill = sys.argv[3] if len(sys.argv) > 3 else "x"
head, mid, tail = f"START-TOKEN-{tag}\n", f"MID-TOKEN-{tag}\n", f"END-TOKEN-{tag}\n"
rest, lines = n - len(head) - len(mid) - len(tail), []
while rest > 0:
    k = min(79, rest - 1)
    lines.append(fill * k + "\n")
    rest -= k + 1
lines.insert(len(lines) // 2, mid)
sys.stdout.write(head + "".join(lines) + tail)
EOF
# context.py <event> <file>: prints a file as additionalContext.
cat > "$dir/bin/context.py" <<'EOF'
#!/usr/bin/env python3
import json, sys
print(json.dumps({"hookSpecificOutput": {"hookEventName": sys.argv[1], "additionalContext": open(sys.argv[2]).read()}}))
EOF
chmod +x "$dir"/bin/* "$dir/with space/slot.sh"
# A slot script that prints a token naming the source of this firing.
TOKEN="echo \"CONTEXT-TOKEN-\$(echo \"\$s\" | python3 -c 'import json,sys;print(json.load(sys.stdin)[\"source\"])')\""

# hooks_json [moved|extra]: writes $CODEX_HOME/hooks.json. `moved` points the
# first hook at a copy of slot.sh in bin2; `extra` adds hooks whose command a
# slot cannot vary.
hooks_json() {
  python3 - "$dir" "${1:-}" > "$CODEX_HOME/hooks.json" <<'EOF'
import json, sys
d, variant = sys.argv[1], sys.argv[2]
c = lambda slot, bin="bin", **kw: {"type": "command", "command": f"{d}/{bin}/slot.sh {slot}", **kw}
start = [
    {"hooks": [c("ss", "bin2" if variant == "moved" else "bin"), c("ss2"), c("ss3")]},
    {"hooks": [c("ss-t2", timeout=2)]},
    {"hooks": [c("ss-l0", additionalContextLimit=0)]},
    {"hooks": [c("ss-l5k", additionalContextLimit=5000)]},
    {"matcher": "startup", "hooks": [c("ss-startup")]},
    {"matcher": "startup|clear|compact", "hooks": [c("ss-scc")]},
]
if variant == "extra":
    start.append({"hooks": [
        {"type": "command", "command": f"'{d}/with space/slot.sh' ss-q"},
        {"type": "command", "command": f"{d}/with space/slot.sh ss-u"},
        {"type": "command", "command": "/nonexistent/quiver hook session-start codex"},
    ]})
print(json.dumps({"hooks": {"SessionStart": start, "SubagentStart": [{"hooks": [c("sub")]}]}}, indent=2))
EOF
}
# trust_computed: trusts the current hooks.json with codex-hook-trust.py.
trust_computed() {
  "$SPIKE/codex-hook-trust.py" "$CODEX_HOME/hooks.json" "$CODEX_HOME/config.toml" > /dev/null
}

# slot <session dir> <slot> <script>: the script the slot runs in that folder.
# It has the hook's stdin in $s and logs it to hook.log first.
slot() {
  mkdir -p "$1/hooks"
  printf '#!/bin/sh\ns=$(cat); echo "$s" >> ./hook.log\n%s\n' "$3" > "$1/hooks/$2"
  chmod +x "$1/hooks/$2"
}

# run <name> <prompt> [codex option]...: one headless session in <dir>/<name>.
run() {
  local name=$1 prompt=$2
  shift 2
  local d=$dir/$name
  mkdir -p "$d"
  local t0=$SECONDS
  (cd "$d" && "${EXEC[@]}" "$@" "$prompt" < /dev/null > out.jsonl 2> err.txt)
  echo "$((SECONDS - t0))" > "$d/seconds.txt"
}
# throttle: waits while six sessions run.
throttle() {
  while [ "$(jobs -rp | wc -l)" -ge 6 ]; do sleep 1; done
}

# trust: the review dialog, a hash computed outside Codex, and a moved binary.
check_trust() {
  hooks_json
  local d=$dir/trust
  slot "$d" ss "cat '$RULE'"
  "$PY" "$SPIKE/codex-hooks-tui.py" trust "$d"
  "$SPIKE/codex-hook-trust.py" "$CODEX_HOME/hooks.json" | while read -r key hash; do
    if grep -A1 -F "\"$key\"" "$CODEX_HOME/config.toml" | grep -qF "$hash"; then
      echo "same $key"
    else
      echo "differs $key"
    fi
  done > "$d/hashes.txt"
  mkdir -p "$dir/bin2"
  cp "$dir/bin/slot.sh" "$dir/bin2/"
  hooks_json moved
  slot "$dir/moved-exec" ss "cat '$RULE'"
  run moved-exec "$CAT"
  slot "$dir/moved-tui" ss "cat '$RULE'"
  "$PY" "$SPIKE/codex-hooks-tui.py" skip "$dir/moved-tui"
  trust_computed
  slot "$dir/moved-computed" ss "cat '$RULE'"
  run moved-computed "$CAT"
  # Moving it back changes the command again; trust it through the app server.
  hooks_json
  "$SPIKE/codex-hook-trust.py" --app-server "$dir" > "$d/app-server.txt"
  slot "$dir/moved-app-server" ss "cat '$RULE'"
  run moved-app-server "$CAT"
}

# prompts: the 10 prompts against the real server. The hook delivers
# reply-style; `both` also assigns it as always-on, so the server's pointer is
# active too.
prompts_run() {
  local name=$1 hook=$2 always=$3
  local d=$dir/$name
  mkdir -p "$d/out"
  export QUIVER_HOME=$d/home
  "$QUIVER" init > /dev/null
  cp -R "$SPIKE"/skills/* "$SPIKE"/always/* "$QUIVER_HOME/local/"
  "$QUIVER" add sql-migration-review release-notes k8s-debug --global > /dev/null
  [ -z "$always" ] || "$QUIVER" add reply-style --global --always > /dev/null
  slot "$d" ss "$hook"
  while IFS='|' read -r id prompt; do
    (cd "$d" && "${EXEC[@]}" \
      -c "mcp_servers.quiver.command=\"$QUIVER\"" \
      -c 'mcp_servers.quiver.args=["mcp"]' \
      -c "mcp_servers.quiver.env={QUIVER_HOME=\"$QUIVER_HOME\"}" \
      -c 'mcp_servers.quiver.default_tools_approval_mode="approve"' \
      "$prompt" < /dev/null > "$d/out/$id.jsonl" 2>&1)
  done < "$SPIKE/prompts.txt"
}
check_prompts() {
  prompts_run prompts-text "cat '$RULE'" "" &
  prompts_run prompts-json "'$dir/bin/context.py' SessionStart '$RULE'" "" &
  prompts_run prompts-both "cat '$RULE'" always &
  wait
}

# size: which parts of a long text reach the model?
check_size() {
  local q='Without using any tool: quote every line in your context that begins with START-TOKEN, MID-TOKEN or END-TOKEN, one per line. If there are none, write NONE.'
  local b=$dir/bodies n t w i
  for n in 5000 9900 10100 20000 50000; do
    "$dir/bin/body.py" $n $n > "$b/$n.txt"
    slot "$dir/size-text-$n" ss "cat '$b/$n.txt'"
    throttle
    run size-text-$n "$q" "${LOW[@]}" &
    slot "$dir/size-json-$n" ss "'$dir/bin/context.py' SessionStart '$b/$n.txt'"
    throttle
    run size-json-$n "$q" "${LOW[@]}" &
  done
  # additionalContextLimit 0 turns the limit off; 5000 raises it.
  slot "$dir/size-l0-50000" ss-l0 "cat '$b/50000.txt'"
  slot "$dir/size-l5k-20000" ss-l5k "cat '$b/20000.txt'"
  slot "$dir/size-l5k-50000" ss-l5k "cat '$b/50000.txt'"
  for t in size-l0-50000 size-l5k-20000 size-l5k-50000; do
    throttle
    run $t "$q" "${LOW[@]}" &
  done
  for t in a1 a2 a3; do "$dir/bin/body.py" 9000 $t > "$b/$t.txt"; done
  slot "$dir/size-three" ss "cat '$b/a1.txt'"
  slot "$dir/size-three" ss2 "cat '$b/a2.txt'"
  slot "$dir/size-three" ss3 "cat '$b/a3.txt'"
  throttle
  run size-three "$q" "${LOW[@]}" &
  # 4,900 and 7,000 characters of ş: 9,800 and 14,000 bytes.
  for n in 4900 7000; do
    "$dir/bin/body.py" $n tr$n ş > "$b/tr$n.txt"
    slot "$dir/size-multibyte-$n" ss "cat '$b/tr$n.txt'"
    throttle
    run size-multibyte-$n "$q" "${LOW[@]}" &
  done
  # Over the limit, with tools: the rule at the start, middle or end of 20 KB.
  python3 -c "
import sys
f = 'x' * 79 + '\n'
r = open(sys.argv[1]).read()
for name, text in (('start', r + f * 250), ('mid', f * 125 + r + f * 125), ('end', f * 250 + r)):
    open(f'{sys.argv[2]}/rule-{name}.txt', 'w').write(text)" "$RULE" "$b"
  for i in 1 2 3; do
    for w in start mid end; do
      slot "$dir/big-rule-$w-$i" ss "cat '$b/rule-$w.txt'"
      throttle
      run big-rule-$w-$i "$CAT" &
    done
  done
  wait
}

# failures: a hook that is slow, fails or cannot start.
check_failures() {
  local q='Without using any tool: quote every line in your context that contains START-TOKEN. If there is none, write NONE.'
  local b=$dir/bodies/fail.txt
  "$dir/bin/body.py" 5000 fail > "$b"
  mkdir -p "$dir/fail-none"
  slot "$dir/fail-fast" ss "cat '$b'"
  slot "$dir/fail-sleep5" ss "sleep 5; cat '$b'"
  slot "$dir/fail-sleep20" ss "sleep 20; cat '$b'"
  slot "$dir/fail-timeout" ss-t2 "cat '$b'; sleep 8"
  slot "$dir/fail-exit1" ss "cat '$b'; exit 1"
  slot "$dir/fail-exit2" ss "cat '$b'; echo boom >&2; exit 2"
  slot "$dir/fail-wrong-event" ss "echo '{\"hookSpecificOutput\":{\"hookEventName\":\"UserPromptSubmit\",\"additionalContext\":\"START-TOKEN-wrong\"}}'"
  slot "$dir/fail-bad-json" ss "echo '{START-TOKEN-badjson'"
  local t
  for t in none fast sleep5 sleep20 timeout exit1 exit2 wrong-event bad-json; do
    throttle
    run fail-$t "$q" "${LOW[@]}" &
  done
  wait
  # A binary path with a space, quoted and not, and a binary that does not
  # exist; trusted with the computed hash. The TUI run shows what the terminal
  # prints for them.
  hooks_json extra
  trust_computed
  for t in path path-tui; do
    slot "$dir/$t" ss-q "echo START-TOKEN-quoted"
    slot "$dir/$t" ss-u "echo START-TOKEN-unquoted"
  done
  run path "$q" "${LOW[@]}"
  "$PY" "$SPIKE/codex-hooks-tui.py" show "$dir/path-tui"
  hooks_json
}

# sources: a headless session, then a resume or fork of it.
chain() {
  local name=$1 second=$2 s=$3 script=$4 prompt=$5
  local d=$dir/$name
  slot "$d" "$s" "$script"
  cd "$d" || return
  "${EXEC[@]}" "Say OK." < /dev/null > 1.jsonl 2> 1.err
  local id
  id=$(python3 -c "import json;print(next(json.loads(l)['thread_id'] for l in open('1.jsonl') if 'thread.started' in l))")
  codex exec "$second" --json --skip-git-repo-check "${SANDBOX[@]}" "$id" "$prompt" < /dev/null > 2.jsonl 2> 2.err
}
check_sources() {
  local q='Without using any tool: quote every line in your context that begins with CONTEXT-TOKEN, one per line.'
  local i
  for i in 1 2 3; do
    chain chain-resume-all-$i resume ss "cat '$RULE'" "$CAT" &
    chain chain-resume-startup-$i resume ss-startup "cat '$RULE'" "$CAT" &
    chain chain-resume-tokens-$i resume ss "$TOKEN" "$q" &
    chain chain-fork-startup-$i fork ss-startup "cat '$RULE'" "$CAT" &
    chain chain-fork-tokens-$i fork ss "$TOKEN" "$q" &
    wait
  done
}

# subagents: does a sub-agent get the text, with and without SubagentStart,
# and with and without its parent's history?
check_subagents() {
  local ask='to start one sub-agent with exactly this message, adding nothing to it: "Suggest three names for a cat." Wait for it to finish, then print its reply verbatim.'
  local all="Use the spawn_agent tool $ask" none="Use the spawn_agent tool with fork_turns \"none\" $ask" i
  for i in 1 2 3; do
    slot "$dir/sub-all-$i" ss "cat '$RULE'"
    slot "$dir/sub-none-$i" ss "cat '$RULE'"
    slot "$dir/sub-none-json-$i" ss "cat '$RULE'"
    slot "$dir/sub-none-json-$i" sub "'$dir/bin/context.py' SubagentStart '$RULE'"
    slot "$dir/sub-none-text-$i" ss "cat '$RULE'"
    slot "$dir/sub-none-text-$i" sub "cat '$RULE'"
    run sub-all-$i "$all" &
    run sub-none-$i "$none" &
    run sub-none-json-$i "$none" &
    run sub-none-text-$i "$none" &
    wait
  done
}

# interactive: a TUI session through /clear and /compact, once per matcher,
# and one through /compact alone.
check_interactive() {
  local v d
  for v in all:ss startup:ss-startup scc:ss-scc; do
    d=$dir/tui-${v%%:*}
    slot "$d" "${v#*:}" "cat '$RULE'"
    "$PY" "$SPIKE/codex-hooks-tui.py" session "$d" &
  done
  for v in all:ss startup:ss-startup; do
    d=$dir/tui-compact-${v%%:*}
    slot "$d" "${v#*:}" "cat '$RULE'"
    "$PY" "$SPIKE/codex-hooks-tui.py" compact "$d" &
  done
  wait
}

# project: a .codex/hooks.json in the session folder, with a slot no user hook
# runs. project-exec is never trusted; project-tui is trusted in the TUI, then
# run headless.
check_project() {
  local d
  for d in "$dir/project-exec" "$dir/project-tui"; do
    mkdir -p "$d/.codex"
    printf '{"hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"%s/bin/slot.sh project"}]}]}}\n' "$dir" > "$d/.codex/hooks.json"
    slot "$d" project "cat '$RULE'"
  done
  run project-exec "$CAT"
  "$PY" "$SPIKE/codex-hooks-tui.py" folder "$dir/project-tui"
  run project-tui "$CAT"
}

[ $# -gt 0 ] || set -- trust prompts size failures sources subagents interactive project
hooks_json
for c in "$@"; do
  (check_"$c")
  echo "done $c"
done
