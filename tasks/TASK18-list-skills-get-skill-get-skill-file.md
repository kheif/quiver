# TASK18 — `list_skills`, `get_skill`, `get_skill_file`

**Phase:** Phase 2 — MCP Server (first usable version)

**Status:** ✅ Done

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
   - `skill.Files` lists symlinks as they are. Drop any entry that resolves outside the skill folder.
2. Ends with this note: `Scripts and files referenced by this skill are in <dir>. Use absolute paths.`
3. An ID that is not served in this project is still returned if it exists in the library, with a note that it is not assigned here. Unknown IDs return an error with suggestions.

**`get_skill_file`** (`id`, `path`):

1. Resolve the path inside the skill folder. Reject any path that escapes it, including through symlinks.
2. Text files up to 256 KB are returned as text. For larger or binary files, return their absolute path and size instead.

**Notes:**
- TASK15 checked `update` only through `library.Get`. Add a test that runs `update` and gets the new content through `get_skill`.
- Remove `addPing` from `internal/server/server.go`. `TestPing` expects exactly one tool; replace it.
- The tools are in `internal/server/skills.go`, the catalog in `catalog.go`. Every call loads the library and the registry again, so skills assigned or updated during the session show at once. `TestGetSkillAfterUpdate` covers `update`.
- `New` now reads the library and the registry to build `instructions`, so every test that calls it must set `QUIVER_HOME`.
- A skill that is both global and assigned to the project has scope `global`. Assigned IDs that are no longer in the library are left out without a warning.
- `list_skills` returns structured output. The SDK also puts the same JSON into the text content.
- The budget counts bytes, which are never fewer than characters (`maxInstructions`). Descriptions are dropped from the last skill backwards, so skills early in ID order keep theirs. TASK19 must pass `catalog` the budget left after its pointer.
- `get_skill_file` also refuses hidden files (a skill at the root of a source repository has `.git` in its folder) and anything that is not a regular file. `get_skill` leaves the same entries out of its file list, so broken symlinks and links to folders are not listed. Text that is not valid UTF-8 or has a NUL byte counts as binary.
- Still open from TASK17: a startup resolve error is only logged; `list_skills` does not mention it.

**Done when:** Harness tests cover all three tools, path traversal attempts, binary files and the `instructions` budget.
