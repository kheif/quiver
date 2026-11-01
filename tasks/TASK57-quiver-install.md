# TASK57: `quiver install`

**Phase:** Phase 6: Session-start delivery

**Status:** Todo

**Depends on:** TASK55, TASK50

Quiver now sets up more than an MCP server, so `quiver install` replaces `quiver mcp install`. Per assistant it sets up the MCP server and, where the assistant has one (TASK54), the session-start hook, and it reports how always-on skills reach each assistant. `quiver mcp` stays: it is the command assistants run. `quiver mcp config` stays too.

The current code is in `internal/cli/mcp.go` (`mcpTargets`, `mcpInstall`, `mcpUninstall`, `installClaude`, `installCodex`, `installJSON`), `internal/assistant` (Claude Code and Codex through their commands, the JSON assistants through `jsonconfig.go`) and `internal/cli/doctor.go` (`checkMCP` and the checks after it).

**Done when:** every subtask is done.

**Subtasks:**
- [TASK57.1](TASK57.1-rename-to-quiver-install-and-quiver-uninstall.md): Rename to `quiver install` and `quiver uninstall`
- [TASK57.2](TASK57.2-claude-code-hook.md): Claude Code hook
- [TASK57.3](TASK57.3-codex-hook.md): Codex hook
- [TASK57.4](TASK57.4-delivery-report-doctor-and-status.md): Delivery report, doctor and status
