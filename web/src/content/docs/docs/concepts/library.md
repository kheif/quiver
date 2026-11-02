---
title: The library
description: Where Quiver keeps skills, how local skills and source skills differ, and how skill IDs are formed.
---

The library is the folder `~/.quiver`, called the Quiver home. Every skill is stored there once, whether you wrote it or it comes from a git repository. Set `QUIVER_HOME` to use another folder.

## Local skills and source skills

| Property | Local skills | Source skills |
| --- | --- | --- |
| Folder | `~/.quiver/local/<name>/` | `~/.quiver/sources/<source>/…` |
| Where they come from | `quiver new`, `quiver import`, `quiver fork`, the `create_skill` tool | `quiver source add` |
| Editable | Yes: `quiver edit`, the `update_skill` tool | No: read-only, `quiver update` replaces them |
| History | Each change is a commit in `local/`, a git repository | The source repository's own history |
| ID | The folder name: `release-notes` | `<source>/<skill>`: `anthropic/pdf` |

`local/` is a git repository that Quiver commits to. Every change to a local skill, from the CLI or from the assistant, is a commit. `git -C ~/.quiver/local log -p` shows them and `git revert` undoes one. After [`quiver sync init`](/docs/guides/multiple-machines/), the repository is `~/.quiver` itself.

To change a source skill, fork it into your own skills with `quiver fork`. See [Updating and pinning](/docs/guides/updating-and-pinning/).

## Skill files

A skill is a folder with a `SKILL.md`. Its YAML frontmatter needs:

- `name`: lowercase letters, digits and dashes, starting with a letter or a digit. Up to 64 characters.
- `description`: when the assistant should use the skill. Up to 1024 characters.

Other frontmatter fields are kept as they are. Any other file in the folder is an extra file of the skill, such as `references/checklist.md` or a script; the assistant reads them with `get_skill_file`.

Quiver also imports a skill written as one markdown file of any name, such as `review.md`, when its frontmatter has a `description`.

## Skill IDs

- A local skill's ID is its folder name: `release-notes`.
- A skill from a source is prefixed with the source's name: `anthropic/pdf`.

IDs are what you pass to `quiver add`, `quiver show` and the other commands, and what the assistant passes to `get_skill`.

## Looking at the library

```sh
quiver list                 # every skill and where it is assigned
quiver list --unassigned    # skills not assigned anywhere
quiver list --source local  # only your own skills
quiver show anthropic/pdf   # one skill: origin, folder, assignments, files, SKILL.md
quiver search "pull request" # IDs, names and descriptions of every skill
```

`list`, `show`, `search`, `status` and `where` take `--json` for scripts.

See [Configuration and file layout](/docs/reference/files/) for every file in the Quiver home.
