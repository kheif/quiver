# TASK11 — Skill discovery

**Phase:** Phase 1 — Remote Sources

**Status:** Todo

**Depends on:** TASK05

`discover.Find(repoDir)` returns the skill folders in a repository.

1. **Marketplace mode.** If `.claude-plugin/marketplace.json` exists:
   1. Read `plugins[]`.
   2. Resolve each plugin's `source` relative to the repo root, taking `metadata.pluginRoot` into account if present.
   3. If the plugin entry has a `skills` array, use those paths.
   4. Otherwise use `<pluginDir>/skills/*/SKILL.md`.
   5. Plugins whose `source` points to another repository are skipped with a warning in v1.
2. **Plain mode.** Otherwise, walk the repo for `SKILL.md` files:
   - Skip `.git`, `node_modules` and hidden folders.
   - Do not look inside a skill's subfolders once its `SKILL.md` is found.
3. **Collisions inside one source:** use `<source>/<plugin>/<name>` for both skills and print a warning.
4. Verify the marketplace format against real repositories, starting with `anthropics/skills`. Save trimmed copies as fixtures.

**Done when:** Fixture tests pass for a marketplace repo, a plain repo, a nested repo and a repo with collisions.
