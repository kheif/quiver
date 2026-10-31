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

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/kheif/quiver/main/install.ps1 | iex
```

The same steps for Windows: it installs `quiver.exe` into `%LOCALAPPDATA%\Programs\quiver` and adds that folder to your user `PATH`, without admin rights; open a new terminal afterwards. `QUIVER_VERSION` and `QUIVER_INSTALL_DIR` work the same way, set with `$env:QUIVER_INSTALL_DIR = '...'` before the line.

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

**New versions.** Once a day, a command run in a terminal asks GitHub for the latest release, while it runs, and remembers the answer in `~/.quiver/state/version-check.json`. When a newer version is out, the command ends with one line on stderr, with the install command that matches how you installed Quiver. Nothing is shown for `quiver mcp`, with `--json`, outside a terminal or for development builds, and a failed check is silent. `quiver doctor` shows the latest version. Set `QUIVER_NO_UPDATE_CHECK=1` to turn the check off.

## Quickstart

```sh
# 1. Create the library in ~/.quiver, register the MCP server with your
#    assistants and import the skills you already have
quiver init

# 2. In a project folder, write a skill for that project
cd ~/code/app
quiver new release-notes
```

In a terminal, `quiver init` asks before each step: registering the MCP server (`quiver mcp install`), importing the skills in `~/.claude/skills` for every folder, and importing those in the current project's `.claude/skills`. Any step can be skipped and run later. `quiver init --yes`, or a run outside a terminal, only creates `~/.quiver`; any other command creates it too.

`quiver new` opens the new `SKILL.md` in `$VISUAL` or `$EDITOR`. Write one sentence in `description` on when the assistant should use the skill, and the instructions below the frontmatter. Quiver commits the skill in `~/.quiver/local` and creates `~/code/app/.quiver.json`. Commit that file, so the project keeps its ID in every clone.

Then start Claude Code in `~/code/app` and ask for the task, such as "write the release notes for this version". The assistant sees `release-notes` in Quiver's skill list and loads it. You can also run it yourself as `/quiver:release-notes`.

`quiver status` shows what the assistant gets in the current folder. `quiver` alone sums it up in three lines, with a next step, before the command list.

## Concepts

- **Library.** `~/.quiver/local/` holds your own skills, in a git repository Quiver commits to. `~/.quiver/sources/` holds cloned skill repositories. Set `QUIVER_HOME` to use another folder.
- **Skill files.** A skill is a folder with a `SKILL.md`. Quiver also imports a skill written as one markdown file of any name, such as `review.md`, when its frontmatter has a `description`.
- **Skill IDs.** A local skill's ID is its folder name, such as `release-notes`. A skill from a source is `<source>/<skill>`, such as `anthropic/pdf`.
- **Assignments.** `registry.json` stores which skills are global and which projects get which skills. A folder that is not a project gets only the global skills.
- **Assignments are yours.** They live in your `~/.quiver`, not in the project. A teammate who clones the project gets its `.quiver.json`, but not your skills or assignments. To share skills, put them in a git repository that everyone adds with `quiver source add`.
- **Projects.** A project is the folder with `.quiver.json`, found by walking up from the current folder. It is created the first time you assign a skill there; in a terminal, `add`, `new` and `import` first ask `Register <folder> as a project?` (`--yes` skips the question). The ID survives moves and renames. Git worktrees of one repository share the committed `.quiver.json`, so they are one project with the same skills. If a folder is copied, both copies have the same ID; `quiver status` warns, and `quiver projects split` gives the copy its own.
- **One copy of each skill.** A skill that Quiver serves and that Claude Code also loads by itself, from `~/.claude/skills`, the project's `.claude/skills` or an enabled plugin, may be loaded twice. `quiver import` therefore deletes the originals it copied, except those committed in a project's git, which teammates may still need (`--keep` keeps every original). `quiver status` and `quiver doctor` report the copies they find, by name or by content, and `quiver doctor --fix` deletes those not committed in git and not in a plugin.
- **Always-on skills.** A skill added with `--always` is loaded before the assistant's first reply, for style or rules that apply to every answer. See the limits below.

## Commands

Run `quiver help <command>` for the details and examples of each.

| Command | What it does |
|---|---|
| `quiver` | Show the project here, how many skills it is served and a next step, then the command list. |
| `quiver init` | Create the Quiver home and anything missing in it; in a terminal, then offer to register the MCP server and import existing skills (`--yes`: only create the home). |
| `quiver new <name>` | Create a local skill and assign it to this project (`--global`: every folder, `--none`: nowhere). |
| `quiver edit <id>` | Open a local skill in your editor, then check and commit it. |
| `quiver delete <id>` | Delete a local skill from the library. An assigned skill needs `--force`. |
| `quiver import` | Move the skills in this project's `.claude/skills` into the library (`--deep`: every folder of the project). Originals committed in git are kept; `--keep` keeps all. |
| `quiver import <folder>` | Move every skill in a folder and its subfolders into the library. Files that are not valid skills are listed at the end. |
| `quiver import <skill>` | Move one skill: its `SKILL.md`, its folder or a markdown file with frontmatter. `--name` renames it. |
| `quiver add <id>...` | Assign skills to this project, or to every folder with `--global`. `--always` makes them always-on. |
| `quiver remove <id>...` | Unassign skills. They stay in the library. |
| `quiver list` | List the library's skills and where each is assigned (`--source`, `--unassigned`, `--json`). |
| `quiver show <id>` | Show a skill: where it comes from, where it is assigned, its files and its `SKILL.md` (`--json`). |
| `quiver search <text>` | Search the IDs, names and descriptions of every library skill, also unassigned ones, and show whether each is served here (`--json`). |
| `quiver status` | Show the project in this folder and the skills served here, with a warning for each copy Claude Code also loads (`--json`). |
| `quiver where <id>` | Show where one skill is assigned (`--json`). |
| `quiver source add <repo>` | Clone a skill repository: `owner/repo` (GitHub), a git URL or a local path. |
| `quiver source list` | List sources with their skill count and commit. |
| `quiver source remove <name>` | Delete a source. Refuses while its skills are assigned, unless `--force`. |
| `quiver update [<name>...]` | Pull the latest skills of all or some sources, and list what changed. |
| `quiver projects` | List projects (`--json`); `rename`, `forget` and `split` manage them. |
| `quiver doctor` | Check the library, the assignments, copies of served skills in Claude Code and its setup (`--fix` applies the safe fixes). |
| `quiver mcp install` | Register the MCP server with every installed assistant: Claude Code and Cursor (`--for <assistant>` for one, such as `claude-desktop`). |
| `quiver mcp uninstall` | Remove it from them (`--for <assistant>` for one). |
| `quiver mcp config` | Print the MCP config entry for any other assistant. |
| `quiver mcp` | Run the MCP server. Assistants start this. |
| `quiver completion <shell>` | Print the completion script for `bash`, `zsh`, `fish` or `powershell`. |

Shell completion suggests real skill IDs, sources and projects. For zsh, for example:

```sh
quiver completion zsh > "${fpath[1]}/_quiver"
```

`quiver help completion` shows the setup for each shell.

## MCP server

`quiver mcp install` sets up Claude Code and Cursor when it finds them, or only the assistant named with `--for`:

| Assistant | `--for` | What it changes |
|---|---|---|
| Claude Code | `claude-code` | Runs `claude mcp add --scope user quiver -- <path to quiver> mcp`. |
| Cursor | `cursor` | Adds `quiver` to `mcpServers` in `~/.cursor/mcp.json`. |
| Claude Desktop | `claude-desktop` | Adds `quiver` to `mcpServers` in `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows). Only with `--for claude-desktop`. |

