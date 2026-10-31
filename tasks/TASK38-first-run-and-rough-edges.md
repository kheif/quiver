# TASK38 — First run and rough edges

**Phase:** Phase 4c — v1.2

**Status:** ✅ Done

**Depends on:** TASK35

From the v1 review: Quiver has many concepts (library, source, global, project, always-on), and a few prompts are hard to read.

1. **First run.** `quiver init` in a terminal becomes a short guided setup: create the library, register the MCP server (`mcp install`), and offer to import existing skills from `~/.claude/skills` and from the `.claude/skills` of the current project. Every step can be skipped. `--yes` and a non-terminal run keep today's plain init.
2. **`quiver` with no arguments,** until the TUI exists, prints three lines: the project here, how many skills it is served, and a next step (`quiver init`, `quiver add`, …), then the command list.
3. **Clearer import question.** `[P/g/s]` becomes words: `Assign to (p)roject, (g)lobal, or (s)kip? [p]`.
4. **Confirm a new project.** When `add`, `import` or `new` would create a `.quiver.json`, show the folder first (`Register /x/y as a project? [Y/n]`), so a command run in the wrong folder does not create a project there. `--yes` skips the question.
5. **`--json`** for `status`, `projects` and `where`, like `list`.
6. **New-session note.** After `add` and `remove`, say that open assistant sessions see the change through `list_skills`, and the catalog in the next session.

**Done when:** tests cover the guided init with each step taken and skipped, the bare `quiver` output, the confirmation, and the JSON output.

**Notes:**
- A run counts as a terminal when stdin is one (`interactive` in `internal/cli/term.go`). The guided init reads every answer through one `bufio.Reader`, so no answer is read ahead. Skills in `~/.claude/skills` default to global, those in the project's `.claude/skills` to the project; an import offer is shown only when there are skills to import, and the project offer is left out in the home folder, where both are the same folder.
- `quiver` alone skips the auto-init, so on a new machine it suggests `quiver init` instead of creating the home. Next steps: `quiver init` without a home, `quiver new`/`quiver source add` with an empty library, `quiver add` when nothing is served here, else `quiver status`. It uses `QUIVER_PROJECT` like `status`.
- The project question is asked only in a terminal, so scripts and pipes keep working without `--yes`. It names the folder `Register` would use: `project.NewRoot`, the nearest folder with `.git`. In `import`, a "no" skips the skill and every later project skill of the run, without asking again.
- JSON fields: `status` gives `folder`, `project` (null outside a project), `skills` (`id`, `scope`, `always_on`, `in_library`) and `warnings`; `projects` gives `projects`, each with `id`, `name`, `skills`, `paths` and `missing`; `where` gives `id`, `in_library`, `global` and `projects` in the same form. `printJSON` in `list.go` is shared by every `--json`.
