---
title: How MCP delivery works
description: How the Quiver MCP server tells the assistant which skills exist and how the assistant loads them.
---

Quiver delivers skills only through its MCP server, `quiver mcp`. The assistant starts it on stdio at the beginning of a session.

## 1. The server finds the project

The server serves the project of its working folder, found by walking up to `.quiver.json`, or of `$QUIVER_PROJECT` when that is set. Without a project it serves the global skills only. It logs the folder and the project to `~/.quiver/logs/mcp.log`.

## 2. The skill list goes into the instructions

MCP servers can send `instructions` when a session starts. Assistants such as Claude Code and Cursor add them to the system prompt, so the model knows the skills from the first message. Quiver's instructions hold:

- the always-on skills, with the request to load them with `get_skill` before the first reply,
- the catalog: the ID and description of every skill served here, project skills first,
- a pointer to `create_skill` for saving new skills.

Instructions are cut at about 4096 characters. A catalog that does not fit is replaced by a pointer: before its first reply, the assistant calls `list_skills`, which returns every skill with its whole description.

Tool descriptions are not used for the catalog: Claude Code defers MCP tools, so the model does not see their descriptions until it looks them up.

## 3. The assistant loads what fits

When a task fits a skill's description, the assistant calls `get_skill` with its ID. The result is the skill's instructions, a list of its extra files, and the skill's folder. The assistant reads an extra file with `get_skill_file`, and runs a skill's scripts itself, from that folder.

Only the skills a task needs are read into the conversation.

## 4. Skills as prompts

Every served skill is also an MCP prompt. In Claude Code it appears as `/quiver:<id>`. Text after the command is passed to the skill as the task.

## 5. Live updates

While a session runs, the server watches the library and the assignments. Skills you add, remove or change show up in `list_skills` and the prompts right away; the server sends `prompts/list_changed`. The catalog in the instructions is built once, so it changes from the next session.

## Writing skills from the conversation

The assistant can also write to your library:

- `create_skill` creates a local skill, commits it and assigns it to this project or globally.
- `update_skill` changes a local skill and commits the change. Source skills are read-only.
- `search_skills` searches the whole library, also skills not assigned here. The assistant may suggest an unassigned skill, but asks before using it.

See [MCP tools and prompts](/docs/reference/mcp/) for the inputs and results of every tool.

## Where it differs by assistant

Not every assistant passes `instructions` to the model up front. Codex passes them only after the model looks for Quiver's tools, Antigravity keeps them in a file the model rarely reads, and Claude Desktop drops them; there skills are used when you ask for them. A session-start hook (coming soon) puts always-on skills in the context in Claude Code and Codex without the instructions; see [Always-on skills](/docs/concepts/always-on/) and [Connect an assistant](/docs/assistants/#what-works-where).
