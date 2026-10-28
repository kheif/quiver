# TASK37 — Pinning

**Phase:** Phase 5 — v2

**Status:** Todo

**Depends on:** TASK15

1. `quiver pin <source/skill> [<commit>]` pins the skill to the current HEAD, or to the given commit.
2. Check the pinned commit out into `sources/.pins/<source>@<commit>/` with `git worktree`, fetching more history if needed.
3. Store pins in `config.json` under `pins`. A pin applies everywhere the skill is assigned.
4. `update` leaves pinned skills alone. `quiver unpin` switches back to the live source.
5. `doctor` removes pin folders that are no longer used.

**Done when:** A pinned skill keeps its old content after `update`, and shows the new content after `unpin`.
