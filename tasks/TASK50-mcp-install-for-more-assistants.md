# TASK50 — `mcp install` for more assistants

**Phase:** Phase 5 — v2

**Status:** Todo

**Depends on:** TASK37

TASK37 added Cursor and Claude Desktop, the shared JSON installer and `--for`.

1. Find out how Codex, Gemini CLI, VS Code and others register stdio MCP servers (config file path and format, or a CLI command). Codex uses TOML.
2. Run the TASK37 spike for each: does it pass the MCP `instructions` to the model?
3. Add each one that does to `mcp install --for <name>`.
4. Confirm that each assistant passes the project folder (roots or working directory) correctly. Record any assistant that needs `QUIVER_PROJECT`.

**Done when:** Install and uninstall work for each supported assistant, and each one sees the correct project's skills.
