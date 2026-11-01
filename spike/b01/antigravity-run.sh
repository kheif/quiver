#!/bin/bash
# usage: antigravity-run.sh <dir>
# Runs every prompt of prompts.txt in a headless Antigravity CLI session
# (agy -p) against the real `quiver mcp`, with the fixture home of
# gui-setup.sh. agy reads MCP servers only from ~/.gemini/config/mcp_config.json,
# so this adds a quiver entry there and removes it at the end; the tool
# permission is given for the session only. Writes <dir>/ws, the workspace,
# and one <dir>/out/<id>.jsonl per prompt. Needs $QUIVER and an agy login.
set -e
SPIKE=$(cd "$(dirname "$0")" && pwd)
dir=$1
if [ ! -d "$dir/home" ]; then
  mkdir -p "$dir/ws"
  "$SPIKE/gui-setup.sh" "$dir" > /dev/null
fi
agy mcp add --env "QUIVER_HOME=$dir/home" quiver "$QUIVER" mcp > /dev/null
trap 'agy mcp remove quiver > /dev/null' EXIT
mkdir -p "$dir/out"
cd "$dir/ws"
while IFS='|' read -r id prompt; do
  agy -p "$prompt" --output-format stream-json \
    --settings '{"permissions":{"allow":["mcp(quiver/*)"]}}' \
    < /dev/null > "$dir/out/$id.jsonl" 2>&1 || true
  echo "done $id"
done < "$SPIKE/prompts.txt"
