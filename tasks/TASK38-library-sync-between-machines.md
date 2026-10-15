# TASK38 — Library sync between machines

**Phase:** Phase 5 — v2

**Status:** Todo

**Depends on:** TASK30

1. `quiver sync init <git-remote>`: turns `~/.quiver` itself into a git repository that tracks `local/`, `config.json` and `registry.json`. `sources/` and `logs/` are ignored, because sources are re-cloned from the config.
2. `quiver sync`: commit, pull with rebase, push. On conflicts in `registry.json`, merge the lists (union) automatically. Stop on any other conflict and explain it.
3. After a pull, clone any source that is in the config but missing on disk.
4. Merge the `local/` git repo into the top-level repo, or keep it as a submodule. Decide in this task and record the reason.

**Done when:** A test with two `QUIVER_HOME` folders and one bare remote syncs a new skill and an assignment from machine A to machine B.
