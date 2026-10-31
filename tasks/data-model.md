# Data Model

File formats and identifiers used by every task. The only file Quiver puts into a project is `.quiver.json`, and it holds just the project's ID.

## `~/.quiver/config.json`

The user's settings.

```json
{
  "version": 1,
  "sources": {
    "anthropic": {
      "repo": "https://github.com/anthropics/skills.git",
      "ids": { "pdf": "skills/pdf", "docx": "skills/docx" }
    },
    "team": { "repo": "git@github.com:my-org/skills.git" }
  }
}
```

`ids` maps each skill ID given out in the source, without the `<source>/` prefix, to the skill's folder in the repo. See Skill ID below. It is written when the source is added and by `quiver update`; a source without it uses the IDs discovery gives.

## `~/.quiver/registry.json`

Assignments: which skills are global, and which projects get which skills.

```json
{
  "version": 2,
  "global": ["caveman/caveman", "commit-rules", "anthropic/pdf"],
  "alwaysOn": ["caveman/caveman"],
  "projects": {
    "app-k3f9q2": {
      "name": "app",
      "skills": ["db-migration", "hga-expert"],
      "alwaysOn": []
    },
    "tool-8xw2mf": {
      "name": "tool",
      "skills": ["quick-notes"]
    }
  }
}
```

`alwaysOn` lists skills that apply from the first message. The MCP server's `instructions` name them and tell the assistant to load them with `get_skill` before its first reply. Every ID in an `alwaysOn` list must also be in the matching `global` or `skills` list.

The file holds only data that is the same on every machine, so it can be synced as it is. Version 1 also kept each project's `paths`. Quiver still reads version 1, and the first write moves the paths into `state/projects.json` and saves version 2. An older Quiver stops on version 2 with "update quiver".

## `~/.quiver/state/`

Machine-local data. It is never synced.

- `projects.json`: the folders where each project was seen on this machine. A project from a synced `registry.json` that this machine has not seen yet has no entry. Projects no longer in `registry.json` are dropped on the next write.
- `version-check.json`: the last check for a new Quiver version (TASK39).

```json
{
  "version": 1,
  "paths": {
    "app-k3f9q2": ["/Users/me/code/app"],
    "tool-8xw2mf": ["/Users/me/scratch/tool"]
  }
}
```

## `<project>/.quiver.json`

The project's identity. The project does not need git; if it uses git, commit this file.

```json
{ "id": "app-k3f9q2" }
```

- The file holds only the ID. Which skills the project gets is stored in `registry.json`, never in the project.
- It is created the first time a skill is assigned to the project, from the CLI, the TUI or `create_skill`.
- The ID is `<folder name>-<6 random characters>`. It does not change when the folder is moved or renamed.
- A folder without `.quiver.json` is not a project. It gets only global skills.

## Project registry fields

- `name` is a display name. It defaults to the folder name.
- `paths` lists the folders where this ID has been seen on this machine, from `state/projects.json`. It is only a hint.
- **Moved folder:** the ID shows up at a new path and the old path is gone. Quiver updates `paths`.
- **Copied folder:** the same ID shows up at two existing paths. Quiver warns, and `quiver projects split` gives the copy its own ID.

## Skill ID

- Local skills: the folder name inside `local/`, such as `db-migration`.
- Remote skills: `<source>/<name>`, such as `anthropic/pdf`.
- Collisions inside one source: `<source>/<plugin>/<name>`, or `<source>/<folder path>` when the skill has no plugin.
- A skill at a repo's root is named after the source: `<source>/<source>`.
- A remote skill keeps its ID: the source's `ids` in `config.json` remember which folder got which ID. When an update adds a skill whose ID is already given, only the new skill gets `<source>/<plugin>/<name>` or its folder path. The ID of a skill removed upstream stays taken, and `quiver update` and `doctor` report the skill as removed upstream.

## Forked skills

`quiver fork <source/skill>` copies a source skill into `local/` and adds `forkedFrom: <id>@<commit>` to its frontmatter, such as `forkedFrom: anthropic/pdf@3f2a…`. Wherever the original ID is assigned or marked always-on, the copy's ID replaces it.

## Skills served in a project

The global skills plus the project's own skills, without duplicates. If two served skills share the same `name`, the project skill wins, and `doctor` reports the clash.
