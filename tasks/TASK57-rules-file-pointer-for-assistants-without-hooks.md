# TASK57: Rules-file pointer for assistants without hooks

**Phase:** Phase 6: Session-start delivery (v1.4)

**Status:** Todo

**Depends on:** TASK52.4, TASK55.4

For an assistant without a usable hook that always loads a rules file, `quiver install` adds a short pointer to that file. In the 2026-11-01 workspace test this raised always-on from 0 to 10 of 10 prompts in Codex and to 9 of 10 in Antigravity.

1. **Block.** Between marker comments, so nothing else in the file changes:
   ```
   <!-- quiver:start -->
   Skills on this machine are served by the MCP server named quiver. Before your first reply in every conversation, load the quiver tools (search for them if they are not loaded) and follow the quiver server's instructions.
   <!-- quiver:end -->
   ```
2. **Global file first.** Use the assistant's global rules file found in TASK52.4, such as `~/.gemini/GEMINI.md`, so no project changes. Only assistants whose level is rules file get it.
3. **Project files, opt in.** For an assistant without a global file, write the block into the project's rules file only when the user turned this on once (`"projectRulesFile": true` in `config.json`). Write it when a project is registered and when the MCP server starts in a registered project whose file lacks it; the first session in a new project may miss it. If Quiver creates the file, add it to the repository's `.git/info/exclude`, as TASK61 does for generated skills. The install report names every file it changed.
4. `uninstall` removes the block, and the file if Quiver created it and nothing else is in it. `doctor` checks the block is there and current.

**Done when:** tests cover a new file, an existing file, an update of the block, removal, the setting and `.git/info/exclude`.
