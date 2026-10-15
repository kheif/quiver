# TASK03 — Config

**Phase:** Phase 0 — Foundation

**Depends on:** TASK02

1. Define Go structs for `config.json` (see [data-model.md](data-model.md)).
2. `config.Load()` returns defaults if the file is missing.
3. `config.Save()` writes atomically under the lock.
4. Validation:
   - Source names must match `^[a-z0-9][a-z0-9-]*$`.
   - `local` is a reserved name.
5. If the `version` field is newer than the binary supports, exit with a clear error.

**Done when:** Load, save and validation are covered by tests, including an invalid source name and an unknown version.
