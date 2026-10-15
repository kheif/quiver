import pathlib, re, sys

OUT = pathlib.Path(sys.argv[1])

PHASES = []  # (phase title, [(id, title, depends, body)])

def phase(title):
    PHASES.append((title, []))

def task(tid, title, depends, body):
    PHASES[-1][1].append((tid, title, depends, body.strip("\n")))

# ---------------------------------------------------------------- Phase 0
phase("Phase 0 — Foundation")

task("TASK01", "Repository setup", "—", """
1. Initialize the Go module: `go mod init github.com/kheif/quiver`.
2. Use this layout. Each package is created by the first task that needs it:
   ```
   cmd/quiver/main.go     # Entry point, only calls cli.Execute()
   internal/cli/          # One file per cobra command
   internal/mcpserver/    # MCP server, tools and prompts
   internal/config/       # config.json
   internal/registry/     # registry.json and assignments
   internal/project/      # Project identification
   internal/skill/        # SKILL.md parsing and validation
   internal/library/      # Aggregates skills from all sources
   internal/source/       # Remote sources, git, discovery
   internal/fsutil/       # Atomic writes, locking, path helpers
   testdata/              # Fixture skills and repositories
   ```
3. Add a root `quiver` command with `--version` and a global `--verbose` flag.
4. Add a `Makefile` with `build`, `test`, `lint` and `install` targets.
5. CI is deferred until the project has a GitHub repository. Then add GitHub Actions that run `go test ./...` and `golangci-lint` on macOS, Linux and Windows.

**Done when:**
- `make build` produces `./bin/quiver`.
- `quiver --version` prints the version.
- `make test` and `make lint` pass.
""")

task("TASK02", "Paths, atomic writes and locking", "TASK01", """
1. `fsutil.QuiverHome()` returns `$QUIVER_HOME` if it is set, otherwise `~/.quiver`.
2. `fsutil.ExpandHome(path)` expands a leading `~`.
3. `fsutil.WriteJSONAtomic(path, v)` writes to a temp file in the same folder, calls `fsync`, then renames it over `path`.
4. `fsutil.ReadJSON(path, v)` returns a typed `ErrNotExist` when the file is missing.
5. `fsutil.WithLock(fn)` runs `fn` while holding `~/.quiver/.lock`. Use `github.com/gofrs/flock`. Every write to `~/.quiver` goes through it.

**Done when:**
- A test proves an interrupted write leaves the old file intact.
- A test proves two concurrent writers do not corrupt the file.
""")

task("TASK03", "Config", "TASK02", """
1. Define Go structs for `config.json` (see [data-model.md](data-model.md)).
2. `config.Load()` returns defaults if the file is missing.
3. `config.Save()` writes atomically under the lock.
4. Validation:
   - Source names must match `^[a-z0-9][a-z0-9-]*$`.
   - `local` is a reserved name.
5. If the `version` field is newer than the binary supports, exit with a clear error.

**Done when:** Load, save and validation are covered by tests, including an invalid source name and an unknown version.
""")

task("TASK04", "`quiver init`", "TASK03", """
1. Create `~/.quiver/`, `local/`, `sources/`, `config.json` and `registry.json`.
2. Run `git init` in `local/` and commit a short README.
3. Make the command idempotent. Running it again only fills in what is missing and never overwrites anything.
4. Every other command, including `quiver mcp`, auto-runs `init` if `~/.quiver` does not exist.

**Done when:**
- Running `init` twice gives the same result.
- `local/` is a git repo with one commit.
""")

task("TASK05", "Skill parsing and validation", "TASK01", """
1. `skill.Load(dir)` reads `dir/SKILL.md` and parses its YAML frontmatter.
2. The `Skill` struct holds `Name`, `Description`, `Dir`, `Source`, `ID`, `Body` (markdown after the frontmatter) and the raw frontmatter as a map, so unknown fields are kept.
3. `skill.Files(s)` lists the skill's extra files as relative paths. It skips `.git` and hidden files.
4. Validation:
   - `SKILL.md` exists.
   - `name` and `description` are present.
   - `name` matches `^[a-z0-9][a-z0-9-]*$`.
   - Soft issues are returned as warnings, for example a folder name that differs from `name`.
5. Errors include the file path.

**Done when:** Tests cover valid skills, missing frontmatter, missing fields, invalid names and unknown fields (which are kept).
""")

task("TASK06", "Project identification (`.quiver.json`)", "TASK02", """
A project is identified by the `id` in its `.quiver.json` (see [data-model.md](data-model.md)).

1. `project.Find(dir)` walks up from `dir` to the first folder containing `.quiver.json`. It stops at `$HOME` and the file system root. It returns `{ID, Root}`, or `ErrNotRegistered`.
2. `project.Register(dir)`:
   - The root is the nearest folder containing `.git`, or `dir` itself if there is none.
   - Generate the ID as `<folder-name>-<6 random base32 chars>`, for example `app-k3f9q2`.
   - Write `{"id": "..."}` to `<root>/.quiver.json`.
   - If the file already exists, return its ID instead.
3. Read the file strictly. A missing or invalid `id` is an error that names the file.
4. The file holds only the ID. Unknown keys are kept when the file is rewritten.

**Done when:** Tests cover a nested subfolder, a missing file, an invalid file, registering twice (same ID) and registering a non-git folder.
""")

