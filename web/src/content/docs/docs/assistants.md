---
title: Connect an assistant
description: Set Quiver up for Claude Code, Codex, Cursor, Claude Desktop, Antigravity, VS Code or any other MCP client, and what each one supports.
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
In the next release, `quiver install` replaces `quiver mcp install`, which still works as a hidden, deprecated alias until the next major version. In v1.3.0, run `quiver mcp install`, which sets up every supported assistant that is installed, or name the assistants to set up, such as `quiver mcp install claude-desktop`. v1.3.0 registers the MCP server only; it sets up no session-start hook and does not support VS Code.
:::

Next to the MCP server, `quiver install` sets up a session-start hook for Claude Code, Codex, Cursor, Antigravity and VS Code, and a sub-agent hook where the assistant has one. The hook is how always-on skills reach the model. Claude Desktop has no hook. See [Always-on skills](/docs/concepts/always-on/).

## Supported assistants

| Assistant | Name | What `quiver install` changes |
| --- | --- | --- |
| Claude Code | `claude-code` | Runs `claude mcp add --scope user quiver -- <path to quiver> mcp`, and adds the session-start and sub-agent hooks. |
| Codex | `codex` | Runs `codex mcp add quiver -- <path to quiver> mcp`, which writes `~/.codex/config.toml` (or `$CODEX_HOME`), and adds the session-start and sub-agent hooks. The Codex CLI and app share them. |
| Cursor | `cursor` | Adds `quiver` to `mcpServers` in `~/.cursor/mcp.json`, and adds the session-start hook. |
| Claude Desktop | `claude-desktop` | Adds `quiver` to `mcpServers` in `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows). It has no hook. |
| Antigravity | `antigravity` | Adds `quiver` to `mcpServers` in `~/.gemini/config/mcp_config.json`, which the Antigravity CLI (`agy`) reads, and adds the session-start and sub-agent hooks. |
| VS Code Copilot | `vscode` | Adds `quiver` to `servers` in `~/Library/Application Support/Code/User/mcp.json` (macOS; `%APPDATA%\Code\User\mcp.json` on Windows, `~/.config/Code/User/mcp.json` on Linux), with `"type": "stdio"`, and adds the session-start and sub-agent hooks in `~/.copilot/hooks/quiver.json`. |

An assistant counts as installed when its command (`claude`, `codex`) is on `PATH`, or when its config folder exists (`~/.gemini/antigravity-cli` for Antigravity, VS Code's user folder for VS Code; VS Code Insiders keeps its files under `Code - Insiders`).

In the JSON files, every other key stays as it is, including an `env` of the `quiver` entry, and the old file is saved next to it as `<file>.quiver-backup`. A file that is not valid JSON is reported and never overwritten. VS Code's `mcp.json` may contain comments: Quiver reads them, and writes the file without them after saving the `.quiver-backup`. When Codex already has a `quiver` entry, its `env` is kept too. Restart the assistant afterwards.

Every assistant but Claude Code gets the path of the binary itself, never the npm launcher: desktop apps do not see your shell's `PATH`, so they cannot find `node`.

Run `quiver install` again after you move the binary. A registration that points to another binary is replaced.

When it is done, `quiver install` prints one line per assistant on how always-on skills reach it, and `quiver status` lists the same. It says "hook" only when the hook is really in place.

## What works where

| Assistant | Skill list reaches the model | Always-on skills | Project skills |
| --- | --- | --- | --- |
| Claude Code | Yes, through the hook | Yes, through the hook | Yes, it starts the server in the project folder |
| Cursor | Yes, through the hook | Yes, through the hook | Yes, it starts the server in the workspace folder |
| Codex | Yes, through the hook | Yes, through the hook | Yes, it starts the server in the working directory |
| Antigravity | Yes, through the hook | Yes, through the hook | Yes, it starts the server in the project folder |
| VS Code | Yes, through the hook | Yes, through the hook | Yes, it sends the workspace as an MCP root |
| Claude Desktop | No | No; when you ask | Only with `QUIVER_PROJECT` |
| Other clients | Depends on the client | Depends on the client | Where it starts the server, or with `QUIVER_PROJECT` |

:::note[Next release]
The table shows the next release. In v1.3.0 there is no hook. Claude Code and Cursor get the skill list in the server's instructions and load always-on skills themselves; Codex and Antigravity get Quiver's skills only when you ask for them in the prompt; VS Code is not supported.
:::

See [Always-on skills](/docs/concepts/always-on/) for how always-on skills reach each assistant.

### Claude Code

A session-start hook adds the always-on skills in full, and the ID and description of every other skill served in the folder, to the context, so the model knows the skills from the first message. Claude Code also adds the server's short instructions to its system prompt. It starts the server in the folder you start Claude Code in, so project skills work.

Every served skill is also a prompt: run `/quiver:<id>`, such as `/quiver:release-notes`. Text after the command is passed to the skill as the task.

### Cursor

Cursor runs the session-start hook, so the skills reach the model from the first message, and starts the server in the workspace folder, so project skills work. Use Agent mode. Cursor's sub-agent event cannot add text, so Quiver sets up only the session-start hook.

### Codex

Codex runs the session-start hook, so the skills reach the model without it looking for Quiver's tools. Codex's hooks are trusted automatically through `codex app-server`; if that fails, `quiver install` prints the "Review hooks" step. Quiver's read tools are marked read-only, so Codex runs them without asking.

### Antigravity

Antigravity runs the session-start hook and starts the server in the project folder, so project skills work.

### VS Code

VS Code starts the server in your home folder and sends the workspace as an MCP root, which Quiver reads after startup to serve that project's skills. Quiver owns one hook file, `~/.copilot/hooks/quiver.json`, which both of VS Code's harnesses read.

### Claude Desktop

Claude Desktop runs Quiver's tools, but it drops the server's instructions and has no hook, so the model does not see the skills up front. A skill is used when you ask for Quiver's skills in the prompt, such as *"check the quiver skills for this"*. `quiver install` says so when it sets it up.

Claude Desktop never starts the server in a project. To serve a project's skills there, set `QUIVER_PROJECT` in the entry (see below).

## Any other MCP client

`quiver mcp config` prints the entry that starts this `quiver`, without changing anything:

```json
{ "mcpServers": { "quiver": { "command": "/path/to/quiver", "args": ["mcp"] } } }
```

Add it to the assistant's MCP config. Whether the model sees which skills exist depends on whether the client passes MCP server `instructions` to the model. If it does not, ask for Quiver's skills in the prompt.

## Serve a project's skills anywhere

The server serves the project of the folder it is started in. An assistant that starts it somewhere else, such as your home folder, gets only the global skills, unless it sends its workspace as an MCP root (VS Code). Set `QUIVER_PROJECT` to a project folder in that assistant's MCP entry:

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

`quiver doctor` checks the entry of each installed assistant and the hooks of Claude Code, Codex, Cursor, Antigravity and VS Code. It reports a hook that is missing or runs another binary, a Codex hook that needs review, and an assistant with fewer hook commands than the text of your skills takes. The server logs to `~/.quiver/logs/mcp.log`; a `server started` line names the folder and the project it serves. The hook logs to `~/.quiver/logs/hook.log`.

## Remove the server

```sh
quiver uninstall                # from every assistant that has it
quiver uninstall claude-desktop # from one assistant
```

This removes the MCP server and Quiver's hooks. Your library and its skills are not touched.

:::note[Next release]
In v1.3.0 the command is `quiver mcp uninstall`.
:::
