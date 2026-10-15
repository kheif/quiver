# TASK28 — `quiver import`

**Phase:** Phase 3 — Local Skills and Management

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

**Done when:** An end-to-end test imports two skills from a fixture project, with one name collision. Nothing is overwritten, and the originals stay unless deletion was confirmed.
