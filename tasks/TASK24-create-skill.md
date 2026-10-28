# TASK24 — `create_skill`

**Phase:** Phase 3 — Local Skills and Management

**Status:** ✅ Done

**Depends on:** TASK18, TASK23

**Input:** `name`, `description`, `body` (markdown without frontmatter), optional `files` (map of relative path to text content), `scope` (`"project"` or `"global"`).

1. Call `library.CreateLocal` (TASK23) with the resolved project. With `scope: "project"` and no `.quiver.json`, register the project first (TASK06), and say in the result that `.quiver.json` was created and should be committed.
2. The tool description tells the assistant to ask the user for the scope if the user did not say it.
3. Return the new skill's ID, folder path and scope.
4. Errors that the assistant can fix (invalid name, name taken) return a clear message and a suggested alternative name.
5. Mark the tool with the MCP annotations `readOnlyHint: false` and `destructiveHint: false`.

**Notes:**
- The tool is in `internal/server/create.go`. Its input schema restricts `scope` to `project` and `global`, and the handler checks it again.
- The served project used to be fixed at startup. It now lives in a `session` with a mutex (`internal/server/server.go`): when `create_skill` registers the folder, every later request in the session serves that project.
- `registry.RegisterProject` finds or registers the project and records it under one lock. `quiver new` uses it too.
- Error hints: an invalid name suggests its slug (`My Notes!` gives `my-notes`); a taken name suggests the first free `<name>-N`. Path errors come from `library.CreateLocal`.
- The server `instructions` now always end with a pointer to `create_skill`, inside the 3800-byte budget (`createHint`, `buildInstructions`). Without it the manual test failed: Claude Code defers MCP tools, the instructions were empty for an empty library, and Claude wrote the skill file itself.
- Manual test, run headless: `claude -p` in a new folder with `--mcp-config` pointing to `bin/quiver` and a temporary `QUIVER_HOME`. Claude called `create_skill` with `scope: "project"`, `.quiver.json` was created, and `quiver status` listed the skill as `project`.
- Backlog B07 (approval for global skills) is still open and must be decided before release (TASK34): a prompt injection can still create a global skill.

**Done when:**
- Harness tests cover both scopes, an unregistered project, a taken name and path traversal in `files`.
- A manual test in Claude Code creates a project skill from a conversation, and `quiver status` shows it.
