# TASK38 — First run and rough edges

**Phase:** Phase 4c — v1.2

**Status:** Todo

**Depends on:** TASK35

From the v1 review: Quiver has many concepts (library, source, global, project, always-on), and a few prompts are hard to read.

1. **First run.** `quiver init` in a terminal becomes a short guided setup: create the library, register the MCP server (`mcp install`), and offer to import existing skills from `~/.claude/skills` and from the `.claude/skills` of the current project. Every step can be skipped. `--yes` and a non-terminal run keep today's plain init.
2. **`quiver` with no arguments,** until the TUI exists, prints three lines: the project here, how many skills it is served, and a next step (`quiver init`, `quiver add`, …), then the command list.
3. **Clearer import question.** `[P/g/s]` becomes words: `Assign to (p)roject, (g)lobal, or (s)kip? [p]`.
4. **Confirm a new project.** When `add`, `import` or `new` would create a `.quiver.json`, show the folder first (`Register /x/y as a project? [Y/n]`), so a command run in the wrong folder does not create a project there. `--yes` skips the question.
5. **`--json`** for `status`, `projects` and `where`, like `list`.
6. **New-session note.** After `add` and `remove`, say that open assistant sessions see the change through `list_skills`, and the catalog in the next session.

**Done when:** tests cover the guided init with each step taken and skipped, the bare `quiver` output, the confirmation, and the JSON output.
