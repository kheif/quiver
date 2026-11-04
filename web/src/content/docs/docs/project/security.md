---
title: Security
description: What Quiver trusts, how to review skills from sources, and how to report a vulnerability.
---

Skills give instructions to your assistant, and a skill can include scripts the assistant may run. A skill from a source is code you run on your own machine. Quiver serves skills from your library and from the sources you added; it does not check what they say.

## Practices

- **Add only sources you trust**, and read their skills before you assign them: `quiver show <id>` prints a skill's files and its `SKILL.md`.
- **Review updates.** `quiver update` downloads new commits before it applies them, lists the changed files of every assigned skill, marks scripts and executable files as **needs review**, and asks before it applies the changes. Read them with `quiver update --diff`; answering no leaves the source as it was.
- **Pin what you depend on.** `quiver pin` keeps a skill at a commit you reviewed. Move the pin when you have read the changes.
- **Fork to change.** Source skills are read-only in Quiver. `quiver fork` copies one into your own skills.
- **Watch global skills.** `create_skill` and `update_skill` let the assistant write skills into your library. A global skill's description reaches the assistant in every folder, so review new global skills with `quiver list` and `quiver edit`.
- **Review what a sync brings.** `quiver sync` takes in what another machine pushed. It lists changes that make a skill always-on, change the files of an always-on skill or add or change a source, and asks before it pulls. Use a private remote.
- **Know what the whole library exposes.** `get_skill` and `search_skills` read the whole library, not only the skills assigned to the project. Assign only what you want a project's sessions to see by name.
- **Always-on skills are changed by you.** `update_skill` changes only skills served in the session's project, and never an always-on skill: every session starts with that text, so change it yourself with `quiver edit`.

Every change to your own skills is a commit in `~/.quiver/local` (in `~/.quiver` once you use `quiver sync`). `git -C ~/.quiver/local log -p` shows them, and `git revert` undoes one.

## What reaches the model

The session-start hook passes the text of your always-on skills and the ID and description of your other skills to the model. A skill that cannot be read is left out and named in a `<quiver-notice>` block, so a session does not start silently without it. Text that reaches the model is cleaned: hidden characters are dropped, and a skill cannot write Quiver's own tags (`<quiver-always-on>`, `<quiver-skills>`, `<quiver-notice>`). `~/.quiver/logs/hook.log` records each run with a hash of the text it passed.

A source skill folder that is a symlink is not discovered, so an update cannot change a skill through files that git reports somewhere else.

`quiver install` changes your assistants' config files: it adds the MCP server and the hooks. `quiver uninstall` removes both.

:::note[Next release]
The hook, the notices, the cleaning of text, the checks on `quiver sync` and `update_skill`, and the symlink rule are new in the next release. In v1.3.0 `quiver mcp install` registers only the MCP server, `quiver sync` pulls without asking, and `update_skill` can change any local skill.
:::

## Installing Quiver

The install scripts download a release archive from GitHub, check it against the release's `checksums.txt`, and place one binary. Read [`install.sh`](https://github.com/kheif/quiver/blob/main/install.sh) or [`install.ps1`](https://github.com/kheif/quiver/blob/main/install.ps1) before you pipe them into a shell, or install with npm, Go or an archive instead.

npm packages are published from CI with npm trusted publishing and provenance; no npm token is stored.

## Network access

Quiver makes network requests only for:

- git operations on the sources and the sync remote you configure,
- a daily check of the latest release on GitHub, off with `QUIVER_NO_UPDATE_CHECK=1`.

There is no telemetry.

## Reporting a vulnerability

Do not open a public issue. Report it privately through GitHub: open the [Security tab](https://github.com/kheif/quiver/security) of the repository and choose **Report a vulnerability**. Include the Quiver version (`quiver --version`), your system and the steps to reproduce. You will get an answer within a week.

Fixes go into the latest release only. The policy is in [SECURITY.md](https://github.com/kheif/quiver/blob/main/SECURITY.md).