An assistant counts as installed when the `claude` command is on `PATH`, or when its config folder exists. In the JSON files, every other key stays as it is, including an `env` of the `quiver` entry, and the old file is saved next to it as `<file>.quiver-backup`. A file that is not valid JSON is reported and never overwritten. Restart Cursor or Claude Desktop afterwards. `quiver doctor` checks the entry of each installed assistant (of Claude Desktop only when it has one). Cursor and Claude Desktop get the path of the binary itself, never the npm launcher, because desktop apps do not see your shell's `PATH` and so cannot find `node`. Cursor passes the skill list to the model and starts the server in the workspace folder, so project skills work there.

**Claude Desktop** runs Quiver's tools but does not pass the server's instructions to the model, so it never sees the skill list or the always-on skills. A skill is used only when you ask for Quiver's skills in the prompt, such as "check the quiver skills for this". That is why `quiver mcp install` sets it up only with `--for claude-desktop`. See [docs/spike-mcp-clients.md](docs/spike-mcp-clients.md).

For any other assistant, `quiver mcp config` prints the entry:

```json
{ "mcpServers": { "quiver": { "command": "/path/to/quiver", "args": ["mcp"] } } }
```

The server serves the project of the folder it is started in, or of `$QUIVER_PROJECT` when that is set. It logs to `~/.quiver/logs/`.

