# TASK29 — `quiver doctor`

**Phase:** Phase 3 — Local Skills and Management

**Status:** ✅ Done

**Depends on:** TASK22, TASK27

Check these, and report each one as `ok`, `warn` or `error`:

1. The `~/.quiver` layout and JSON files are valid.
2. `local/` is a git repo. Uncommitted changes are a warning.
3. Every library skill passes validation.
4. Every assigned ID exists in the library. An ID can go missing when a source is removed or a skill is renamed upstream.
5. Every source in `config.json` has its folder in `sources/`. The library only warns "not downloaded" when it is missing. Leftover `sources/.add-*` and `sources/.remove-*` folders are a warning; `--fix` deletes them.
6. Projects whose paths all no longer exist on this machine are listed (they may still exist on another machine, so this is only a warning).
7. Skill name clashes in the served set of any project.
8. The MCP server is registered with Claude Code (`claude mcp get quiver`).

`--fix` applies the safe fixes: committing `local/` (after showing the changes) and unassigning missing IDs (after confirmation).

**Implementation notes:**
- `doctor` overrides the root `PersistentPreRunE`, so it checks the home as it is instead of repairing it first. A missing part of the home is an error; `--fix` runs `setup.Init`, which never overwrites.
- Checks that need an invalid `config.json` or `registry.json` are shown as skipped. Exit code 1 when errors remain; warnings alone exit 0.
- Assigned skills of a configured source that is not downloaded are listed but not unassigned by `--fix`: adding the source again is the fix.
- Name clashes in the global set are reported once as "every folder", not again per project.
- The MCP check warns when Claude Code runs another quiver binary than this one, and when the `claude` command is missing.

**Done when:** Each check has a fixture that triggers it, and `--fix` resolves the fixable ones.
