# TASK33 — `update_skill` MCP tool

**Phase:** Phase 5 — v2

**Depends on:** TASK24

1. Input: `id`, and any of `description`, `body`, `files` (path to content; empty content deletes the file).
2. Only local skills can be updated. For remote skills, return an error suggesting `fork`.
3. Commit in `local/` with the message `update <name> (via MCP)`.
4. The tool description asks the assistant to confirm the change with the user first.

**Done when:** Harness tests cover a body change, a file add and delete, a remote skill and path traversal.
