---
title: Introduction
description: 'What Quiver is, the problem it solves, and the three ideas it is built on: one library, layers and always-on skills.'
---

Quiver is a skill manager for AI assistants. It keeps your agent skills in one library and gives every folder the skills it needs, in Claude Code, Codex, Cursor and other MCP clients.

## The problem

Agent skills are instruction files that teach an assistant how your work is done: how you review a migration, write release notes or debug a crashing pod.

- Each assistant keeps them in its own place: Claude Code in `.claude/skills`, Cursor in its rules, Codex in `.codex/skills` and `AGENTS.md`.
- When another project needs a skill, it is copied by hand, and the copies drift apart.
- Archiving or deleting a project deletes its skills too.
- Skills shared on GitHub have to be found, copied and kept up to date by hand.

## Three ideas

**One library.** Every skill is stored once, in `~/.quiver`: your own in `local/`, a git repository Quiver commits to, and skills from git repositories, called *sources*, in `sources/`. Skills are never copied into projects; a project only gets a small `.quiver.json` file with its ID. See [The library](/docs/concepts/library/).

**Layers.** Every folder gets the skills of its layers: yours, which apply in every folder, and the project's. The team and company layers come with Quiver for teams, which is planned. See [Layers](/docs/concepts/layers/).

**Always-on skills.** Some skills should never be optional: security rules, team standards, the way you write. An always-on skill is in the conversation before the first reply. See [Always-on skills](/docs/concepts/always-on/).

## How it fits together

1. **Library.** You write skills, import the ones you have, or add a git repository of skills.
2. **Assignments.** You put each skill on a layer: global for you, or one project, and optionally always on. Assignments live in your `~/.quiver`, not in the project.
3. **MCP server.** `quiver install` sets Quiver up for every assistant it finds. When a session starts, the server finds the project of its folder and serves its skills.
4. **The assistant decides.** The server tells the assistant each skill's ID and description. When a skill fits the task, the assistant loads it with the `get_skill` tool. Always-on skills are there from the start.

Read [How MCP delivery works](/docs/concepts/mcp-delivery/) for the details.

## Why not the assistant's own skills folder?

If you use one assistant and keep a handful of skills, `~/.claude/skills` and the project's `.claude/skills` may be all you need. Quiver is for when that stops being enough:

- **More than one assistant.** One library serves Claude Code, Codex, Cursor and other MCP clients; nothing is copied into each one's folder.
- **Skills that outlive projects.** A skill lives in your library, in git, not in a project folder that gets archived.
- **Skills from other people.** Sources are git repositories you add once and update with a review of what changed.
- **Rules that must apply.** Always-on skills are in the conversation from the start, not left to the model to pick.

`quiver import` moves the skills you already have into the library, so you do not start over.

## What else it does

- **Skills from any git repository:** a GitHub `owner/repo`, any git URL or a local folder. Repositories with a Claude Code plugin marketplace work as they are. See [Sources](/docs/guides/sources/).
- **Import the skills you have:** `quiver import` moves the skills in `.claude/skills` into the library, so the assistant does not load them twice. See [Importing skills](/docs/guides/importing-skills/).
- **Skills written from the conversation:** ask the assistant to keep what you just did, and it saves a skill with the `create_skill` tool, committed in your library.
- **Updates you review first:** `quiver update` shows what changed and asks before it applies anything; `quiver pin` keeps a skill at one commit, `quiver fork` makes your own copy. See [Updating and pinning](/docs/guides/updating-and-pinning/).
- **Every machine:** keep your library in a private git remote with `quiver sync`. See [Multiple machines](/docs/guides/multiple-machines/).
- **Check and repair:** `quiver doctor` finds broken setups and copies of skills an assistant would load twice.

## Skill format

Quiver uses the standard Agent Skills format: a folder with a `SKILL.md` file, whose YAML frontmatter has a `name` and a `description`, plus any extra files. Quiver does not define a format of its own, so existing skills and skill repositories work as they are.

```markdown title="release-notes/SKILL.md"
---
name: release-notes
description: Write release notes from the commits since the last tag.
---

# Release notes

1. List the commits since the last tag with `git log $(git describe --tags --abbrev=0)..HEAD`.
2. Group them into Features, Fixes and Other.
```

## What Quiver is not

- **Not a hosted service.** Quiver is a single binary and your library is a folder on your machine. There is no account. (For Quiver for teams, hosting by us is coming soon.)
- **Not a copy tool.** It never writes skills into project folders. The MCP server is the only way skills reach the assistant.
- **Not a script runner.** The assistant runs a skill's scripts itself, using the paths that `get_skill` returns.
- **Not a marketplace.** Sources are plain git repositories that you choose.

## Next steps

- [Install Quiver](/docs/installation/)
- [Follow the quickstart](/docs/quickstart/)
- [Connect your assistant](/docs/assistants/)