task("TASK07", "Registry and assignments", "TASK02, TASK06", """
1. Define Go structs for `registry.json` (see [data-model.md](data-model.md)).
2. Provide these functions:
   - `Assign(projectID, ids...)`, `Unassign(projectID, ids...)`
   - `AssignGlobal(ids...)`, `UnassignGlobal(ids...)`
   - `EnsureProject(id, root)`: adds the project, or records a new path for it
   - `Served(projectID)`: the global skills plus the project's skills, without duplicates
   - `ProjectsUsing(skillID)`
   - `RenameProject`, `ForgetProject`
3. **Moved and copied folders.** When `EnsureProject` sees a known ID at a new path:
   - If the old path no longer has a `.quiver.json` with the same ID, the folder was moved. Replace the path.
   - If it still has one, the folder was probably copied. Keep both paths and return a `PossibleCopy` warning.
4. Keep every list sorted and free of duplicates.
5. Every change runs under the lock.
   - `fsutil.WithLock` is not re-entrant: calling `config.Save`, `config.Update` or another locking function inside it blocks forever. When one change touches both files, take the lock once and use the unlocked write helpers inside it.
6. Replace the placeholder `emptyRegistry` in `internal/setup` with the registry's real default value.
7. `project.Find` and `project.Register` return `filepath.Abs` paths; they do not resolve symlinks. The same folder reached through a symlink (macOS `/var` vs `/private/var`) gives a different `Root`. `EnsureProject` compares paths with `filepath.EvalSymlinks` before deciding a folder was moved or copied.

**Done when:** Tests cover every function, a moved folder, a copied folder, concurrent writers and `Served` with overlapping lists.
""")

# ---------------------------------------------------------------- Phase 1
phase("Phase 1 — Library and CLI")

task("TASK08", "Library (local skills)", "TASK05", """
1. `library.Load()` scans `~/.quiver/local/*/SKILL.md` and returns every skill with `Source = "local"` and `ID = <folder name>` (see [data-model.md](data-model.md); folder names are unique, `name` values may not be).
2. `library.Get(id)` returns the skill, or `ErrSkillNotFound` with "did you mean" suggestions based on edit distance.
3. Invalid skills are reported as warnings and do not stop the scan.
4. Design the API so remote sources can be plugged in later (TASK26) without changing callers.

**Notes:**
- A symlink inside `local/` is skipped without a warning, because `os.ReadDir` reports it as a file. If users link skills into `local/`, follow links that point to folders and warn about the rest.
- `Library` is a snapshot. Long-running callers (the MCP server, TASK21) must call `Load` again after a change.

**Done when:** Tests pass with a fixture library containing valid and invalid skills.
""")

task("TASK09", "`quiver new` and `quiver edit`", "TASK04, TASK07, TASK08", """
`new <name>` and the MCP tool `create_skill` (TASK19) share one function: `library.CreateLocal(name, description, body, files, scope, projectID)`.

**`library.CreateLocal`:**

1. Validate the name with the rule from TASK05. Fail if `local/<name>` already exists.
2. Write `local/<name>/SKILL.md` and any extra files. Reject file paths that are absolute or contain `..`.
3. Assign the skill: `scope: "global"` goes to the global list; `scope: "project"` goes to the given project.
4. Commit in `local/` with the message `add <name>`.

**`quiver new <name>`:**

1. Flags:
   - `--description "<text>"`
   - `--global`: assign globally
   - `--project <path>`: assign to that project instead of the current one
   - `--none`: create without assigning
2. The default scope is the current project.
3. Without `--description`, open the new file in `$EDITOR`.
4. Template:
   ```markdown
   ---
   name: <name>
   description: TODO — one sentence on when the assistant should use this skill.
   ---

   # <Name>

   Instructions go here.
   ```

**`quiver edit <name>`:**

1. Opens `local/<name>/SKILL.md` in `$EDITOR`.
2. After the editor closes, validates the skill and commits with the message `edit <name>`.
3. For remote skills, refuses through `library.CheckEditable` and suggests `fork` (TASK34).

**Done when:** Create, assign and commit are tested for all three scopes. Duplicate names and path traversal are rejected.
""")

task("TASK10", "`quiver add` / `quiver remove`", "TASK07, TASK08", """
**`add <id>...`:**

1. Resolve every ID through the library. Fail on any unknown ID before changing anything.
2. Without `--global`: find the current project (or `--project <path>`). If it has no `.quiver.json`, register it (TASK06) and print that the file was created and should be committed. Then call `EnsureProject` and `Assign`.
   - `project.Register` takes no lock. Two processes registering the same new folder at once can each write an ID; the last write wins and the other caller holds a stale ID. Run `Register`, `EnsureProject` and `Assign` inside one `registry.Update` call.
3. With `--global`: call `AssignGlobal`.
4. If an added skill has the same `name` as another skill served in that project, warn about the clash.
5. Print what the project serves now, as `status` does.

**`remove <id>...`:**

1. Same project and `--global` resolution as `add`.
2. Unassign only. The skill stays in the library.
3. Removing an ID that is not assigned prints a note, not an error.

**Done when:** Tests cover project and global assignment, `--project`, unknown IDs and name clashes.
""")

task("TASK11", "`quiver list` and `quiver status`", "TASK07, TASK08", """
**`list`:**

1. Print one row per library skill: ID, description (cut to terminal width), and where it is assigned (`global`, `3 projects`, or blank).
2. Flags:
   - `--source <name>`
   - `--unassigned`
   - `--json`
3. Print library warnings at the end, in dim text.

**`status`:**

1. Find the current project and print its ID, name and root.
2. Print the skills the MCP server would serve here (`registry.Served`), each marked `global` or `project`.
3. If there is no `.quiver.json`, print `not registered; only global skills are served`.
4. Show the `PossibleCopy` warning from TASK07 with the fix: `quiver projects split`.

**Done when:** Output matches golden files for a fixture library and registry.
""")

