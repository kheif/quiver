# TASK07 — Registry and assignments

**Phase:** Phase 0 — Foundation

**Status:** Todo

**Depends on:** TASK02, TASK06

1. Define Go structs for `registry.json` (see [data-model.md](data-model.md)).
2. Provide these functions:
   - `Assign(projectID, ids...)`, `Unassign(projectID, ids...)`
   - `AssignGlobal(ids...)`, `UnassignGlobal(ids...)`
   - `EnsureProject(id, root)`: adds the project, or records a new path for it
   - `Served(projectID)`: the global skills plus the project's skills, without duplicates
   - `ProjectsUsing(skillID)`
   - `RenameProject`, `ForgetProject`
3. **Moved and copied folders.** When `EnsureProject` sees a known ID at a new path:
   - If the old path no longer has a `.quiver.json` with the same ID, the folder was moved. Replace the path.
   - If it still has one, the folder was probably copied. Keep both paths and return a `PossibleCopy` warning.
4. Keep every list sorted and free of duplicates.
5. Every change runs under the lock.

**Done when:** Tests cover every function, a moved folder, a copied folder, concurrent writers and `Served` with overlapping lists.
