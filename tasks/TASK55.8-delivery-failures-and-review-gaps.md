# TASK55.8: Delivery failures and review gaps

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK55](TASK55-quiver-install.md): `quiver install`

**Status:** ✅ Done

**Depends on:** TASK55.7

A review of the hook delivery with real binaries, in isolated homes, found places where a session could start without its skills and nothing said so, where a change reached the model unreviewed, and where the cost grew with the size of the library instead of the size of the text. This task closes them.

1. **A hook never leaves a session silent.** A skill that cannot be read is left out and named, and the others and the catalog still arrive. Anything the hook could not do (a corrupt registry, a `.quiver.json` it cannot read, a skill that is assigned but missing, a project ID shared with another folder) goes into a `<quiver-notice>` block in part 1, so the assistant tells the user. The hook reads stdin for at most 2 seconds, so an assistant that never closes it cannot hold a session start. It does not create the Quiver home. `hook.log` keeps one earlier file, is not readable by others, and records a short hash of each part.
2. **Parts of one session start say the same.** Part 1 builds the text and leaves its parts in `~/.quiver/cache/hook`; the later parts take theirs from there, and build their own only when nothing is there. A skill edited between two parts no longer gives them different texts, and 231 parts of a 2 MB text arrive in 3.5 s instead of most of them hitting the hook's 10 s limit. The wait before part N is 50 ms for each of the first 20 parts, 10 ms after that, and at most 3 s. A hook reads only the skills it passes on (`library.LoadFor`). `PartsNeeded` builds each skill's text, size and catalog line once and only counts parts: `quiver add` at 2,000 skills, 500 projects and 100 always-on skills takes 1 s instead of 21 s.
3. **Text that reaches the model is clean.** Invalid UTF-8 is replaced before the text is measured, so a part never exceeds its limit. Control characters, zero-width characters, direction overrides and Unicode tag characters are dropped, and a skill, description or ID cannot write Quiver's own tags (`<quiver-always-on>`, `<quiver-skills>`, `<quiver-notice>`): they read as `&lt;quiver-...`. A `.quiver.json` must be a regular file of at most 64 KB.
4. **Reviews cannot be skipped.** A source skill folder that is a symlink is not discovered, so an update cannot change a skill through files that git reports somewhere else. `quiver sync` and `quiver sync init` list changes that make a skill always-on, change the files of an always-on skill, or add or change a source, and ask before they pull (`--yes` takes them; without a terminal and without `--yes`, nothing is pulled).
5. **The assistant cannot rewrite what every session starts with.** `update_skill` changes only skills served in the session's project, and not always-on skills, which the user changes with `quiver edit`.
6. **Install and doctor tell the truth.** The delivery report and `quiver status` say "hook" only when the assistant's config holds Quiver's hook. VS Code's `mcp.json`, which allows comments, is read with them and written without them, after the old file is saved next to it. `quiver doctor` checks Cursor's hook. A hook command with a shell suffix such as `2>/dev/null || true` is still Quiver's, so install does not add a second one.

**Not done:** Windows quoting of the hook command for `cmd.exe` and PowerShell (TASK64.2); `get_skill` and `search_skills` still read the whole library, as documented.

**Done when:** unit tests cover each point above, including the unreadable skill, the forged tags, the hidden characters, invalid UTF-8, the symlinked skill folder, the sync confirmation, the `update_skill` refusals, commented `mcp.json`, wrapped hook commands, and parts sharing one snapshot; and all 231 parts of a 2 MB text, started at once, were delivered in a real run.
