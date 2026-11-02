# TASK54: Always-on pointer becomes the fallback

**Phase:** Phase 6: Session-start delivery (v1.4)

**Status:** Todo

**Depends on:** TASK53

With the hook installed, the pointer in `instructions` (TASK19) makes the model load each always-on skill a second time with `get_skill`: in 6 of 10 B01 prompts in Claude Code (TASK52.1), and in 0 of 10 in Codex (TASK52.2). Without the hook it is still needed.

1. Change `pointer` in `internal/server/catalog.go` to:
   ```
   Always-on skills: <ids>. If the conversation already has a <quiver-always-on> block, these skills are loaded: follow them. Otherwise call the get_skill tool of this server for each before your first reply and follow it for the whole conversation.
   ```
2. The catalog and its budget (`maxInstructions`) stay as they are; check that the longer pointer still leaves room for the catalog.

**Done when:** the server tests check the new pointer, and a Claude Code session with the hook (TASK55.2, or a hook set by hand) follows `reply-style` in 10 of 10 B01 prompts with no `get_skill` call for it. Record the counts in the task notes.
