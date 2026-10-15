# TASK30 — End-to-end test suite

**Phase:** Phase 4 — v1 Release

**Status:** Todo

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

**Done when:** All flows pass in CI.
