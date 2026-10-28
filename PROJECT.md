# Quiver

A central skill manager for AI assistants, delivered as an MCP server.

All skills live in one library. The library holds your own skills and skills from remote sources. Quiver decides which skills each project gets and serves them to the assistant over MCP. The assistant discovers them, loads the ones it needs into the conversation, and can create new ones.

Skills are never copied into projects. A project only gets a tiny `.quiver.json` that holds its ID.

## Problem

- Skills sit scattered inside projects. A good skill written for one project gets forgotten there.
- When another project needs the same skill, it gets copied by hand. The copies drift apart over time.
- Deleting or archiving a project deletes its skills too.
- Skills shared on GitHub have to be found, copied and kept up to date manually.
- Every assistant (Claude Code, Cursor, …) wants skills in its own folder layout.

## How It Works

```
          ┌──────────────────────────┐
          │    Quiver TUI / CLI      │   manage: which skills, which projects
          └────────────┬─────────────┘
                       │
          ┌────────────▼─────────────┐
          │      ~/.quiver/          │   library + assignments
          │  local/   sources/       │
          │  registry.json           │
          └────────────┬─────────────┘
                       │
          ┌────────────▼─────────────┐
          │    Quiver MCP server     │   serves the right skills per project
          └────────────┬─────────────┘
                       │  tools / prompts
     ┌─────────────────┼─────────────────┐
  Claude Code       Cursor          Claude Desktop …
```

1. **Library.** Every skill is stored once in `~/.quiver/`. A skill is either your own (`local/`) or comes from a remote git repository (`sources/`). Quiver treats both the same way.
2. **Assignments.** You decide which skills are **global** (available in every project) and which belong to specific **projects**. These assignments live in Quiver, not in the project.
3. **MCP server.** You add Quiver to your assistant once, globally. When a session starts, the server reads the project's `.quiver.json` and offers the global skills plus that project's skills.
4. **The assistant decides.** The server's `instructions` give it each skill's name and description. When a skill fits the task, the assistant calls a tool to load the skill into the conversation.

## Skill Format

Quiver uses the standard Agent Skills format: a folder with a `SKILL.md` file (YAML frontmatter with `name` and `description`) plus any extra files. Quiver does not invent a new format, so existing skills and skill repositories work as they are.

## Library Layout

```
~/.quiver/
├── config.json          # Remote sources and settings
├── registry.json        # Global skills, projects and their skills
├── local/               # Your own skills (a git repo)
│   ├── db-migration/
│   │   └── SKILL.md
│   └── hga-expert/
│       ├── SKILL.md
│       └── references/
└── sources/             # Remote sources, cloned and kept updated
    ├── anthropic/
    └── team/
```

**Skill IDs:**
- Local skills use their name: `db-migration`.
- Remote skills are prefixed with their source: `anthropic/pdf`.

## Projects

A project is identified by a small `.quiver.json` file in its root:

```json
{ "id": "app-k3f9q2" }
```

- The file holds only the ID. Which skills the project gets is stored in Quiver.
- It is created the first time you assign a skill to the project, from the TUI, the CLI or a conversation.
- The project does not need git. The ID survives moves and renames. If the project uses git, commit the file so clones are recognized too.
- A folder without `.quiver.json` gets only the global skills.
- If a project folder is copied, both copies have the same ID. Quiver notices this and `quiver projects split` gives the copy its own ID.

## MCP Interface

### Skill catalog

At session start, the server's `instructions` list the available skills (global and project) with their descriptions, and tell the assistant to load a fitting skill with `get_skill`. The assistant adds `instructions` to its system prompt, so it knows the skills from the first message.

Tool descriptions are not used for this. Claude Code defers MCP tools, so the model does not see their descriptions (see [docs/spike-b01-results.md](docs/spike-b01-results.md)). `instructions` are cut at about 4096 characters, so long catalogs are shortened and end with a hint to call `list_skills`.

### Tools

| Tool | Purpose |
|---|---|
| `list_skills` | Lists the skills available in this project (global and project), including ones assigned during the session. |
| `get_skill` | Returns a skill's `SKILL.md` content, plus a list of its extra files and the skill's folder path. |
| `get_skill_file` | Returns one extra file of a skill (`references/…`, templates, examples). |
| `create_skill` | Creates a new skill from the conversation. `scope: "project"` assigns it to the current project only; `scope: "global"` makes it available everywhere. |
| `update_skill` | Edits an existing local skill. Remote skills are read-only. |
| `search_skills` | Searches the whole library, including skills not assigned to this project. Lets the assistant suggest one. |

### Always-on skills

Some skills must apply from the first message, without the assistant deciding to load them, for example a communication style like caveman. Mark them as always-on:

```bash
quiver add caveman/caveman --global --always
```

The server's `instructions` name them and tell the assistant to load them with `get_skill` before its first reply. The full text is not put into `instructions`, because they are cut at about 4096 characters. Changes take effect in the next session. Plugin hooks are not run; only the skill text is delivered. Subagents may not follow always-on skills.

