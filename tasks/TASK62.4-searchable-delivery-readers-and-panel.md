# TASK62.4: Searchable skills in delivery, readers and the panel

**Phase:** Phase 7b: Skill discovery (v1.6)

**Part of:** [TASK62](TASK62-finding-unassigned-skills.md): Finding unassigned skills

**Status:** ✅ Done

**Depends on:** TASK62.2, TASK62.3

The rest of the Searchable level (see TASK62.2 and TASK62.3): what the MCP server serves, every place that reads assignments, the panel, and the documentation.

1. **Delivery.** A Searchable skill is not in the session-start catalog and not always-on. The MCP server serves it through `search_skills` and `get_skill` (TASK62.5, TASK62.8); `get_skill` loads it without the "not assigned" note. `list_skills` keeps answering what is listed in the folder, and its description says "every listed skill". Searchable skills are not MCP prompts. `update_skill` changes only skills that are served in the folder, as today (`checkChangeable`): the assistant cannot rewrite a Searchable skill, which it would then load without asking; its error for a Searchable skill says to make the skill listed with `quiver add <id>` first, and the website's security page says so. `create_skill` and the skills it makes are unchanged (TASK62.8, item 7).
2. **Other readers of assignments.** Every place that reads the registry lists counts Searchable skills as assigned: `quiver source remove` and the panel's `x` (they refuse while a skill of the source is assigned, or list the assignments that go with it), `quiver delete` of a local skill, `quiver update` (the preview and what it waits for, `assignedWhere` in `internal/cli/update.go`), `quiver fork` (it replaces the original in every list), `quiver doctor` (assigned but missing, name clashes), `quiver projects` (list, show and forget, which count a project's skills), the bare `quiver` summary and its "Next:" hint, shell completion, the panel's loading and detail, and the review that `quiver sync` shows before applying changes from another machine (`internal/homesync/review.go`), which lists Searchable skills that arrive and changes to their files, as it does for always-on skills, because the assistant loads them without asking.
3. **Panel.** The scope setting of a skill in the Skills section (TASK58.1) has four states, in this order: *Not assigned*, *Searchable*, *On demand*, *Always on*, with the same safeguards as today (a change for every folder asks first). A Searchable skill appears in the view of what is available here (marked), in the views of the project's and of the global assignments, and in the whole library; the Sources views of TASK58.3 count it as used. Its separate `AddSource` path asks the question of TASK62.3, item 2, once the clone is done. What `n` and `i` assign by default is changed in TASK62.8. The panel tour data of the website (`web/src/data/panel.json`) is regenerated.
4. **Docs.** README, PROJECT.md (the assignment levels), `quiver add --help`, `quiver source add --help`, the website's concept, reference and security pages and the panel guide list the level. The website gets a guide page, "Finding skills", that starts with the levels, the question when a source is added and `--searchable` (TASK62.6 and TASK62.7 extend it); website pages follow the rule of TASK62.

**Done when:** tests cover `get_skill` of a Searchable skill without the note, `list_skills` without it, `update_skill` refusing it with the new error, a Searchable skill not being a prompt; each reader of item 2; and the panel's four states with the safeguards, the views, the counts and the question after `a`.
