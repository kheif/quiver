---
title: Projects
description: How .quiver.json identifies a project, and what happens when projects are moved, copied, cloned or forgotten.
---

A project is a folder with a `.quiver.json` file. Quiver finds it by walking up from the current folder, the same way git finds a repository.

```json title=".quiver.json"
{ "id": "app-6ts23a" }
```

The file holds only the ID. Which skills the project gets is stored in your `~/.quiver/registry.json`.

## Creating a project

A folder becomes a project the first time you assign a skill there, with `quiver add`, `quiver new` or `quiver import`, or when the assistant creates a project skill with `create_skill`. In a terminal, the commands ask first:

```text
Register ~/code/app as a project?
```

`--yes` skips the question. The project does not need git.

## Commit the file

If the project uses git, commit `.quiver.json`. Every clone then has the same ID, and Quiver recognizes it on your other machines too.

Git worktrees of one repository share the committed `.quiver.json`, so they are one project with the same skills.

## Moves, renames and copies

- **Moving or renaming** the folder keeps the project: the ID travels with the file.
- **Copying** the folder copies the ID, so both copies are one project. `quiver status` warns about it. Run `quiver projects split` in the copy to give it its own ID; it starts with the skills of the original.

```sh
cp -R ~/code/app ~/code/app-v2
cd ~/code/app-v2
quiver projects split
```

## Managing projects

```sh
quiver projects                          # name, ID, skill count and folders
quiver projects rename app-6ts23a web    # change the name Quiver shows
quiver projects forget old-site          # remove the project and its assignments
```

The name is only a label. A forgotten project's `.quiver.json` is left alone; using the folder again registers it again, with the same ID and no skills.

The folders where each project was seen are machine-local, in `~/.quiver/state/projects.json`, so the same project can live in different folders on different machines.

## Serving a project from elsewhere

The MCP server serves the project of the folder it starts in. If an assistant starts it somewhere else, set `QUIVER_PROJECT` to the project folder in that assistant's MCP entry. `quiver status` honors `QUIVER_PROJECT` too:

```sh
QUIVER_PROJECT=~/code/api quiver status
```
