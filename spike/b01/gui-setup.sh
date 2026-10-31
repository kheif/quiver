#!/bin/bash
# usage: gui-setup.sh <dir>
# Prepares the TASK37 spike for assistants with a GUI (Cursor, Claude
# Desktop): a fresh Quiver home in <dir>/home with the B01 fixture skills,
# all global, so they are served wherever the assistant starts the server.
# Prints the mcpServers entry to paste into the assistant's MCP config. Needs
# $QUIVER. See docs/spike-mcp-clients.md.
set -e
SPIKE=$(cd "$(dirname "$0")" && pwd)
export QUIVER_HOME=$1/home
if [ -e "$QUIVER_HOME" ]; then
  echo "$QUIVER_HOME exists; use a new folder" >&2
  exit 1
fi
"$QUIVER" init > /dev/null
cp -R "$SPIKE"/skills/* "$SPIKE"/always/* "$QUIVER_HOME/local/"
"$QUIVER" add sql-migration-review release-notes k8s-debug --global > /dev/null
"$QUIVER" add reply-style --global --always > /dev/null
cat <<J
{
  "mcpServers": {
    "quiver": {
      "command": "$QUIVER",
      "args": ["mcp"],
      "env": { "QUIVER_HOME": "$QUIVER_HOME" }
    }
  }
}
J
