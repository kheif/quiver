# TASK63.4: Journey: failures

**Phase:** Phase 8: Test phase

**Part of:** [TASK63](TASK63-end-to-end-user-journeys.md): End-to-end user journeys

**Status:** Todo

**Depends on:** TASK56, TASK30

Each case ends with a clear message, no half-written file, and a `doctor` report that names the fix:
- an unreachable remote: `source add`, `update` and `sync` time out (the git timeouts) and say so;
- an assistant config that is not valid JSON, and a Codex `config.toml` the codex command rejects: `install` skips that assistant, sets up the others, and exits with an error;
- a `registry.json` that is not valid JSON, and one with a newer version: commands refuse it, and the hook prints nothing and exits 0;
- a skill folder deleted by hand;
- a read-only Quiver home;
- a quiver process killed in the middle of a write (the file is the old or the new one, never a mix);
- the lock held by another process: the command waits, then fails with a message that names the lock file.

**Done when:** every case passes on all three systems.
