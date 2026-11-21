# TASK63.3: Ownership ledger

**Phase:** Phase 7c: Tool bundles (v1.7)

**Part of:** [TASK63](TASK63-tool-bundles.md): Tool bundles

**Status:** Todo

**Depends on:** TASK63.2

The record of what Quiver wrote for each bundle, so that removal is exact and a change by someone else is visible.

1. **File.** `~/.quiver/state/bundles.json`, machine-local and never synced like the rest of `state/`, with a format version (the rule of TASK62.2). For each installed bundle and scope (global, or a project by its project ID in `registry.json`, not by its path): the bundle's ID, source, version and commit, the time, the commands the user agreed to (the argv arrays of install, upgrade, uninstall and status, with the working directory and the first element resolved to an absolute path), and a list of entries. Removal runs the commands from the ledger, so it works when the source or its manifest is gone or has changed.
2. **Entry.** The file, the place inside it (a JSON key path, or the ID of a marked block), the kind of component, and a hash of exactly what Quiver wrote. A file Quiver created is marked so that removal may delete it; a file that existed before is never deleted.
3. **Operations.** `record` (after a write), `verify` (compare each entry to the file: intact, changed, missing), `reverse` (remove what is intact, report the rest). `reverse` never removes an entry whose hash differs and never touches a key or block that is not in the list.
4. **Safety.** The file is written atomically with a backup, like the other state files; a ledger from a newer Quiver is refused; a broken ledger stops every bundle command with a message and is never rewritten silently. Two Quiver processes at once cannot corrupt it (the same lock as `registry.json`).
5. `quiver doctor` reads it (TASK63.5).

**Done when:** tests cover record, verify and reverse for each component type, the commands read back from the ledger, an entry changed by hand, a file deleted by hand, a file that existed before, a newer ledger, a broken ledger, and two processes writing at once.
