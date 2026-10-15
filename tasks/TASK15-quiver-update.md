# TASK15 — `quiver update`

**Phase:** Phase 1 — Remote Sources

**Status:** Todo

**Depends on:** TASK13

1. `update [<name>...]` runs `git pull --ff-only` for the given sources, or all of them.
2. Record HEAD before and after. Print the skills that were added, changed or removed.
3. If an assigned skill disappeared upstream, warn and list the projects using it. Do not unassign it automatically; `doctor` reports it.
4. If a source cannot be fast-forwarded, report it and continue with the other sources.
5. Running MCP servers pick up the change through live updates (TASK26).

**Notes:**
- Sources are shallow clones. After an upstream force-push, `--ff-only` fails and the old HEAD may be gone, so `DiffNames` cannot run. Offer a re-clone in that case.

**Done when:** A test pushes a change to a fixture bare repo, runs `update`, and `get_skill` returns the new content.
