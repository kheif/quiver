# TASK20 — `quiver mcp install`

**Phase:** Phase 2 — MCP Server (first usable version)

**Status:** ✅ Done

**Depends on:** TASK16

Registers the server with assistants, so the user never edits MCP config by hand.

1. **Claude Code:** run `claude mcp add --scope user quiver -- <absolute path to quiver> mcp`. If the `claude` command is not found, print the exact command for the user to run.
2. `--print`: only print the config snippet for manual setup.
3. `mcp uninstall` removes the registration again.
4. Detect an existing registration and update it if the binary path changed (for example after a Homebrew upgrade).

**Notes:**
- The `claude` calls are in `internal/assistant/claude.go`; the commands in `internal/cli/mcp.go`. The binary path is `os.Executable()` without resolving symlinks, so Homebrew's stable link is kept.
- `claude mcp get` shows only the registration that wins in the current folder: a local or project one hides the user one. Install therefore looks the user one up from a new empty temp folder, then warns if the current folder has a local or project registration that runs another binary.
- `claude mcp get` starts the server to check its health, so each lookup takes about a second.
- Tests use a fake `claude` script on PATH (`internal/assistant/assistanttest`) that models the user scope and one local registration.
- Checked against the real Claude Code CLI: install, install again (unchanged), `claude mcp get` from another folder shows the user registration as connected, uninstall twice. A new session with the tools is checked in TASK21.

**Done when:**
- On a machine with Claude Code, `install` followed by a new session shows the Quiver tools.
- `uninstall` removes them.
