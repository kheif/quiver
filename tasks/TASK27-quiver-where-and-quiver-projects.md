# TASK27 — `quiver where` and `quiver projects`

**Phase:** Phase 3 — Local Skills and Management

**Status:** ✅ Done

**Depends on:** TASK07

**`where <id>`:** Print `global` if applicable, then each project that has the skill, with its name and paths.

**Notes:**
- `source remove` (TASK13) has its own lookup, `library.usage`, by source prefix. Share the output format with it.

**`projects`:**

1. `projects` lists every project: name, ID, skill count and paths. Paths that no longer exist on this machine are marked `(missing)`.
2. `projects rename <id|name> <new-name>`
3. `projects forget <id|name>`: removes the project and its assignments. Asks for confirmation. Does not touch the `.quiver.json` file.
4. `projects split` (run inside a copied folder): gives this folder a new ID, copies the assignments of the old ID to it, and rewrites its `.quiver.json`.

**Implementation notes:**
- Both formats use `registry.ProjectLabel`: `<name> (<paths>)`. `where` prints `global`, then `project <label>` per project; the `source remove` error prints `project <label>: <skills>`.
- `registry.Lookup` resolves `<id|name>`: an ID wins over a name, and a name used by several projects is an error that lists their IDs.
- `forget` takes `--yes` for scripts. A "no" answer is not an error, like `source add`.
- `split` refuses when no other folder still holds the ID (`ErrNotCopied`). The new project is named after its folder and copies the skills and always-on marks. `.quiver.json` is written before the registry; if the registry write fails, the old ID is put back.

**Done when:** Each subcommand has a test. After `split`, changing one copy's assignments does not affect the other.