Claude Code starts the server in the project folder. Other assistants may start it somewhere else, such as your home folder; there it serves only the global skills. Cursor starts it in the workspace folder. Claude Desktop never starts it in a project. Set `QUIVER_PROJECT` in that assistant's MCP config to serve a project's skills.

**Instructions.** When a session starts, the server tells the assistant which skills exist here: their IDs and descriptions, the always-on skills, and that new skills are saved with `create_skill`.

**Tools:**

| Tool | Input | Result |
|---|---|---|
| `list_skills` | — | Every skill served here, project skills first: ID, name, whole description, `global` or `project`, and whether it is always-on. |
| `get_skill` | `id` | The skill's instructions, its extra files and its folder. |
| `get_skill_file` | `id`, `path` | One extra file of a skill, such as `references/checklist.md`. Large and binary files return their path instead. |
| `create_skill` | `name`, `description`, `body`, `scope` (`project` or `global`), optional `files` | Creates a local skill, commits it and assigns it. In a folder that is not a project yet, `project` scope registers it. |
| `update_skill` | `id`, any of `description`, `body`, `files` (an empty content deletes the file) | Changes a local skill and commits it as `update <name> (via MCP)`. Other frontmatter fields are kept. Source skills are read-only. |
| `search_skills` | `query` | Searches the whole library, also skills not assigned here, and marks each match as assigned here (`global` or `project`) or not. The assistant may suggest an unassigned skill, but asks before using it. |

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
- `create_skill` and `update_skill` let the assistant write skills into your library. A global skill's description reaches the assistant in every folder, so review new global skills with `quiver list` and `quiver edit`. Every change is a commit in `~/.quiver/local`, so `git -C ~/.quiver/local log -p` shows it and `git revert` undoes it.

To report a vulnerability in Quiver itself, see [SECURITY.md](SECURITY.md).

## Uninstall

The last command deletes your own skills too. To keep them, first push `~/.quiver/local` somewhere: it is a git repository.

```sh
quiver mcp uninstall      # remove the MCP server from every assistant
rm "$(command -v quiver)" # remove the binary; with npm: npm uninstall -g @kheif/quiver
rm -rf ~/.quiver          # delete the library
```

On Windows, after `quiver mcp uninstall`, delete `%LOCALAPPDATA%\Programs\quiver` and `%USERPROFILE%\.quiver`, and remove the folder from your user `PATH` in the environment variables settings.

Projects keep their `.quiver.json`; delete it there if you no longer need it.

## Development

```sh
make build   # bin/quiver
make test    # go test ./..., including the end-to-end suite in e2e/
make lint    # golangci-lint
```

See [CONTRIBUTING.md](CONTRIBUTING.md), [PROJECT.md](PROJECT.md) for the design and [tasks/](tasks/README.md) for the plan.

## License

Apache License 2.0. See [LICENSE](LICENSE).
