# TASK24 — `create_skill`

**Phase:** Phase 3 — Local Skills and Management

**Depends on:** TASK18, TASK23

**Input:** `name`, `description`, `body` (markdown without frontmatter), optional `files` (map of relative path to text content), `scope` (`"project"` or `"global"`).

1. Call `library.CreateLocal` (TASK23) with the resolved project. With `scope: "project"` and no `.quiver.json`, register the project first (TASK06), and say in the result that `.quiver.json` was created and should be committed.
2. The tool description tells the assistant to ask the user for the scope if the user did not say it.
3. Return the new skill's ID, folder path and scope.
4. Errors that the assistant can fix (invalid name, name taken) return a clear message and a suggested alternative name.
5. Mark the tool with the MCP annotations `readOnlyHint: false` and `destructiveHint: false`.

**Done when:**
- Harness tests cover both scopes, an unregistered project, a taken name and path traversal in `files`.
- A manual test in Claude Code creates a project skill from a conversation, and `quiver status` shows it.
