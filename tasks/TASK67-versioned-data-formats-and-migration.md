# TASK67: Versioned data formats and migration

**Phase:** Phase 8: Test phase

**Status:** Todo

**Depends on:** TASK56, TASK63.8

Facts: `registry.json` is at version 2, or 3 once it holds a Searchable skill (TASK62.2), and `registry.Load` upgrades version 1 (`internal/registry/registry.go`); the state file is at version 1 and a newer one is refused; `config.json` is at version 1; `.quiver.json` holds only `id`; the hook's block (TASK53) has no version.

1. TASK62.2 sets the rule for every file that is new from v1.6; this task brings the files that exist already under it. Every file Quiver writes in the Quiver home has a version, and a file from a newer Quiver is refused with a message that names the version needed, as the state file does. A file under `cache/` is derived data: a format from another version is ignored and rebuilt, not refused.
2. `.quiver.json` is committed in users' repositories, so it gets a version only when its format changes; a file without one is version 1.
3. **Golden homes.** The layout, the script and the test come from TASK62.2, which made the home of v1.5.0. This task adds the homes of v1.0.0, v1.1.0, v1.2.0, v1.3.0 and v1.4.0 and of every release since that no other task made (v1.5.0, v1.6.0 and v1.7.0 are made by TASK62.2, TASK62.10 and TASK63.8), each made by running that release's binary through the same script, and the same test upgrades each one.
4. Describe each format version in data-model.md.
5. Add "add the release's golden home" to docs/releasing.md.
6. **Surface.** The list of MCP tools (names and input schemas) and the list of commands and flags of each release are kept under `testdata/surface/<version>/`, and a test fails when a later version removes or renames one without the hidden deprecated alias that the versioning rule of docs/releasing.md requires.

**Done when:** every golden home upgrades in the tests, and every file from a newer version is refused clearly.
