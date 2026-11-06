# TASK58.2: Projects section

**Phase:** Phase 7: TUI and slash commands (v1.5)

**Part of:** [TASK58](TASK58-interactive-tui-quiver-with-no-arguments.md): Interactive panel (`quiver` with no arguments)

**Status:** ✅ Done

**Depends on:** TASK58.1

1. **List.** The name, the number of distinct skills assigned directly, and tags (`managing`, `here`, `missing`). `/` searches by name, ID and folder.
2. **Detail.** The name, the ID, every known folder, the skills assigned directly, and the skills that are available only through global assignments, labeled as such. The technical ID is shown here, not in the header.
3. **Actions.** *Manage skills* makes the project the target of the panel and opens Skills in *Project assignments*; it changes neither the working directory nor the registry. *Rename* and *Forget* run the same code as `quiver projects rename` and `forget`; forget explains its effect (the assignments go, the folder and `.quiver.json` stay) before it asks. *Split from a copy* is offered only for the project of the current folder and runs `quiver projects split`, which refuses a project that is not a copy.
4. A project without a folder on this machine can be read but not changed, and the screen says why.

**Done when:** tests cover choosing a project without a registry change, changes in a project other than the current one, a missing folder, a project without a folder, rename, forget and split.
