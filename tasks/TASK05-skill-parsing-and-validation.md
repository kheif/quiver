# TASK05 — Skill parsing and validation

**Phase:** Phase 0 — Foundation

**Status:** ✅ Done

**Depends on:** TASK01

1. `skill.Load(dir)` reads `dir/SKILL.md` and parses its YAML frontmatter.
2. The `Skill` struct holds `Name`, `Description`, `Dir`, `Source`, `ID`, `Body` (markdown after the frontmatter) and the raw frontmatter as a map, so unknown fields are kept.
3. `skill.Files(s)` lists the skill's extra files as relative paths. It skips `.git` and hidden files.
4. Validation:
   - `SKILL.md` exists.
   - `name` and `description` are present.
   - `name` matches `^[a-z0-9][a-z0-9-]*$`.
   - Soft issues are returned as warnings, for example a folder name that differs from `name`.
5. Errors include the file path.

**Done when:** Tests cover valid skills, missing frontmatter, missing fields, invalid names and unknown fields (which are kept).
