# TASK42 — Split machine-local state out of the registry

**Phase:** Phase 5 — v2

**Status:** ✅ Done

**Depends on:** TASK41

Backlog B04. `registry.json` holds each project's absolute `paths`. Synced between machines (Phase 5's library sync), the paths are wrong on the other machine and both machines rewrite the file, so every sync conflicts.

1. Keep only portable data in `registry.json`: global and always-on skills, and per project its ID, name, skills and always-on skills.
2. Move `paths` and other machine-local data into `~/.quiver/state/` (ignored by git). The version check of TASK39 is already there.
3. Migrate: registry version 2. Reading version 1 moves the paths into the state on the first write; an older quiver reading version 2 already stops with "update quiver".
4. Update data-model.md.

**Done when:** tests cover the migration, and two copies of `~/.quiver` with different project paths share one `registry.json` without changes to it.
