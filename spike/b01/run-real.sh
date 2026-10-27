#!/bin/bash
# usage: run-real.sh <id> <prompt>
# Runs one headless Claude Code session against the real `quiver mcp`, with
# the B01 fixture skills in a fresh Quiver home. Needs $OUT and $QUIVER.
# Set RESUME=<session id> to continue a session instead of starting one.
set -e
SPIKE=$(cd "$(dirname "$0")" && pwd)
d=$OUT/real/$1
export QUIVER_HOME=$d/home
if [ ! -d "$QUIVER_HOME" ]; then
  mkdir -p "$d/proj"
  "$QUIVER" init > /dev/null
  cp -R "$SPIKE"/skills/* "$SPIKE"/always/* "$QUIVER_HOME/local/"
  (cd "$d/proj" && "$QUIVER" add sql-migration-review release-notes k8s-debug > /dev/null \
    && "$QUIVER" add reply-style --always > /dev/null)
fi
cat > "$d/mcp.json" <<J
{"mcpServers":{"quiver":{"command":"$QUIVER","args":["mcp"],"env":{"QUIVER_HOME":"$QUIVER_HOME"}}}}
J
cd "$d/proj"
n=$(ls "$d"/out*.jsonl 2>/dev/null | wc -l | tr -d ' ')
claude -p "$2" ${RESUME:+--resume "$RESUME"} --mcp-config "$d/mcp.json" --strict-mcp-config \
  --setting-sources project --allowedTools "mcp__quiver__*" \
  --output-format stream-json --verbose < /dev/null > "$d/out$n.jsonl" 2>&1 || true
echo "done $1 ($n)"
