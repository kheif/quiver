# TASK41 — Background update checks

**Phase:** Phase 5 — v2

**Depends on:** TASK40, TASK15

Inside the TUI, check the remote sources for updates quietly, one at a time.

1. Start a background worker when the TUI opens. It checks **one source at a time**, with a short pause between sources.
2. For each source, run `git ls-remote <url> HEAD` and compare it with the local HEAD. This downloads nothing.
3. Skip the whole run when there is no network. Each check has a 10 s timeout. A failure is logged and skipped, never shown as an error.
4. Store the last check time per source in `~/.quiver/state.json`. Check a source again only after the interval (default 6 hours, configurable as `updateCheckInterval`).
5. When a source has updates, show a small notice in the bottom-right corner: `anthropic: 3 skills updated — press u to pull`. Several sources are shown as a count.
6. `u` opens a list of the sources with updates and the changed skills (TASK37 preview). Pulling runs `quiver update` for the selected sources.
7. The worker never pulls by itself. Pulling always needs a key press.

**Done when:**
- A test with a fixture remote that gains a commit shows the notice.
- A test with no network shows nothing and does not block the TUI.
- Checks respect the interval.
