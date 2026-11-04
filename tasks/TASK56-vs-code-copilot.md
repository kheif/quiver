# TASK56: VS Code Copilot

**Phase:** Phase 6: Session-start delivery (v1.4)

**Status:** Done

**Depends on:** TASK52.3, TASK53, TASK55.1

Add VS Code to `quiver install` with the delivery level found in TASK52.3.

1. **MCP.** Add `quiver` to `servers` in VS Code's user `mcp.json`: `~/Library/Application Support/Code/User/mcp.json` on macOS, `%APPDATA%\Code\User\mcp.json` on Windows, `~/.config/Code/User/mcp.json` on Linux. The entry has `"type": "stdio"` and no `cwd`: with `"cwd": "${workspaceFolder}"` the server did not start (TASK52.3). Extend `JSONAssistant` with the name of the servers key (today always `mcpServers`) and the extra fields of an entry. Do not use `code --add-mcp`; it cannot remove an entry.
2. **Project.** VS Code starts the server in the home folder but sends the workspace folder as an MCP root when the server is initialized (TASK52.3). When `QUIVER_PROJECT` is unset and the working directory is in no project, `server.Resolve` (`internal/server/resolve.go`) asks the client for its roots after initialization and serves the project of the first root that is in one. Clients that start the server in the project behave as before. Update the comment there that says roots are not used.
3. **Hook.** Quiver owns the file `~/.copilot/hooks/quiver.json`, which VS Code's Local and Copilot harnesses both read (and the Copilot CLI too): a `SessionStart` and a `SubagentStart` entry in the flat form, `{"type": "command", "command": "<quiver> hook session-start vscode", "timeoutSec": 10}` and the same with `subagent-start`, `<quiver>` being the binary itself (`appCommand()`), quoted with `assistant.ShellQuote` since the command runs with `/bin/sh`. `uninstall` removes the file. It is written whole, so no merging is needed.
4. **Installed** means its user folder exists. VS Code Insiders keeps its files under `Code - Insiders` in place of `Code` (VS Code's documented user folder; not measured).
5. The MCP command is the binary itself (`appCommand()`), as for the other desktop apps.

**Done when:** install, update, uninstall and doctor are tested for VS Code; a Copilot agent session in VS Code follows an always-on fixture skill from the hook, in the Local and in the Copilot harness; and a skill assigned to the open workspace's project is served.
