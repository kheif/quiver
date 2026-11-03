# TASK65: Versioned data formats and migration

**Phase:** Phase 8: Test phase

**Status:** Todo

**Depends on:** TASK56

Facts: `registry.json` is at version 2 and `registry.Load` upgrades version 1 (`internal/registry/registry.go`); the state file is at version 1 and a newer one is refused; `config.json` is at version 1; `.quiver.json` holds only `id`; the hook's block (TASK53) has no version.

1. Every file Quiver writes in the Quiver home has a version, and a file from a newer Quiver is refused with a message that names the version needed, as the state file does.
2. `.quiver.json` is committed in users' repositories, so it gets a version only when its format changes; a file without one is version 1.
3. **Golden homes.** Keep a Quiver home as written by v1.0.0, v1.1.0, v1.2.0 and every later release under `testdata/formats/<version>/`, made by running that release's binary through the same script of commands (init, source add, add, add --always, a project). A test copies each, runs the current code on it, and checks the upgraded files and a clean `doctor`.
4. Describe each format version in data-model.md.
5. Add "add the release's golden home" to docs/releasing.md.

**Done when:** every golden home upgrades in the tests, and every file from a newer version is refused clearly.
