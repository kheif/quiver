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

**At session start, through a hook.** `quiver install` sets up a session-start hook for Claude Code, Codex, Cursor, Antigravity and VS Code. When a session starts, the hook adds the full text of the always-on skills to the context itself, inside a `<quiver-always-on>` block, global skills first. After them it adds the ID and description of every other skill served in the folder, inside a `<quiver-skills>` block. The model does not have to decide to load anything, however many always-on skills there are and however large they are.

An assistant takes only so much text from one hook command (Claude Code 10,000 characters). Quiver therefore splits the text over as many hook commands as it takes, and nothing is left out. Each part says which one it is, and a skill is cut between parts only when it is longer than one. `quiver install` and every command that changes skills set the number of commands; `quiver doctor` warns when an assistant has fewer commands than the text takes. The parts of one session start share one text, so a skill edited meanwhile cannot give them different texts.

Claude Code, Codex, Antigravity and VS Code also run the hook for each sub-agent. Cursor's sub-agent event cannot add text, so it has only the session-start hook.

**A session never starts silently without its skills.** A skill that cannot be read is left out and named, and so is anything else the hook could not do, such as an unreadable registry or a folder that has the project ID of another folder. The assistant is told in a `<quiver-notice>` block and tells you. Text that reaches the model is cleaned: hidden characters are dropped, and a skill cannot write Quiver's own tags. `~/.quiver/logs/hook.log` records each run with a hash of the text it passed.

**Through Quiver's instructions.** In a client without a hook that adds the MCP server's instructions to the prompt, the instructions name the always-on skills and ask the model to load them with `get_skill` before its first reply, if the conversation has no `<quiver-always-on>` block.

**When you ask.** Claude Desktop has no hook and drops the server's instructions, so ask for Quiver's skills in the prompt, such as *"check the quiver skills for this"*.

`quiver install` ends with one line per assistant on how always-on skills reach it, and `quiver status` lists the same. It says "hook" only when the hook is really in place. For an assistant without a hook it names the always-on skills, which do not load by themselves there.

:::note[Next release]
The hook, the parts, the notices and the short instructions are new in the next release. In v1.3.0 there is no hook: Claude Code and Cursor get the skill list in the server's instructions and load the always-on skills themselves, and in Codex, Claude Desktop and Antigravity you ask for Quiver's skills in the prompt.
:::

See [Connect an assistant](/docs/assistants/#what-works-where) for the table.

## Limits

- Changes take effect in the next session, because the hook runs when a session starts.
- The model sees the text; it is guidance, not enforcement. A rule that must never break also needs a check outside the model, such as a hook that blocks a command or a CI check.
- Only the skill's text is delivered. A plugin's own hooks, such as a per-message reminder, are not run.
- Sub-agents get the text from the sub-agent hook, in the assistants that have one (Claude Code, Codex, Antigravity, VS Code). Cursor has none.
- The assistant cannot change an always-on skill through `update_skill`. Change it yourself with `quiver edit`.
