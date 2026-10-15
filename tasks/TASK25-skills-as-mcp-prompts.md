# TASK25 — Skills as MCP prompts

**Phase:** Phase 3 — Local Skills and Management

**Status:** Todo

**Depends on:** TASK18

1. Expose every served skill as an MCP prompt:
   - prompt name: the skill ID, with `/` replaced by `__`
   - prompt description: the skill description
2. Getting a prompt returns the skill body as a user message, followed by the same folder path note as `get_skill`.
3. Accept an optional `args` argument. If it is given, append it after the skill content, prefixed with `Task:`.

**Done when:**
- Harness tests list and get prompts.
- In Claude Code, `/mcp__quiver__<skill>` loads the skill.
