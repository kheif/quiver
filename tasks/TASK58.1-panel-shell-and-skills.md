# TASK58.1: Panel shell and Skills section

**Phase:** Phase 7: TUI and slash commands (v1.5)

**Part of:** [TASK58](TASK58-interactive-tui-quiver-with-no-arguments.md): Interactive panel (`quiver` with no arguments)

**Status:** ✅ Done

**Depends on:** TASK45, TASK14

The frame of the panel and its main section.

1. **Frame.** A header with the project being managed (or "no project"), the sections with the current one marked in a way that does not depend on color, the views with their result counts, the search line, the two panes, a status line and the key line. It fits any size from 40×12; below that it says "terminal too small" and how to quit. Widths are measured in terminal cells, so wide characters and long names do not break it, and a resize keeps the selection and the focus.
2. **Views.** *Available here* (the union of the project's and the global assignments, each skill once), *Project assignments*, *Global assignments* and *All skills*. `←` `→` switch; the query stays. The active view and the number of results are always on screen.
3. **Search.** `/` searches the active view with the ranking of TASK45 (words in any order, space and hyphen alike). Typing is text, never a command. With no result the screen says "No matches in <view>" and offers the same query in all skills (`a`, or `ctrl+a` while typing). An empty library, an empty view and an empty result are different messages.
4. **Rows.** The skill ID and where it is assigned in words (*Project*, *Global*, *Project + global*, *Other projects* when only other projects use it, *Unassigned*, with *always on* when there is room). No codes to decode. Without a search the list is grouped by where the skills come from (your own first, then each source), each group under a header with its count and its rows indented; a search shows one list in the order of the best match. A skill that is always on stands out, one that is used nowhere here is dimmed. The detail names the other projects that use the skill, so a skill is never called unused because it is not used here.
5. **Detail.** The ID, the source, whether it can be edited, the pin, the two assignment settings, the description once and the start of the instructions. `r` reads the whole `SKILL.md`, scrollable.
6. **Assignments.** `enter` (or `space`) on a skill focuses its assignment rows; `enter` on a row opens a choice of the three modes. The change goes through `add` and `remove`: *On demand → Always on* keeps the assignment and sets the mark, *Always on → On demand* removes only the mark. A change for every folder asks first. The message names the skill and what changed, for example "Removed from api. Still available globally."
7. **Edit, fork and delete.** `e` edits a local skill and returns to the same row. A skill from a source is read-only; `f` forks it into a local copy and asks for its name (the name is checked at once). `d` deletes a local skill like `quiver delete --force`: the question lists where the skill is assigned, since the deletion unassigns it there, and says that the removal is committed in the Quiver home. A source skill cannot be deleted on its own; the message points to removing the source.
8. **Pin.** `p` keeps a source skill at its current commit like `quiver pin`, or, for a pinned skill, serves it from the source again like `quiver unpin`. Both ask first.
9. **New and import.** `n` asks for a name (checked at once: a bad name or one that is taken is refused before anything else is asked), a description and where the skill goes: the managed project as on demand (the default), every folder, or nowhere yet. Without a description it opens the editor. `i` asks for a folder or file (empty: the `.claude/skills` of the folder), checks that it exists, and, after a confirmation, imports like `quiver import --yes`. The message says how many skills were imported, not the list of every file.
10. Outside a project the project row says what to do and the global scope stays usable.
11. **Always-on honesty.** When an assistant has no session-start hook, a change to *Always on* says so and points to Health instead of promising "from the next session".

**Done when:** tests cover the frame at several sizes, the four views and the query across them, the search keys, the empty states, every key of the list and the detail, the independence of the two scopes (including the "always on in one project only" case), and the selection after a row leaves the view.
