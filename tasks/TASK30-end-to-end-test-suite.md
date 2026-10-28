# TASK30 — End-to-end test suite

**Phase:** Phase 4 — v1 Release

**Status:** ✅ Done

**Depends on:** TASK20, TASK15, TASK29

1. Build a harness that:
   - creates a temp `HOME` and `QUIVER_HOME`,
   - creates fixture projects and bare git repositories,
   - runs the compiled binary for CLI steps,
   - connects an MCP client to `quiver mcp` as a subprocess for MCP steps.
2. Cover these flows:
   - `new` → `add` → `list_skills` shows it → `remove` → gone
   - `create_skill` with project scope, then with global scope
   - `import` from a project with `.claude/skills`
   - `source add` → `add anthropic/x` → `update` → new content served
   - a moved project folder keeps its skills; a copied one gets the `PossibleCopy` warning, and `projects split` separates it
   - `doctor --fix`
3. Run it on all three OSes in CI.

**Implementation notes:**
- The suite is in `e2e/`. `TestMain` builds the binary; each test gets its own `HOME`, `QUIVER_HOME` and git identity, and a `PATH` with only git's folder, so the real `claude` is never run. MCP steps use the SDK's `CommandTransport` on `quiver mcp`. It runs with `go test ./...`.
- It found a bug: a server started in a folder that was not registered yet never served the project that `quiver add` then registered there. `session.projectID` now looks for `.quiver.json` again while the session has no project.
- CI (`.github/workflows/ci.yml`, the deferred step of TASK01) runs `go test ./...` on Linux, macOS and Windows, and `golangci-lint` on Linux. On Windows the tests that need the fake `claude` shell script, or a symbolic link, are skipped.

**Done when:** All flows pass in CI.
