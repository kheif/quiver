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

npm installs the binary for your platform; no install script runs. If you switch Node versions with nvm, fnm or volta, run `quiver install` again afterwards, because the path of `quiver` changes with the Node version.

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

In a terminal, `quiver init` asks before each step: registering the MCP server (`quiver install`), importing the skills in `~/.claude/skills` for every folder, and importing those in the current project's `.claude/skills`. Any step can be skipped and run later. `quiver init --yes`, or a run outside a terminal, only creates `~/.quiver`; any other command creates it too.

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
| `quiver fork <source/skill>` | Copy a source skill into your own skills to change it; `--as` names the copy. The copy takes the original's place wherever it is assigned, and `quiver update` no longer changes it. |
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
| `quiver update [<name>...]` | Download the latest skills of all or some sources and show what would change: the changed files of each assigned skill, with scripts marked "needs review". Asks before it applies changes to assigned skills (`--yes` does not ask, `--diff` shows the whole change). Pinned skills stay at their commit. |
| `quiver pin <source/skill> [<commit>]` | Keep a source skill at its current commit, or the one given, so `quiver update` leaves it alone. `quiver unpin` goes back to the source's current version. |
| `quiver sync init <git-remote>` | Keep the Quiver home in a git remote, so your other machines get the same skills, sources and assignments. Run it on each machine; then `quiver sync` commits, pulls and pushes, merges assignment changes from both machines and clones sources added elsewhere. Before it pulls changes that make a skill always-on, change an always-on skill's files or add a source, it lists them and asks (`--yes` takes them). |
| `quiver projects` | List projects (`--json`); `rename`, `forget` and `split` manage them. |
| `quiver doctor` | Check the library, the assignments, copies of served skills in Claude Code and its setup (`--fix` applies the safe fixes). |
| `quiver install [assistant...]` | Register the MCP server and add the session-start hook where the assistant has one, in every supported assistant that is installed, or only in the assistants named. See [MCP server](#mcp-server). |
| `quiver uninstall [assistant...]` | Remove it from them, or only from the assistants named. |
| `quiver mcp config` | Print the MCP config entry for any other assistant. |
| `quiver mcp` | Run the MCP server. Assistants start this. |
| `quiver completion <shell>` | Print the completion script for `bash`, `zsh`, `fish` or `powershell`. |

Shell completion suggests real skill IDs, sources and projects. For zsh, for example:

```sh
quiver completion zsh > "${fpath[1]}/_quiver"
```

`quiver help completion` shows the setup for each shell.

## MCP server

`quiver install` sets up every supported assistant that is installed. Name assistants to set up only those, such as `quiver install codex cursor`. The supported assistants:

| Assistant | Name | What it changes |
|---|---|---|
| Claude Code | `claude-code` | Runs `claude mcp add --scope user quiver -- <path to quiver> mcp`. |
| Codex | `codex` | Runs `codex mcp add quiver -- <path to quiver> mcp`, which writes `~/.codex/config.toml` (or `$CODEX_HOME`). The Codex CLI and app share it. |
| Cursor | `cursor` | Adds `quiver` to `mcpServers` in `~/.cursor/mcp.json`. |
| Claude Desktop | `claude-desktop` | Adds `quiver` to `mcpServers` in `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows). |
| Antigravity | `antigravity` | Adds `quiver` to `mcpServers` in `~/.gemini/config/mcp_config.json`, which the Antigravity CLI (`agy`) reads. |
| VS Code (Copilot) | `vscode` | Adds `quiver` to `servers` in `~/Library/Application Support/Code/User/mcp.json` (macOS; `%APPDATA%\Code\User\mcp.json` on Windows, `~/.config/Code/User/mcp.json` on Linux), with `"type": "stdio"`. |

An assistant counts as installed when its command (`claude`, `codex`) is on `PATH`, or when its config folder exists (`~/.gemini/antigravity-cli` for Antigravity, VS Code's user folder for VS Code; VS Code Insiders keeps its files under `Code - Insiders`). In the JSON files, every other key stays as it is, including an `env` of the `quiver` entry, and the old file is saved next to it as `<file>.quiver-backup`. A file that is not valid JSON is reported and never overwritten. When Codex already has a `quiver` entry, its `env` is kept too. Restart the assistant afterwards. `quiver doctor` checks the entry of each installed assistant. Every assistant but Claude Code gets the path of the binary itself, never the npm launcher, because desktop apps do not see your shell's `PATH` and so cannot find `node`. Cursor passes the skill list to the model and starts the server in the workspace folder, so project skills work there. The earlier names `quiver mcp install` and `quiver mcp uninstall` still work as hidden, deprecated aliases.

`quiver install` ends with one line per assistant saying how always-on skills reach it, and `quiver status` lists the same. For an assistant without a session-start hook it names the always-on skills, which do not load by themselves there.

**Claude Code, Codex, Cursor, Antigravity and VS Code** get a session-start hook that adds the always-on skills' text at the start of every session, so they always reach the model. Claude Code, Codex, Antigravity and VS Code also run it for each sub-agent; Cursor's sub-agent event cannot add text, so it has only the session-start hook. VS Code's hook is one file Quiver owns, `~/.copilot/hooks/quiver.json`, which both its harnesses read. Codex's hooks are trusted automatically through `codex app-server`; if that fails, `quiver install` prints the "Review hooks" step. `quiver doctor` reports a Claude Code, Codex, Antigravity or VS Code hook that is missing, runs another binary, or (Codex) needs review.

The hook passes the always-on skills in full, global ones first, and then the ID and description of every other skill served in the folder. An assistant takes only so much text from one hook command (Claude Code 10,000 characters, Antigravity about 48,900 bytes), so Quiver splits the text over as many hook commands as the assistant needs and nothing is left out. Each part says which one it is, and a skill is cut between parts only when it is longer than one. `quiver install`, and every command that changes skills, sets the number of commands to what your skills take (a project's own always-on skills count too); `quiver doctor` warns when an assistant has fewer commands than the text takes. The parts of one session start share one text: the first command builds it and the others read it, so a skill edited meanwhile cannot give them different texts.

A session never starts silently without its skills. A skill that cannot be read is left out and named, and so is anything else the hook could not do (an unreadable registry, a `.quiver.json` that is not a small file, a folder that has the project ID of another folder, as a copied repository does); the assistant is told in a `<quiver-notice>` block and tells you. Text that reaches the model is cleaned: hidden characters are dropped and a skill cannot write Quiver's own tags. `~/.quiver/logs/hook.log` records each run with a hash of the text it passed.

**Claude Desktop** uses Quiver's skills only when you ask for them in the prompt, such as "check the quiver skills for this"; `quiver install` says so, because Claude Desktop drops the server's instructions. Antigravity starts the server in the project folder, so project skills work there. VS Code starts it in your home folder and sends the workspace as an MCP root, which Quiver reads after startup to serve that project's skills. See [docs/spike-mcp-clients.md](docs/spike-mcp-clients.md).

For any other assistant, `quiver mcp config` prints the entry:

```json
{ "mcpServers": { "quiver": { "command": "/path/to/quiver", "args": ["mcp"] } } }
```

The server serves the project of the folder it is started in, or of `$QUIVER_PROJECT` when that is set. It logs to `~/.quiver/logs/`.

Claude Code starts the server in the project folder. Other assistants may start it somewhere else, such as your home folder; there it serves only the global skills, unless the assistant sends its workspace as an MCP root (VS Code), which Quiver reads to find the project. Cursor starts it in the workspace folder. Claude Desktop never starts it in a project. Set `QUIVER_PROJECT` in that assistant's MCP config to serve a project's skills.

**Instructions.** The server's instructions stay short: how to use its tools (`get_skill`, `get_skill_file`, `list_skills`, `search_skills`), that new skills are saved with `create_skill`, and the names of the always-on skills, which an assistant without a session-start hook loads with `get_skill`. The skills themselves come from the hook above: the always-on skills in full and the others as ID and description.

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

**Live updates.** While a session runs, the server watches the library. Skills you add, remove or change show up in `list_skills` and the prompts right away. The hook's text is built when a session starts, so it changes from the next session on.

## Always-on skills: limits

- Changes take effect in the next session, because the hook runs when a session starts.
- Only the skill's text is delivered. Plugin hooks, such as a per-message reminder, are not run.
- Sub-agents get the text from the sub-agent hook, in the assistants that have one (Claude Code, Codex, Antigravity, VS Code); Cursor has none.

## Security

Skills give instructions to your assistant, and a skill can include scripts the assistant may run. A skill from a source is code you run on your own machine.

- Add only sources you trust, and read their skills before you assign them.
- `quiver update` downloads new commits before it applies them, lists the changed files of every assigned skill, marks scripts and executable files "needs review", and asks before it applies the changes. Read them with `quiver update --diff`; answering no leaves the source as it was. To review changes to a skill before it reaches your sessions, pin it with `quiver pin` and move the pin when you have read them.
- Source skills are read-only in Quiver. To change one, fork it into your own skills with `quiver fork`.
- `quiver sync` takes in what another machine pushed, so it lists changes that make a skill always-on, change an always-on skill or add a source, and asks first; use a private remote.
- `get_skill` and `search_skills` read the whole library, not only the skills assigned to the project; assign only what you want a project's sessions to see by name.
- `update_skill` changes only skills served in the session's project, and never an always-on skill: change those yourself with `quiver edit`.
- `create_skill` and `update_skill` let the assistant write skills into your library. A global skill's description reaches the assistant in every folder, so review new global skills with `quiver list` and `quiver edit`. Every change is a commit in `~/.quiver/local` (in `~/.quiver` once you use `quiver sync`), so `git -C ~/.quiver/local log -p` shows it and `git revert` undoes it.

To report a vulnerability in Quiver itself, see [SECURITY.md](SECURITY.md).

## Uninstall

The last command deletes your own skills too. `quiver init` and `quiver import` moved the skills you had in `~/.claude/skills` and in projects' `.claude/skills` into `~/.quiver/local`, so that folder holds the only copy of them. To keep them, first push `~/.quiver/local` somewhere: it is a git repository.

```sh
quiver uninstall          # remove the MCP server and the session-start hooks from every assistant
rm "$(command -v quiver)" # remove the binary; with npm: npm uninstall -g @kheif/quiver
rm -rf ~/.quiver          # delete the library
```

On Windows, after `quiver uninstall`, delete `%LOCALAPPDATA%\Programs\quiver` and `%USERPROFILE%\.quiver`, and remove the folder from your user `PATH` in the environment variables settings.

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
