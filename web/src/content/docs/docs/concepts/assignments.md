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

An always-on skill is in the conversation before the first reply. How it gets there depends on the assistant, and changes take effect in the next session; see [Always-on skills](/docs/concepts/always-on/).

## Removing assignments

```sh
quiver remove release-notes             # from this project
quiver remove caveman --global          # from every folder
quiver remove caveman --global --always # keep it global, drop only the always-on mark
```

The skills stay in the library. To delete a local skill, use `quiver delete`.

## Assignments are yours

Assignments are stored in `~/.quiver/registry.json`, not in the project. A teammate who clones a project gets its `.quiver.json`, but not your skills or your assignments.

To share skills with a team, put them in a git repository that everyone adds with [`quiver source add`](/docs/guides/sources/). Each person then assigns the skills they want.

## Where a skill is assigned

```sh
quiver where release-notes   # global, and each project with its folders
quiver status                # every skill served in this folder
```

Running assistant sessions see assignment changes through `list_skills` and the prompts right away. The skill list in the server's instructions, and the always-on skills, change from the next session.
