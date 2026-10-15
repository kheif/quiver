# TASK12 — Library with remote sources

**Phase:** Phase 1 — Remote Sources

**Depends on:** TASK08, TASK11

1. Extend `library.Load()` to run discovery on every source listed in the config.
2. Remote skills get `Source = <name>` and `ID = <name>/<skill>`.
3. Cache discovery results in `sources/<name>/.quiver-index.json`, keyed by the HEAD commit. Rescan only when HEAD changes.
4. `edit` and `update_skill` refuse remote skills.

**Done when:**
- `list` and `list_skills` show local and remote skills together.
- A second load uses the cache, verified by a test hook.
