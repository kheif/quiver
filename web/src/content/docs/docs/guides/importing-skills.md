---
title: Importing skills
description: Move existing skills from .claude/skills, a folder or a single file into the Quiver library.
---

`quiver import` moves skills you already have into the library and assigns them. Afterwards the assistant gets them from Quiver, in every project you assign them to.

## What it imports

```sh
quiver import                 # the .claude/skills of this project
quiver import --deep          # every folder of this project
quiver import ~/skill-pack    # every skill in a folder and its subfolders
quiver import notes/review.md # one skill: its SKILL.md, its folder or a markdown file
```

A skill is a folder with a `SKILL.md`, or a markdown file of any name whose frontmatter has a `description`. Its name is the one in the frontmatter, or the folder or file name. Markdown files without frontmatter, such as READMEs, are not skills. Skills without a description are skipped and listed at the end.

`quiver init` offers the two most common imports: `~/.claude/skills` for every folder, and the current project's `.claude/skills`.

## Where imported skills are assigned

For each skill, `quiver import` asks where to assign it, and asks for a new name if the name is taken.

| Flag | Effect |
| --- | --- |
| `--global` | Make every folder the default instead of the project. |
| `--name <name>` | Import one skill under another name. |
| `--yes` | Ask nothing. Taken names are skipped. |
| `--keep` | Keep every original. |

```sh
quiver import ~/Downloads/greeter/SKILL.md --name demo
```

## What happens to the originals

A skill that Quiver serves and that Claude Code also loads by itself, from `~/.claude/skills`, the project's `.claude/skills` or an enabled plugin, may be loaded twice. So after a skill is imported, its original is deleted; the copy in the library is committed in git.

An original committed in a project's git is kept, unless you say so, because teammates who do not use Quiver may still need it. `--yes` keeps those too. `--keep` keeps every original.

## Finding copies later

`quiver status` and `quiver doctor` report copies of served skills that Claude Code also loads, by name or by content.

```sh
quiver doctor --fix
```

This deletes the copies that are not committed in git and not part of a plugin, after asking. Copies committed in git and copies in plugins are only reported.
