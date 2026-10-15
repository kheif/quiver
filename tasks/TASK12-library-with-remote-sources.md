# TASK12 — Library with remote sources

**Phase:** Phase 1 — Remote Sources

**Status:** Todo

**Depends on:** TASK08, TASK11

1. Extend `library.Load()` to run discovery on every source listed in the config.
2. Remote skills get `Source = <name>` and `ID = <name>/<skill>`.
3. Cache discovery results in `sources/<name>/.quiver-index.json`, keyed by the HEAD commit. Rescan only when HEAD changes.
4. `edit` and `update_skill` refuse remote skills.

**Notes:**
- `source.Discover` only finds folders. Load each one with `skill.Load` and `skill.Validate`, and turn problems into warnings as `addLocal` does.
- IDs from discovery can hold slashes (`<plugin>/<name>` on collision), so a remote ID can have three parts. Code that splits IDs must split on the first `/` only.

**Done when:**
- `list` and `list_skills` show local and remote skills together.
- A second load uses the cache, verified by a test hook.
