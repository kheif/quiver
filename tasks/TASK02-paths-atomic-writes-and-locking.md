# TASK02 — Paths, atomic writes and locking

**Phase:** Phase 0 — Foundation

**Status:** Todo

**Depends on:** TASK01

1. `fsutil.QuiverHome()` returns `$QUIVER_HOME` if it is set, otherwise `~/.quiver`.
2. `fsutil.ExpandHome(path)` expands a leading `~`.
3. `fsutil.WriteJSONAtomic(path, v)` writes to a temp file in the same folder, calls `fsync`, then renames it over `path`.
4. `fsutil.ReadJSON(path, v)` returns a typed `ErrNotExist` when the file is missing.
5. `fsutil.WithLock(fn)` runs `fn` while holding `~/.quiver/.lock`. Use `github.com/gofrs/flock`. Every write to `~/.quiver` goes through it.

**Done when:**
- A test proves an interrupted write leaves the old file intact.
- A test proves two concurrent writers do not corrupt the file.