task("TASK12", "`quiver where` and `quiver projects`", "TASK07", """
**`where <id>`:** Print `global` if applicable, then each project that has the skill, with its name and paths.

**`projects`:**

1. `projects` lists every project: name, ID, skill count and paths. Paths that no longer exist on this machine are marked `(missing)`.
2. `projects rename <id|name> <new-name>`
3. `projects forget <id|name>`: removes the project and its assignments. Asks for confirmation. Does not touch the `.quiver.json` file.
4. `projects split` (run inside a copied folder): gives this folder a new ID, copies the assignments of the old ID to it, and rewrites its `.quiver.json`.

**Done when:** Each subcommand has a test. After `split`, changing one copy's assignments does not affect the other.
""")

task("TASK13", "`quiver import`", "TASK09, TASK10", """
Brings existing skills that are scattered across projects into the library.

1. `import [<dir>]` looks for skill folders under `<dir>` (default: the current project):
   - `.claude/skills/*/SKILL.md`
   - any other `SKILL.md`, if `--deep` is given
2. Skip `node_modules`, `.git` and `~/.quiver`.
3. Show what was found and ask per skill:
   - scope: this project or global
   - name, if it collides with an existing library skill
4. For each confirmed skill:
   1. If a library skill with the same name has identical content, assign it and skip the copy.
   2. Otherwise copy it into `local/<name>` and assign it.
   3. Commit in `local/` with the message `import <name> from <project>`.
5. Ask whether to delete the original folder. Never delete it without a clear yes. With `--yes`, the originals are kept.

**Done when:** An end-to-end test imports two skills from a fixture project, with one name collision. Nothing is overwritten, and the originals stay unless deletion was confirmed.
""")

task("TASK14", "`quiver doctor`", "TASK11, TASK12", """
Check these, and report each one as `ok`, `warn` or `error`:

1. The `~/.quiver` layout and JSON files are valid.
2. `local/` is a git repo. Uncommitted changes are a warning.
3. Every library skill passes validation.
4. Every assigned ID exists in the library. An ID can go missing when a source is removed or a skill is renamed upstream.
5. Every source in `config.json` has its folder in `sources/`. The library only warns "not downloaded" when it is missing.
6. Projects whose paths all no longer exist on this machine are listed (they may still exist on another machine, so this is only a warning).
7. Skill name clashes in the served set of any project.
8. The MCP server is registered with Claude Code (`claude mcp get quiver`).

`--fix` applies the safe fixes: committing `local/` (after showing the changes) and unassigning missing IDs (after confirmation).

**Done when:** Each check has a fixture that triggers it, and `--fix` resolves the fixable ones.
""")

# ---------------------------------------------------------------- Phase 2
phase("Phase 2 — MCP Server")

task("TASK15", "MCP server skeleton (`quiver mcp`)", "TASK04, TASK08", """
1. `quiver mcp` starts an MCP server on stdio using the official Go SDK.
2. Server info: name `quiver`, plus the binary version.
3. **Never write to stdout** except MCP messages. Send all logs to stderr and to `~/.quiver/logs/mcp.log`.
4. Recover from panics in tool handlers and return them as tool errors, so the session survives.
5. Register a placeholder `ping` tool to prove the wiring. Remove it in TASK17.
6. Add an in-memory test harness that connects a client to the server through the SDK's in-memory transport, with no subprocess. Every later MCP task uses it.

**Done when:**
- The harness test can list tools and call `ping`.
- Running `claude mcp add quiver -- ./bin/quiver mcp` by hand shows the server as connected in Claude Code.
""")

task("TASK16", "Project resolution in the server", "TASK15, TASK06, TASK07", """
The server must know which project it is serving. Resolve it once, at startup, because `instructions` (TASK17) are built before the first request:

1. `QUIVER_PROJECT` environment variable, if set.
2. The process working directory. Claude Code starts stdio servers in the project folder (confirmed in the B01 spike).

Then:

- Run `project.Find` on the result. It walks up to the nearest `.quiver.json`, so subfolders and monorepos work. Remember the project ID for the session.
- If there is no `.quiver.json`, serve only global skills. Do not create the file until the user creates or assigns a skill there. Remember the folder, so `create_skill` can register it.
- Do not use MCP roots. Under protocol version 2026-07-28 the server cannot send `roots/list` while serving a request (SEP-2322), and Claude Code never triggered the `initialized` handler in the spike. See `docs/spike-b01-results.md`.

**Done when:**
- Harness tests cover both sources and a start inside a subfolder.
- The resolved key is logged at startup.
""")

