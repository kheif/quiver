# TASK28 — `quiver import`

**Phase:** Phase 3 — Local Skills and Management

**Status:** ✅ Done

**Depends on:** TASK23, TASK14

Brings existing skills that are scattered across projects into the library.

1. `import [<dir>]` looks for skill folders under `<dir>` (default: the current project):
   - `.claude/skills/*/SKILL.md`
   - any other `SKILL.md`, if `--deep` is given
2. Skip `node_modules`, `.git` and `~/.quiver`.
3. Show what was found and ask per skill:
   - scope: this project or global
   - name, if it collides with an existing library skill
4. For each confirmed skill:
   1. If a library skill with the same name has identical content, assign it and skip the copy.
   2. Otherwise copy it into `local/<name>` and assign it.
   3. Commit in `local/` with the message `import <name> from <project>`.
5. Ask whether to delete the original folder. Never delete it without a clear yes. With `--yes`, the originals are kept.

**Implementation notes:**
- `library.FindSkills` finds the folders, `library.ImportLocal` copies one skill under the lock: hidden temp folder, validate, rename, assign, commit. Links and `.git` folders are not copied (a warning names each skipped link). The source folder is never changed.
- A new name is written into the `name:` line of the copied `SKILL.md`.
- "Identical content" means the same files with the same bytes, `.git` ignored, under the same name.
- A found skill that cannot be loaded is listed as skipped. A taken or invalid name is asked again until it is free; an empty answer skips the skill.
- `--yes` asks nothing: scope is the project (or every folder with `--global`), taken names are skipped, originals are kept. Without `--yes`, `--global` only changes the default answer.
- The project is registered only when the first skill is assigned to it.

**Done when:** An end-to-end test imports two skills from a fixture project, with one name collision. Nothing is overwritten, and the originals stay unless deletion was confirmed.
