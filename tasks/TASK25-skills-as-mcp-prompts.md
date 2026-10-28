# TASK25 — Skills as MCP prompts

**Phase:** Phase 3 — Local Skills and Management

**Status:** ✅ Done

**Depends on:** TASK18

1. Expose every served skill as an MCP prompt:
   - prompt name: the skill ID, with `/` replaced by `__`
   - prompt description: the skill description
2. Getting a prompt returns the skill body as a user message, followed by the same folder path note as `get_skill`.
3. Accept an optional `args` argument. If it is given, append it after the skill content, prefixed with `Task:`.

**Notes:**
- The prompts are in `internal/server/prompts.go`; `get_skill` and the prompts share `skillText`. The prompt name keeps the `/` of a source skill's ID (`acme/review`): Claude Code uses prompt names as given, and `/mcp__quiver__acme/review` was checked to run.
- Claude Code lists a prompt as `/quiver:<id> (MCP)`; the server name is the prefix and cannot be dropped. A prefix-free `/<id>` would need skills synced into Claude Code's own skills folder: a separate task.
- Claude Code splits the text after the command on spaces, gives one word to each declared argument and drops the rest; quotes do not help. So the task is declared as `args`, `args2` … `args32` and joined again (`joinArgs`): up to 32 words.
- In `claude -p` only the `/mcp__quiver__<id>` form runs a prompt; `/quiver:<id>` is the interactive menu form.
- The prompt list is read at startup. A skill is read again on every get, so edits are seen; a deleted skill returns an error.

**Done when:**
- Harness tests list and get prompts.
- In Claude Code, `/mcp__quiver__<skill>` loads the skill.
