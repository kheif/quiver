# TASK14 — `quiver add` / `quiver remove`

**Phase:** Phase 1 — Remote Sources

**Depends on:** TASK07, TASK08

**`add <id>...`:**

1. Resolve every ID through the library. Fail on any unknown ID before changing anything.
2. Without `--global`: find the current project (or `--project <path>`). If it has no `.quiver.json`, register it (TASK06) and print that the file was created and should be committed. Then call `EnsureProject` and `Assign`.
3. With `--global`: call `AssignGlobal`.
4. If an added skill has the same `name` as another skill served in that project, warn about the clash.
5. Print what the project serves now, as `status` does.

**`remove <id>...`:**

1. Same project and `--global` resolution as `add`.
2. Unassign only. The skill stays in the library.
3. Removing an ID that is not assigned prints a note, not an error.

**Done when:** Tests cover project and global assignment, `--project`, unknown IDs and name clashes.
