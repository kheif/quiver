# TASK09 — Git wrapper

**Phase:** Phase 1 — Remote Sources

**Status:** Todo

**Depends on:** TASK01

1. Provide `git.Clone(url, dir)`, `git.Pull(dir)`, `git.Fetch(dir)`, `git.Head(dir)`, `git.DiffNames(dir, from, to)`, `git.Diff(dir, from, to, paths)`, `git.Commit(dir, msg, paths)`.
2. Run the system `git` binary with `GIT_TERMINAL_PROMPT=0`, so missing credentials fail instead of hanging.
3. Include stderr in errors. If `git` is missing, say so clearly.
4. Clone with `--depth 1`.

**Done when:** Tests pass against local bare repositories created in `testdata/`, with no network access.
