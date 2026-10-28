# TASK22 — `quiver list` and `quiver status`

**Phase:** Phase 3 — Local Skills and Management

**Status:** ✅ Done

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

**Notes:**
- `printServed` in `internal/cli/assign.go` already prints the served list after `add` and `remove`. Use it here, so both outputs stay the same.
- The commands are in `internal/cli/list.go` and `internal/cli/status.go`. Columns are ID, ASSIGNED, DESCRIPTION: the description comes last, so cutting it to the terminal width keeps the table aligned. A skill both global and in projects shows `global, 2 projects`.
- Width and dim text come from `golang.org/x/term` (`internal/cli/term.go`). Output that is not a terminal is never cut or coloured, and `NO_COLOR` turns dim text off.
- `--json` prints `{"skills": [...], "warnings": [...]}`; each skill has `id`, `name`, `source`, `description`, `global` and `projects` (project IDs).
- `status` finds the project with `server.Resolve`, so it honours `$QUIVER_PROJECT` like the server. It checks for a `PossibleCopy` on an in-memory copy of the registry and writes nothing. It also warns about assigned skills that are no longer in the library, which the server leaves out.
- Golden files are in `internal/cli/testdata`; rewrite them with `go test ./internal/cli -update`.

**Done when:** Output matches golden files for a fixture library and registry.
