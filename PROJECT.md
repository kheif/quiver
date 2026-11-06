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
├── state/               # Machine-local: project folders, version check
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

At session start, the server's `instructions` list the available skills (global and project) with their descriptions, and tell the assistant to load a fitting skill with `get_skill`. The assistant adds `instructions` to its system prompt, so it knows the skills from the first message. Claude Code and Cursor do. Codex and Antigravity pass them only after the model looks for Quiver's tools, and Claude Desktop drops them, so there skills are used only on request (see [docs/spike-mcp-clients.md](docs/spike-mcp-clients.md)).

Tool descriptions are not used for this. Claude Code defers MCP tools, so the model does not see their descriptions (see [docs/spike-b01-results.md](docs/spike-b01-results.md)). `instructions` are cut at about 4096 characters. A catalog that does not fit is replaced by a pointer: before its first reply, the assistant calls `list_skills`, which returns every skill with its whole description.

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
- **Updates:** Sources are git clones. `quiver update` downloads them, shows what changes in each assigned skill and asks before it applies the changes; then every project gets the new version.
- **Sharing:** `~/.quiver/local/` is a git repository. Push it to GitHub and others can add it as a source. The same repository can also serve as a Claude Code marketplace.
- **Customizing:** Remote skills are read-only. `quiver fork anthropic/pdf` copies one into `local/`, where you can edit it.
- **Security:** Remote skills give instructions to your assistant and can include scripts. Add only sources you trust. `quiver update` lists the changed files of assigned skills before it applies them, marks scripts "needs review", and asks first.

## Interactive Panel (TUI)

Running `quiver` with no arguments in a terminal opens a management panel with four sections:

- **Skills.** `n` creates a skill, `i` imports skills from a folder, `e` edits one, `f` forks a source skill, `d` deletes a local one, `p` pins a source skill or unpins it. Four views: what is available here (global and project assignments together), the project's assignments, the global assignments, and the whole library. `/` searches the active view. The detail of a skill shows two independent settings, for the managed project and for every folder, each *Not assigned*, *On demand* or *Always on*.
- **Projects.** The registered projects with their folders and skills. Choose one to manage (the header then says `Managing: api`), rename it, forget it, or split a copied folder.
- **Sources.** `a` adds a source. The remote sources, which of them the project and the global scope actually use, and their update state. Checking a source shows the same preview as `quiver update` (`d` adds the patches, like `--diff`) and waits for a yes when an assigned skill changes; a rewritten history asks before it takes the remote one (`--reset`).
- **Health.** The checks of `quiver doctor`, how always-on skills reach each assistant, and `i`, `x`, `f` to set Quiver up in the assistants, remove it, or apply the safe fixes. A header badge counts the problems. Without a Quiver home, the panel opens on a first-run screen that sets it up.
- **Update notices.** While the panel is open, Quiver quietly checks the remote sources in the background, one at a time and only when online. If a source has changes, a small notice appears in the bottom-right corner: `anthropic: 3 skills updated — press u to pull`. Nothing is pulled without a key press.

Every action in the panel is also a command, and the panel runs the same code. `quiver sync` and the `--json` outputs stay in the CLI. Sections for activity and settings are planned (see Roadmap).

## CLI

Every action in the TUI is also a command, for scripts and quick use:

```bash
# Setup
quiver init                         # Create ~/.quiver; in a terminal, also register the MCP server and import existing skills
quiver install                      # Register the MCP server with every installed assistant
quiver                              # In a terminal: the interactive screen. Otherwise: the project here, its skill count and a next step

# Skills
quiver list                         # All skills, and where each one is assigned
quiver new <name>                   # Create a local skill
quiver edit <name>                  # Open a local skill in $EDITOR
quiver show <skill>                 # Where a skill comes from, where it is assigned, its SKILL.md
quiver search <text>
quiver fork <source/skill> [--as <name>]  # Copy a source skill into local/ to change it

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
quiver update [<source>...] [--diff] [--yes] [--reset]  # Preview, confirm, then pull; --reset takes a force-pushed history
quiver pin <source/skill> [<commit>]  # Keep a source skill at one commit; quiver unpin undoes it

# Maintenance
quiver sync init <git-remote>       # Keep ~/.quiver in a git remote; quiver sync syncs it
quiver import [<dir>]               # Pull existing skills (e.g. .claude/skills) into the library
quiver doctor                       # Check the library and fix problems
```

## Multiple Machines

`quiver sync init <git-remote>` turns `~/.quiver/` into a git repository, and `quiver sync` commits, pulls with rebase and pushes. `local/` holds your skills. `config.json` and `registry.json` hold sources and assignments; when both machines changed them, the changes are merged. Projects carry their ID in `.quiver.json`, so assignments carry over to another machine as they are. Sources are cloned again from `config.json`. `state/` holds what differs per machine, such as the folders where each project was seen, and is not synced.

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

**v1.2**
- `show`, `search`, every skill in the catalog: through `list_skills` when it does not fit in `instructions`
- `mcp install` for Cursor, and for Claude Desktop when named (it drops `instructions`)
- Import removes originals; `doctor` and `status` find native copies of served skills
- Guided first run, notice of a new Quiver version, `install.ps1` for Windows
- Git timeouts, dependency and vulnerability checks

**v1.3**
- Machine-local state out of `registry.json`, stable skill IDs
- `update_skill`, `search_skills`, `fork`
- Pinning remote skills to a commit
- Update preview with diffs
- Library sync between machines
- More assistants for `mcp install`, which sets up every supported assistant that is installed

**v1.4: Session-start delivery**
- Always-on skills added to the context by a session-start hook, in assistants that have one; the other skills stay on the MCP server
- `quiver install` replaces `quiver mcp install` and reports, per assistant, how always-on skills reach it
- VS Code Copilot
- A pointer in the rules file of assistants without hooks

**v1.5**
- Interactive panel with sections for skills, projects and sources
- Background update checks with notices
- Slash commands without the `quiver:` prefix in Claude Code

**Test phase** (before the enterprise track)
- End-to-end user journeys, Windows without skipped tests, versioned data formats with migration tests
- A benchmark of assistant behaviour, run before every release
- Fuzzing, parallel processes, performance budgets and a security review

**Enterprise**
- To be planned after the test phase (see Direction)

**Later**
- More panel pages: activity (hook runs) and settings
- Graphical app
- Search and a marketplace: people can find and add skills others published

## Direction

Quiver grows in two tracks.

- **Individual.** One person's skills, kept in the local library, or pushed to their own GitHub repository to keep them remote and in sync between machines. v1 is this track.
- **Enterprise.** A company keeps its skills and their structure in a central place, such as a GitHub repository or S3, under its own standards. Each developer's Quiver MCP pulls them from there. Later, a company can run its own Quiver server: developers' MCP connects to it, and it serves each developer only the skills they are allowed to use.

Decisions in the individual track should not block the enterprise one. Sources are git repositories today; other storage (such as S3) is a new source type. A company server needs MCP over HTTP instead of stdio, and identity and permissions on top.

## Out of Scope

- Copying or linking skills into projects. The MCP server is the only delivery path, and `.quiver.json` is the only file in a project.
- Running scripts. The assistant runs a skill's scripts itself, using the paths that `get_skill` returns.
- Defining a new skill format.
- Hosting a central marketplace. Sources are plain git repositories. Search and a marketplace may come later (see Roadmap).
