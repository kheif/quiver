# TASK16 — MCP server skeleton (`quiver mcp`)

**Phase:** Phase 2 — MCP Server (first usable version)

**Status:** ✅ Done

**Depends on:** TASK04, TASK08

1. `quiver mcp` starts an MCP server on stdio using the official Go SDK.
2. Server info: name `quiver`, plus the binary version.
3. **Never write to stdout** except MCP messages. Send all logs to stderr and to `~/.quiver/logs/mcp.log`.
4. Recover from panics in tool handlers and return them as tool errors, so the session survives.
5. Register a placeholder `ping` tool to prove the wiring. Remove it in TASK18.
6. Add an in-memory test harness that connects a client to the server through the SDK's in-memory transport, with no subprocess. Every later MCP task uses it.

**Notes:**
- The SDK (v1.8.0) needs Go 1.25, so `go.mod` moved from 1.23 to 1.25. See decisions.md.
- The harness is `connect` and `call` in `internal/server/harness_test.go`.
- `logs/mcp.log` is never rotated, and every server process appends to it. `doctor` (TASK29) could warn when it gets large, or the server could rotate it at startup.
- The log records the folder each server starts in, so it lists the user's project paths. Mention this before anyone shares a log.
- `recoverPanics` only covers the handler's own goroutine. A panic in a goroutine that a handler or the server starts (for example the watcher in TASK26) still ends the process.
- `TestMCPStdoutHasOnlyMessages` re-runs the test binary as `quiver mcp`. If the server ever stops answering, the test hangs until the `go test` timeout.

**Done when:**
- The harness test can list tools and call `ping`.
- Running `claude mcp add quiver -- ./bin/quiver mcp` by hand shows the server as connected in Claude Code.
