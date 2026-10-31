# TASK48 — Update preview

**Phase:** Phase 5 — v2

**Status:** ✅ Done

**Depends on:** TASK15

1. `update` fetches first, without merging.
2. Show a per-skill change summary for every assigned skill that changed. `--diff` shows the full diff.
3. Flag changes to scripts (`*.sh`, `*.py`, `*.js`, files with the executable bit) as **needs review**.
4. Ask for confirmation, then merge. `--yes` skips the question.

**Done when:** A test where a fixture adds a script to a skill shows the "needs review" flag and requires confirmation.
