# TASK63.3: Journey: two machines

**Phase:** Phase 8: Test phase

**Part of:** [TASK63](TASK63-end-to-end-user-journeys.md): End-to-end user journeys

**Status:** Todo

**Depends on:** TASK56, TASK30

Two homes (machine A and B) and one bare remote:
1. `quiver sync init <remote>` on both.
2. A adds a local skill and marks it always-on; B, after `quiver sync`, has the skill, and B's hook prints it.
3. A and B change assignments of the same project apart from each other; after both sync, both changes are kept (the three-way merge of `internal/jsonmerge`).
4. A adds a source; B's next `sync` clones it.
5. A removes a skill B still has assigned: B's `doctor` reports it.

**Done when:** the journey passes on all three systems.