task("TASK17", "`list_skills`, `get_skill`, `get_skill_file`", "TASK16", """
**`list_skills`** (no input):

1. Returns the served skills: `id`, `name`, `description`, `scope` (`global` or `project`). It always reads the current registry, so it also shows skills assigned during the session.

**Skill catalog in `instructions`:**

Claude Code defers MCP tools behind its tool search, so the model never sees tool descriptions until it loads a tool. A skill list inside a tool description was used 0 times out of 6 in the spike. The same list in the server's `instructions` was used 6 out of 6 times, with no false positives. So the catalog goes into `instructions`:

```
Skills (reusable instructions for specific tasks). Before starting a task, check whether one fits. If it does, call the get_skill tool of this server with its id and follow it.
- db-migration: Rules for writing and running database migrations.
- anthropic/pdf: Read, create and edit PDF files.
```

1. Build it at startup from the project resolved in TASK16. Always-on pointers (TASKA1) come first, the catalog second.
2. **Budget:** Claude Code cuts `instructions` at about 4096 characters and appends `… [truncated]`. Keep the whole text under 3800 characters: shorten each description to 150 characters, then drop descriptions, then end with `…and N more: call list_skills`.
3. The catalog is fixed for the session. Skills assigned later appear in `list_skills` and in the next session.

**`get_skill`** (`id`):

1. Returns the `SKILL.md` body, the list of extra files, and the absolute path of the skill folder.
   - `skill.Files` lists symlinks as they are. Drop any entry that resolves outside the skill folder.
2. Ends with this note: `Scripts and files referenced by this skill are in <dir>. Use absolute paths.`
3. An ID that is not served in this project is still returned if it exists in the library, with a note that it is not assigned here. Unknown IDs return an error with suggestions.

**`get_skill_file`** (`id`, `path`):

1. Resolve the path inside the skill folder. Reject any path that escapes it, including through symlinks.
2. Text files up to 256 KB are returned as text. For larger or binary files, return their absolute path and size instead.

**Done when:** Harness tests cover all three tools, path traversal attempts, binary files and the `instructions` budget.
""")

task("TASK18", "Behaviour check against the real server", "TASK17, TASKA1", """
The B01 spike (`spike/b01`, results in `docs/spike-b01-results.md`) answered the design questions with a throwaway server. Repeat its checks against the real `quiver mcp` before calling the first milestone done.

1. Point `spike/b01/run.sh` and `run2.sh` at `quiver mcp` with fixture skills, or port them to a `make spike` target.
2. Expect the spike numbers: fitting prompts load the skill, non-fitting prompts load nothing, `get_skill_file` is read when `SKILL.md` refers to a file, and the always-on skill holds across five turns.
3. Check what the spike did not cover: context compaction, and a real caveman session with the caveman plugin disabled.
4. Add the results to `docs/spike-b01-results.md`.

**Done when:** The real server matches the spike results, and compaction behaviour is recorded.
""")

task("TASK19", "`create_skill`", "TASK17, TASK09", """
**Input:** `name`, `description`, `body` (markdown without frontmatter), optional `files` (map of relative path to text content), `scope` (`"project"` or `"global"`).

1. Call `library.CreateLocal` (TASK09) with the resolved project. With `scope: "project"` and no `.quiver.json`, register the project first (TASK06), and say in the result that `.quiver.json` was created and should be committed.
2. The tool description tells the assistant to ask the user for the scope if the user did not say it.
3. Return the new skill's ID, folder path and scope.
4. Errors that the assistant can fix (invalid name, name taken) return a clear message and a suggested alternative name.
5. Mark the tool with the MCP annotations `readOnlyHint: false` and `destructiveHint: false`.

**Done when:**
- Harness tests cover both scopes, an unregistered project, a taken name and path traversal in `files`.
- A manual test in Claude Code creates a project skill from a conversation, and `quiver status` shows it.
""")

task("TASK20", "Skills as MCP prompts", "TASK17", """
1. Expose every served skill as an MCP prompt:
   - prompt name: the skill ID, with `/` replaced by `__`
   - prompt description: the skill description
2. Getting a prompt returns the skill body as a user message, followed by the same folder path note as `get_skill`.
3. Accept an optional `args` argument. If it is given, append it after the skill content, prefixed with `Task:`.

**Done when:**
- Harness tests list and get prompts.
- In Claude Code, `/mcp__quiver__<skill>` loads the skill.
""")

task("TASK21", "Live updates", "TASK19, TASK20", """
1. Watch `registry.json`, `config.json`, `local/` and `sources/` with `fsnotify`. Debounce events by 300 ms.
2. When the served set changes, reload it and send `notifications/prompts/list_changed`. `list_skills` returns the new set on its next call. The catalog in `instructions` stays as it was until the next session (TASK17).
3. Content changes to a skill need no notification, because `get_skill` always reads from disk.
4. A failed reload keeps the last good state and logs the error.

**Done when:** A harness test runs `quiver add` from another goroutine and receives the notification. The new skill appears in `list_skills`.
""")

task("TASK22", "`quiver mcp install`", "TASK15", """
Registers the server with assistants, so the user never edits MCP config by hand.

1. **Claude Code:** run `claude mcp add --scope user quiver -- <absolute path to quiver> mcp`. If the `claude` command is not found, print the exact command for the user to run.
2. `--print`: only print the config snippet for manual setup.
3. `mcp uninstall` removes the registration again.
4. Detect an existing registration and update it if the binary path changed (for example after a Homebrew upgrade).

**Done when:**
- On a machine with Claude Code, `install` followed by a new session shows the Quiver tools.
- `uninstall` removes them.
""")

# ---------------------------------------------------------------- Phase 3
phase("Phase 3 — Remote Sources")

task("TASK23", "Git wrapper", "TASK01", """
1. Provide `git.Clone(url, dir)`, `git.Pull(dir)`, `git.Fetch(dir)`, `git.Head(dir)`, `git.DiffNames(dir, from, to)`, `git.Diff(dir, from, to, paths...)`, `git.Commit(dir, msg, paths...)`. `Init` and `HasCommit` serve `quiver init`.
2. Run the system `git` binary with `GIT_TERMINAL_PROMPT=0`, so missing credentials fail instead of hanging.
3. Include stderr in errors. If `git` is missing, return `ErrNotInstalled`.
4. Clone with `--depth 1`. Pull with `--ff-only`.
5. `Commit` does nothing when nothing changed, and commits as "Quiver" when the user has no git identity.

**Notes:**
- `run` passes the user's environment through. A `GIT_DIR` or `GIT_WORK_TREE` set by a git hook or IDE would point every command at the wrong repository. Unset them in `run` if this shows up.
- `Clone` puts `--` before the URL, but other dangerous inputs (`ext::`, `file://` from untrusted input) must be rejected by URL parsing (TASK24).

**Done when:** Tests pass against local bare repositories created in temporary folders, with no network access.
""")