### Prompts

Every available skill is also exposed as an MCP prompt. In Claude Code, this means a skill can be invoked directly as a slash command, such as `/mcp__quiver__db-migration`.

### Live updates

When the library or the assignments change, the server notifies the assistant (`prompts/list_changed`), and `list_skills` returns the new list. The catalog in `instructions` is updated in the next session.

## Creating Skills From a Project

While working in a project, you can tell the assistant:

> "Turn what we just did into a skill. It's only for this project."

> "Write a skill for our commit message rules and make it global."

The assistant calls `create_skill` with the matching scope. The skill is saved into the central library (`local/`) and committed. It is never stored inside the project, so it cannot be lost with it.

A project skill can later be made global, or assigned to other projects, from the CLI.

## Remote Sources and Sharing

Skills can come from any git repository, including Claude Code plugin marketplaces:

```bash
quiver source add anthropics/skills
quiver source add my-org/skills --name team
quiver update
```

- **Discovery:** If a repository has `.claude-plugin/marketplace.json`, Quiver reads the plugins' skills from it. Otherwise it collects every folder with a `SKILL.md`.
- **Updates:** Sources are git clones. `quiver update` pulls them, and every project immediately gets the new version.
- **Sharing:** `~/.quiver/local/` is a git repository. Push it to GitHub and others can add it as a source. The same repository can also serve as a Claude Code marketplace.
- **Customizing:** Remote skills are read-only. `quiver fork anthropic/pdf` copies one into `local/`, where you can edit it.
- **Security:** Remote skills give instructions to your assistant and can include scripts. Add only sources you trust. `quiver update` lists what changed in each source after pulling.

## Interactive Screen (TUI)

Running `quiver` with no arguments opens an interactive screen:

- **Search** the whole library as you type, with a preview of each skill.
- **Toggle** a skill for the current project (`space`) or globally (`g`).
- **Browse** projects and sources.
- **Update notices.** While the screen is open, Quiver quietly checks the remote sources in the background, one at a time and only when online. If a source has changes, a small notice appears in the bottom-right corner: `anthropic: 3 skills updated — press u to pull`. Nothing is pulled without a key press.

## CLI

Every action in the TUI is also a command, for scripts and quick use:

```bash
# Setup
quiver init                         # Create ~/.quiver
quiver mcp install                  # Register the MCP server with Claude Code / Cursor

# Skills
quiver list                         # All skills, and where each one is assigned
quiver new <name>                   # Create a local skill
quiver edit <name>                  # Open a local skill in $EDITOR
quiver search <text>
quiver fork <source/skill>

# Assignments (current project unless --project is given)
quiver add <skill>... [--global] [--always]    # Assign to this project or globally; --always loads it every session
quiver remove <skill>... [--global] [--always] # Unassign (the skill stays in the library); --always removes only the mark
quiver status                       # What the MCP server serves in this project
quiver where <skill>                # Which projects use this skill
quiver projects                     # List projects; rename, forget or split them

# Sources
quiver source add <repo> [--name <n>]
quiver source list
quiver source remove <name>
quiver update [<source>...] [--reset]  # Pull sources; --reset takes a force-pushed history

# Maintenance
quiver import [<dir>]               # Pull existing skills (e.g. .claude/skills) into the library
quiver doctor                       # Check the library and fix problems
```

## Multiple Machines

`~/.quiver/` is meant to be versioned with git. `local/` holds your skills. `config.json` and `registry.json` hold sources and assignments. Projects carry their ID in `.quiver.json`, so assignments carry over to another machine as they are.

## Roadmap

**First usable version** (remote skills first)
- Remote sources: `source add`, `update`
- Global and project assignment, always-on skills
- MCP server with `list_skills` and `get_skill`, `mcp install`
- Goal: `quiver source add JuliusBrussee/caveman --name caveman` → `quiver add caveman/caveman --global --always` works

**v1**
- Library, registry, project identification
- CLI: `init`, `new`, `edit`, `list`, `add`, `remove`, `status`, `where`, `projects`, `import`, `doctor`
- MCP server: `list_skills`, `get_skill`, `get_skill_file`, `create_skill`, prompts, live updates
- Remote sources: `source add/list/remove`, `update`
- `mcp install` for Claude Code

**v2**
- `update_skill`, `search_skills`, `search`, `fork`
- Pinning remote skills to a commit
- Update preview with diffs
- Library sync between machines
- `mcp install` for more assistants
- Interactive TUI with search
- Background update checks with notices

**Later**
- Graphical app
- Team libraries with shared assignments

## Out of Scope

- Copying or linking skills into projects. The MCP server is the only delivery path, and `.quiver.json` is the only file in a project.
- Running scripts. The assistant runs a skill's scripts itself, using the paths that `get_skill` returns.
- Defining a new skill format.
- Hosting a central marketplace. Sources are plain git repositories.
