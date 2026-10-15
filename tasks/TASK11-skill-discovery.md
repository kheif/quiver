# TASK11 — Skill discovery

**Phase:** Phase 1 — Remote Sources

**Status:** ✅ Done

**Depends on:** TASK05

`source.Discover(repoDir)` returns the skill folders in a repository, sorted by ID, plus warnings. Each result has an ID without the `<source>/` prefix, the folder relative to the repo root and the plugin that lists it.

1. **Marketplace mode.** If `.claude-plugin/marketplace.json` exists:
   1. Read `plugins[]`.
   2. Resolve each plugin's `source` relative to the repo root. `metadata.pluginRoot` is prepended to sources that do not start with `./`.
   3. If the plugin entry has `skills` (a path or a list), use those paths. A path may be a skill folder or a folder of skills.
   4. Otherwise use `<pluginDir>/skills/*/SKILL.md`.
   5. Plugins whose `source` points to another repository are skipped with a warning in v1.
   6. Paths that leave the repository, also through symlinks, are skipped with a warning. A folder listed twice is kept once.
   7. A broken manifest gives a warning and falls back to plain mode.
2. **Plain mode.** Otherwise, walk the repo for `SKILL.md` files:
   - Skip `.git`, `node_modules` and hidden folders.
   - Do not look inside a skill's subfolders once its `SKILL.md` is found.
   - A skill at the repo root is named after the repo folder, which is the source name.
3. **Collisions inside one source:** use `<plugin>/<name>` for every skill that shares the name and print a warning. Without a plugin, or when that is still not unique, use the folder path.
4. Verify the marketplace format against real repositories, starting with `anthropics/skills`. Save trimmed copies as fixtures.

**Notes:**
- Checked 2026-10-15: `anthropics/skills` gives 19 skills and `JuliusBrussee/caveman` 22, both without warnings. Their trimmed copies are in `internal/source/testdata`.
- Collision IDs can change: a new skill with the same name in a later commit renames an existing one, and its assignments break. See backlog B05; TASK15 must detect this.
- A plugin's own `.claude-plugin/plugin.json` can also list skills. It is ignored in v1; only the marketplace entry counts.
- A `SKILL.md` that is a symlink is not found, and the plain walk does not follow symlinked folders.

**Done when:** Fixture tests pass for a marketplace repo, a plain repo, a nested repo and a repo with collisions.
