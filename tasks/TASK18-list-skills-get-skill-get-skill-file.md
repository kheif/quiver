# TASK18 — `list_skills`, `get_skill`, `get_skill_file`

**Phase:** Phase 2 — MCP Server (first usable version)

**Status:** Todo

**Depends on:** TASK17

**`list_skills`** (no input):

1. Returns the served skills: `id`, `name`, `description`, `scope` (`global` or `project`). It always reads the current registry, so it also shows skills assigned during the session.

**Skill catalog in `instructions`:**

Claude Code defers MCP tools behind its tool search, so the model never sees tool descriptions until it loads a tool. A skill list inside a tool description was used 0 times out of 6 in the spike. The same list in the server's `instructions` was used 6 out of 6 times, with no false positives. So the catalog goes into `instructions`:

```
Skills (reusable instructions for specific tasks). Before starting a task, check whether one fits. If it does, call the get_skill tool of this server with its id and follow it.
- db-migration: Rules for writing and running database migrations.
- anthropic/pdf: Read, create and edit PDF files.
```

1. Build it at startup from the project resolved in TASK17. Always-on pointers (TASK19) come first, the catalog second.
2. **Budget:** Claude Code cuts `instructions` at about 4096 characters and appends `… [truncated]`. Keep the whole text under 3800 characters: shorten each description to 150 characters, then drop descriptions, then end with `…and N more: call list_skills`.
3. The catalog is fixed for the session. Skills assigned later appear in `list_skills` and in the next session.

**`get_skill`** (`id`):

1. Returns the `SKILL.md` body, the list of extra files, and the absolute path of the skill folder.
2. Ends with this note: `Scripts and files referenced by this skill are in <dir>. Use absolute paths.`
3. An ID that is not served in this project is still returned if it exists in the library, with a note that it is not assigned here. Unknown IDs return an error with suggestions.

**`get_skill_file`** (`id`, `path`):

1. Resolve the path inside the skill folder. Reject any path that escapes it, including through symlinks.
2. Text files up to 256 KB are returned as text. For larger or binary files, return their absolute path and size instead.

**Done when:** Harness tests cover all three tools, path traversal attempts, binary files and the `instructions` budget.
