#!/bin/bash
# usage: codex-rules-run.sh <dir> [global|both]...
# Measures Codex's global rules file, $CODEX_HOME/AGENTS.md. CODEX_HOME must
# name a throwaway Codex home with its own `codex login`; the script writes
# its AGENTS.md and removes it at the end, and never touches ~/.codex.
# global: the 10 prompts with a pointer to Quiver in the global file only.
# both: a rule in the global file and another in the workspace's AGENTS.md.
# Summarize global with codex-analyze.py <dir>/global/out. Needs $QUIVER.
# See docs/spike-hooks.md.
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
trap 'rm -f "$CODEX_HOME/AGENTS.md"' EXIT

# run <name> <id> <prompt>: one codex exec session in <dir>/<name>/ws with
# the fixture server.
run() {
  local w=$dir/$1
  mkdir -p "$w/out"
  (cd "$w/ws" && codex exec --json --skip-git-repo-check -s read-only \
    -c "mcp_servers.quiver.command=\"$QUIVER\"" \
    -c 'mcp_servers.quiver.args=["mcp"]' \
    -c "mcp_servers.quiver.env={QUIVER_HOME=\"$dir/home\"}" \
    -c 'mcp_servers.quiver.default_tools_approval_mode="approve"' \
    "$3" < /dev/null > "$w/out/$2.jsonl" 2>&1)
}

[ -d "$dir/home" ] || { mkdir -p "$dir"; "$SPIKE/gui-setup.sh" "$dir" > /dev/null; }

check_global() {
  local id prompt
  mkdir -p "$dir/global/ws"
  echo 'Before your first reply, load the tools of the quiver MCP server and follow its instructions.' > "$CODEX_HOME/AGENTS.md"
  while IFS='|' read -r id prompt; do run global "$id" "$prompt"; done < "$SPIKE/prompts.txt"
  rm "$CODEX_HOME/AGENTS.md"
}

check_both() {
  mkdir -p "$dir/both/ws"
  echo 'Start every reply with the line [global-rule].' > "$CODEX_HOME/AGENTS.md"
  echo 'End every reply with the line [project-rule].' > "$dir/both/ws/AGENTS.md"
  run both cat-1 'Suggest three names for a cat.'
  run both cat-2 'Suggest three names for a cat.'
  rm "$CODEX_HOME/AGENTS.md"
}

[ $# -gt 0 ] || set -- global both
for c in "$@"; do
  (check_"$c")
  echo "done $c"
done
