# TASK54: Always-on pointer becomes the fallback

**Phase:** Phase 6: Session-start delivery (v1.4)

**Status:** ✅ Done

**Depends on:** TASK53

With the hook installed, the pointer in `instructions` (TASK19) makes the model load each always-on skill a second time with `get_skill`: in 6 of 10 B01 prompts in Claude Code (TASK52.1) and in VS Code (TASK52.3), and in 0 of 10 in Codex (TASK52.2). Without the hook it is still needed.

1. Change `pointer` in `internal/server/catalog.go` to:
   ```
   Always-on skills: <ids>. If the conversation already has a <quiver-always-on> block, these skills are loaded: follow them. Otherwise call the get_skill tool of this server for each before your first reply and follow it for the whole conversation.
   ```
2. The catalog and its budget (`maxInstructions`) stay as they are; check that the longer pointer still leaves room for the catalog.

**Done when:** the server tests check the new pointer, and a Claude Code session with the hook (TASK55.2, or a hook set by hand) follows `reply-style` in 10 of 10 B01 prompts with no `get_skill` call for it. Record the counts in the task notes.

**Notes:**
- `pointer` in `internal/server/catalog.go` now names the `<quiver-always-on>` block as the primary path and `get_skill` as the fallback. Server tests updated (`TestCatalogBudget`, `TestInstructionsTooManySkills`); the budget test still passes, so the longer pointer leaves room for the catalog under `maxInstructions` (3800).
- B01, Claude Code 2.1.308, `spike/b01/prompts.txt`, both the hook (`quiver hook session-start claude-code`) and the server's pointer active. With the real hook emitting the `<quiver-always-on>` block: always-on 10 of 10, fitting 6 of 6, other 0 of 4, `get_skill(reply-style)` **0 of 10**. Done-when met.
- For comparison, with the hook delivering the raw SKILL.md (no `<quiver-always-on>` block): always-on 10 of 10, `get_skill(reply-style)` 2 of 10. The old pointer with the same setup was 6 of 10 (TASK52.1). The block plus the new pointer removes the redundant second load.
