---
title: MCP tools and prompts
description: The instructions, tools and prompts that the Quiver MCP server exposes to assistants.
---

`quiver mcp` runs the MCP server on stdin and stdout. Assistants start it; you do not run it yourself. It serves the skills of the project in its working folder, or in `$QUIVER_PROJECT` when that is set, and logs to `~/.quiver/logs/mcp.log`.

## Instructions

When a session starts, the server's `instructions` tell the assistant:

- which skills are always-on, and to load them with `get_skill` before the first reply,
- the ID and description of each skill served here,
- that new skills are saved with `create_skill`.

They are cut at about 4096 characters. When the catalog does not fit, it is replaced by a request to call `list_skills` before the first reply.

## Tools

| Tool | Input | Result |
| --- | --- | --- |
| `list_skills` | — | Every skill served here, project skills first: ID, name, whole description, `global` or `project`, and whether it is always-on. |
| `get_skill` | `id` | The skill's instructions, its extra files and its folder. |
| `get_skill_file` | `id`, `path` | One extra file of a skill, such as `references/checklist.md`. Large and binary files return their path instead. |
| `create_skill` | `name`, `description`, `body`, `scope` (`project` or `global`), optional `files` | Creates a local skill, commits it and assigns it. In a folder that is not a project yet, `project` scope registers it. |
| `update_skill` | `id`, any of `description`, `body`, `files` (an empty content deletes the file) | Changes a local skill and commits it as `update <name> (via MCP)`. Other frontmatter fields are kept. Source skills are read-only. |
| `search_skills` | `query` | Searches the whole library, also skills not assigned here, and marks each match as assigned here (`global` or `project`) or not. The assistant may suggest an unassigned skill, but asks before using it. |

## Prompts

Every served skill is also an MCP prompt, named by its ID. In Claude Code it appears as `/quiver:<id>`. Text after the command is passed to the skill as the task.

## Notifications

While a session runs, the server watches the library and the assignments. When they change, it sends `prompts/list_changed`, and `list_skills` returns the new list. The instructions do not change during a session.
