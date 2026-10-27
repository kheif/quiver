# TASK19 — Always-on skills (MCP instructions)

**Phase:** Phase 2 — MCP Server (first usable version)

**Status:** ✅ Done

**Depends on:** TASK18, TASK14

Some skills must be active from the first message, without the assistant deciding to load them. Caveman is the example: a communication style that applies to every answer. For these, Quiver puts a short pointer into the MCP server's `instructions`, and the assistant loads the skill with `get_skill` before its first reply.

The full skill text does not go into `instructions`: Claude Code cuts them at about 4096 characters, and caveman's `SKILL.md` alone is 4678. The pointer approach held in 10 of 10 single-turn spike runs and across a five-turn session.

1. **Data:** add an `alwaysOn` list next to `global` in `registry.json`, and per project (see [data-model.md](data-model.md)). An always-on skill must also be assigned in the same place.
2. **CLI:** `quiver add <id> --always` (with or without `--global`) assigns the skill and marks it always-on. `quiver remove <id> --always` removes only the mark.
3. **Server:** start `instructions` with the pointer, before the skill catalog (TASK18):
   ```
   Always-on skills: caveman/caveman. Before your first reply in this conversation, call the get_skill tool of this server for each always-on skill and follow it for the whole conversation.
   ```
4. **Project at startup:** use the project resolved in TASK17. Global always-on skills do not depend on it.
5. **Limits, to document in the README:**
   - Changes take effect in the next session, because `instructions` cannot change during a session.
   - Plugin hooks (such as caveman's per-message reminder) are not run. Only the skill text is delivered.
   - Subagents are not reliable: in the spike, one of two subagents followed the always-on skill.

**Notes:**
- `add` and `remove` share their flags through `target` in `internal/cli/assign.go`. Add `--always` there, and mark always-on skills in `printServed`.
- `quiver add --always` assigns and marks in one registry update; `quiver remove --always` only unmarks. The registry methods are `MarkAlwaysOn`, `UnmarkAlwaysOn`, their `Global` variants and `AlwaysOnServed`, in `internal/registry/assign.go`. Marking a skill that is not assigned in the same place is an error.
- `remove --always` in a project, for a skill that is always-on globally, prints a note to use `--global`. `printServed` adds an `always-on` column.
- The pointer is built by `pointer` in `internal/server/catalog.go`. `catalog` gets the budget left after it. Always-on IDs that are no longer in the library are left out, like in the catalog.
- The README does not exist yet. TASK31 must document the three limits above.

**Done when:**
- A harness test gets the pointer in `instructions`, ahead of the catalog.
- In Claude Code, a new session follows a global always-on caveman skill from the first message, with the caveman plugin disabled (checked in TASK21).
