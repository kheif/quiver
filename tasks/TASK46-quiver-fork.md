# TASK46 — `quiver fork`

**Phase:** Phase 5 — v2

**Status:** Todo

**Depends on:** TASK12, TASK14

1. `fork <source/skill> [--as <name>]` copies the skill into `local/<name>` and adds `forkedFrom: <id>@<commit>` to its frontmatter.
2. In the global list and in every project, replace the original ID with the local one.
3. Commit in `local/`.

**Done when:** After `fork`, editing the local copy changes what projects get, and `update` no longer affects them.
