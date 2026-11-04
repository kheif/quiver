---
title: Working across machines
description: Keep your skills, sources and assignments the same on every machine with quiver sync.
---

`quiver sync` keeps the Quiver home in a git remote, so each of your machines has the same local skills, sources and assignments.

**Before you start:** create an empty **private** repository for it, such as `me/quiver-home`. Your library and your list of projects go there.

## Set it up

Run this on every machine, with the same remote:

```sh
quiver sync init git@github.com:me/quiver-home.git
```

It turns `~/.quiver` into a git repository with that remote, and pushes it. The history of `local/` is kept; `local/` stops being a repository of its own.

When the remote already holds another machine's home, this machine's skills, sources and assignments are merged with it. If the same local skill differs on both machines, it stops and changes nothing.

## Sync

```sh
quiver sync
```

It commits the changes on this machine, pulls the others with rebase, and pushes.

- When `registry.json` or `config.json` changed on both machines, the changes are merged: a skill assigned on either machine stays assigned.
- Any other conflict stops the sync with nothing pulled or pushed, and says how to resolve it.
- Sources added on another machine are cloned afterwards.

### Changes that ask first

Whatever another machine pushed becomes live here, so `quiver sync` lists some changes and asks before it pulls them: a change that makes a skill always-on, a change to the files of an always-on skill, and a source that is added or changed. `quiver sync --yes` takes them without asking. Without a terminal and without `--yes`, `quiver sync` pulls nothing.

`quiver sync init` asks the same when it merges another machine's home, and takes `--yes` too. Use a private repository: whoever can push to it can change what your assistants are told.

:::note[Next release]
The question is new in the next release. In v1.3.0 `quiver sync` pulls what the remote holds without asking.
:::

## What is synced

| Synced | Not synced |
| --- | --- |
| `local/`: your skills | `sources/`: cloned again from `config.json` |
| `config.json`: your sources | `state/`: the folders where each project was seen |
| `registry.json`: your assignments | `logs/` and `cache/` |

Projects carry their ID in `.quiver.json`, so assignments carry over as they are, even when a project lives in a different folder on each machine.
