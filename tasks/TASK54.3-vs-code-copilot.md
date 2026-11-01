# TASK54.3: VS Code Copilot

**Phase:** Phase 6: Session-start delivery

**Part of:** [TASK54](TASK54-spike-session-start-hooks-per-assistant.md): Spike: session-start hooks per assistant

**Status:** Todo

**Depends on:** TASK50

Facts seen on 2026-11-01 with VS Code 1.143:
- MCP servers are read from the user's `mcp.json` (`~/Library/Application Support/Code/User/mcp.json` on macOS) and a workspace's `.vscode/mcp.json`. The servers sit under `servers`, not `mcpServers`, and each has `"type": "stdio"`.
- `code --add-mcp '<json>'` adds a server to the user profile; there is no remove command.
- `code chat -m agent "<prompt>"` opens a chat with a prompt. VS Code has no headless mode, so the answers are read from the chat window or from the chat session files in the workspace storage.
- TASK50 did not measure Copilot.

Steps, with the fixture server in a throwaway workspace's `.vscode/mcp.json`:
1. Run the 10 prompts in Copilot Chat's agent mode, one new chat each. Does the model get `instructions` (always-on count), does a fitting prompt load the skill?
2. With the server in the user's `mcp.json` instead: which folder does it start in? If not the workspace folder, does `"cwd": "${workspaceFolder}"` fix it?
3. Does Copilot in VS Code have a session-start hook whose output reaches the model?
4. Does VS Code load a user-level instructions file in every workspace (for example an `.instructions.md` file in the user's prompts folder with `applyTo: "**"`)? Does a pointer there make the model load Quiver?
5. Where is the `mcp.json` on Windows and Linux, and for VS Code Insiders?

**Done when:** the results are in `docs/spike-hooks.md`.
