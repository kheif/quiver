# TASK15 — `quiver update`

**Phase:** Phase 1 — Remote Sources

**Status:** ✅ Done

**Depends on:** TASK13

1. `update [<name>...]` runs `git pull --ff-only` for the given sources, or all of them.
2. Record HEAD before and after. Print the skills that were added, changed or removed.
3. If an assigned skill disappeared upstream, warn and list the projects using it. Do not unassign it automatically; `doctor` reports it.
4. If a source cannot be fast-forwarded, report it and continue with the other sources.
5. Running MCP servers pick up the change through live updates (TASK26).

**Notes:**
- Sources are shallow clones. After an upstream force-push, `--ff-only` fails. `update` reports it with a hint, and `update <name> --reset` runs `fetch --depth 1` and `reset --hard @{u}` instead of a re-clone. The old HEAD stays in the object store, so `DiffNames` still works right after the reset.
- The discovery index lives in `.git/quiver-index.json` (TASK12), outside the working tree, so it never blocks a pull. A source without its own `.git` folder (for example a submodule) gets no index and is scanned on every load.
- The test checks the new content with `library.Get`, because `get_skill` does not exist yet. TASK18 repeats the check through `get_skill`.
- `update` takes no lock while it pulls, so a `source remove` at the same time can move the folder away mid-pull.
- `--reset` drops local edits in a source folder. Quiver never edits sources (`CheckEditable`), but hand edits are lost.
- Only a diverged history gets the `--reset` hint. A pull that fails because of local edits or the network shows git's error only.
- A skill at the repo root (folder `.`) counts as changed on every new commit.
- A collision rename (B05) shows as one skill removed and one added, with the warning if the old ID is assigned. Nothing moves the assignment; B05 is still open.

**Done when:** A test pushes a change to a fixture bare repo, runs `update`, and `get_skill` returns the new content.
