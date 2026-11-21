# TASK66.4: Journey: failures

**Phase:** Phase 8: Test phase

**Part of:** [TASK66](TASK66-end-to-end-user-journeys.md): End-to-end user journeys

**Status:** Todo

**Depends on:** TASK56, TASK30, TASK63.8, TASK65.1, TASK65.2

Each case ends with a clear message, no half-written file, and a `doctor` report that names the fix:
- an unreachable remote: `source add`, `update` and `sync` time out (the git timeouts) and say so;
- an assistant config that is not valid JSON, and a Codex `config.toml` the codex command rejects: `install` skips that assistant, sets up the others, and exits with an error;
- a `registry.json` that is not valid JSON, and one with a newer version: commands refuse it, and the hook prints nothing and exits 0;
- a skill folder deleted by hand;
- a read-only Quiver home;
- a `state/search.json` with a newer version: search runs as `words`, the message appears once and `doctor` names it (TASK62.5);
- a model file with a wrong hash or cut off: search answers with the `words` matches and one line, the engine stays `local`, and `doctor` names the step (TASK62.6, TASK62.7);
- a broken `state/bundles.json`: every `tool` command stops with a message and nothing is rewritten (TASK63.3);
- a bundle install command that fails: the ledger is as it was and no config file changed (TASK63.5, item 7);
- a quiver process killed in the middle of a write (the file is the old or the new one, never a mix);
- the lock held by another process: the command waits, then fails with a message that names the lock file. `fsutil.WithLock` waits without a limit today (`flock.Lock`); this task gives it a limit of 30 seconds, a constant that tests shorten, with `TryLockContext`.

**Done when:** every case passes on all three systems.
