# TASK06 — Project identification (`.quiver.json`)

**Phase:** Phase 0 — Foundation

**Depends on:** TASK02

A project is identified by the `id` in its `.quiver.json` (see [data-model.md](data-model.md)).

1. `project.Find(dir)` walks up from `dir` to the first folder containing `.quiver.json`. It stops at `$HOME` and the file system root. It returns `{ID, Root}`, or `ErrNotRegistered`.
2. `project.Register(dir)`:
   - The root is the nearest folder containing `.git`, or `dir` itself if there is none.
   - Generate the ID as `<folder-name>-<6 random base32 chars>`, for example `app-k3f9q2`.
   - Write `{"id": "..."}` to `<root>/.quiver.json`.
   - If the file already exists, return its ID instead.
3. Read the file strictly. A missing or invalid `id` is an error that names the file.
4. The file holds only the ID. Unknown keys are kept when the file is rewritten.

**Done when:** Tests cover a nested subfolder, a missing file, an invalid file, registering twice (same ID) and registering a non-git folder.
