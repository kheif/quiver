# Data Model

File formats and identifiers used by every task. The only file Quiver puts into a project is `.quiver.json`, and it holds just the project's ID.

## `~/.quiver/config.json`

The user's settings.

```json
{
  "version": 1,
  "sources": {
    "anthropic": { "repo": "https://github.com/anthropics/skills.git" },
    "team": { "repo": "git@github.com:my-org/skills.git" }
  }
}
```

## `~/.quiver/registry.json`

Assignments: which skills are global, and which projects get which skills.

```json
{
  "version": 1,
  "global": ["caveman/caveman", "commit-rules", "anthropic/pdf"],
  "alwaysOn": ["caveman/caveman"],
  "projects": {
    "app-k3f9q2": {
      "name": "app",
      "paths": ["/Users/me/code/app"],
      "skills": ["db-migration", "hga-expert"],
      "alwaysOn": []
    },
    "tool-8xw2mf": {
      "name": "tool",
      "paths": ["/Users/me/scratch/tool"],
      "skills": ["quick-notes"]
    }
  }
}
```

`alwaysOn` lists skills that apply from the first message. The MCP server's `instructions` name them and tell the assistant to load them with `get_skill` before its first reply. Every ID in an `alwaysOn` list must also be in the matching `global` or `skills` list.

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
- `paths` lists the folders where this ID has been seen. It is only a hint and can differ between machines.
- **Moved folder:** the ID shows up at a new path and the old path is gone. Quiver updates `paths`.
- **Copied folder:** the same ID shows up at two existing paths. Quiver warns, and `quiver projects split` gives the copy its own ID.

## Skill ID

- Local skills: the folder name inside `local/`, such as `db-migration`.
- Remote skills: `<source>/<name>`, such as `anthropic/pdf`.
- Collisions inside one source: `<source>/<plugin>/<name>`.

## Skills served in a project

The global skills plus the project's own skills, without duplicates. If two served skills share the same `name`, the project skill wins, and `doctor` reports the clash.
