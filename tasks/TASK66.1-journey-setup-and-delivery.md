# TASK66.1: Journey: setup and delivery

**Phase:** Phase 8: Test phase

**Part of:** [TASK66](TASK66-end-to-end-user-journeys.md): End-to-end user journeys

**Status:** Todo

**Depends on:** TASK56, TASK30, TASK63.8, TASK65.1, TASK65.2

1. A fresh home: `quiver init` with its questions answered on stdin.
2. `quiver source add` of a local bare repository; `quiver add <id> --global --always`, and in a project folder `quiver add <id> --always`.
3. `quiver install` with fake Claude Code and Codex: both report the hook level; a JSON assistant without a hook reports its level and names the always-on skills.
4. `quiver hook session-start claude-code` with the project folder on stdin prints both skills, global first; with another folder only the global one.
5. An MCP session in the project: `instructions` has the fallback pointer and the catalog; `list_skills` and `get_skill` agree with the hook.
6. A folder with one Searchable skill and nothing listed: the hook prints the cue alone, and in a folder without a Searchable skill it prints no cue (TASK62.8). In an MCP session `search_skills` finds the skill and `get_skill` loads it without the "not assigned" note.
7. `quiver uninstall`: no Quiver entry or hook is left, and every other entry is.

**Done when:** the journey passes on all three systems.
