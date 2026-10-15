# TASK26 — Live updates

**Phase:** Phase 3 — Local Skills and Management

**Status:** Todo

**Depends on:** TASK24, TASK25

1. Watch `registry.json`, `config.json`, `local/` and `sources/` with `fsnotify`. Debounce events by 300 ms.
2. When the served set changes, reload it and send `notifications/prompts/list_changed`. `list_skills` returns the new set on its next call. The catalog in `instructions` stays as it was until the next session (TASK18).
3. Content changes to a skill need no notification, because `get_skill` always reads from disk.
4. A failed reload keeps the last good state and logs the error.

**Done when:** A harness test runs `quiver add` from another goroutine and receives the notification. The new skill appears in `list_skills`.
