# TASK64.1: Journey: setup and delivery

**Phase:** Phase 8: Test phase

**Part of:** [TASK64](TASK64-end-to-end-user-journeys.md): End-to-end user journeys

**Status:** Todo

**Depends on:** TASK57, TASK30

1. A fresh home: `quiver init` with its questions answered on stdin.
2. `quiver source add` of a local bare repository; `quiver add <id> --global --always`, and in a project folder `quiver add <id> --always`.
3. `quiver install` with fake Claude Code and Codex: both report the hook level; a JSON assistant without a hook reports its level and names the always-on skills.
4. `quiver hook session-start claude-code` with the project folder on stdin prints both skills, global first; with another folder only the global one.
5. An MCP session in the project: `instructions` has the fallback pointer and the catalog; `list_skills` and `get_skill` agree with the hook.
6. `quiver uninstall`: no Quiver entry, hook or rules-file block is left, and every other entry is.

**Done when:** the journey passes on all three systems.