task("TASK24", "Source URL parsing", "—", """
`source.Parse(input)` returns a clone URL and a suggested name:

| Input | URL | Default name |
|---|---|---|
| `anthropics/skills` | `https://github.com/anthropics/skills.git` | `anthropics` |
| `github.com/a/b` | `https://github.com/a/b.git` | `a` |
| `https://gitlab.com/a/b` | `https://gitlab.com/a/b.git` | `a` |
| `git@github.com:a/b.git` | unchanged | `a` |
| `/path/to/repo` | unchanged (local path) | folder name |

- Local paths start with `/`, `./`, `../` or `~/` and are made absolute.
- URL schemes: `https`, `http`, `ssh`, `git`, `file`. HTTP(S) URLs get `.git` appended.
- The default name is lowercased and cleaned to match `config.ValidateSourceName`.
- Input that starts with `-` or uses a git transport helper (`ext::`, `fd::`) is rejected. Both can run commands through git (see TASK23).

**Notes:**
- `Parse` keeps credentials in a URL such as `https://user:token@host/a/b`. `source add` must refuse or strip them, or the token ends up in `config.json` as plain text.

**Done when:** Table tests cover every row and invalid input.
""")

task("TASK25", "Skill discovery", "TASK05", """
`source.Discover(repoDir)` returns the skill folders in a repository, sorted by ID, plus warnings. Each result has an ID without the `<source>/` prefix, the folder relative to the repo root and the plugin that lists it.

1. **Marketplace mode.** If `.claude-plugin/marketplace.json` exists:
   1. Read `plugins[]`.
   2. Resolve each plugin's `source` relative to the repo root. `metadata.pluginRoot` is prepended to sources that do not start with `./`.
   3. If the plugin entry has `skills` (a path or a list), use those paths. A path may be a skill folder or a folder of skills.
   4. Otherwise use `<pluginDir>/skills/*/SKILL.md`.
   5. Plugins whose `source` points to another repository are skipped with a warning in v1.
   6. Paths that leave the repository, also through symlinks, are skipped with a warning. A folder listed twice is kept once.
   7. A broken manifest gives a warning and falls back to plain mode.
2. **Plain mode.** Otherwise, walk the repo for `SKILL.md` files:
   - Skip `.git`, `node_modules` and hidden folders.
   - Do not look inside a skill's subfolders once its `SKILL.md` is found.
   - A skill at the repo root is named after the repo folder, which is the source name.
3. **Collisions inside one source:** use `<plugin>/<name>` for every skill that shares the name and print a warning. Without a plugin, or when that is still not unique, use the folder path.
4. Verify the marketplace format against real repositories, starting with `anthropics/skills`. Save trimmed copies as fixtures.

**Notes:**
- Checked 2026-10-15: `anthropics/skills` gives 19 skills and `JuliusBrussee/caveman` 22, both without warnings. Their trimmed copies are in `internal/source/testdata`.
- Collision IDs can change: a new skill with the same name in a later commit renames an existing one, and its assignments break. See backlog B05; TASK28 must detect this.
- A plugin's own `.claude-plugin/plugin.json` can also list skills. It is ignored in v1; only the marketplace entry counts.
- A `SKILL.md` that is a symlink is not found, and the plain walk does not follow symlinked folders.

**Done when:** Fixture tests pass for a marketplace repo, a plain repo, a nested repo and a repo with collisions.
""")

task("TASK26", "Library with remote sources", "TASK08, TASK25", """
1. Extend `library.Load()` to run discovery on every source listed in the config, in name order.
2. Load each found folder with `skill.Load` and `skill.Validate`; problems become warnings, as for local skills.
3. Remote skills get `Source = <name>` and `ID = <name>/<discovered ID>`.
4. A source folder that is missing is a warning, not an error.
5. Cache discovery results in `sources/<name>/.quiver-index.json`, keyed by the HEAD commit and an index version. Rescan only when either changes. A folder that is not a git repository is scanned every time.
6. `library.CheckEditable` refuses remote skills with `ErrReadOnly`. `edit` and `update_skill` must call it.

**Notes:**
- IDs from discovery can hold slashes (`<plugin>/<name>` on collision), so a remote ID can have three parts. Code that splits IDs must split on the first `/` only.
- Bump `indexVersion` whenever discovery rules change, or old caches keep the old result.
- The index is an untracked file in the source's working tree. `git status` shows it, and an upstream file with the same name would block `git pull`. TASK28 must handle both.
- `Load` now reads `config.json`, so a broken config fails every load, also for local skills.

**Done when:**
- `Library.All` returns local and remote skills together; `list` and `list_skills` build on it.
- A second load uses the cache, verified by a test hook.
""")

task("TASK27", "`quiver source add | list | remove`", "TASK23, TASK24, TASK26", """
**`add <repo> [--name <n>]`:**

1. Parse the input (TASK24) and check that the name is free. Validate a `--name` value with `config.ValidateSourceName`.
2. Clone into `sources/<name>`.
3. Run discovery. If no skills are found, ask whether to keep the source anyway.
4. Save it to the config and print the skills that were found.
5. Print a reminder that skills from this source give instructions to the assistant and may contain scripts.

**`list`:** Print each source's name, URL, skill count, current commit and last update time.

**`remove <name>`:**

1. If any skill from the source is assigned, refuse and list where (using `where`).
2. `--force` unassigns those skills first.
3. Delete `sources/<name>` and remove it from the config.

**Notes:**
- The default name is the repository owner, so two repos from one owner (`anthropics/skills`, `anthropics/other`) collide. When the name is taken, suggest `--name` in the error.

**Done when:** An end-to-end test adds a source from a local bare repo, assigns one of its skills, gets refused on remove, then succeeds with `--force`.
""")

