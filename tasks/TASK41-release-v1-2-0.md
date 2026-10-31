# TASK41 — Release v1.2.0

**Phase:** Phase 4c — v1.2

**Status:** ✅ Done

**Depends on:** TASK34, TASK35, TASK36, TASK37, TASK38, TASK39, TASK40

1. README and help text cover every change of this phase; PROJECT.md's roadmap moves the pulled-in items from v2 to v1.2.
2. Mark the done items in `tasks/backlog.md` (B08, B09, B13).
3. Tag `v1.2.0` only after the maintainer confirms. The release workflow publishes to GitHub and npm and installs on clean machines.

**Done when:** the release workflow is green.

**Notes:**
- The roadmap and backlog B08, B09 and B13 were updated by the tasks themselves; PROJECT.md's CLI list gained bare `quiver`, `show` and the guided `init`.
- Tagged `v1.2.0` on 2026-10-31. The release workflow's `verify-powershell` job ran `install.ps1` from the tag for the first time.
