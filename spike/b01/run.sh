#!/bin/bash
# usage: run.sh <variant> <id> <prompt>   (variant: desc | catalog | notoolsearch)
W=$OUT; v=$1; id=$2; p=$3
d=$W/$v/$id; mkdir -p $d/proj
env=""
case $v in
  catalog) extra=',"QV_CATALOG":"1"';;
  *) extra='';;
esac
cat > $d/mcp.json <<J
{"mcpServers":{"quiver-spike":{"command":"$SPIKE/quiver-spike","env":{"QV_LOG":"$d/log.jsonl","QV_ALWAYS":"1"$extra}}}}
J
cd $d/proj
if [ $v = notoolsearch ]; then export ENABLE_TOOL_SEARCH=false; fi
claude -p "$p" --mcp-config $d/mcp.json --strict-mcp-config --setting-sources project --allowedTools "mcp__quiver-spike__*" --output-format stream-json --verbose < /dev/null > $d/out.jsonl 2>&1
echo "done $v $id"
