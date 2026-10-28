# TASK26 — Live updates

**Phase:** Phase 3 — Local Skills and Management

**Status:** ✅ Done

**Depends on:** TASK24, TASK25

1. Watch `registry.json`, `config.json`, `local/` and `sources/` with `fsnotify`. Debounce events by 300 ms.
2. When the served set changes, reload it and send `notifications/prompts/list_changed`. `list_skills` returns the new set on its next call. The catalog in `instructions` stays as it was until the next session (TASK18).
3. Content changes to a skill need no notification, because `get_skill` always reads from disk.
4. A failed reload keeps the last good state and logs the error.

**Notes:**
- `recoverPanics` (TASK16) does not cover the watcher goroutine. Recover inside it, so a bad event does not end the server.
- The watcher is in `internal/server/watch.go`. fsnotify does not watch subfolders, so it watches the home folder (only `registry.json` and `config.json` count), `local/` and each skill folder in it, `sources/`, and each source's `.git` folder, which a pull or reset changes. New skill folders and new sources are added as they appear. Hidden names (`.lock`, `local/.new-*`) are ignored.
- `prompts.sync` diffs the served skills against the registered prompts and adds, replaces or removes only what changed, so a description change also updates the prompt. The SDK batches the `list_changed` notifications.
- `New` still returns a plain `*mcp.Server`; `newServer` also returns what `watch` needs. `Run` starts the watcher; tests start it with a 20 ms debounce.
- The harness tests change the registry with `registry.Update` instead of running `quiver add`: the server package cannot import the CLI. A run of the real binary showed `skill list changed` in the log about a second after `quiver add`.

**Done when:** A harness test runs `quiver add` from another goroutine and receives the notification. The new skill appears in `list_skills`.
