# TASK14 — `quiver add` / `quiver remove`

**Phase:** Phase 1 — Remote Sources

**Status:** ✅ Done

**Depends on:** TASK07, TASK08

**`add <id>...`:**

1. Resolve every ID through the library. Fail on any unknown ID before changing anything.
2. Without `--global`: find the current project (or `--project <path>`). If it has no `.quiver.json`, register it (TASK06) and print that the file was created and should be committed. Then call `EnsureProject` and `Assign`.
   - `project.Register` takes no lock. Two processes registering the same new folder at once can each write an ID; the last write wins and the other caller holds a stale ID. Run `Register`, `EnsureProject` and `Assign` inside one `registry.Update` call.
3. With `--global`: call `AssignGlobal`.
4. If an added skill has the same `name` as another skill served in that project, warn about the clash.
5. Print what the project serves now, as `status` does.

**`remove <id>...`:**

1. Same project and `--global` resolution as `add`.
2. Unassign only. The skill stays in the library.
3. Removing an ID that is not assigned prints a note, not an error.

**Notes:**
- `add --global` checks name clashes only among the global skills. A global skill can still clash with a skill of some project; `doctor` (TASK29) reports those.
- If the registry write fails after `.quiver.json` was created, the file stays without a registry entry. The next `add` reuses its ID, so nothing breaks.
- `remove` does not look IDs up in the library, so a skill deleted from its source can still be unassigned. The cost: a typo only prints a note and exits 0.
- `remove` does not call `EnsureProject`, so a moved project keeps its old path in the registry until the next `add` or until the server resolves it (TASK17).

**Done when:** Tests cover project and global assignment, `--project`, unknown IDs and name clashes.
