# TASK56: VS Code Copilot

**Phase:** Phase 6: Session-start delivery (v1.4)

**Status:** Todo

**Depends on:** TASK52.3, TASK55.1

Add VS Code to `quiver install` with the delivery level found in TASK52.3.

1. **MCP.** Add `quiver` to `servers` in VS Code's user `mcp.json`: `~/Library/Application Support/Code/User/mcp.json` on macOS, `%APPDATA%\Code\User\mcp.json` on Windows, `~/.config/Code/User/mcp.json` on Linux. The entry has `"type": "stdio"` and, if TASK52.3 found it needed, `"cwd": "${workspaceFolder}"`. Extend `JSONAssistant` with the name of the servers key (today always `mcpServers`) and the extra fields of an entry. Use `code --add-mcp` only if TASK52.3 shows it is better; it cannot remove an entry.
2. **Delivery.** Add the hook or the rules-file pointer if TASK52.3 found one; otherwise VS Code is at the instructions or on-request level.
3. **Installed** means its user folder exists. VS Code Insiders is included if TASK52.3 found its paths.
4. The command is the binary itself (`appCommand()`), as for the other desktop apps.

**Done when:** install, update, uninstall and doctor are tested for VS Code, and a Copilot agent session uses the fixture skills at the measured level.
