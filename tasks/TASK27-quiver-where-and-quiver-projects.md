# TASK27 — `quiver where` and `quiver projects`

**Phase:** Phase 3 — Local Skills and Management

**Status:** Todo

**Depends on:** TASK07

**`where <id>`:** Print `global` if applicable, then each project that has the skill, with its name and paths.

**`projects`:**

1. `projects` lists every project: name, ID, skill count and paths. Paths that no longer exist on this machine are marked `(missing)`.
2. `projects rename <id|name> <new-name>`
3. `projects forget <id|name>`: removes the project and its assignments. Asks for confirmation. Does not touch the `.quiver.json` file.
4. `projects split` (run inside a copied folder): gives this folder a new ID, copies the assignments of the old ID to it, and rewrites its `.quiver.json`.

**Done when:** Each subcommand has a test. After `split`, changing one copy's assignments does not affect the other.
