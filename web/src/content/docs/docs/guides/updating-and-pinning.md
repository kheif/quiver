---
title: Updating and pinning
description: Review and apply updates from sources, pin a source skill to a commit, or fork it to change it.
sidebar:
  badge:
    text: Next
    variant: default
---

:::note[Next release]
The update preview, `quiver pin`, `quiver unpin` and `quiver fork` are on the main branch and ship after v1.2.0. In v1.2.0, `quiver update` pulls first, then lists what changed.
:::

## Update sources

```sh
quiver update            # every source
quiver update anthropic  # some sources
```

For each source, `quiver update` first downloads the new commits without applying them, and lists the skills that would be added, changed or removed. For each **assigned** skill that changes, it lists the changed files, and marks scripts, such as `.sh`, `.py` and `.js` files and executable files, as **needs review**: the assistant may run them.

When an assigned skill would change or go away, it asks before it applies the changes. Answering no leaves the source as it was. Changes to skills that are not assigned are applied right away. Pinned skills are left alone.

| Flag | Effect |
| --- | --- |
| `--diff` | Show the whole change of each assigned skill that changes. |
| `--yes` | Apply the changes without asking. |
| `--reset` | Take the remote history of a source that was force-pushed. |

A source whose history was rewritten is not updated until you run it with `--reset`.

## Pin a skill to a commit

A pin keeps a source skill at one commit, so its text changes only when you decide:

```sh
quiver pin anthropic/pdf          # at the source's current commit
quiver pin acme/review 3f2a9c1    # at a given hash, tag or branch
```

The pin applies everywhere the skill is assigned, and the skill keeps its ID. The pinned commit is checked out into `~/.quiver/sources/.pins`; an older commit is downloaded first when needed.

To review changes before they reach your sessions: pin the skill, read the changes with `quiver update --diff`, then move the pin to the new commit.

```sh
quiver unpin anthropic/pdf   # serve the source's current version again
```

## Fork a skill to change it

Source skills are read-only. To change one, copy it into your own skills:

```sh
quiver fork anthropic/pdf
quiver fork acme/review --as team-review
```

The copy is named after the skill, or `--as`. Its frontmatter records where it came from, as `forkedFrom: <id>@<commit>`. Wherever the original is assigned, globally or in a project, the copy takes its place, always-on marks included.

From then on `quiver edit` changes what those projects get, and `quiver update` no longer does.
