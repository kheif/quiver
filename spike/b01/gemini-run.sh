#!/bin/bash
# usage: gemini-run.sh <dir> [<model>]
# Runs every prompt of prompts.txt in a headless Gemini CLI session
# (gemini -p) against the real `quiver mcp`, with the fixture home of
# gui-setup.sh. Writes <dir>/ws, the workspace with a project-level
# .gemini/settings.json that trusts only Quiver, and one <dir>/out/<id>.jsonl
# per prompt. Needs $QUIVER and a Gemini CLI login.
set -e
SPIKE=$(cd "$(dirname "$0")" && pwd)
dir=$1
model=${2:+--model $2}
if [ ! -d "$dir/home" ]; then
  mkdir -p "$dir/ws/.gemini"
  "$SPIKE/gui-setup.sh" "$dir" | python3 -c '
import json, sys
c = json.load(sys.stdin)
c["mcpServers"]["quiver"]["trust"] = True
json.dump(c, sys.stdout, indent=2)' > "$dir/ws/.gemini/settings.json"
fi
mkdir -p "$dir/out"
cd "$dir/ws"
while IFS='|' read -r id prompt; do
  # shellcheck disable=SC2086
  gemini -p "$prompt" $model --output-format stream-json \
    --allowed-mcp-server-names quiver < /dev/null > "$dir/out/$id.jsonl" 2>&1 || true
  echo "done $id"
done < "$SPIKE/prompts.txt"
