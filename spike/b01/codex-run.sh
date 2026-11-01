#!/bin/bash
# usage: codex-run.sh <dir> [<codex option>...]
# Runs every prompt of prompts.txt in a headless Codex session (codex exec)
# against the real `quiver mcp`, with the fixture home of gui-setup.sh. The
# server is passed with -c, so ~/.codex/config.toml is not changed; its tools
# are approved, since codex exec cannot ask. Writes <dir>/ws, the workspace,
# and one <dir>/out/<id>.jsonl per prompt. Extra options go to codex exec.
# Needs $QUIVER and a codex login.
set -e
SPIKE=$(cd "$(dirname "$0")" && pwd)
dir=$1
shift
if [ ! -d "$dir/home" ]; then
  mkdir -p "$dir/ws"
  "$SPIKE/gui-setup.sh" "$dir" > /dev/null
fi
mkdir -p "$dir/out"
cd "$dir/ws"
while IFS='|' read -r id prompt; do
  codex exec --json --ephemeral --skip-git-repo-check -s read-only \
    -c "mcp_servers.quiver.command=\"$QUIVER\"" \
    -c 'mcp_servers.quiver.args=["mcp"]' \
    -c "mcp_servers.quiver.env={QUIVER_HOME=\"$dir/home\"}" \
    -c 'mcp_servers.quiver.default_tools_approval_mode="approve"' \
    "$@" "$prompt" < /dev/null > "$dir/out/$id.jsonl" 2>&1 || true
  echo "done $id"
done < "$SPIKE/prompts.txt"
