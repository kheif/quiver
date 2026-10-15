# TASK16 — MCP server skeleton (`quiver mcp`)

**Phase:** Phase 2 — MCP Server (first usable version)

**Depends on:** TASK04, TASK08

1. `quiver mcp` starts an MCP server on stdio using the official Go SDK.
2. Server info: name `quiver`, plus the binary version.
3. **Never write to stdout** except MCP messages. Send all logs to stderr and to `~/.quiver/logs/mcp.log`.
4. Recover from panics in tool handlers and return them as tool errors, so the session survives.
5. Register a placeholder `ping` tool to prove the wiring. Remove it in TASK18.
6. Add an in-memory test harness that connects a client to the server through the SDK's in-memory transport, with no subprocess. Every later MCP task uses it.

**Done when:**
- The harness test can list tools and call `ping`.
- Running `claude mcp add quiver -- ./bin/quiver mcp` by hand shows the server as connected in Claude Code.
