# TASK13 — `quiver source add | list | remove`

**Phase:** Phase 1 — Remote Sources

**Depends on:** TASK09, TASK10, TASK12

**`add <repo> [--name <n>]`:**

1. Parse the input (TASK10) and check that the name is free.
2. Clone into `sources/<name>`.
3. Run discovery. If no skills are found, ask whether to keep the source anyway.
4. Save it to the config and print the skills that were found.
5. Print a reminder that skills from this source give instructions to the assistant and may contain scripts.

**`list`:** Print each source's name, URL, skill count, current commit and last update time.

**`remove <name>`:**

1. If any skill from the source is assigned, refuse and list where (using `where`).
2. `--force` unassigns those skills first.
3. Delete `sources/<name>` and remove it from the config.

**Done when:** An end-to-end test adds a source from a local bare repo, assigns one of its skills, gets refused on remove, then succeeds with `--force`.
