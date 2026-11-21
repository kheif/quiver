# TASK66.3: Journey: two machines

**Phase:** Phase 8: Test phase

**Part of:** [TASK66](TASK66-end-to-end-user-journeys.md): End-to-end user journeys

**Status:** Todo

**Depends on:** TASK56, TASK30, TASK63.8, TASK65.1, TASK65.2

Two homes (machine A and B) and one bare remote:
1. `quiver sync init <remote>` on both.
2. A adds a local skill and marks it always-on; B, after `quiver sync`, has the skill, and B's hook prints it.
3. A and B change assignments of the same project apart from each other; after both sync, both changes are kept (the three-way merge of `internal/jsonmerge`).
4. A adds a source; B's next `sync` clones it.
5. A removes a skill B still has assigned: B's `doctor` reports it.
6. A makes a skill Searchable; after `quiver sync` B has it Searchable. A skill that A made Searchable and B made On demand ends On demand on both (TASK62.2).

**Done when:** the journey passes on all three systems.
