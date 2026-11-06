# TASK58.3: Sources section

**Phase:** Phase 7: TUI and slash commands (v1.5)

**Part of:** [TASK58](TASK58-interactive-tui-quiver-with-no-arguments.md): Interactive panel (`quiver` with no arguments)

**Status:** ✅ Done

**Depends on:** TASK58.1

Sources belong to the central library; only skill assignments have a scope. The panel therefore never shows a "project source": the views are derived from the real assignments.

1. **Views.** *All sources*, *Used by project* (a skill of the source is assigned to the managed project directly) and *Used globally*. A source used in both scopes is in both views. Local skills are not a source; a library without remote sources says so and counts its local skills.
2. **Detail.** The repository, the status, the commit and its date, the pinned skills, and its skills with the scopes they are used in.
3. **Status.** *Not checked* until a check ran in this session; then *Up to date*, *Changes available* or *Check failed*. The panel never says "up to date" without a check.
4. **Check and update.** `u` fetches the source in the background (the panel stays usable) and runs the same preview as `quiver update`. A change that touches no assigned skill applies at once, as in the CLI; a change to an assigned skill shows the preview (changed files, scripts marked for review) and waits for `y`. `d` in the preview shows the patch of each assigned skill that changes, like `quiver update --diff`, and `d` again goes back to the summary. `U` checks every source and applies nothing. A source whose history was rewritten asks whether to take the remote history, like `quiver update <name> --reset`, and then shows the preview as usual.
5. **Remove.** `x` lists the assignments that go with the source before it asks. A long list is read in the body, one skill per line, and scrolls.
6. **Add.** `a` asks for a repository and a name and clones it like `quiver source add`. A repository without skills is not kept. The prompt warns that skills can include scripts.

**Done when:** tests cover the views, a source without assignments, the status before and after a check, an update that applies at once, one that waits for a yes, a failing remote, a repeated key, and remove.
