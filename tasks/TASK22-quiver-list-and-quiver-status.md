# TASK22 — `quiver list` and `quiver status`

**Phase:** Phase 3 — Local Skills and Management

**Depends on:** TASK07, TASK08

**`list`:**

1. Print one row per library skill: ID, description (cut to terminal width), and where it is assigned (`global`, `3 projects`, or blank).
2. Flags:
   - `--source <name>`
   - `--unassigned`
   - `--json`
3. Print library warnings at the end, in dim text.

**`status`:**

1. Find the current project and print its ID, name and root.
2. Print the skills the MCP server would serve here (`registry.Served`), each marked `global` or `project`.
3. If there is no `.quiver.json`, print `not registered; only global skills are served`.
4. Show the `PossibleCopy` warning from TASK07 with the fix: `quiver projects split`.

**Done when:** Output matches golden files for a fixture library and registry.
