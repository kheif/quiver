# TASK19 — Always-on skills (MCP instructions)

**Phase:** Phase 2 — MCP Server (first usable version)

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

**Done when:**
- A harness test gets the pointer in `instructions`, ahead of the catalog.
- In Claude Code, a new session follows a global always-on caveman skill from the first message, with the caveman plugin disabled (checked in TASK21).