task("TASK28", "`quiver update`", "TASK27", """
1. `update [<name>...]` runs `git pull --ff-only` for the given sources, or all of them.
2. Record HEAD before and after. Print the skills that were added, changed or removed.
3. If an assigned skill disappeared upstream, warn and list the projects using it. Do not unassign it automatically; `doctor` reports it.
4. If a source cannot be fast-forwarded, report it and continue with the other sources.
5. Running MCP servers pick up the change through live updates (TASK21).

**Notes:**
- Sources are shallow clones. After an upstream force-push, `--ff-only` fails and the old HEAD may be gone, so `DiffNames` cannot run. Offer a re-clone in that case.
- Each source holds an untracked `.quiver-index.json` (TASK26). Do not treat it as a dirty tree, and never run `git clean`. If upstream adds a file with that name, delete the local one and pull again.

**Done when:** A test pushes a change to a fixture bare repo, runs `update`, and `get_skill` returns the new content.
""")

# ---------------------------------------------------------------- Phase 4
phase("Phase 4 — v1 Release")

task("TASK29", "End-to-end test suite", "TASK22, TASK28", """
1. Build a harness that:
   - creates a temp `HOME` and `QUIVER_HOME`,
   - creates fixture projects and bare git repositories,
   - runs the compiled binary for CLI steps,
   - connects an MCP client to `quiver mcp` as a subprocess for MCP steps.
2. Cover these flows:
   - `new` → `add` → `list_skills` shows it → `remove` → gone
   - `create_skill` with project scope, then with global scope
   - `import` from a project with `.claude/skills`
   - `source add` → `add anthropic/x` → `update` → new content served
   - a moved project folder keeps its skills; a copied one gets the `PossibleCopy` warning, and `projects split` separates it
   - `doctor --fix`
3. Run it on all three OSes in CI.

**Done when:** All flows pass in CI.
""")

task("TASK30", "README and help text", "TASK29", """
1. Write a README with: what Quiver is, install, a 2-minute quickstart (`init` → `mcp install` → `new` → ask the assistant), the command reference, the MCP tool reference and a security note about remote sources.
2. Give every cobra command a `Short`, a `Long` and an `Example`.
3. Add shell completion with `quiver completion zsh|bash|fish`. Skill, source and project arguments complete with real values.

**Done when:** `quiver help <cmd>` is useful for every command, and completion suggests skill IDs.
""")

task("TASK31", "Release", "TASK30", """
1. Set up GoReleaser for darwin/linux/windows on amd64 and arm64.
2. Publish GitHub Releases with checksums.
3. Add a Homebrew tap: `brew install <owner>/tap/quiver`.
4. After a Homebrew upgrade, `quiver mcp install` must still point to the right binary. Use the stable Homebrew path, not the versioned Cellar path.
5. Tag `v1.0.0`.

**Done when:** `brew install` on a clean machine works, and the quickstart runs.
""")

# ---------------------------------------------------------------- Phase 5
phase("Phase 5 — v2")

task("TASK32", "`update_skill` MCP tool", "TASK19", """
1. Input: `id`, and any of `description`, `body`, `files` (path to content; empty content deletes the file).
2. Only local skills can be updated. Check with `library.CheckEditable`; for remote skills, return an error suggesting `fork`.
3. Commit in `local/` with the message `update <name> (via MCP)`.
4. The tool description asks the assistant to confirm the change with the user first.

**Done when:** Harness tests cover a body change, a file add and delete, a remote skill and path traversal.
""")

task("TASK33", "`quiver search` and `search_skills`", "TASK26, TASK17", """
1. Do a case-insensitive match on ID, name and description across the whole library.
2. Rank results: exact name match, then name prefix, then description match.
3. CLI: `quiver search <text>`, with `--json`.
4. MCP: `search_skills` (`query`) returns the matches and whether each one is assigned here. The description tells the assistant it may suggest an unassigned skill to the user but should not assume it applies.

**Done when:** Ranking tests pass, and the MCP tool marks assigned and unassigned skills correctly.
""")

task("TASK34", "`quiver fork`", "TASK26, TASK10", """
1. `fork <source/skill> [--as <name>]` copies the skill into `local/<name>` and adds `forkedFrom: <id>@<commit>` to its frontmatter.
2. In the global list and in every project, replace the original ID with the local one.
3. Commit in `local/`.

**Done when:** After `fork`, editing the local copy changes what projects get, and `update` no longer affects them.
""")

task("TASK35", "Pinning", "TASK28", """
1. `quiver pin <source/skill> [<commit>]` pins the skill to the current HEAD, or to the given commit.
2. Check the pinned commit out into `sources/.pins/<source>@<commit>/` with `git worktree`, fetching more history if needed.
3. Store pins in `config.json` under `pins`. A pin applies everywhere the skill is assigned.
4. `update` leaves pinned skills alone. `quiver unpin` switches back to the live source.
5. `doctor` removes pin folders that are no longer used.

**Done when:** A pinned skill keeps its old content after `update`, and shows the new content after `unpin`.
""")

