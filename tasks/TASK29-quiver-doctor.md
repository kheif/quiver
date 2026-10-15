# TASK29 — `quiver doctor`

**Phase:** Phase 3 — Local Skills and Management

**Status:** Todo

**Depends on:** TASK22, TASK27

Check these, and report each one as `ok`, `warn` or `error`:

1. The `~/.quiver` layout and JSON files are valid.
2. `local/` is a git repo. Uncommitted changes are a warning.
3. Every library skill passes validation.
4. Every assigned ID exists in the library. An ID can go missing when a source is removed or a skill is renamed upstream.
5. Projects whose paths all no longer exist on this machine are listed (they may still exist on another machine, so this is only a warning).
6. Skill name clashes in the served set of any project.
7. The MCP server is registered with Claude Code (`claude mcp get quiver`).

`--fix` applies the safe fixes: committing `local/` (after showing the changes) and unassigning missing IDs (after confirmation).

**Done when:** Each check has a fixture that triggers it, and `--fix` resolves the fixable ones.
