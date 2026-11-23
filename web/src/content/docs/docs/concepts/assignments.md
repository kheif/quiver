---
title: Assignments
description: Global skills, project skills and always-on skills, and where assignments are stored.
---

A skill in the library is served nowhere until you assign it. An assignment says where the MCP server serves a skill.

| Assignment | Served in | Set with |
| --- | --- | --- |
| Project | The folders of one project | `quiver add <id>` in the project |
| Global | Every folder | `quiver add <id> --global` |
| Always-on | Where it is assigned, and loaded before the first reply | `quiver add <id> --always` |
| Searchable | Not listed, not loaded: the assistant finds it by searching | `quiver add <id> --searchable` |

A folder that is not a project gets only the global skills.

## Project skills

```sh
cd ~/code/app
quiver add release-notes anthropic/docx
```

The skills are served in this project only. If the folder is not a project yet, it becomes one: in a terminal, Quiver first asks `Register <folder> as a project?` (`--yes` skips the question). See [Projects](/docs/concepts/projects/).

`--project <folder>` assigns to another project without changing directory:

```sh
quiver add tdd --project ~/code/api
```

## Global skills

```sh
quiver add commit-style --global
```

A global skill is served in every folder, projects included. Its description reaches the assistant in every session, so keep global skills few and their descriptions precise.

## Always-on skills

Most skills are loaded when a task needs them; the assistant decides. Some must apply from the first message without that decision, such as a communication style or house rules. Mark them always-on:

```sh
quiver add caveman --global --always
```

An always-on skill is in the conversation before the first reply. In Claude Code, Codex, Cursor, Antigravity and VS Code, a session-start hook adds its full text. Changes take effect in the next session; see [Always-on skills](/docs/concepts/always-on/).

An assistant cannot change an always-on skill through `update_skill`. Change it yourself with `quiver edit`.

## Searchable skills

A skill that is only needed when its topic comes up, such as a style guide for Slack messages, should cost nothing until then. Make it Searchable. The skills you create with `quiver new` and import with `quiver import` are Searchable from the start; a skill the assistant creates with `create_skill` is on demand until you decide otherwise, and a fork takes the level of the original:

```sh
quiver add slack-style --searchable --global
quiver add --source anthropic --searchable --global
```

A Searchable skill is not listed at the start of a session and not loaded. The assistant finds it with `search_skills`, which returns the ID and description of the best matches, and loads the one that fits with `get_skill`, without asking you: assigning a skill is your statement that you trust it. See [Finding skills](/docs/guides/finding-skills/).

A skill is at one level in a scope: `--searchable` takes it off the listed skills of that scope, always-on mark included, and `quiver add` without the flag takes it off the Searchable ones. Across the global and the project scope, its level in a folder is the most served of the two. An assistant cannot change a Searchable skill through `update_skill`, because it would load the new text without asking; make the skill listed with `quiver add <id>` first, or use `quiver edit`.

## Removing assignments

```sh
quiver remove release-notes             # from this project
quiver remove caveman --global          # from every folder
quiver remove caveman --global --always # keep it global, drop only the always-on mark
quiver remove --source anthropic --global # every skill of a source, at every level
```

`quiver remove` takes a skill off every level of the scope. The skills stay in the library. To delete a local skill, use `quiver delete`.

## Assignments are yours

Assignments are stored in `~/.quiver/registry.json`, not in the project. A teammate who clones a project gets its `.quiver.json`, but not your skills or your assignments.

To share skills with a team, put them in a git repository that everyone adds with [`quiver source add`](/docs/guides/sources/). Each person then assigns the skills they want.

## Where a skill is assigned

```sh
quiver where release-notes   # global, and each project with its folders
quiver status                # every skill served in this folder, and the Searchable ones
```

Running assistant sessions see assignment changes through `list_skills` and the prompts right away. The text the session-start hook adds, the always-on skills and the list of other skills, changes from the next session.