task("TASK36", "Update preview", "TASK28", """
1. `update` fetches first, without merging.
2. Show a per-skill change summary for every assigned skill that changed. `--diff` shows the full diff.
3. Flag changes to scripts (`*.sh`, `*.py`, `*.js`, files with the executable bit) as **needs review**.
4. Ask for confirmation, then merge. `--yes` skips the question.

**Done when:** A test where a fixture adds a script to a skill shows the "needs review" flag and requires confirmation.
""")

task("TASK37", "Library sync between machines", "TASK29", """
1. `quiver sync init <git-remote>`: turns `~/.quiver` itself into a git repository that tracks `local/`, `config.json` and `registry.json`. `sources/` and `logs/` are ignored, because sources are re-cloned from the config.
2. `quiver sync`: commit, pull with rebase, push. On conflicts in `registry.json`, merge the lists (union) automatically. Stop on any other conflict and explain it.
3. After a pull, clone any source that is in the config but missing on disk.
4. Merge the `local/` git repo into the top-level repo, or keep it as a submodule. Decide in this task and record the reason.

**Done when:** A test with two `QUIVER_HOME` folders and one bare remote syncs a new skill and an assignment from machine A to machine B.
""")

task("TASK38", "`mcp install` for more assistants", "TASK22", """
1. Find out how Cursor, Claude Desktop, Codex and Gemini CLI register stdio MCP servers (config file path and format, or a CLI command).
2. Add each one to `mcp install --target <name>`. Without `--target`, detect the installed assistants and ask which ones to set up.
3. Edit JSON config files safely: keep unknown keys, write atomically, and back up the original first.
4. Confirm that each assistant passes the project folder (roots or working directory) correctly. Record any assistant that needs `QUIVER_PROJECT`.

**Done when:** Install and uninstall work for each supported assistant, and each one sees the correct project's skills.
""")

task("TASK39", "Interactive TUI (`quiver` with no arguments)", "TASK33, TASK10", """
Running `quiver` with no arguments in a terminal opens an interactive screen. Build it with Bubble Tea (`github.com/charmbracelet/bubbletea`, `bubbles` and `lipgloss`).

**Layout:**

```
┌ Quiver ─ project: app (app-k3f9q2) ──────────────────────────┐
│ Search: migr_                                                │
│                                                              │
│  [P] db-migration        Rules for database migrations       │
│  [ ] team/migrate-check  Checks a migration before merge     │
│  [G] commit-rules        Commit message conventions          │
│                                                              │
│ ┌ Preview ───────────────────────────────────────────────┐   │
│ │ SKILL.md of the selected skill                         │   │
│ └────────────────────────────────────────────────────────┘   │
│                                  ┌───────────────────────┐   │
│ space: project  g: global        │ anthropic: 3 updated  │   │
│ e: edit  u: updates  q: quit     │ press u to pull       │   │
└──────────────────────────────────┴───────────────────────┴───┘
```

1. **Search:** typing filters the whole library live, using the ranking from TASK33.
2. **Markers:** `[G]` global, `[P]` this project, `[ ]` not assigned here.
3. **Keys:**
   - `space`: toggle the skill for the current project
   - `g`: toggle global
   - `enter`: full preview
   - `e`: edit in `$EDITOR` (local skills only)
   - `/`: focus search
   - `tab`: switch between the views Skills, Projects and Sources
   - `q`: quit
4. Every change goes through the same functions as the CLI commands. The TUI contains no separate logic.
5. Outside a project, `space` asks which project to assign to.
6. When stdout is not a terminal, `quiver` prints the help text instead of opening the TUI.

**Done when:** Tests drive the model with key messages (no real terminal) for search, toggling and editing. A manual check on macOS, Linux and Windows terminals shows no layout glitches.
""")

task("TASK40", "Background update checks", "TASK39, TASK28", """
Inside the TUI, check the remote sources for updates quietly, one at a time.

1. Start a background worker when the TUI opens. It checks **one source at a time**, with a short pause between sources.
2. For each source, run `git ls-remote <url> HEAD` and compare it with the local HEAD. This downloads nothing.
3. Skip the whole run when there is no network. Each check has a 10 s timeout. A failure is logged and skipped, never shown as an error.
4. Store the last check time per source in `~/.quiver/state.json`. Check a source again only after the interval (default 6 hours, configurable as `updateCheckInterval`).
5. When a source has updates, show a small notice in the bottom-right corner: `anthropic: 3 skills updated — press u to pull`. Several sources are shown as a count.
6. `u` opens a list of the sources with updates and the changed skills (TASK36 preview). Pulling runs `quiver update` for the selected sources.
7. The worker never pulls by itself. Pulling always needs a key press.

**Done when:**
- A test with a fixture remote that gains a commit shows the notice.
- A test with no network shows nothing and does not block the TUI.
- Checks respect the interval.
""")

