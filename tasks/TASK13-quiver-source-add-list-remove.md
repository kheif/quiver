# TASK13 — `quiver source add | list | remove`

**Phase:** Phase 1 — Remote Sources

**Status:** ✅ Done

**Depends on:** TASK09, TASK10, TASK12

**`add <repo> [--name <n>]`:**

1. Parse the input (TASK10) and check that the name is free, before the slow clone. Validate a `--name` value with `config.ValidateSourceName`.
2. `library.StageSource` clones into `sources/.add-<name>-*/<name>` and runs discovery.
3. If no skills are found, ask whether to keep the source anyway. No answer means no.
4. `Staged.Keep` moves the clone to `sources/<name>` and saves the config under one lock. Print the skills that were found.
5. Print a reminder that skills from this source give instructions to the assistant and may contain scripts.

**`list`:** Print each source's name, skill count, short commit, commit date and URL. A missing folder shows as `missing`.

**`remove <name>`:**

1. `library.RemoveSource` finds assigned IDs that start with `<name>/`, global and per project. If there are any, it refuses and lists where.
2. `--force` unassigns those skills first, always-on marks included.
3. Move `sources/<name>` aside, write the registry and the config, then delete the folder. A failed write puts the folder back.

**Notes:**
- The default name is the repository owner, so two repos from one owner (`anthropics/skills`, `anthropics/other`) collide. When the name is taken, suggest `--name` in the error.
- An interrupted add or remove leaves a `sources/.add-*` or `sources/.remove-*` folder. TASK29 should report and delete them.
- `git.Head` walks up to a parent repository. If `~/.quiver` is inside a git repo (dotfiles) and a source folder is not a repo, `list` and the index cache read the parent's HEAD.
- `list` shows the HEAD commit date, not the time of the last pull.
- If the config write fails after the registry write in `remove --force`, the skills stay unassigned while the source remains.

**Done when:** An end-to-end test adds a source from a local bare repo, assigns one of its skills, gets refused on remove, then succeeds with `--force`.
