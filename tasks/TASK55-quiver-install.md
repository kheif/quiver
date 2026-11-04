# TASK55: `quiver install`

**Phase:** Phase 6: Session-start delivery (v1.4)

**Status:** ✅ Done

**Depends on:** TASK53, TASK50

Quiver now sets up more than an MCP server, so `quiver install` replaces `quiver mcp install`. Per assistant it sets up the MCP server and, where the assistant has one (TASK52), the session-start hook, and it reports how always-on skills reach each assistant. `quiver mcp` stays: it is the command assistants run. `quiver mcp config` stays too.

The current code is in `internal/cli/mcp.go` (`mcpTargets`, `mcpInstall`, `mcpUninstall`, `installClaude`, `installCodex`, `installJSON`), `internal/assistant` (Claude Code and Codex through their commands, the JSON assistants through `jsonconfig.go`) and `internal/cli/doctor.go` (`checkMCP` and the checks after it).

**Done when:** every subtask is done.

**Subtasks:**
- [TASK55.1](TASK55.1-rename-to-quiver-install-and-quiver-uninstall.md): Rename to `quiver install` and `quiver uninstall`
- [TASK55.2](TASK55.2-claude-code-hook.md): Claude Code hook
- [TASK55.3](TASK55.3-codex-hook.md): Codex hook
- [TASK55.4](TASK55.4-delivery-report-doctor-and-status.md): Delivery report, doctor and status
- [TASK55.5](TASK55.5-cursor-hook.md): Cursor hook
- [TASK55.6](TASK55.6-antigravity-hook.md): Antigravity hook
- [TASK55.7](TASK55.7-hook-text-in-parts.md): Hook text in parts
- [TASK55.8](TASK55.8-delivery-failures-and-review-gaps.md): Delivery failures and review gaps
