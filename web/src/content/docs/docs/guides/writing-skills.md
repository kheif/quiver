---
title: Writing skills
description: Create, edit and delete your own skills with the CLI or from a conversation with your assistant.
---

Your own skills live in `~/.quiver/local`, a git repository. Quiver commits every change.

## Create a skill

```sh
cd ~/code/app
quiver new release-notes
```

`quiver new` creates `~/.quiver/local/release-notes/SKILL.md`, opens it in `$VISUAL` or `$EDITOR`, then checks and commits it. By default the skill is assigned to the project in the current folder.

| Flag | Effect |
| --- | --- |
| `--global` | Serve it in every folder instead. |
| `--none` | Create it without assigning it. |
| `--project <folder>` | Assign it to another project. |
| `--description "<text>"` | Set the description and skip the editor. |
| `--yes` | Register a new project without asking. |

```sh
quiver new commit-style --global --description "Write commit messages in our style."
```

Names use lowercase letters, digits and dashes, and start with a letter or digit.

## Write a good description

The description is all the assistant sees before it decides to load a skill. Write one sentence on **when** to use it, in the words a request would use:

```yaml
description: Write release notes from the commits since the last tag. Use when asked for release notes, a changelog entry or a version summary.
```

The instructions below the frontmatter are read only after the skill is loaded, so they can be as long as the task needs. Put long material in extra files, such as `references/checklist.md`, and point to them from `SKILL.md`; the assistant reads them with `get_skill_file`.

## Edit a skill

```sh
quiver edit release-notes
```

When the editor closes, Quiver checks the skill and commits the change. Running sessions see the new text the next time the assistant loads the skill.

Source skills are read-only, because `quiver update` would overwrite edits. Fork one to change it:

```sh
quiver fork anthropic/pdf
```

See [Updating and pinning](/docs/guides/updating-and-pinning/#fork-a-skill-to-change-it).

## Delete a skill

```sh
quiver delete old-draft
```

It asks first and commits the removal, so git history still has the skill. An assigned skill needs `--force`, which also unassigns it everywhere. To take a skill away from one project only, use `quiver remove`.

## Let the assistant write skills

While you work, you can ask the assistant to keep what you did:

> "Turn what we just did into a skill. It's only for this project."

> "Write a skill for our commit message rules and make it global."

The assistant calls the `create_skill` tool with the matching scope. The skill is saved in `~/.quiver/local` and committed, never in the project, so it cannot be lost with it. In a folder that is not a project yet, a project skill registers it.

The assistant can also change a local skill with `update_skill`. Each change is a commit named `update <name> (via MCP)`.

:::caution[Review what the assistant writes]
A global skill's description reaches the assistant in every folder. Review new global skills with `quiver list` and `quiver show`, and change them with `quiver edit`. `git -C ~/.quiver/local log -p` shows every change, and `git revert` undoes one.
:::