task("TASKA1", "Always-on skills (MCP instructions)", "TASK17, TASK10", """
Some skills must be active from the first message, without the assistant deciding to load them. Caveman is the example: a communication style that applies to every answer. For these, Quiver puts a short pointer into the MCP server's `instructions`, and the assistant loads the skill with `get_skill` before its first reply.

The full skill text does not go into `instructions`: Claude Code cuts them at about 4096 characters, and caveman's `SKILL.md` alone is 4678. The pointer approach held in 10 of 10 single-turn spike runs and across a five-turn session.

1. **Data:** add an `alwaysOn` list next to `global` in `registry.json`, and per project (see [data-model.md](data-model.md)). An always-on skill must also be assigned in the same place.
2. **CLI:** `quiver add <id> --always` (with or without `--global`) assigns the skill and marks it always-on. `quiver remove <id> --always` removes only the mark.
3. **Server:** start `instructions` with the pointer, before the skill catalog (TASK17):
   ```
   Always-on skills: caveman/caveman. Before your first reply in this conversation, call the get_skill tool of this server for each always-on skill and follow it for the whole conversation.
   ```
4. **Project at startup:** use the project resolved in TASK16. Global always-on skills do not depend on it.
5. **Limits, to document in the README:**
   - Changes take effect in the next session, because `instructions` cannot change during a session.
   - Plugin hooks (such as caveman's per-message reminder) are not run. Only the skill text is delivered.
   - Subagents are not reliable: in the spike, one of two subagents followed the always-on skill.

**Done when:**
- A harness test gets the pointer in `instructions`, ahead of the catalog.
- In Claude Code, a new session follows a global always-on caveman skill from the first message, with the caveman plugin disabled (checked in TASK18).
""")

# ---------------------------------------------------------------- renumber
ORDER = [
    ("Phase 0 — Foundation", ["TASK01", "TASK02", "TASK03", "TASK04", "TASK05", "TASK06", "TASK07"]),
    ("Phase 1 — Remote Sources", ["TASK08", "TASK23", "TASK24", "TASK25", "TASK26", "TASK27", "TASK10", "TASK28"]),
    ("Phase 2 — MCP Server (first usable version)", ["TASK15", "TASK16", "TASK17", "TASKA1", "TASK22", "TASK18"]),
    ("Phase 3 — Local Skills and Management", ["TASK11", "TASK09", "TASK19", "TASK20", "TASK21", "TASK12", "TASK13", "TASK14"]),
    ("Phase 4 — v1 Release", ["TASK29", "TASK30", "TASK31"]),
    ("Phase 5 — v2", ["TASK32", "TASK33", "TASK34", "TASK35", "TASK36", "TASK37", "TASK38", "TASK39", "TASK40"]),
]
ALL = {t[0]: t for _, ts in PHASES for t in ts}
flat = [tid for _, ids in ORDER for tid in ids]
assert sorted(flat) == sorted(ALL), set(ALL) ^ set(flat)
REMAP = {old: f"TASK{i:02d}" for i, old in enumerate(flat, 1)}
remap = lambda s: re.sub(r"TASK(?:\d\d|A1)", lambda m: REMAP[m.group(0)], s)
MARK = {"x": "✅", "~": "🚧"}
STATUS = {"TASK01": "x", "TASK02": "x", "TASK03": "x", "TASK04": "x", "TASK05": "x", "TASK06": "x", "TASK07": "x", "TASK08": "x", "TASK09": "x", "TASK10": "x", "TASK11": "x", "TASK12": "x"}  # final IDs; x done, ~ in progress
EXTRA_DEPS = {"TASK29": ", TASK14"}  # e2e also waits for the last management task
PHASES = [
    (ptitle, [(REMAP[o], remap(ALL[o][1]), remap(ALL[o][2] + EXTRA_DEPS.get(o, "")), remap(ALL[o][3])) for o in ids])
    for ptitle, ids in ORDER
]

# ---------------------------------------------------------------- write
def slug(s):
    s = s.replace("`", "")
    return re.sub(r"[^a-z0-9]+", "-", s.lower()).strip("-")

index = []
for ptitle, tasks in PHASES:
    rows = []
    for tid, title, dep, body in tasks:
        fname = f"{tid}-{slug(title)}.md"
        status = {"x": "✅ Done", "~": "🚧 In progress"}.get(STATUS.get(tid), "Todo")
        (OUT / fname).write_text(f"# {tid} — {title}\n\n**Phase:** {ptitle}\n\n**Status:** {status}\n\n**Depends on:** {dep}\n\n{body}\n")
        rows.append(f"| {MARK.get(STATUS.get(tid), '')} | [{tid}]({fname}) | {title.replace('|', chr(92) + '|')} | {dep} |")
    index.append(f"### {ptitle}\n\n| Status | Task | Title | Depends on |\n|---|---|---|---|\n" + "\n".join(rows))

readme = """# Quiver — Implementation Tasks

This folder breaks [PROJECT.md](../PROJECT.md) into ordered tasks, one file per task. Work through them top to bottom.

- [decisions.md](decisions.md): technical decisions that apply to every task
- [data-model.md](data-model.md): file formats, project IDs and skill IDs
- [backlog.md](backlog.md): open concerns from the design review; check it before starting a related task
- [docs/spike-b01-results.md](../docs/spike-b01-results.md): measured assistant behaviour that shaped the MCP design

## Conventions

- **Status** is set in `scripts/gen_tasks.py` (`STATUS`) and shown in the index below and at the top of each task: ✅ done, 🚧 in progress, empty for todo.
- **Depends on:** tasks that must be finished first.
- **Done when:** acceptance criteria. A task is not done until every item passes and has a test.
- Every command needs unit tests. Every user-facing flow also needs an end-to-end test (see TASK30).
- MCP tasks are tested through the in-memory harness from TASK16.

## First Milestone

Phases 0–2 are the first usable version. After TASK21 you can run:

```bash
quiver source add JuliusBrussee/caveman
quiver add caveman/caveman --global --always
quiver mcp install
```

and every new assistant session follows caveman from the first message. Use Quiver this way every day before you start Phase 3.

## Index

""" + "\n\n".join(index) + """

## Later (not yet planned in detail)

- Graphical app with the same features as the TUI.
- Team libraries with shared assignments.
"""
(OUT / "README.md").write_text(readme)
print(sum(len(t) for _, t in PHASES), "tasks")
