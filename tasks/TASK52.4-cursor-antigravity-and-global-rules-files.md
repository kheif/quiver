# TASK52.4: Cursor, Antigravity and global rules files

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK52](TASK52-spike-session-start-hooks-per-assistant.md): Spike: session-start hooks per assistant

**Status:** ✅ Done

**Depends on:** TASK50

Facts: in a workspace test on 2026-11-01, a one-line `AGENTS.md` telling the model to load Quiver's tools before the first reply raised always-on from 0 to 10 of 10 prompts in Codex and from 0 to 9 of 10 in Antigravity, and fitting skills from 0 and 1 to 6 of 6. That test put the file in the workspace only.

Steps:
1. Cursor: does it have a hooks file with an event at session start or before the first prompt whose output reaches the model? Cursor already gets `instructions`, so a hook only matters if it is more reliable.
2. Antigravity CLI (`agy` 1.4): does it have hooks, for example in its plugins (`agy plugin`)? Where does the Antigravity app, as opposed to the CLI, keep its MCP config?
3. Global rules files: does Codex load `~/.codex/AGENTS.md` (or `$CODEX_HOME/AGENTS.md`) in every project, and Antigravity `~/.gemini/GEMINI.md`? Repeat the workspace test with only the global file.
4. Does a project's own `AGENTS.md` add to the global one or replace it?

**Done when:** the results are in `docs/spike-hooks.md`.

**Notes:**
- Results in docs/spike-hooks.md (`cursor-hooks-run.sh`, `antigravity-hooks-run.sh`, `codex-rules-run.sh`). Delivery level: hook for both.
- Cursor CLI 2026.10.22: a `sessionStart` hook in `.cursor/hooks.json` printing `{"additional_context": ...}` gave always-on 10 of 10, fitting 6 of 6, other 0 of 4, no `get_skill` for `reply-style`. `beforeSubmitPrompt` did not run in the CLI, and `.claude/settings.json` was not read. User file: `~/.cursor/hooks.json`. The Cursor app was not measured.
- Antigravity `agy` 1.4.12: hooks live in `hooks.json` in a customization root, the global `~/.gemini/config/` or the workspace's `.agents/`, as `{"<name>": {"SessionStart": [{"type": "command", "command": "..."}]}}`. Output `{"injectSteps": [{"systemMessage": {"systemMessage": "..."}}]}` gave always-on 10 of 10 (fitting 0 of 6, as without the hook); plain stdout does nothing. The text is cut after about 48,900 bytes. Each sub-agent conversation runs `SessionStart` too. The global `hooks.json` worked. `.agents/mcp_config.json` in a workspace works, which TASK50 missed.
- Global rules files: a pointer in `$CODEX_HOME/AGENTS.md` gave 10 of 10 in Codex; in `~/.gemini/config/GEMINI.md` 7 of 10 in Antigravity, where 3 headless sessions stopped on a permission. `~/.gemini/config/AGENTS.md` and `~/.gemini/GEMINI.md` are read too. A project's `AGENTS.md` adds to the global file in both.
- The Antigravity app was not installed; by `agy`'s built-in docs it shares `~/.gemini/config/`.
