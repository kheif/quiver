# TASK33 — v1.1: import one skill, version for go install

**Phase:** Phase 4b — v1.1

**Status:** ✅ Done

**Depends on:** TASK32

Stabilization before v2, from using v1.0.0 (2026-10-28). Released as v1.1.0: a new feature, so a minor version.

1. `quiver import <skill>` imports one skill: its `SKILL.md`, its folder, or any markdown file with `name` and `description` frontmatter. No `--deep` needed. `--name` sets its name in the library and is written into the copy's frontmatter. A `SKILL.md` brings its folder (`scripts/`, `references/`); a lone markdown file is imported alone, without the other files of its folder. The skill is assigned to the project of the current folder, or to every folder.
2. `quiver --version` of a `go install` build shows the module version (`1.1.0`), not `dev`. A plain `go build` shows Go's VCS version; release builds keep the ldflags version.

**Not done (decided 2026-10-28):** prefix-free commands (TASK43) stay in v2. The user asked whether the assistant reliably follows a pointer file to `get_skill`, and chose to keep the MCP prompts as they are for now.

**Notes:**
- `importOne` in `internal/cli/import.go` handles one skill before the folder search; `library.ReadFound` reads a skill folder for both paths. A lone markdown file is copied to a temp folder as `SKILL.md` first. Deleting the original deletes the skill folder, or only the lone file.
- `import` now completes files as well as folders.
- `currentVersion` in `internal/cli/root.go` reads `debug.ReadBuildInfo` when no ldflags version is set.

**Done when:** tests cover a lone file with `--name`, a `SKILL.md` with its folder, and the errors; `go install github.com/kheif/quiver/cmd/quiver@v1.1.0` prints `quiver 1.1.0`.
