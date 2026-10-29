# TASK33 — v1.1: import one skill or a folder, version for go install

**Phase:** Phase 4b — v1.1

**Status:** ✅ Done

**Depends on:** TASK32

Stabilization before v2, from using v1.0.0 (2026-10-28). Released as v1.1.0: a new feature, so a minor version.

1. `quiver import <skill>` imports one skill: its `SKILL.md`, its folder, or any markdown file with skill frontmatter. `--name` sets its name in the library and is written into the copy's frontmatter; without it, the frontmatter `name` is used, then the folder or file name. A `SKILL.md` brings its folder (`scripts/`, `references/`); a lone markdown file is imported alone, without the other files of its folder.
1b. `quiver import <folder>` imports every skill in the folder and its subfolders, without `--deep`. A skill is a folder with a `SKILL.md`, or a markdown file of any name that starts with frontmatter (the content counts, not the file name). Markdown files without frontmatter, such as READMEs, are not skills and are not mentioned. Files that look like skills but have no description are skipped and listed at the end of the output. A skill folder is not searched further: its markdown files belong to it.
1c. Imported skills are assigned to the project of the current folder (or to every folder), also when a folder elsewhere is imported. `quiver import` without a path keeps searching this project's `.claude/skills`, or every folder of it with `--deep`.
2. `quiver --version` of a `go install` build shows the module version (`1.1.0`), not `dev`. A plain `go build` shows Go's VCS version; release builds keep the ldflags version.

**Not done (decided 2026-10-28):** prefix-free commands (TASK53) stay in v2. It is not measured yet whether the assistant reliably follows a pointer file to `get_skill`, so the MCP prompts stay as they are for now.

**Notes:**
- `importOne` in `internal/cli/import.go` handles one skill before the folder search; `library.ReadFound` reads a skill folder for both paths. A lone markdown file is copied to a temp folder as `SKILL.md` first. Deleting the original deletes the skill folder, or only the lone file.
- `import` now completes files as well as folders.
- `currentVersion` in `internal/cli/root.go` reads `debug.ReadBuildInfo` when no ldflags version is set.

- `library.FindSkills` returns `Found` with `File` set for a lone markdown file; `Found.Stage` copies it into a temp folder as `SKILL.md` for `ImportLocal`. `skill.LoadFile` and `skill.HasFrontmatter` read such files. `Found.Err` is set for an unreadable skill or one without a description; a missing or invalid name is not an error, because the import asks for one.

**Released 2026-10-29 as v1.1.0:** the first release published to npm by CI with trusted publishing (with provenance). npm took about 6 minutes to show every package, longer than the install check waited, so it now waits up to 15 minutes.

**Done when:** tests cover a lone file with `--name` and with its frontmatter name, a `SKILL.md` with its folder, a folder with nested skills, lone files, a README and invalid skills, and the errors; `go install github.com/kheif/quiver/cmd/quiver@v1.1.0` prints `quiver 1.1.0`.
