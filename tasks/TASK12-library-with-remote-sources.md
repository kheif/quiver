# TASK12 — Library with remote sources

**Phase:** Phase 1 — Remote Sources

**Status:** ✅ Done

**Depends on:** TASK08, TASK11

1. Extend `library.Load()` to run discovery on every source listed in the config, in name order.
2. Load each found folder with `skill.Load` and `skill.Validate`; problems become warnings, as for local skills.
3. Remote skills get `Source = <name>` and `ID = <name>/<discovered ID>`.
4. A source folder that is missing is a warning, not an error.
5. Cache discovery results in `sources/<name>/.quiver-index.json`, keyed by the HEAD commit and an index version. Rescan only when either changes. A folder that is not a git repository is scanned every time.
6. `library.CheckEditable` refuses remote skills with `ErrReadOnly`. `edit` and `update_skill` must call it.

**Notes:**
- IDs from discovery can hold slashes (`<plugin>/<name>` on collision), so a remote ID can have three parts. Code that splits IDs must split on the first `/` only.
- Bump `indexVersion` whenever discovery rules change, or old caches keep the old result.
- The index is an untracked file in the source's working tree. `git status` shows it, and an upstream file with the same name would block `git pull`. TASK15 must handle both.
- `Load` now reads `config.json`, so a broken config fails every load, also for local skills.

**Done when:**
- `Library.All` returns local and remote skills together; `list` and `list_skills` build on it.
- A second load uses the cache, verified by a test hook.
