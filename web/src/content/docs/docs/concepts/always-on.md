---
title: Always-on skills
description: Skills that are in the conversation before the first reply, and how they get there in each assistant.
sidebar:
  badge:
    text: Soon
    variant: caution
---

Most skills load when a task needs them: the assistant reads the catalog and decides. Some skills should never wait for that decision, such as security rules, team standards or the way you write. Mark them always on:

```sh
quiver add security-review --global --always
quiver add reply-style --always          # only in this project
```

An always-on skill is in the conversation before the first reply, in every session where it is assigned. `quiver remove <id> --always` takes the mark off and keeps the assignment.

## How they reach the model

**At session start, through a hook.** In assistants that run a command when a session starts, Quiver's hook adds the full text of the always-on skills to the context itself, inside a `<quiver-always-on>` block. The model does not have to decide to load them.

:::note[Coming soon]
Session-start hooks for Claude Code and Codex are planned; `quiver install` will set them up.
:::

**Through Quiver's instructions.** In assistants that add the MCP server's instructions to the prompt, the instructions name the always-on skills and ask the model to load them with `get_skill` before its first reply. This works today in Claude Code and Cursor.

**When you ask.** Where the model does not see Quiver's instructions up front (Codex today, Claude Desktop, Antigravity), ask for Quiver's skills in the prompt, such as *"check the quiver skills for this"*. In Codex and Antigravity the model then follows the always-on skills too.

See [Connect an assistant](/docs/assistants/#what-works-where) for the table.

## Limits

- Changes take effect in the next session.
- The model sees the text; it is guidance, not enforcement. A rule that must never break also needs a check outside the model, such as a hook that blocks a command or a CI check.
- Only the skill's text is delivered. A plugin's own hooks, such as a per-message reminder, are not run.
- Sub-agents are not reliable: in testing, one of two sub-agents followed the always-on skill.
