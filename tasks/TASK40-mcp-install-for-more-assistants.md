# TASK40 — `mcp install` for more assistants

**Phase:** Phase 5 — v2

**Status:** Todo

**Depends on:** TASK20

1. Find out how Cursor, Claude Desktop, Codex and Gemini CLI register stdio MCP servers (config file path and format, or a CLI command).
2. Add each one to `mcp install --target <name>`. Without `--target`, detect the installed assistants and ask which ones to set up.
3. Edit JSON config files safely: keep unknown keys, write atomically, and back up the original first.
4. Confirm that each assistant passes the project folder (roots or working directory) correctly. Record any assistant that needs `QUIVER_PROJECT`.

**Done when:** Install and uninstall work for each supported assistant, and each one sees the correct project's skills.
