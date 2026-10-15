# TASK08 — Library (local skills)

**Phase:** Phase 1 — Remote Sources

**Depends on:** TASK05

1. `library.Load()` scans `~/.quiver/local/*/SKILL.md` and returns every skill with `Source = "local"` and `ID = name`.
2. `library.Get(id)` returns the skill, or `ErrSkillNotFound` with "did you mean" suggestions based on edit distance.
3. Invalid skills are reported as warnings and do not stop the scan.
4. Design the API so remote sources can be plugged in later (TASK12) without changing callers.

**Done when:** Tests pass with a fixture library containing valid and invalid skills.
