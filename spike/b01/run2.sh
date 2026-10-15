#!/bin/bash
# usage: run2.sh <id> <pad> <prompt>
d=$OUT/$1; mkdir -p $d/proj
cat > $d/mcp.json <<J
{"mcpServers":{"quiver-spike":{"command":"$SPIKE/quiver-spike","env":{"QV_LOG":"$d/log.jsonl","QV_ALWAYS":"${QV_ALWAYS_MODE:-1}","QV_CATALOG":"1","QV_PAD":"$2"}}}}
J
cd $d/proj
claude -p "$3" --mcp-config $d/mcp.json --strict-mcp-config --setting-sources project --allowedTools "mcp__quiver-spike__*" --output-format stream-json --verbose < /dev/null > $d/out.jsonl 2>&1
echo "done $1"
