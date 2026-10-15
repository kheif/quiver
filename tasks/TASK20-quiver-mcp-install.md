# TASK20 — `quiver mcp install`

**Phase:** Phase 2 — MCP Server (first usable version)

**Status:** Todo

**Depends on:** TASK16

Registers the server with assistants, so the user never edits MCP config by hand.

1. **Claude Code:** run `claude mcp add --scope user quiver -- <absolute path to quiver> mcp`. If the `claude` command is not found, print the exact command for the user to run.
2. `--print`: only print the config snippet for manual setup.
3. `mcp uninstall` removes the registration again.
4. Detect an existing registration and update it if the binary path changed (for example after a Homebrew upgrade).

**Done when:**
- On a machine with Claude Code, `install` followed by a new session shows the Quiver tools.
- `uninstall` removes them.
