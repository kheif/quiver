---
title: MCP tools and prompts
description: The instructions, tools and prompts that the Quiver MCP server exposes to assistants.
---

`quiver mcp` runs the MCP server on stdin and stdout. Assistants start it; you do not run it yourself. It serves the skills of the project in its working folder, or in `$QUIVER_PROJECT` when that is set, and logs to `~/.quiver/logs/mcp.log`.

## Instructions

When a session starts, the server's `instructions` are short. They tell the assistant:

- how to use the tools: `get_skill`, `get_skill_file`, `list_skills` and `search_skills`,
- that new skills are saved with `create_skill`,
- which skills are always-on, for an assistant without a session-start hook, and to load them with `get_skill` before the first reply,
- to call `list_skills` before the first reply when the conversation has no `<quiver-skills>` block.

The skills themselves come from the session-start hook: the always-on skills in full, and the ID and description of the others. See [Always-on skills](/docs/concepts/always-on/).

## Tools

| Tool | Input | Result |
| --- | --- | --- |
| `list_skills` | — | Every listed skill here, project skills first: ID, name, whole description, `global` or `project`, and whether it is always-on. Searchable skills are not listed. |
| `get_skill` | `id` | The skill's instructions, its extra files and its folder. |
| `get_skill_file` | `id`, `path` | One extra file of a skill, such as `references/checklist.md`. Large and binary files return their path instead. |
| `create_skill` | `name`, `description`, `body`, `scope` (`project` or `global`), optional `files` | Creates a local skill, commits it and assigns it. In a folder that is not a project yet, `project` scope registers it. |
| `update_skill` | `id`, any of `description`, `body`, `files` (an empty content deletes the file) | Changes a local skill and commits it as `update <name> (via MCP)`. Other frontmatter fields are kept. Source skills are read-only. It changes only skills served in the session's project, and never an always-on or Searchable skill: change those with `quiver edit`, or make a Searchable skill listed with `quiver add <id>` first. |
| `search_skills` | `query` | Searches the skills that are [Searchable](/docs/guides/finding-skills/) in this project, ranked by words (and by meaning with the `local` engine), and returns at most 10, with `more` for the matches left out. Every match is assigned here (`assigned` is true, `scope` is `project` or `global`); the assistant loads the one that fits with `get_skill`. Listed skills and skills that are not assigned are not searched. |

## Prompts

Every served skill is also an MCP prompt, named by its ID. In Claude Code it appears as `/quiver:<id>`. Text after the command is passed to the skill as the task.

## Notifications

While a session runs, the server watches the library and the assignments. When they change, it sends `prompts/list_changed`, and `list_skills` returns the new list. The hook's text and the instructions do not change during a session.
