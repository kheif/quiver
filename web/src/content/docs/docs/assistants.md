---
title: Connect an assistant
description: Set Quiver up for Claude Code, Codex, Cursor, Claude Desktop, Antigravity or any other MCP client, and what each one supports.
---

Quiver reaches your assistant through one MCP server, `quiver mcp`. You set it up once per assistant; the assistant starts it for each session.

```sh
quiver install
```

This sets up every supported assistant that is installed. Name assistants to set up only those:

```sh
quiver install codex cursor
```

:::note[Next release]
In the next release, `quiver install` replaces `quiver mcp install`, and Codex and Antigravity are supported. In v1.2.0, run `quiver mcp install`, which sets up Claude Code and Cursor, and name one other assistant with `--for`, such as `quiver mcp install --for claude-desktop`.
:::

## Supported assistants

| Assistant | Name | What `quiver install` changes |
| --- | --- | --- |
| Claude Code | `claude-code` | Runs `claude mcp add --scope user quiver -- <path to quiver> mcp`. |
| Codex | `codex` | Runs `codex mcp add quiver -- <path to quiver> mcp`, which writes `~/.codex/config.toml` (or `$CODEX_HOME`). The Codex CLI and app share it. |
| Cursor | `cursor` | Adds `quiver` to `mcpServers` in `~/.cursor/mcp.json`. |
| Claude Desktop | `claude-desktop` | Adds `quiver` to `mcpServers` in `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows). |
| Antigravity | `antigravity` | Adds `quiver` to `mcpServers` in `~/.gemini/config/mcp_config.json`, which the Antigravity CLI (`agy`) reads. |
| VS Code Copilot | | Planned. |

An assistant counts as installed when its command (`claude`, `codex`) is on `PATH`, or when its config folder exists (`~/.gemini/antigravity-cli` for Antigravity).

In the JSON files, every other key stays as it is, including an `env` of the `quiver` entry, and the old file is saved next to it as `<file>.quiver-backup`. A file that is not valid JSON is reported and never overwritten. When Codex already has a `quiver` entry, its `env` is kept too. Restart the assistant afterwards.

Every assistant but Claude Code gets the path of the binary itself, never the npm launcher: desktop apps do not see your shell's `PATH`, so they cannot find `node`.

Run `quiver install` again after you move the binary. A registration that points to another binary is replaced.

## What works where

| Assistant | Skill list reaches the model | Always-on skills | Project skills |
| --- | --- | --- | --- |
| Claude Code | Yes | Yes; at session start through a hook, coming soon | Yes, it starts the server in the project folder |
| Cursor | Yes | Yes | Yes, it starts the server in the workspace folder |
| Codex | When the model looks for Quiver's tools | At session start through a hook, coming soon; today when you ask | Yes, it starts the server in the working directory |
| Claude Desktop | No | No | Only with `QUIVER_PROJECT` |
| Antigravity | Rarely | When you ask | Yes, it starts the server in the working directory |
| Other clients | Depends on the client | Depends on the client | Where it starts the server, or with `QUIVER_PROJECT` |

See [Always-on skills](/docs/concepts/always-on/) for how always-on skills reach each assistant.

### Claude Code

Claude Code adds the server's instructions to its system prompt, so the model knows the skills from the first message. It starts the server in the folder you start Claude Code in, so project skills work.

Every served skill is also a prompt: run `/quiver:<id>`, such as `/quiver:release-notes`. Text after the command is passed to the skill as the task.

### Cursor

Cursor passes the skill list to the model and starts the server in the workspace folder, so project skills work. Use Agent mode.

### Codex

Codex loads an MCP server's tools, and with them its instructions, only when the model searches for them. Until a session-start hook loads the always-on skills (coming soon), ask for Quiver's skills in the prompt, such as *"check the quiver skills for this"*; from then on the model follows the always-on skills too. Quiver's read tools are marked read-only, so Codex runs them without asking.

### Claude Desktop and Antigravity

Both run Quiver's tools, but the model does not see the skill list up front: Claude Desktop drops the server's instructions, and Antigravity keeps them in a file the model rarely reads. A skill is used when you ask for Quiver's skills in the prompt. `quiver install` says so when it sets them up.

Claude Desktop never starts the server in a project. To serve a project's skills there, set `QUIVER_PROJECT` in the entry (see below).

## Any other MCP client

`quiver mcp config` prints the entry that starts this `quiver`, without changing anything:

```json
{ "mcpServers": { "quiver": { "command": "/path/to/quiver", "args": ["mcp"] } } }
```

Add it to the assistant's MCP config. Whether the skill list reaches the model depends on whether the client passes MCP server `instructions` to the model. If it does not, ask for Quiver's skills in the prompt.

## Serve a project's skills anywhere

The server serves the project of the folder it is started in. An assistant that starts it somewhere else, such as your home folder, gets only the global skills. Set `QUIVER_PROJECT` to a project folder in that assistant's MCP entry:

```json
{
  "mcpServers": {
    "quiver": {
      "command": "/path/to/quiver",
      "args": ["mcp"],
      "env": { "QUIVER_PROJECT": "/Users/me/code/app" }
    }
  }
}
```

`quiver install` keeps this `env` when it updates the entry.

## Check the setup

```sh
quiver doctor
```

`quiver doctor` checks the entry of each installed assistant. The server logs to `~/.quiver/logs/mcp.log`; a `server started` line names the folder and the project it serves.

## Remove the server

```sh
quiver uninstall                # from every assistant that has it
quiver uninstall claude-desktop # from one assistant
```

Your library and its skills are not touched.
