#!/bin/bash
# usage: cursor-run.sh <dir> [<model>]
# Runs every prompt of prompts.txt in a headless Cursor CLI session
# (cursor-agent -p) against the real `quiver mcp`, with the fixture home of
# gui-setup.sh. Writes <dir>/ws, the workspace with a project-level
# .cursor/mcp.json and a cli.json that allows only Quiver's tools, and one
# <dir>/out/<id>.jsonl per prompt. Needs $QUIVER and a cursor-agent login.
set -e
SPIKE=$(cd "$(dirname "$0")" && pwd)
dir=$1
model=${2:-auto}
if [ ! -d "$dir/home" ]; then
  mkdir -p "$dir/ws/.cursor"
  "$SPIKE/gui-setup.sh" "$dir" > "$dir/ws/.cursor/mcp.json"
  echo '{"permissions":{"allow":["Mcp(quiver:*)"],"deny":[]}}' > "$dir/ws/.cursor/cli.json"
fi
mkdir -p "$dir/out"
cd "$dir/ws"
while IFS='|' read -r id prompt; do
  cursor-agent -p "$prompt" --model "$model" --output-format stream-json \
    --approve-mcps --trust < /dev/null > "$dir/out/$id.jsonl" 2>&1 || true
  echo "done $id"
done < "$SPIKE/prompts.txt"
