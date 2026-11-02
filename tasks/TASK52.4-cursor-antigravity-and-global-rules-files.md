# TASK52.4: Cursor, Antigravity and global rules files

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK52](TASK52-spike-session-start-hooks-per-assistant.md): Spike: session-start hooks per assistant

**Status:** Todo

**Depends on:** TASK50

Facts: in a workspace test on 2026-11-01, a one-line `AGENTS.md` telling the model to load Quiver's tools before the first reply raised always-on from 0 to 10 of 10 prompts in Codex and from 0 to 9 of 10 in Antigravity, and fitting skills from 0 and 1 to 6 of 6. That test put the file in the workspace only.

Steps:
1. Cursor: does it have a hooks file with an event at session start or before the first prompt whose output reaches the model? Cursor already gets `instructions`, so a hook only matters if it is more reliable.
2. Antigravity CLI (`agy` 1.4): does it have hooks, for example in its plugins (`agy plugin`)? Where does the Antigravity app, as opposed to the CLI, keep its MCP config?
3. Global rules files: does Codex load `~/.codex/AGENTS.md` (or `$CODEX_HOME/AGENTS.md`) in every project, and Antigravity `~/.gemini/GEMINI.md`? Repeat the workspace test with only the global file.
4. Does a project's own `AGENTS.md` add to the global one or replace it?

**Done when:** the results are in `docs/spike-hooks.md`.
