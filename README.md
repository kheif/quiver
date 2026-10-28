# Quiver

A central skill manager for AI assistants, delivered as an MCP server.

Quiver keeps all your skills in one library, `~/.quiver`: your own skills and skills from git repositories. You decide which skills are **global** (served in every folder) and which belong to a **project**. The assistant starts Quiver's MCP server, sees the skills of the folder it works in, and loads the ones a task needs.

Skills are never copied into projects. A project only gets a small `.quiver.json` with its ID.

Skills use the standard Agent Skills format: a folder with a `SKILL.md` (YAML frontmatter with `name` and `description`) and any extra files. Existing skills and skill repositories work as they are.

## Install

Quiver is a single binary. It needs `git` on your `PATH`.

**macOS and Linux:**

```sh
curl -fsSL https://raw.githubusercontent.com/kheif/quiver/main/install.sh | sh
```

The script downloads the latest release for your system, checks its checksum and installs `quiver` into `~/.local/bin`. Set `QUIVER_VERSION=v1.0.0` for a given release, or `QUIVER_INSTALL_DIR` for another folder. Run it again to update.

**npm (macOS, Linux and Windows):**

```sh
npm install -g @kheif/quiver
```

npm installs the binary for your platform; no install script runs. If you switch Node versions with nvm, fnm or volta, run `quiver mcp install` again afterwards, because the path of `quiver` changes with the Node version.

**Go:**

```sh
go install github.com/kheif/quiver/cmd/quiver@latest
```

