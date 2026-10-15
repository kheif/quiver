# TASK09 — Git wrapper

**Phase:** Phase 1 — Remote Sources

**Status:** ✅ Done

**Depends on:** TASK01

1. Provide `git.Clone(url, dir)`, `git.Pull(dir)`, `git.Fetch(dir)`, `git.Head(dir)`, `git.DiffNames(dir, from, to)`, `git.Diff(dir, from, to, paths...)`, `git.Commit(dir, msg, paths...)`. `Init` and `HasCommit` serve `quiver init`.
2. Run the system `git` binary with `GIT_TERMINAL_PROMPT=0`, so missing credentials fail instead of hanging.
3. Include stderr in errors. If `git` is missing, return `ErrNotInstalled`.
4. Clone with `--depth 1`. Pull with `--ff-only`.
5. `Commit` does nothing when nothing changed, and commits as "Quiver" when the user has no git identity.

**Notes:**
- `run` passes the user's environment through. A `GIT_DIR` or `GIT_WORK_TREE` set by a git hook or IDE would point every command at the wrong repository. Unset them in `run` if this shows up.
- `Clone` puts `--` before the URL, but other dangerous inputs (`ext::`, `file://` from untrusted input) must be rejected by URL parsing (TASK10).

**Done when:** Tests pass against local bare repositories created in temporary folders, with no network access.
