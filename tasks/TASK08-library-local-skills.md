# TASK08 — Library (local skills)

**Phase:** Phase 1 — Remote Sources

**Status:** ✅ Done

**Depends on:** TASK05

1. `library.Load()` scans `~/.quiver/local/*/SKILL.md` and returns every skill with `Source = "local"` and `ID = <folder name>` (see [data-model.md](data-model.md); folder names are unique, `name` values may not be).
2. `library.Get(id)` returns the skill, or `ErrSkillNotFound` with "did you mean" suggestions based on edit distance.
3. Invalid skills are reported as warnings and do not stop the scan.
4. Design the API so remote sources can be plugged in later (TASK12) without changing callers.

**Notes:**
- A symlink inside `local/` is skipped without a warning, because `os.ReadDir` reports it as a file. If users link skills into `local/`, follow links that point to folders and warn about the rest.
- `Library` is a snapshot. Long-running callers (the MCP server, TASK26) must call `Load` again after a change.

**Done when:** Tests pass with a fixture library containing valid and invalid skills.
