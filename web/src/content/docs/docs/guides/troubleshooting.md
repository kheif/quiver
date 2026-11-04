---
title: Troubleshooting
description: Diagnose common problems with quiver doctor, the MCP log and quiver status.
---

## Start with `quiver doctor`

```sh
quiver doctor
```

It checks the Quiver home, the library, the assignments and the setup of each installed assistant, including the session-start hooks of Claude Code, Codex, Cursor, Antigravity and VS Code, and reports each check as `ok`, `warn` or `error`. The last line names the latest Quiver release.

```sh
quiver doctor --fix
```

This applies the safe fixes: it creates missing parts of the Quiver home, commits uncommitted changes in `local/`, and deletes folders left over from an interrupted `source add` or `remove`. After asking, it unassigns skills that are not in the library and deletes copies of served skills that Claude Code also loads by itself. Copies committed in git and copies in plugins are only reported.

## The assistant does not use a skill

1. **Is the skill served in this folder?** Run `quiver status` in the folder the assistant works in. If the skill is missing, assign it with `quiver add`.
2. **Did the server start in the right folder?** Open `~/.quiver/logs/mcp.log`. The `server started` line names the folder and the project. An assistant that starts the server outside the project serves only global skills; set `QUIVER_PROJECT` in its MCP entry. See [Connect an assistant](/docs/assistants/#serve-a-projects-skills-anywhere).
3. **Was the skill assigned during the session?** The skill list the session-start hook adds is built when the session starts. Start a new session, or ask the assistant to call `list_skills`.
4. **Does the description say when to use it?** The assistant decides from the description alone. Name the situations and words a request would use. See [Writing skills](/docs/guides/writing-skills/#write-a-good-description).
5. **Is it Claude Desktop?** It has no hook and drops the server's instructions, so the model does not get the skill list up front. Ask for Quiver's skills in the prompt; see [Connect an assistant](/docs/assistants/#what-works-where).
6. **Did the assistant say Quiver could not load something?** A skill or setting the hook could not read is named in a `<quiver-notice>` block, and the assistant tells you. Run `quiver doctor`, and look at `~/.quiver/logs/hook.log`.

## An always-on skill does not load

1. **Check how the assistant gets it.** `quiver status` lists, per assistant, how always-on skills reach it. It says `hook` only when the session-start hook is in place. If it says the hook is missing, run `quiver install`.
2. **Start a new session.** The hook runs when a session starts, so a change reaches the model from the next session.
3. **Check for too few hook commands.** An assistant takes only so much text from one hook command, so Quiver splits the text over several commands. `quiver doctor` warns when an assistant has fewer commands than your skills take. `quiver install`, or any command that changes skills, sets the number; run `quiver install <assistant>` to set it now.
4. **Check that the hook runs this binary.** After you move the binary or switch Node versions, `quiver doctor` reports a hook that runs another `quiver`. Run `quiver install` again.
5. **Is it Claude Desktop?** It has no hook. Ask for Quiver's skills in the prompt.

:::note[Next release]
The hook, the `hook` line in `quiver status` and the hook checks of `quiver doctor` are new in the next release. In v1.3.0 `quiver install` is `quiver mcp install`, and there is no hook.
:::

## A skill is loaded twice

Claude Code also loads skills from `~/.claude/skills`, the project's `.claude/skills` and enabled plugins. If the same skill is there and in Quiver, it may load twice. `quiver status` warns about each copy; `quiver doctor --fix` deletes the copies that are safe to delete.

## The assistant cannot start `quiver`

- **Installed with npm and a Node version manager:** the path of `quiver` changes with the Node version. Run `quiver install` again after switching.
- **Moved the binary:** run `quiver install` again; it replaces a registration that points to another binary.
- **Desktop apps** do not see your shell's `PATH`. `quiver install` writes the absolute path of the binary for them. For a hand-written entry, use the path that `quiver mcp config` prints.

## `quiver install` refuses a config file

A Cursor, Claude Desktop, Antigravity or VS Code config that is not valid JSON is reported and never overwritten. VS Code's `mcp.json` may hold comments: Quiver reads them, and writes the file without them after saving `<file>.quiver-backup`. Fix the file, or restore it from `<file>.quiver-backup`, and run the command again. For Codex, Quiver runs `codex mcp add`; if that fails, `codex mcp list` shows whether Codex can read its `config.toml`.

## Two folders are the same project

A copied project folder keeps the original's `.quiver.json`. `quiver status` warns about it. Run `quiver projects split` in the copy to give it its own ID.

## `quiver update` stops on a source

A source whose history was rewritten with a force-push is not updated. If you trust the new history, run:

```sh
quiver update <source> --reset
```

## Still stuck

Run the failing command with `-v` for more details, and [open an issue](https://github.com/kheif/quiver/issues/new/choose) with the output of `quiver --version` and `quiver doctor`. Report security problems privately; see [Security](/docs/project/security/).
