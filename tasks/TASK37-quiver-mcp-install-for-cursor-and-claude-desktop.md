# TASK37 — `quiver mcp install` for Cursor and Claude Desktop

**Phase:** Phase 4c — v1.2

**Status:** Todo

**Depends on:** TASK33

Today `mcp install` registers Quiver only with Claude Code (`internal/assistant/claude.go`). Pulls the first part of the v2 task "`mcp install` for more assistants" into v1.2.

1. **Spike first, per assistant.** The catalog and always-on skills reach the assistant through the MCP `instructions`. Check that Cursor and Claude Desktop pass `instructions` to the model, as in docs/spike-b01-results.md: does a fitting prompt load the skill, does an always-on skill apply from the first reply? Record the results in `docs/`. If an assistant ignores `instructions`, decide with the user before adding it.
2. **Shared JSON installer.** Most assistants keep MCP servers in a JSON file under `mcpServers`. One function adds, updates and removes the `quiver` entry, keeps every other key and the file's indentation, backs up the original, and writes atomically. A broken JSON file is an error, never overwritten.
   - Cursor: `~/.cursor/mcp.json`.
   - Claude Desktop: `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS, `%APPDATA%\Claude\claude_desktop_config.json` on Windows.
3. **CLI.** `quiver mcp install [--for claude-code|cursor|claude-desktop]`. Without `--for`, install for every assistant that is installed, and list what was done. `mcp uninstall` the same. `quiver mcp config` prints the JSON snippet for any other assistant.
4. The command is the absolute path of the binary (backlog B09), or the npm shim as today. Claude Desktop does not run in a project folder, so it gets only global skills: say so after installing.
5. `doctor` checks each installed assistant's entry.

**Done when:** tests cover install, update, uninstall, other keys kept and a broken file for each JSON assistant; the spike results are written down; the README lists the assistants.
