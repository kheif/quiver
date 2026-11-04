---
title: How MCP delivery works
description: How the Quiver MCP server tells the assistant which skills exist and how the assistant loads them.
---

Quiver delivers skills only through its MCP server, `quiver mcp`. The assistant starts it on stdio at the beginning of a session.

## 1. The server finds the project

The server serves the project of its working folder, found by walking up to `.quiver.json`, or of `$QUIVER_PROJECT` when that is set. Without a project it serves the global skills only. It logs the folder and the project to `~/.quiver/logs/mcp.log`.

## 2. The skills reach the model

MCP servers can send `instructions` when a session starts. Quiver's instructions are short. They tell the assistant:

- how to use the tools: `get_skill`, `get_skill_file`, `list_skills` and `search_skills`,
- that new skills are saved with `create_skill`,
- the names of the always-on skills, for an assistant without a session-start hook,
- to call `list_skills` before its first reply when the conversation has no `<quiver-skills>` block.

The skills themselves come from a session-start hook, which `quiver install` sets up for Claude Code, Codex, Cursor, Antigravity and VS Code. It adds the always-on skills in full, and the ID and description of every other skill served here, project skills first, to the conversation. An assistant takes only so much text from one hook command, so Quiver splits it over as many hook commands as needed and leaves nothing out. A skill that cannot be read is named in a `<quiver-notice>` block. See [Always-on skills](/docs/concepts/always-on/).

Tool descriptions are not used for the catalog: Claude Code defers MCP tools, so the model does not see their descriptions until it looks them up.

:::note[Next release]
The hook and the short instructions are new in the next release. In v1.3.0 the instructions hold the always-on skills with the request to load them with `get_skill`, and the catalog: the ID and description of every skill served here. They are cut at about 4096 characters, and a catalog that does not fit is replaced by a pointer to `list_skills`.
:::

## 3. The assistant loads what fits

When a task fits a skill's description, the assistant calls `get_skill` with its ID. The result is the skill's instructions, a list of its extra files, and the skill's folder. The assistant reads an extra file with `get_skill_file`, and runs a skill's scripts itself, from that folder.

Only the skills a task needs are read into the conversation.

## 4. Skills as prompts

Every served skill is also an MCP prompt. In Claude Code it appears as `/quiver:<id>`. Text after the command is passed to the skill as the task.

## 5. Live updates

While a session runs, the server watches the library and the assignments. Skills you add, remove or change show up in `list_skills` and the prompts right away; the server sends `prompts/list_changed`. The hook's text is built when a session starts, so the skill list and the always-on skills in it change from the next session.

## Writing skills from the conversation

The assistant can also write to your library:

- `create_skill` creates a local skill, commits it and assigns it to this project or globally.
- `update_skill` changes a local skill served in this project and commits the change. Source skills are read-only, and so are always-on skills: change those with `quiver edit`.
- `search_skills` searches the whole library, also skills not assigned here. The assistant may suggest an unassigned skill, but asks before using it.

See [MCP tools and prompts](/docs/reference/mcp/) for the inputs and results of every tool.

## Where it differs by assistant

Not every assistant passes `instructions` to the model up front. Claude Desktop drops them and has no hook, so there skills are used when you ask for them. The other supported assistants run the session-start hook, so the skills reach the model without the instructions; see [Always-on skills](/docs/concepts/always-on/) and [Connect an assistant](/docs/assistants/#what-works-where).
