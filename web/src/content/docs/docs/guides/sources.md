---
title: Managing sources
description: Add skills from GitHub, any git URL or a local folder, share your own skills, and remove sources.
---

A source is a git repository of skills. Quiver clones it into `~/.quiver/sources/<name>` and serves its skills as `<name>/<skill>`.

:::caution[A source is code you run]
Skills give instructions to your assistant and can include scripts it may run. A skill from a source is code from someone else, on your machine. Add only sources you trust, and read their skills before you assign them. See [Security](/docs/project/security/).
:::

## Add a source

```sh
quiver source add anthropics/skills --name anthropic
```

The repository can be:

- `owner/repo` on GitHub, or `host/owner/repo`,
- any git URL, such as `git@github.com:acme/skills.git`,
- a local path, such as `~/code/team-skills`.

The source name defaults to the repository owner; `--name` sets another one. Quiver lists the skills it found:

```text
Cloning https://github.com/anthropics/skills.git as source "anthropic"...
Added source "anthropic" with 19 skills:
  anthropic/docx
  anthropic/pdf
  …
Note: skills give instructions to your assistant and may contain scripts. Review them before you assign them.
```

Nothing is assigned yet. Read a skill, then assign it:

```sh
quiver show anthropic/pdf
quiver add anthropic/pdf --global
```

## How skills are found

- If the repository has `.claude-plugin/marketplace.json`, a Claude Code plugin marketplace, Quiver reads the skills of its plugins from it.
- Otherwise it collects every folder with a `SKILL.md`.

## List and remove sources

```sh
quiver source list            # name, skill count, commit in use and its date, repository
quiver source remove acme     # refuses while its skills are assigned
quiver source remove acme --force # unassigns its skills first
```

## Keep sources current

`quiver update` downloads new commits, shows what changes in your assigned skills, and asks before it applies them. See [Updating and pinning](/docs/guides/updating-and-pinning/).

## Share your own skills

`~/.quiver/local` is a git repository. Push it to GitHub, and others can add it as a source:

```sh
git -C ~/.quiver/local remote add origin git@github.com:me/skills.git
git -C ~/.quiver/local push -u origin HEAD
```

```sh
# on a teammate's machine
quiver source add me/skills
```

For a team, keep shared skills in one repository that everyone adds as a source. Each person assigns the skills they want; assignments are not shared.

To keep your own library the same on several machines, use [`quiver sync`](/docs/guides/multiple-machines/) instead.