The archives for every platform are also on the [releases page](https://github.com/kheif/quiver/releases). To build from a clone: `make build`, which writes `bin/quiver`.

## Quickstart

```sh
# 1. Create the library in ~/.quiver (any other command does this too)
quiver init

# 2. Register the MCP server with Claude Code, once for all projects
quiver mcp install

# 3. In a project folder, write a skill for that project
cd ~/code/app
quiver new release-notes
```

`quiver new` opens the new `SKILL.md` in `$VISUAL` or `$EDITOR`. Write one sentence in `description` on when the assistant should use the skill, and the instructions below the frontmatter. Quiver commits the skill in `~/.quiver/local` and creates `~/code/app/.quiver.json`. Commit that file, so the project keeps its ID in every clone.

Then start Claude Code in `~/code/app` and ask for the task, such as "write the release notes for this version". The assistant sees `release-notes` in Quiver's skill list and loads it. You can also run it yourself as `/quiver:release-notes`.

`quiver status` shows what the assistant gets in the current folder.

## Concepts

- **Library.** `~/.quiver/local/` holds your own skills, in a git repository Quiver commits to. `~/.quiver/sources/` holds cloned skill repositories. Set `QUIVER_HOME` to use another folder.
- **Skill IDs.** A local skill's ID is its folder name, such as `release-notes`. A skill from a source is `<source>/<skill>`, such as `anthropic/pdf`.
- **Assignments.** `registry.json` stores which skills are global and which projects get which skills. A folder that is not a project gets only the global skills.
- **Assignments are yours.** They live in your `~/.quiver`, not in the project. A teammate who clones the project gets its `.quiver.json`, but not your skills or assignments. To share skills, put them in a git repository that everyone adds with `quiver source add`.
- **Projects.** A project is the folder with `.quiver.json`, found by walking up from the current folder. It is created the first time you assign a skill there. The ID survives moves and renames. Git worktrees of one repository share the committed `.quiver.json`, so they are one project with the same skills. If a folder is copied, both copies have the same ID; `quiver status` warns, and `quiver projects split` gives the copy its own.
- **Always-on skills.** A skill added with `--always` is loaded before the assistant's first reply, for style or rules that apply to every answer. See the limits below.

## Commands

Run `quiver help <command>` for the details and examples of each.

| Command | What it does |
|---|---|
| `quiver init` | Create the Quiver home and anything missing in it. |
| `quiver new <name>` | Create a local skill and assign it to this project (`--global`: every folder, `--none`: nowhere). |
| `quiver edit <id>` | Open a local skill in your editor, then check and commit it. |
| `quiver delete <id>` | Delete a local skill from the library. An assigned skill needs `--force`. |
| `quiver import [<dir>]` | Copy the skills in a project's `.claude/skills` into the library (`--deep`: every `SKILL.md`). |
| `quiver add <id>...` | Assign skills to this project, or to every folder with `--global`. `--always` makes them always-on. |
| `quiver remove <id>...` | Unassign skills. They stay in the library. |
| `quiver list` | List the library's skills and where each is assigned (`--source`, `--unassigned`, `--json`). |
| `quiver status` | Show the project in this folder and the skills served here. |
| `quiver where <id>` | Show where one skill is assigned. |
| `quiver source add <repo>` | Clone a skill repository: `owner/repo` (GitHub), a git URL or a local path. |
| `quiver source list` | List sources with their skill count and commit. |
| `quiver source remove <name>` | Delete a source. Refuses while its skills are assigned, unless `--force`. |
| `quiver update [<name>...]` | Pull the latest skills of all or some sources, and list what changed. |
| `quiver projects` | List projects; `rename`, `forget` and `split` manage them. |
| `quiver doctor` | Check the library, the assignments and the Claude Code setup (`--fix` applies the safe fixes). |
| `quiver mcp install` | Register the MCP server with Claude Code (`--print` shows the setup for other assistants). |
| `quiver mcp uninstall` | Remove it from Claude Code. |
| `quiver mcp` | Run the MCP server. Assistants start this. |
| `quiver completion <shell>` | Print the completion script for `bash`, `zsh`, `fish` or `powershell`. |

Shell completion suggests real skill IDs, sources and projects. For zsh, for example:

```sh
quiver completion zsh > "${fpath[1]}/_quiver"
```

`quiver help completion` shows the setup for each shell.

## MCP server

`quiver mcp install` runs `claude mcp add --scope user quiver -- <path to quiver> mcp`. For an assistant configured with JSON, `quiver mcp install --print` prints the entry:

```json
{ "mcpServers": { "quiver": { "command": "/path/to/quiver", "args": ["mcp"] } } }
```

The server serves the project of the folder it is started in, or of `$QUIVER_PROJECT` when that is set. It logs to `~/.quiver/logs/`.

Claude Code starts the server in the project folder. Other assistants may start it somewhere else, such as your home folder; there it serves only the global skills. Set `QUIVER_PROJECT` in that assistant's MCP config to serve a project's skills.

**Instructions.** When a session starts, the server tells the assistant which skills exist here: their IDs and descriptions, the always-on skills, and that new skills are saved with `create_skill`.

**Tools:**

| Tool | Input | Result |
|---|---|---|
| `list_skills` | — | The skills served here: ID, name, description, and `global` or `project`. |
| `get_skill` | `id` | The skill's instructions, its extra files and its folder. |
| `get_skill_file` | `id`, `path` | One extra file of a skill, such as `references/checklist.md`. Large and binary files return their path instead. |
| `create_skill` | `name`, `description`, `body`, `scope` (`project` or `global`), optional `files` | Creates a local skill, commits it and assigns it. In a folder that is not a project yet, `project` scope registers it. |

**Prompts.** Every served skill is also an MCP prompt. In Claude Code it appears as `/quiver:<id>`; text after the command is passed to the skill as the task.

**Live updates.** While a session runs, the server watches the library. Skills you add, remove or change show up in `list_skills` and the prompts right away. The skill list in the instructions is built once, so it changes from the next session on.

## Always-on skills: limits

- Changes take effect in the next session, because the instructions cannot change during a session.
- Only the skill's text is delivered. Plugin hooks, such as a per-message reminder, are not run.
- Subagents are not reliable: in testing, one of two subagents followed the always-on skill.

## Security

Skills give instructions to your assistant, and a skill can include scripts the assistant may run. A skill from a source is code you run on your own machine.

- Add only sources you trust, and read their skills before you assign them.
- `quiver update` prints the old and new commit of each source, and the skills that were added, changed or removed. To read the changes of a source you do not control: `git -C ~/.quiver/sources/<name> diff <old> <new>`.
- Source skills are read-only in Quiver. To change one, copy it into your own skills.
- `create_skill` lets the assistant write skills into your library. A global skill's description reaches the assistant in every folder, so review new global skills with `quiver list` and `quiver edit`.

## Uninstall

The last command deletes your own skills too. To keep them, first push `~/.quiver/local` somewhere: it is a git repository.

```sh
quiver mcp uninstall      # remove the MCP server from Claude Code
rm "$(command -v quiver)" # remove the binary; with npm: npm uninstall -g @kheif/quiver
rm -rf ~/.quiver          # delete the library
```

Projects keep their `.quiver.json`; delete it there if you no longer need it.

## Development

```sh
make build   # bin/quiver
make test    # go test ./..., including the end-to-end suite in e2e/
make lint    # golangci-lint
```

See [PROJECT.md](PROJECT.md) for the design and [tasks/](tasks/README.md) for the plan.
