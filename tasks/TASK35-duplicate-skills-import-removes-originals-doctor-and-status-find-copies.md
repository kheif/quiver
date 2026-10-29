# TASK35 — Duplicate skills: import removes originals, doctor and status find copies

**Phase:** Phase 4c — v1.2

**Status:** Todo

**Depends on:** TASK33

A skill that is both in Quiver and in a native skill folder is loaded twice (backlog B13). Not everyone runs `quiver doctor`, so import should not leave copies behind.

1. **Import removes originals.** After a skill is imported, its original is deleted by default; the copy is safe in the library's git.
   - Exception: an original tracked by the git of a project (a committed `.claude/skills/x`) is shared with teammates who may not use Quiver. For it the question defaults to no and says why, and `--yes` keeps it.
   - `--keep` keeps every original. `--yes` deletes the untracked ones.
   - Use `git.Tracked` to tell the cases apart.
2. **doctor.** A new check finds skills that Quiver serves here and that also exist natively: in `~/.claude/skills`, in this project's `.claude/skills`, and in enabled Claude Code plugins. Match by name, and by same content. `--fix` offers to delete untracked copies; tracked ones are reported only.
3. **status.** `quiver status` prints a warning line for each native copy of a served skill, with the hint `quiver doctor --fix`.

**Done when:** tests cover import of tracked and untracked originals with and without `--yes` and `--keep`, the doctor check and fix, and the status warning; the README and import help describe the new default.
