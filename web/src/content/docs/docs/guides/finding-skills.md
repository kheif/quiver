---
title: Finding skills
description: Keep skills out of the session-start list and let the assistant find them by searching, with the Searchable level.
sidebar:
  badge:
    text: Next
    variant: default
---

:::note[Next release]
The Searchable level is on the main branch and ships after v1.5.0.
:::

Listed skills cost tokens in every session: each one adds its ID and description to the text the assistant starts with. A skill that is needed only when its topic comes up, such as a style guide for Slack messages or a guide to writing like a person, should cost nothing until then. Make it **Searchable**.

## The four levels

Each scope, every folder or one project, holds a skill at one of four levels:

| Level | At the start of a session | Set with |
| --- | --- | --- |
| Not assigned | Nothing | `quiver remove <id>` |
| Searchable | Nothing: the assistant finds it by searching | `quiver add <id> --searchable` |
| On demand | Its ID and description are listed; the assistant loads it when a task fits | `quiver add <id>` |
| Always on | Loaded in full | `quiver add <id> --always` |

A skill moves between them: `--searchable` takes it off the listed skills of that scope, always-on mark included, and `quiver add` without the flag takes it off the Searchable ones. Across the global and the project scope, the level in a folder is the most served of the two. The [panel](/docs/guides/panel/) sets the same four levels in the detail of a skill.

## How the assistant finds a skill

A Searchable skill is not an MCP prompt and is not in `list_skills`. The assistant calls `search_skills` with a query that names what it needs, and gets the ID and description of the best matches, not their text. If one fits, it calls `get_skill` and follows it. It does not ask you first: assigning a skill is your statement that you trust it, which is why you should read a skill with `quiver show <id>` before you make it Searchable, and why the assistant cannot change one through `update_skill`.

## Make skills Searchable

```sh
quiver add slack-style --searchable             # in this project
quiver add slack-style --searchable --global    # in every folder
quiver add --source anthropic --searchable --global
```

`--source <name>` makes every skill of a source Searchable that is not assigned yet in that scope. A skill that is listed or always on there keeps its level. `--searchable` cannot be combined with `--always`.

When you add a source, Quiver asks once, in a terminal, whether to make its skills Searchable in every folder. The default is no, and `--searchable` answers yes:

```sh
quiver source add anthropics/skills --name anthropic --searchable
```

The output prints the command that undoes it, `quiver remove --source anthropic --global`. `quiver remove <id>` takes a skill off every level of the scope, and `--source <name>` does it for a whole source.

## See where a skill is Searchable

```sh
quiver list         # the ASSIGNED column shows searchable (global, 2 projects)
quiver where slack-style
quiver status       # the Searchable skills of this folder
```

`quiver list --unassigned` leaves Searchable skills out, because they are assigned. `quiver update` previews changes to a Searchable skill like those of any assigned skill, and `quiver sync` lists skills that another machine made Searchable before it pulls them.
