# TASK52.3: VS Code Copilot

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK52](TASK52-spike-session-start-hooks-per-assistant.md): Spike: session-start hooks per assistant

**Status:** ✅ Done

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

**Notes:**
- Results in docs/spike-hooks.md (VS Code 1.143.1, Copilot Chat 0.71.0, `spike/b01/vscode-hooks-run.sh`). Delivery level: hook. A second VS Code with its own profile runs the prompts through a small driver extension, since `code chat` ignores `--user-data-dir`.
- Server only: always-on 4 of 10, fitting skills 6 of 6, other skills 0 of 4. With a `SessionStart` hook: 10 of 10, 6 of 6, 0 of 4, and no `get_skill` for `reply-style`; with the pointer also active, the model loaded it again in 6 of 10.
- VS Code has two harnesses: Local (the default) takes only `hookSpecificOutput.additionalContext`, the Copilot harness only a top-level `additionalContext`; plain stdout reaches neither. A command that prints both works in both, once. Both read `~/.copilot/hooks/*.json`; the Copilot CLI reads it too.
- No size limit up to 200,000 characters (Local). `SubagentStart` reaches sub-agents, `SessionStart` does not. A failing hook never stops a session. The command runs with `/bin/sh`; the default timeout is 30 seconds.
- `.claude/settings.json` files count only with `chat.useClaudeHooks` (off by default).
- From the user's `mcp.json` the server starts in the home folder; with `"cwd": "${workspaceFolder}"` it did not start. VS Code sends the workspace folder as an MCP root.
- A user instructions file with a pointer raised always-on to 8 of 10.
- Not measured: trust (workspace trust was off), managed hooks, compaction, Windows, Linux, Insiders.
