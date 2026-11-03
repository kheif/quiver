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
5. CI is deferred until the project has a GitHub repository. Then add GitHub Actions that run `go test ./...` and `golangci-lint` on macOS, Linux and Windows. Done in TASK29.

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

**Notes:**
- `library.CreateLocal` (`internal/library/create.go`) returns the validation warnings. It writes into a hidden `local/.new-<name>-*` folder, which the library skips, validates, then renames it into place, all under the Quiver lock. Nothing is left behind or assigned when it fails before the commit. With `scope: "project"` the project must already be in the registry; `new` registers the folder first, as `add` does.
- Extra file paths must pass `filepath.IsLocal` and may not contain a backslash, a `.git` part, or be `SKILL.md`.
- `skill.ValidateName` is the name rule for new skills; a name over 64 characters is an error here, not a warning.
- `new` checks the name is free before it registers a project, so a typo does not create a `.quiver.json`. Without `--description` the skill is created and committed with the template (`add <name>`), then opened; a change is committed as `edit <name>`. Both `new` and `edit` warn while the description still starts with `TODO`.
- `edit` opens a local skill from its folder, not through the library, so a skill that an earlier edit broke can still be fixed. An invalid edit stays on disk, uncommitted. The editor is `$VISUAL`, then `$EDITOR`, then `vi`, split on spaces so `code --wait` works.
- The read-only error already suggests `quiver fork`, which comes with TASK34.
- Tests replace `runEditor` in `internal/cli/new.go`.

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

**Notes:**
- `add --global` checks name clashes only among the global skills. A global skill can still clash with a skill of some project; `doctor` (TASK14) reports those.
- If the registry write fails after `.quiver.json` was created, the file stays without a registry entry. The next `add` reuses its ID, so nothing breaks.
- `remove` does not look IDs up in the library, so a skill deleted from its source can still be unassigned. The cost: a typo only prints a note and exits 0.
- `remove` does not call `EnsureProject`, so a moved project keeps its old path in the registry until the next `add` or until the server resolves it (TASK16).

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

**Notes:**
- `printServed` in `internal/cli/assign.go` already prints the served list after `add` and `remove`. Use it here, so both outputs stay the same.
- The commands are in `internal/cli/list.go` and `internal/cli/status.go`. Columns are ID, ASSIGNED, DESCRIPTION: the description comes last, so cutting it to the terminal width keeps the table aligned. A skill both global and in projects shows `global, 2 projects`.
- Width and dim text come from `golang.org/x/term` (`internal/cli/term.go`). Output that is not a terminal is never cut or coloured, and `NO_COLOR` turns dim text off.
- `--json` prints `{"skills": [...], "warnings": [...]}`; each skill has `id`, `name`, `source`, `description`, `global` and `projects` (project IDs).
- `status` finds the project with `server.Resolve`, so it honours `$QUIVER_PROJECT` like the server. It checks for a `PossibleCopy` on an in-memory copy of the registry and writes nothing. It also warns about assigned skills that are no longer in the library, which the server leaves out.
- Golden files are in `internal/cli/testdata`; rewrite them with `go test ./internal/cli -update`.

**Done when:** Output matches golden files for a fixture library and registry.
""")

task("TASK12", "`quiver where` and `quiver projects`", "TASK07", """
**`where <id>`:** Print `global` if applicable, then each project that has the skill, with its name and paths.

**Notes:**
- `source remove` (TASK27) has its own lookup, `library.usage`, by source prefix. Share the output format with it.

**`projects`:**

1. `projects` lists every project: name, ID, skill count and paths. Paths that no longer exist on this machine are marked `(missing)`.
2. `projects rename <id|name> <new-name>`
3. `projects forget <id|name>`: removes the project and its assignments. Asks for confirmation. Does not touch the `.quiver.json` file.
4. `projects split` (run inside a copied folder): gives this folder a new ID, copies the assignments of the old ID to it, and rewrites its `.quiver.json`.

**Implementation notes:**
- Both formats use `registry.ProjectLabel`: `<name> (<paths>)`. `where` prints `global`, then `project <label>` per project; the `source remove` error prints `project <label>: <skills>`.
- `registry.Lookup` resolves `<id|name>`: an ID wins over a name, and a name used by several projects is an error that lists their IDs.
- `forget` takes `--yes` for scripts. A "no" answer is not an error, like `source add`.
- `split` refuses when no other folder still holds the ID (`ErrNotCopied`). The new project is named after its folder and copies the skills and always-on marks. `.quiver.json` is written before the registry; if the registry write fails, the old ID is put back.

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

**Implementation notes:**
- `library.FindSkills` finds the folders, `library.ImportLocal` copies one skill under the lock: hidden temp folder, validate, rename, assign, commit. Links and `.git` folders are not copied (a warning names each skipped link). The source folder is never changed.
- A new name is written into the `name:` line of the copied `SKILL.md`.
- "Identical content" means the same files with the same bytes, `.git` ignored, under the same name.
- A found skill that cannot be loaded is listed as skipped. A taken or invalid name is asked again until it is free; an empty answer skips the skill.
- `--yes` asks nothing: scope is the project (or every folder with `--global`), taken names are skipped, originals are kept. Without `--yes`, `--global` only changes the default answer.
- The project is registered only when the first skill is assigned to it.

**Done when:** An end-to-end test imports two skills from a fixture project, with one name collision. Nothing is overwritten, and the originals stay unless deletion was confirmed.
""")

task("TASK14", "`quiver doctor`", "TASK11, TASK12", """
Check these, and report each one as `ok`, `warn` or `error`:

1. The `~/.quiver` layout and JSON files are valid.
2. `local/` is a git repo. Uncommitted changes are a warning.
3. Every library skill passes validation.
4. Every assigned ID exists in the library. An ID can go missing when a source is removed or a skill is renamed upstream.
5. Every source in `config.json` has its folder in `sources/`. The library only warns "not downloaded" when it is missing. Leftover `sources/.add-*` and `sources/.remove-*` folders are a warning; `--fix` deletes them.
6. Projects whose paths all no longer exist on this machine are listed (they may still exist on another machine, so this is only a warning).
7. Skill name clashes in the served set of any project.
8. The MCP server is registered with Claude Code (`claude mcp get quiver`).

`--fix` applies the safe fixes: committing `local/` (after showing the changes) and unassigning missing IDs (after confirmation).

**Implementation notes:**
- `doctor` overrides the root `PersistentPreRunE`, so it checks the home as it is instead of repairing it first. A missing part of the home is an error; `--fix` runs `setup.Init`, which never overwrites.
- Checks that need an invalid `config.json` or `registry.json` are shown as skipped. Exit code 1 when errors remain; warnings alone exit 0.
- Assigned skills of a configured source that is not downloaded are listed but not unassigned by `--fix`: adding the source again is the fix.
- Name clashes in the global set are reported once as "every folder", not again per project.
- The MCP check warns when Claude Code runs another quiver binary than this one, and when the `claude` command is missing.

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

**Notes:**
- The SDK (v1.8.0) needs Go 1.25, so `go.mod` moved from 1.23 to 1.25. See decisions.md.
- The harness is `connect` and `call` in `internal/server/harness_test.go`.
- `logs/mcp.log` is never rotated, and every server process appends to it. `doctor` (TASK14) could warn when it gets large, or the server could rotate it at startup.
- The log records the folder each server starts in, so it lists the user's project paths. Mention this before anyone shares a log.
- `recoverPanics` only covers the handler's own goroutine. A panic in a goroutine that a handler or the server starts (for example the watcher in TASK21) still ends the process.
- `TestMCPStdoutHasOnlyMessages` re-runs the test binary as `quiver mcp`. If the server ever stops answering, the test hangs until the `go test` timeout.

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

**Notes:**
- `server.Resolve` returns a `Where`: `Dir` (the folder to register later), `From`, and `Project` (nil without `.quiver.json`).
- No tool reads `Where` yet, so its tests call `Resolve` directly. TASK17 should cover the served set per project through the harness.
- A missing `QUIVER_PROJECT` folder or a broken `.quiver.json` does not stop the server. It serves the global skills and only logs a warning, so the assistant never hears about it. TASK17 could mention it in `list_skills`.
- At startup the server also records the folder of a known project (`recordPath`), so a moved project gets its new path. It writes `registry.json` only when the paths change, and does not add projects the registry does not know.
- `Where` is fixed for the session. If `quiver add` registers the folder during a session, the server keeps serving only the global skills until the next session, unless TASK21 resolves again.
- Checked by hand: Claude Code starts the server in the project folder (`dir=` in `logs/mcp.log`).

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

**Notes:**
- TASK28 checked `update` only through `library.Get`. Add a test that runs `update` and gets the new content through `get_skill`.
- Remove `addPing` from `internal/server/server.go`. `TestPing` expects exactly one tool; replace it.
- The tools are in `internal/server/skills.go`, the catalog in `catalog.go`. Every call loads the library and the registry again, so skills assigned or updated during the session show at once. `TestGetSkillAfterUpdate` covers `update`.
- `New` now reads the library and the registry to build `instructions`, so every test that calls it must set `QUIVER_HOME`.
- A skill that is both global and assigned to the project has scope `global`. Assigned IDs that are no longer in the library are left out without a warning.
- `list_skills` returns structured output. The SDK also puts the same JSON into the text content.
- The budget counts bytes, which are never fewer than characters (`maxInstructions`). Descriptions are dropped from the last skill backwards, so skills early in ID order keep theirs. TASKA1 must pass `catalog` the budget left after its pointer.
- `get_skill_file` also refuses hidden files (a skill at the root of a source repository has `.git` in its folder) and anything that is not a regular file. `get_skill` leaves the same entries out of its file list, so broken symlinks and links to folders are not listed. Text that is not valid UTF-8 or has a NUL byte counts as binary.
- Still open from TASK16: a startup resolve error is only logged; `list_skills` does not mention it.

**Done when:** Harness tests cover all three tools, path traversal attempts, binary files and the `instructions` budget.
""")

task("TASK18", "Behaviour check against the real server", "TASK17, TASKA1", """
The B01 spike (`spike/b01`, results in `docs/spike-b01-results.md`) answered the design questions with a throwaway server. Repeat its checks against the real `quiver mcp` before calling the first milestone done.

1. Point `spike/b01/run.sh` and `run2.sh` at `quiver mcp` with fixture skills, or port them to a `make spike` target.
2. Expect the spike numbers: fitting prompts load the skill, non-fitting prompts load nothing, `get_skill_file` is read when `SKILL.md` refers to a file, and the always-on skill holds across five turns.
3. Check what the spike did not cover: context compaction, and a real caveman session with the caveman plugin disabled.
4. Add the results to `docs/spike-b01-results.md`.

**Notes:**
- `make spike` runs `spike/b01/real-all.sh`: the ten prompts, a five-turn session and a compacted session, each in a fresh `QUIVER_HOME` with the fixture skills in `local/`. `run-real.sh` runs one session. Every run is a real `claude -p` session and costs tokens.
- Results on 2026-10-27 match the spike: 6 of 6, 0 of 4, `get_skill_file` 2 of 2, always-on 10 of 10 and 5 of 5 turns. A second full pass gave the same numbers.
- Always-on survives `/compact` (4 of 4) because the compact summary restates the skill's rules. `get_skill` is not called again. A summary that drops the rules would lose the skill; the server cannot detect that.
- Caveman from the real source (`caveman/caveman`, always-on, user plugins off) loaded in turn one and held after compaction. Replies were about 15% shorter than the same session with caveman assigned but not always-on.
- `get_skill` ends with the skill's absolute folder, so the model may try `Read` before `get_skill_file`. In the headless runs `Read` was not allowed and it fell back to `get_skill_file`.

**Done when:** The real server matches the spike results, and compaction behaviour is recorded.
""")

task("TASK19", "`create_skill`", "TASK17, TASK09", """
**Input:** `name`, `description`, `body` (markdown without frontmatter), optional `files` (map of relative path to text content), `scope` (`"project"` or `"global"`).

1. Call `library.CreateLocal` (TASK09) with the resolved project. With `scope: "project"` and no `.quiver.json`, register the project first (TASK06), and say in the result that `.quiver.json` was created and should be committed.
2. The tool description tells the assistant to ask the user for the scope if the user did not say it.
3. Return the new skill's ID, folder path and scope.
4. Errors that the assistant can fix (invalid name, name taken) return a clear message and a suggested alternative name.
5. Mark the tool with the MCP annotations `readOnlyHint: false` and `destructiveHint: false`.

**Notes:**
- The tool is in `internal/server/create.go`. Its input schema restricts `scope` to `project` and `global`, and the handler checks it again.
- The served project used to be fixed at startup. It now lives in a `session` with a mutex (`internal/server/server.go`): when `create_skill` registers the folder, every later request in the session serves that project.
- `registry.RegisterProject` finds or registers the project and records it under one lock. `quiver new` uses it too.
- Error hints: an invalid name suggests its slug (`My Notes!` gives `my-notes`); a taken name suggests the first free `<name>-N`. Path errors come from `library.CreateLocal`.
- The server `instructions` now always end with a pointer to `create_skill`, inside the 3800-byte budget (`createHint`, `buildInstructions`). Without it the manual test failed: Claude Code defers MCP tools, the instructions were empty for an empty library, and Claude wrote the skill file itself.
- Manual test, run headless: `claude -p` in a new folder with `--mcp-config` pointing to `bin/quiver` and a temporary `QUIVER_HOME`. Claude called `create_skill` with `scope: "project"`, `.quiver.json` was created, and `quiver status` listed the skill as `project`.
- Backlog B07 (approval for global skills) is not planned: a skill created with `scope: "global"` is active at once.

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

**Notes:**
- The prompts are in `internal/server/prompts.go`; `get_skill` and the prompts share `skillText`. The prompt name keeps the `/` of a source skill's ID (`acme/review`): Claude Code uses prompt names as given, and `/mcp__quiver__acme/review` was checked to run.
- Claude Code lists a prompt as `/quiver:<id> (MCP)`; the server name is the prefix and cannot be dropped. A prefix-free `/<id>` would need skills synced into Claude Code's own skills folder: a separate task.
- Claude Code splits the text after the command on spaces, gives one word to each declared argument and drops the rest; quotes do not help. So the task is declared as `args`, `args2` … `args32` and joined again (`joinArgs`): up to 32 words.
- In `claude -p` only the `/mcp__quiver__<id>` form runs a prompt; `/quiver:<id>` is the interactive menu form.
- The prompt list is read at startup. A skill is read again on every get, so edits are seen; a deleted skill returns an error.

**Done when:**
- Harness tests list and get prompts.
- In Claude Code, `/mcp__quiver__<skill>` loads the skill.
""")

task("TASK21", "Live updates", "TASK19, TASK20", """
1. Watch `registry.json`, `config.json`, `local/` and `sources/` with `fsnotify`. Debounce events by 300 ms.
2. When the served set changes, reload it and send `notifications/prompts/list_changed`. `list_skills` returns the new set on its next call. The catalog in `instructions` stays as it was until the next session (TASK17).
3. Content changes to a skill need no notification, because `get_skill` always reads from disk.
4. A failed reload keeps the last good state and logs the error.

**Notes:**
- `recoverPanics` (TASK15) does not cover the watcher goroutine. Recover inside it, so a bad event does not end the server.
- The watcher is in `internal/server/watch.go`. fsnotify does not watch subfolders, so it watches the home folder (only `registry.json` and `config.json` count), `local/` and each skill folder in it, `sources/`, and each source's `.git` folder, which a pull or reset changes. New skill folders and new sources are added as they appear. Hidden names (`.lock`, `local/.new-*`) are ignored.
- `prompts.sync` diffs the served skills against the registered prompts and adds, replaces or removes only what changed, so a description change also updates the prompt. The SDK batches the `list_changed` notifications.
- `New` still returns a plain `*mcp.Server`; `newServer` also returns what `watch` needs. `Run` starts the watcher; tests start it with a 20 ms debounce.
- The harness tests change the registry with `registry.Update` instead of running `quiver add`: the server package cannot import the CLI. A run of the real binary showed `skill list changed` in the log about a second after `quiver add`.

**Done when:** A harness test runs `quiver add` from another goroutine and receives the notification. The new skill appears in `list_skills`.
""")

task("TASK22", "`quiver mcp install`", "TASK15", """
Registers the server with assistants, so the user never edits MCP config by hand.

1. **Claude Code:** run `claude mcp add --scope user quiver -- <absolute path to quiver> mcp`. If the `claude` command is not found, print the exact command for the user to run.
2. `--print`: only print the config snippet for manual setup.
3. `mcp uninstall` removes the registration again.
4. Detect an existing registration and update it if the binary path changed (for example after a Homebrew upgrade).

**Notes:**
- The `claude` calls are in `internal/assistant/claude.go`; the commands in `internal/cli/mcp.go`. The binary path is `os.Executable()` without resolving symlinks, so Homebrew's stable link is kept.
- `claude mcp get` shows only the registration that wins in the current folder: a local or project one hides the user one. Install therefore looks the user one up from a new empty temp folder, then warns if the current folder has a local or project registration that runs another binary.
- `claude mcp get` starts the server to check its health, so each lookup takes about a second.
- Tests use a fake `claude` script on PATH (`internal/assistant/assistanttest`) that models the user scope and one local registration.
- Checked against the real Claude Code CLI: install, install again (unchanged), `claude mcp get` from another folder shows the user registration as connected, uninstall twice. A new session with the tools is checked in TASK18.

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
5. Cache discovery results in `sources/<name>/.git/quiver-index.json`, keyed by the HEAD commit and an index version. Rescan only when either changes. A folder that is not a git repository is scanned every time.
6. `library.CheckEditable` refuses remote skills with `ErrReadOnly`. `edit` and `update_skill` must call it.

**Notes:**
- IDs from discovery can hold slashes (`<plugin>/<name>` on collision), so a remote ID can have three parts. Code that splits IDs must split on the first `/` only.
- Bump `indexVersion` whenever discovery rules change, or old caches keep the old result.
- The index first lived in the working tree, where `git status` showed it and an upstream file with the same name would block `git pull`. TASK28 moved it into `.git/`.
- `Load` now reads `config.json`, so a broken config fails every load, also for local skills.

**Done when:**
- `Library.All` returns local and remote skills together; `list` and `list_skills` build on it.
- A second load uses the cache, verified by a test hook.
""")

task("TASK27", "`quiver source add | list | remove`", "TASK23, TASK24, TASK26", """
**`add <repo> [--name <n>]`:**

1. Parse the input (TASK24) and check that the name is free, before the slow clone. Validate a `--name` value with `config.ValidateSourceName`.
2. `library.StageSource` clones into `sources/.add-<name>-*/<name>` and runs discovery.
3. If no skills are found, ask whether to keep the source anyway. No answer means no.
4. `Staged.Keep` moves the clone to `sources/<name>` and saves the config under one lock. Print the skills that were found.
5. Print a reminder that skills from this source give instructions to the assistant and may contain scripts.

**`list`:** Print each source's name, skill count, short commit, commit date and URL. A missing folder shows as `missing`.

**`remove <name>`:**

1. `library.RemoveSource` finds assigned IDs that start with `<name>/`, global and per project. If there are any, it refuses and lists where.
2. `--force` unassigns those skills first, always-on marks included.
3. Move `sources/<name>` aside, write the registry and the config, then delete the folder. A failed write puts the folder back.

**Notes:**
- The default name is the repository owner, so two repos from one owner (`anthropics/skills`, `anthropics/other`) collide. When the name is taken, suggest `--name` in the error.
- An interrupted add or remove leaves a `sources/.add-*` or `sources/.remove-*` folder. TASK14 should report and delete them.
- `git.Head` walks up to a parent repository. If `~/.quiver` is inside a git repo (dotfiles) and a source folder is not a repo, `list` and the index cache read the parent's HEAD.
- `list` shows the HEAD commit date, not the time of the last pull.
- If the config write fails after the registry write in `remove --force`, the skills stay unassigned while the source remains.

**Done when:** An end-to-end test adds a source from a local bare repo, assigns one of its skills, gets refused on remove, then succeeds with `--force`.
""")

task("TASK28", "`quiver update`", "TASK27", """
1. `update [<name>...]` runs `git pull --ff-only` for the given sources, or all of them.
2. Record HEAD before and after. Print the skills that were added, changed or removed.
3. If an assigned skill disappeared upstream, warn and list the projects using it. Do not unassign it automatically; `doctor` reports it.
4. If a source cannot be fast-forwarded, report it and continue with the other sources.
5. Running MCP servers pick up the change through live updates (TASK21).

**Notes:**
- Sources are shallow clones. After an upstream force-push, `--ff-only` fails. `update` reports it with a hint, and `update <name> --reset` runs `fetch --depth 1` and `reset --hard @{u}` instead of a re-clone. The old HEAD stays in the object store, so `DiffNames` still works right after the reset.
- The discovery index lives in `.git/quiver-index.json` (TASK26), outside the working tree, so it never blocks a pull. A source without its own `.git` folder (for example a submodule) gets no index and is scanned on every load.
- The test checks the new content with `library.Get`, because `get_skill` does not exist yet. TASK17 repeats the check through `get_skill`.
- `update` takes no lock while it pulls, so a `source remove` at the same time can move the folder away mid-pull.
- `--reset` drops local edits in a source folder. Quiver never edits sources (`CheckEditable`), but hand edits are lost.
- Only a diverged history gets the `--reset` hint. A pull that fails because of local edits or the network shows git's error only.
- A skill at the repo root (folder `.`) counts as changed on every new commit.
- A collision rename (B05) shows as one skill removed and one added, with the warning if the old ID is assigned. Nothing moves the assignment; B05 is still open.

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

**Implementation notes:**
- The suite is in `e2e/`. `TestMain` builds the binary; each test gets its own `HOME`, `QUIVER_HOME` and git identity, and a `PATH` with only git's folder, so the real `claude` is never run. MCP steps use the SDK's `CommandTransport` on `quiver mcp`. It runs with `go test ./...`.
- It found a bug: a server started in a folder that was not registered yet never served the project that `quiver add` then registered there. `session.projectID` now looks for `.quiver.json` again while the session has no project.
- CI (`.github/workflows/ci.yml`, the deferred step of TASK01) runs `go test ./...` on Linux, macOS and Windows, and `golangci-lint` on Linux. On Windows the tests that need the fake `claude` shell script, or a symbolic link, are skipped.

**Done when:** All flows pass in CI.
""")

task("TASK30", "README and help text", "TASK29", """
1. Write a README with: what Quiver is, install, a 2-minute quickstart (`init` → `mcp install` → `new` → ask the assistant), the command reference, the MCP tool reference, the limits of always-on skills (see TASKA1) and a security note about remote sources.
2. Give every cobra command a `Short`, a `Long` and an `Example`.
3. Add shell completion with `quiver completion zsh|bash|fish`. Skill, source and project arguments complete with real values.

**Implementation notes:**
- Completion functions are in `internal/cli/complete.go`. They read the home without the lock and never write; an error gives no suggestions. The root's auto-init is skipped for completion requests, so a tab press never creates `~/.quiver`.
- `add`, `where` and `edit` (local only) complete library IDs with their descriptions; `remove` completes what is assigned in the target (`--global`, `--project` or this folder); `update`, `source remove` and `list --source` complete sources; `projects rename|forget` complete IDs and unique names; `--project` and `import` complete folders. Commands without arguments suggest nothing instead of file names.
- `TestHelpText` fails when a command lacks `Short`, `Long` or `Example`.
- The README describes `go install` until the release adds Homebrew (TASK31). PROJECT.md said `quiver update` shows changes before applying; it lists them after pulling, so the README and PROJECT.md now say that.

**Done when:** `quiver help <cmd>` is useful for every command, and completion suggests skill IDs.
""")

task("TASK31", "Release", "TASK30", """
Ship v1 through three channels: GitHub Releases, a curl install script and npm. No Homebrew tap (decided 2026-10-28).

**Before the release** (fixes from the user-perspective review on 2026-10-28; A to D done):

A. **Git worktrees are not copies.** Folders with the same project ID that are worktrees of one git repository (same `git rev-parse --git-common-dir`) are one project: no `PossibleCopy` warning, and `projects split` refuses there, because it would change the committed `.quiver.json`.
B. **`quiver delete <id>`** removes a local skill from the library and commits the removal in `local/`. It asks first (`--yes` skips). If the skill is assigned, it lists where and refuses unless `--force`, which unassigns it too. Source skills cannot be deleted; `quiver source remove` removes the whole source.
C. **Windows editor:** without `$VISUAL` and `$EDITOR`, `quiver new` and `quiver edit` open `notepad` on Windows instead of `vi`.
D. **README:** assignments are per user and machine (a teammate's clone gets `.quiver.json` but no skills; share skills through a source), how to uninstall, and that assistants other than Claude Code may not start the server in the project folder, so they get only global skills unless `QUIVER_PROJECT` is set.

A to D notes: `git.CommonDir` and `registry.SameRepository` compare worktrees; `SplitProject` returns `ErrWorktree`. `library.DeleteLocal` moves the folder aside before the registry write, and commits only when git tracked the folder. The editor default is `defaultEditor` in `internal/cli/new.go`.

**Release:**

1. Set up GoReleaser (the free version) for darwin/linux/windows on amd64 and arm64. A tag `v*` runs it in GitHub Actions and publishes a GitHub Release with the archives and a checksums file.
2. **curl:** add `install.sh` to the repository. It detects the OS and CPU, downloads the archive of the latest (or a given) release, checks it against the checksums file, and installs `quiver` into `~/.local/bin` (overridable). It prints how to add that folder to `PATH` when it is missing. Usage: `curl -fsSL https://raw.githubusercontent.com/kheif/quiver/main/install.sh | sh`. Windows users install with npm or from the release archive.
3. **npm:** GoReleaser's npm publishing is Pro only, so publish with a script in the release workflow. Use the esbuild layout: one package per platform (such as `@<scope>/quiver-darwin-arm64`, with `os` and `cpu` set) holding the binary, and a main package `@<scope>/quiver` that lists them as `optionalDependencies` and has a small `bin` script that runs the right one. No postinstall download: pnpm 10 skips install scripts by default, and installs keep working offline from a cache. `quiver` itself is taken on npm, so the package is scoped.
4. **npm auth:** publish from GitHub Actions with npm trusted publishing (OIDC, `id-token: write`), so no npm token is stored. Trusted publishing only works for packages that already exist, so the user publishes a first version of each package by hand, then adds the trusted publisher on npmjs.com.
5. **Stable path for `mcp install`:** an npm install lives under the Node version (nvm, fnm, volta), so the binary path changes when the user switches Node. When `quiver mcp install` runs from inside `node_modules`, register the path of the `quiver` command on `PATH` instead, and print a note that a Node switch needs `quiver mcp install` again. The curl install path (`~/.local/bin/quiver`) is stable.
6. Update the README's Install section: curl first, then npm, then `go install`.
7. Decided 2026-10-28: B07 stays in the backlog for v1. Decide backlog B07 (approval for global `create_skill`) before tagging. A global skill's description reaches the instructions in every folder. The smallest fix: leave unapproved global skills out of the catalog.
8. Update the GitHub Actions to versions that run on Node 24 (CI warns about Node 20).
9. Tag `v1.0.0`. Pushing the tag publishes everything, so only after the user confirms.

**Release setup notes (steps 1 to 6 and 8 done):** `.goreleaser.yaml` names archives `quiver_<os>_<arch>` without the version, so `install.sh` can use `releases/latest/download`. `install.sh` takes `QUIVER_DOWNLOAD_URL` for mirrors and tests. The npm launcher is `npm/quiver/bin/quiver.js`; `scripts/npm-publish.mjs` builds the seven packages from `dist/` and skips versions already published. The launcher sets `QUIVER_NPM=1`, and `mcp install` then registers the `quiver` on `PATH` (`command` in `internal/cli/mcp.go`). `.github/workflows/release.yml` publishes on a `v*` tag and installs both ways on clean machines; CI's `release-dry-run` job does the same without publishing on every push. A `TestMain` in `internal/cli` now points `HOME` and `QUIVER_HOME` at a temp folder, because some `mcp install` tests had used the real `~/.quiver`.

**License:** Apache-2.0 (decided 2026-10-28, with the enterprise track in mind). `LICENSE` is in the repository, the archives and every npm package.

**Released 2026-10-28:** v1.0.0 is on GitHub Releases and npm. The npm packages were published by hand with 2FA (npm requires 2FA on the account to publish). The Release workflow passed: the install checks with curl on Linux and macOS and with npm on Linux, macOS and Windows. Trusted publishing is not set up yet; do it right before v1.0.1 (docs/releasing.md).

**The maintainer does by hand:** create the npm scope (the npm username or an organization), publish the first version of each package, and add the trusted publisher for `kheif/quiver` and the release workflow on npmjs.com.

**Done when:** on a clean macOS and Linux machine, the curl install and `npm install -g @<scope>/quiver` both give a working `quiver`, on Windows the npm install does, and the README quickstart runs.
""")


task("TASKC1", "v1.1: import one skill or a folder, version for go install", "TASK31", """
Stabilization before v2, from using v1.0.0 (2026-10-28). Released as v1.1.0: a new feature, so a minor version.

1. `quiver import <skill>` imports one skill: its `SKILL.md`, its folder, or any markdown file with skill frontmatter. `--name` sets its name in the library and is written into the copy's frontmatter; without it, the frontmatter `name` is used, then the folder or file name. A `SKILL.md` brings its folder (`scripts/`, `references/`); a lone markdown file is imported alone, without the other files of its folder.
1b. `quiver import <folder>` imports every skill in the folder and its subfolders, without `--deep`. A skill is a folder with a `SKILL.md`, or a markdown file of any name that starts with frontmatter (the content counts, not the file name). Markdown files without frontmatter, such as READMEs, are not skills and are not mentioned. Files that look like skills but have no description are skipped and listed at the end of the output. A skill folder is not searched further: its markdown files belong to it.
1c. Imported skills are assigned to the project of the current folder (or to every folder), also when a folder elsewhere is imported. `quiver import` without a path keeps searching this project's `.claude/skills`, or every folder of it with `--deep`.
2. `quiver --version` of a `go install` build shows the module version (`1.1.0`), not `dev`. A plain `go build` shows Go's VCS version; release builds keep the ldflags version.

**Not done (decided 2026-10-28):** prefix-free commands (TASKB1) stay for a later version. It is not measured yet whether the assistant reliably follows a pointer file to `get_skill`, so the MCP prompts stay as they are for now.

**Notes:**
- `importOne` in `internal/cli/import.go` handles one skill before the folder search; `library.ReadFound` reads a skill folder for both paths. A lone markdown file is copied to a temp folder as `SKILL.md` first. Deleting the original deletes the skill folder, or only the lone file.
- `import` now completes files as well as folders.
- `currentVersion` in `internal/cli/root.go` reads `debug.ReadBuildInfo` when no ldflags version is set.

- `library.FindSkills` returns `Found` with `File` set for a lone markdown file; `Found.Stage` copies it into a temp folder as `SKILL.md` for `ImportLocal`. `skill.LoadFile` and `skill.HasFrontmatter` read such files. `Found.Err` is set for an unreadable skill or one without a description; a missing or invalid name is not an error, because the import asks for one.

**Released 2026-10-29 as v1.1.0:** the first release published to npm by CI with trusted publishing (with provenance). npm took about 6 minutes to show every package, longer than the install check waited, so it now waits up to 15 minutes.

**Done when:** tests cover a lone file with `--name` and with its frontmatter name, a `SKILL.md` with its folder, a folder with nested skills, lone files, a README and invalid skills, and the errors; `go install github.com/kheif/quiver/cmd/quiver@v1.1.0` prints `quiver 1.1.0`.
""")

# ---------------------------------------------------------------- Phase 4c
phase("Phase 4c — v1.2")

task("TASKD1", "Hardening: git timeouts, race and fuzz tests, dependency checks", "TASKC1", """
From the v1 review (2026-10-29). Small fixes that make Quiver safer to run and to release.

1. **Git timeouts (backlog B08).** `internal/git` already sets `GIT_TERMINAL_PROMPT=0`, but a stalled network makes `git clone` or `git pull` wait forever, and `quiver update` hangs. Run git with `exec.CommandContext`: 5 minutes for clone, fetch and pull, 30 seconds for local commands. On a timeout, say which source and that the network did not answer.
2. **Race detector.** CI runs `go test -race ./...` (the server watches files while it answers requests).
3. **Fuzz test.** `FuzzParse` for the frontmatter parser in `internal/skill`: skills come from the internet, so no input may make it panic. A short `-fuzztime` run in CI.
4. **govulncheck.** A CI job runs `govulncheck ./...` and fails on a reachable known vulnerability.
5. **Dependabot.** `.github/dependabot.yml` for `gomod`, `github-actions` and `npm` (`npm/quiver`), monthly, with all updates of an ecosystem grouped into one PR.
6. **Pin actions to commits.** Every `uses:` in `.github/workflows/` names a full commit SHA with the version as a comment (`actions/checkout@<sha> # v7`). A tag can be moved to other code; a SHA cannot, and the release job can publish to npm. Dependabot keeps the SHAs current.
7. **Open-source files.** `SECURITY.md` (how to report a vulnerability privately), `.github/ISSUE_TEMPLATE/` (bug report with `quiver --version` and `quiver doctor` output, feature request), `CONTRIBUTING.md` (build, test, the task files).

**Done when:** a test makes a fake git hang and gets a timeout error; CI runs race, fuzz and govulncheck; Dependabot opens grouped PRs; no workflow uses a tag.

**Notes:**
- `internal/git` runs local commands with a 1 minute limit and clone, fetch and pull with 5 minutes (`runNetwork`); `WaitDelay` stops waiting for helpers such as `git-remote-https` that keep the output open.
- Releases up to v1.1.0 were built with Go 1.25.0, because `setup-go` read the `go` line of `go.mod`, which is only the minimum. govulncheck found over 30 fixed standard library vulnerabilities in that binary. The workflows now use `go-version: stable` with `check-latest`, and the release job runs govulncheck before publishing.
- The race detector runs on Linux and macOS; the Windows runner has no cgo.
- `skill.parse` is the parser without file access; `FuzzParse` found nothing in 10 million runs.
- Private vulnerability reporting must be turned on in the repository settings for SECURITY.md's link to work.
""")

task("TASKD2", "Duplicate skills: import removes originals, doctor and status find copies", "TASKC1", """
A skill that is both in Quiver and in a native skill folder is loaded twice (backlog B13). Not everyone runs `quiver doctor`, so import should not leave copies behind.

1. **Import removes originals.** After a skill is imported, its original is deleted by default; the copy is safe in the library's git.
   - Exception: an original tracked by the git of a project (a committed `.claude/skills/x`) is shared with teammates who may not use Quiver. For it the question defaults to no and says why, and `--yes` keeps it.
   - `--keep` keeps every original. `--yes` deletes the untracked ones.
   - Use `git.Tracked` to tell the cases apart.
2. **doctor.** A new check finds skills that Quiver serves here and that also exist natively: in `~/.claude/skills`, in this project's `.claude/skills`, and in enabled Claude Code plugins. Match by name, and by same content. `--fix` offers to delete untracked copies; tracked ones are reported only.
3. **status.** `quiver status` prints a warning line for each native copy of a served skill, with the hint `quiver doctor --fix`.

**Done when:** tests cover import of tracked and untracked originals with and without `--yes` and `--keep`, the doctor check and fix, and the status warning; the README and import help describe the new default.

**Notes:**
- `library.NativeSkills` reads `~/.claude/skills` (or `$CLAUDE_CONFIG_DIR/skills`), the project's `.claude/skills`, and the `skills/` folder of each plugin in `plugins/installed_plugins.json` that the user, project or local settings enable; a later settings file wins, as in Claude Code. A project-scoped plugin counts only in its project.
- Plugin copies are reported, never deleted: a plugin update would bring them back. The hint is to disable the plugin.
- An import question that gets no input at all, as at the end of a script's input, keeps the original.
- Paths in the doctor and status lines are absolute, so a copy in `~/.claude/skills` and one in the project are told apart.
""")

task("TASKD3", "`quiver show`, `quiver search`, every skill in the catalog", "TASKC1", """
1. **`quiver show <id>`** prints a skill: ID, where it comes from (local, or source and commit), where it is assigned, always-on, its files, then the `SKILL.md` text. `--json`. Completion of IDs.
2. **`quiver search <text>`** searches the whole library, also unassigned skills: case-insensitive match on ID, name and description. Rank: exact name, name prefix, name contains, description contains. Show whether each match is assigned here. `--json`. Put the search in `internal/library`, so `search_skills` (Phase 5) and the TUI use the same function.
3. **Every skill in the catalog, with its description.** Claude Code cuts the server's `instructions` at about 4096 characters (`maxInstructions` is 3800), and hook output at about 10,000. Today `catalog` in `internal/server/catalog.go` then drops descriptions from the end (about 20 skills keep one) and leaves skills out after about 150. A tool result has no such limit, and the always-on pointer showed that the assistant calls a tool before its first reply when told to (10 of 10, and after `/compact`). Decided 2026-10-29:
   - When the catalog fits, it stays in `instructions` as today.
   - When it does not fit, `instructions` hold no list, only a pointer like the always-on one: before your first reply, call `list_skills` of this server; it lists the skills of this folder, and when one fits a task, load it with `get_skill`. The always-on pointer stays first.
   - `list_skills` returns every served skill with its full description (project skills first, then global), and marks always-on ones.
   - Remove the shortening and dropping in `catalog`; keep the `maxDescription` cut only as a guard against huge descriptions.
   - Measure it like the always-on spike (docs/spike-b01-results.md): with a catalog over the budget, a fitting prompt in a new session calls `list_skills` and then loads the skill, in at least 9 of 10 runs, also after `/compact`. Record the result there.
   - Too many skills for one tool result (hundreds) is for later: group them by area, and list an area on request.

**Done when:** tests cover show for a local and a remote skill, search ranking, the catalog in `instructions` when it fits and the `list_skills` pointer when it does not, and `list_skills` with 200 skills.

**Notes:**
- `catalog` lists every served skill or none: a list over the budget becomes `listPointer`. `maxDescription` is 1024, the Agent Skills limit.
- `list_skills` adds `always_on` and sorts project skills first; `loadServed` keeps each part sorted by ID.
- `library.Search` ranks an exact ID like an exact name, and an ID that contains the text like a name that contains it.
- `quiver search` shows `global` or `project` under HERE; `quiver show` gives the HEAD of `local/` or of the source as the commit.
- Measured with 44 skills: 10 of 10 fitting prompts called `list_skills` and loaded the skill, 4 of 4 after `/compact` (docs/spike-b01-results.md, `make spike-catalog`).
""")

task("TASKD4", "`quiver mcp install` for Cursor and Claude Desktop", "TASKC1", """
Today `mcp install` registers Quiver only with Claude Code (`internal/assistant/claude.go`). Pulls the first part of TASK38 ("`mcp install` for more assistants") into v1.2.

1. **Spike first, per assistant.** The catalog and always-on skills reach the assistant through the MCP `instructions`. Check that Cursor and Claude Desktop pass `instructions` to the model, as in docs/spike-b01-results.md: does a fitting prompt load the skill, does an always-on skill apply from the first reply? Record the results in `docs/`. If an assistant ignores `instructions`, decide with the user before adding it.
2. **Shared JSON installer.** Most assistants keep MCP servers in a JSON file under `mcpServers`. One function adds, updates and removes the `quiver` entry, keeps every other key and the file's indentation, backs up the original, and writes atomically. A broken JSON file is an error, never overwritten.
   - Cursor: `~/.cursor/mcp.json`.
   - Claude Desktop: `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS, `%APPDATA%\\Claude\\claude_desktop_config.json` on Windows.
3. **CLI.** `quiver mcp install [--for claude-code|cursor|claude-desktop]`. Without `--for`, install for every assistant that is installed, and list what was done. `mcp uninstall` the same. `quiver mcp config` prints the JSON snippet for any other assistant.
4. The command is the absolute path of the binary (backlog B09), or the npm shim as today. Claude Desktop does not run in a project folder, so it gets only global skills: say so after installing.
5. `doctor` checks each installed assistant's entry.

**Done when:** tests cover install, update, uninstall, other keys kept and a broken file for each JSON assistant; the spike results are written down; the README lists the assistants.

**Notes:**
- Spike results in docs/spike-mcp-clients.md. Cursor (CLI 2026.10.22, `spike/b01/cursor-run.sh`): 6 of 6 fitting prompts loaded the skill, 0 of 4 others, always-on in 10 of 10; the server starts in the workspace folder, so project skills work. Claude Desktop 4.4212.0 (Chat): the tools work, but the model never gets `instructions`.
- Decided 2026-10-31: Claude Desktop is set up only with `--for claude-desktop` (`JSONAssistant.NoInstructions`), with a note that skills are used only on request. `doctor` checks its entry only when there is one; `mcp uninstall` without `--for` removes it.
- `internal/assistant/jsonconfig.go` keeps key order, the file's indentation and mode, and the entry's other fields such as `env`; it writes `<file>.quiver-backup` before each change. `fsutil.WriteFileAtomic` is the byte version of `WriteJSONAtomic`.
- The JSON assistants always get the binary itself (`appCommand`), also from an npm install: with a desktop app's `PATH` the npm launcher fails with `env: node: No such file or directory`. Claude Code keeps the launcher on `PATH`.
- `mcp install --print` is deprecated in favour of `quiver mcp config`.
""")

task("TASKD5", "First run and rough edges", "TASKD2", """
From the v1 review: Quiver has many concepts (library, source, global, project, always-on), and a few prompts are hard to read.

1. **First run.** `quiver init` in a terminal becomes a short guided setup: create the library, register the MCP server (`mcp install`), and offer to import existing skills from `~/.claude/skills` and from the `.claude/skills` of the current project. Every step can be skipped. `--yes` and a non-terminal run keep today's plain init.
2. **`quiver` with no arguments,** until the TUI exists, prints three lines: the project here, how many skills it is served, and a next step (`quiver init`, `quiver add`, …), then the command list.
3. **Clearer import question.** `[P/g/s]` becomes words: `Assign to (p)roject, (g)lobal, or (s)kip? [p]`.
4. **Confirm a new project.** When `add`, `import` or `new` would create a `.quiver.json`, show the folder first (`Register /x/y as a project? [Y/n]`), so a command run in the wrong folder does not create a project there. `--yes` skips the question.
5. **`--json`** for `status`, `projects` and `where`, like `list`.
6. **New-session note.** After `add` and `remove`, say that open assistant sessions see the change through `list_skills`, and the catalog in the next session.

**Done when:** tests cover the guided init with each step taken and skipped, the bare `quiver` output, the confirmation, and the JSON output.

**Notes:**
- A run counts as a terminal when stdin is one (`interactive` in `internal/cli/term.go`). The guided init reads every answer through one `bufio.Reader`, so no answer is read ahead. Skills in `~/.claude/skills` default to global, those in the project's `.claude/skills` to the project; an import offer is shown only when there are skills to import, and the project offer is left out in the home folder, where both are the same folder.
- `quiver` alone skips the auto-init, so on a new machine it suggests `quiver init` instead of creating the home. Next steps: `quiver init` without a home, `quiver new`/`quiver source add` with an empty library, `quiver add` when nothing is served here, else `quiver status`. It uses `QUIVER_PROJECT` like `status`.
- The project question is asked only in a terminal, so scripts and pipes keep working without `--yes`. It names the folder `Register` would use: `project.NewRoot`, the nearest folder with `.git`. In `import`, a "no" skips the skill and every later project skill of the run, without asking again.
- JSON fields: `status` gives `folder`, `project` (null outside a project), `skills` (`id`, `scope`, `always_on`, `in_library`) and `warnings`; `projects` gives `projects`, each with `id`, `name`, `skills`, `paths` and `missing`; `where` gives `id`, `in_library`, `global` and `projects` in the same form. `printJSON` in `list.go` is shared by every `--json`.
""")

task("TASKD6", "Notice of a new Quiver version", "TASKC1", """
1. At most once a day, a command asks GitHub for the latest release (`api.github.com/repos/kheif/quiver/releases/latest`) with a short timeout, in the background, and stores the answer in `~/.quiver/state/version-check.json`.
2. When a newer version is known, commands end with one line on stderr: `Quiver 1.3.0 is out (you have 1.2.0): npm install -g @kheif/quiver` (or the curl line, or `go install`, depending on how it was installed; `QUIVER_NPM` tells npm).
3. Never in `quiver mcp` (stdout is the protocol), with `--json`, outside a terminal, for `dev` builds, or with `QUIVER_NO_UPDATE_CHECK=1`. A failed check is silent.
4. `quiver doctor` shows the latest version as an info line.

**Done when:** tests with a fake release endpoint cover newer, same, failed and disabled, and that the check runs at most once a day.

**Notes:**
- `internal/release` does the check: `Check` asks GitHub only when the saved check is a day old (or from the future), saves a failed request too, so it is tried again only the next day, and keeps the version found before. A broken state file is replaced; without a Quiver home nothing is asked. `Newer` compares major.minor.patch; a release is newer than its pre-releases. `LatestURL` is replaced in tests.
- `cli.execute` starts the check before the command and waits for it at the end; `release.Timeout` (2 seconds) bounds the wait, once a day at most. `quiver mcp` and its shell completion never start it; `mcp install` and the other subcommands do. A command with `--json` set runs the check but prints no line.
- A development build is `dev`, a Go pseudo-version (a commit between releases) or a version with `+` (`+dirty`). The install line: npm (`QUIVER_NPM=1`), `go install` (no ldflags version), the releases page on Windows until TASKD7, else the curl line, with `QUIVER_INSTALL_DIR` when the binary is not in `~/.local/bin`. `assistant.ShellQuote` is now exported for it.
- `doctor` shows the latest version as an `info` line, which counts as neither warning nor error, and skips it with `QUIVER_NO_UPDATE_CHECK` or for a development build.
- Checked against the real GitHub API on 2026-10-31: a 1.0.0 build reported 1.1.0.
""")

task("TASKD7", "`install.ps1` for Windows", "TASKC1", """
1. `install.ps1`, used as `irm https://raw.githubusercontent.com/kheif/quiver/main/install.ps1 | iex`: the same steps as `install.sh` for the Windows zip (`quiver_windows_<arch>.zip`), with `QUIVER_VERSION`, `QUIVER_INSTALL_DIR` (default `%LOCALAPPDATA%\\Programs\\quiver`) and `QUIVER_DOWNLOAD_URL`.
2. Check the SHA-256 against `checksums.txt`, and add the folder to the user's `PATH` (user scope, no admin).
3. CI's `release-dry-run` and the release workflow's install checks run it on Windows.
4. README: the Windows install line; `install.sh`'s Windows error mentions it.

**Done when:** a clean Windows runner installs with the script and `quiver --version` works in a new shell.

**Notes:**
- `irm | iex` runs the script in the caller's shell, so everything is in one `& { }` block and errors `throw`: `exit` would close the shell. It works in Windows PowerShell 5.1 (TLS 1.2 is turned on, the progress bar off) and in PowerShell 7.
- `QUIVER_DOWNLOAD_URL` also takes a local folder or a `file://` URL, which are copied, because `Invoke-WebRequest` does not read files; CI installs from `dist/` this way.
- A running `quiver.exe` cannot be overwritten but can be renamed: the old one becomes `quiver.exe.old` and is deleted on the next install.
- The `PATH` is changed in `HKCU:\Environment` with the raw value and its registry type, so entries such as `%USERPROFILE%\bin` stay unexpanded; setting and clearing a variable through `[Environment]` then tells Explorer, so new shells get the new `PATH`.
- CI's `release-dry-run-windows` installs the snapshot archive in Windows PowerShell 5.1, again over it in PowerShell 7, and runs `quiver --version` with the `PATH` read from the registry, as a new shell gets it; a step keeps the runner's `PATH`. The release workflow's `verify-powershell` does the same with `irm | iex` from the tag.
- The new-version line on Windows is the `irm` line, with `$env:QUIVER_INSTALL_DIR` when `quiver.exe` is not in its default folder.
""")

task("TASKD8", "Release v1.2.0", "TASKD1, TASKD2, TASKD3, TASKD4, TASKD5, TASKD6, TASKD7", """
1. README and help text cover every change of this phase; PROJECT.md's roadmap moves the pulled-in items from v2 to v1.2.
2. Mark the done items in `tasks/backlog.md` (B08, B09, B13).
3. Tag `v1.2.0` only after the maintainer confirms. The release workflow publishes to GitHub and npm and installs on clean machines.

**Done when:** the release workflow is green.

**Notes:**
- The roadmap and backlog B08, B09 and B13 were updated by the tasks themselves; PROJECT.md's CLI list gained bare `quiver`, `show` and the guided `init`.
- Tagged `v1.2.0` on 2026-10-31. The release workflow's `verify-powershell` job ran `install.ps1` from the tag for the first time.
""")

# ---------------------------------------------------------------- Phase 5
phase("Phase 5 — v1.3")

task("TASKE1", "Split machine-local state out of the registry", "TASKD8", """
Backlog B04. `registry.json` holds each project's absolute `paths`. Synced between machines (Phase 5's library sync), the paths are wrong on the other machine and both machines rewrite the file, so every sync conflicts.

1. Keep only portable data in `registry.json`: global and always-on skills, and per project its ID, name, skills and always-on skills.
2. Move `paths` and other machine-local data into `~/.quiver/state/` (ignored by git). The version check of TASKD6 is already there.
3. Migrate: registry version 2. Reading version 1 moves the paths into the state on the first write; an older quiver reading version 2 already stops with "update quiver".
4. Update data-model.md.

**Done when:** tests cover the migration, and two copies of `~/.quiver` with different project paths share one `registry.json` without changes to it.
""")

task("TASKE2", "Stable skill IDs", "TASKE1", """
Backlog B05. A remote skill's ID is `<source>/<name>`, and `<source>/<plugin>/<name>` only when names collide. If an update adds a skill with the same name, the existing ID changes and its assignments point at nothing.

1. Store each remote skill's ID when it is first discovered (in the source's entry in `config.json` or in the state), and keep it across updates. Only new skills get the qualified form on a clash.
2. A skill that disappears from its source keeps its ID; `quiver update` and `doctor` report it as removed upstream, as today.
3. Update data-model.md.

**Done when:** a test updates a source so that a second `pdf` appears, and the existing `src/pdf` assignment still works.
""")

task("TASK32", "`update_skill` MCP tool", "TASK19", """
1. Input: `id`, and any of `description`, `body`, `files` (path to content; empty content deletes the file).
2. Only local skills can be updated. Check with `library.CheckEditable`; for remote skills, return an error suggesting `fork`.
3. Commit in `local/` with the message `update <name> (via MCP)`.
4. The tool description asks the assistant to confirm the change with the user first.

**Done when:** Harness tests cover a body change, a file add and delete, a remote skill and path traversal.
""")

task("TASK33", "`search_skills` MCP tool", "TASKD3, TASK17", """
`quiver search` and the search in `internal/library` are done in TASKD3.

1. MCP: `search_skills` (`query`) returns the matches of the library search and whether each one is assigned here. The description tells the assistant it may suggest an unassigned skill to the user but should not assume it applies.

**Done when:** the MCP tool marks assigned and unassigned skills correctly.
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

task("TASK38", "`mcp install` for more assistants", "TASKD4", """
TASKD4 added Cursor and Claude Desktop and the shared JSON installer. `mcp install` and `mcp uninstall` take assistant names as arguments.

1. Find out how Codex, Gemini CLI, VS Code and others register stdio MCP servers (config file path and format, or a CLI command). Codex uses TOML.
2. Run the TASKD4 spike for each: does it pass the MCP `instructions` to the model?
3. Add each one that does as a name for `mcp install <name>`.
4. Confirm that each assistant passes the project folder (roots or working directory) correctly. Record any assistant that needs `QUIVER_PROJECT`.

**Done when:** Install and uninstall work for each supported assistant, and each one sees the correct project's skills.

**Notes:**
- Spike results in docs/spike-mcp-clients.md (`spike/b01/codex-run.sh`, `spike/b01/antigravity-run.sh`). Codex 0.166 defers MCP tools, so its model gets `instructions` only after it searches for Quiver's tools: 0 of 6 fitting prompts loaded a skill unprompted, every one did when the prompt named Quiver. Antigravity CLI 1.4.12 writes `instructions` to a file the model rarely reads: 1 of 6. Both start the server in the working directory, so project skills work. Gemini CLI 0.66 could not be measured: its Google sign-in is refused. VS Code is not measured yet.
- Codex is set up with `codex mcp add`, which keeps `config.toml`'s comments; it replaces the whole entry, so `internal/assistant/codex.go` passes an existing `env` again. Antigravity is a JSON assistant (`~/.gemini/config/mcp_config.json`), installed when `~/.gemini/antigravity-cli` exists.
- Decided 2026-11-01: `mcp install` without names sets up every supported assistant that is installed, Claude Desktop included (`JSONAssistant.NoInstructions` became `OnRequest`). Codex, Claude Desktop and Antigravity get a note that they use Quiver's skills only when asked. `doctor` checks every installed assistant.
- The read tools (`list_skills`, `get_skill`, `get_skill_file`, `search_skills`) are marked `readOnlyHint`, so Codex runs them without asking.
- `internal/cli` tests drop the folders holding `codex` from `PATH` and use `assistanttest.InstallCodex`; `TestRealCodex` checks the fake against the real codex in a throwaway `CODEX_HOME` when it is installed.
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

task("TASKB1", "Prefix-free commands in Claude Code", "TASK20, TASK10, TASK22, TASKF4.1", """
MCP prompts (TASK20) appear in Claude Code as `/quiver:<id>`, and the server name cannot be dropped. Claude Code's own skills have no prefix: a folder `.claude/skills/<name>/SKILL.md` becomes `/<name>`. Quiver writes a small Claude Code skill for every served skill, so the user types `/review ship 1.2`.

Facts from the Claude Code docs (checked 2026-10-28):
- Personal skills live in `~/.claude/skills/<name>/`, project skills in `<project>/.claude/skills/<name>/`. On a name clash, personal wins over project.
- `$ARGUMENTS` holds all the text after the command; quotes keep words together. No 32-word limit as with prompts.
- `disable-model-invocation: true` keeps Claude from loading the skill by itself, so it does not repeat the MCP catalog.
- Claude Code watches these folders and picks up changes in the running session.

1. For each served skill, write one Claude Code skill:
   - global skills: `~/.claude/skills/<command>/SKILL.md`
   - project skills: `<project>/.claude/skills/<command>/SKILL.md` in every recorded project path, and add the folder to the repository's `.git/info/exclude` so it is never committed
2. The file has frontmatter `name`, `description`, `disable-model-invocation: true` and a marker `quiver-skill: <id>`. Its body tells the assistant to load `<id>` with the quiver server's `get_skill` tool, follow it, and use `$ARGUMENTS` as the task.
3. The command is the skill's `name`: `acme/review` gives `/review`. If two served skills share a name, both use their ID with `/` replaced by `-` (`/acme-review`).
4. Never change or delete a folder without the marker. If the user's own skill has the name, skip it and warn in `quiver status` and `quiver doctor`.
5. Sync after every change to what is served: `add`, `remove`, `new`, `create_skill`, `update`, `source remove`. Delete generated folders whose skill is no longer served.
6. Opt in: `quiver install` asks once (or `--commands`), saved in the config. `quiver uninstall` removes every generated folder.

**Decide first:** the body could instead hold a copy of the skill. That needs no tool call and works when the server is down, but extra files (`references/…`) are then missing, so the pointer is the default.

**Done when:**
- Tests cover naming, name clashes, the marker rule, removal of stale folders and `.git/info/exclude`.
- In Claude Code, `/review ship version 1.2` loads the skill and gets the whole task.
""")

task("TASKE3", "Release v1.3.0", "TASKE1, TASKE2, TASK32, TASK33, TASK34, TASK35, TASK36, TASK37, TASK38", """
Versions follow Semantic Versioning: only a breaking change needs a new major version. Everything since v1.2.0 adds features except one dropped flag, so this phase is released as v1.3.0 once that flag is back.

1. **`--for` again.** v1.2.0 named one assistant with `quiver mcp install --for <assistant>`; TASK38 made the assistants arguments and dropped the flag. Bring it back on `install` and `uninstall` as a hidden, deprecated flag (`MarkDeprecated`, like `--print`) that counts as one more argument, so `quiver mcp install --for claude-desktop` does what it did in v1.2.0 and points to `quiver mcp install claude-desktop`. Test both commands with the flag.
2. **Upgrade check.** Make a Quiver home with the v1.2.0 binary (init, source add, add, add --always, a project), run the new binary on it, and check that the same skills, assignments and always-on skills are served and `doctor` is clean.
3. README and help text cover every change of this phase; PROJECT.md's roadmap lists them under v1.3.
4. The GitHub release notes list the changes since v1.2.0 and the deprecated `--for`.
5. Tag `v1.3.0` only after the maintainer confirms. The release workflow publishes to GitHub and npm and installs on clean machines.
6. **Website.** The status marks follow the release (`latestRelease`, `unreleased` and the **Next release** asides; see `web/README.md`). `quiver install` is not in v1.3.0, so its asides stay. Website changes are committed only after the maintainer confirms.

**Done when:** the release workflow is green and `quiver mcp install --for <assistant>` works in the release.

**Notes:**
- `addForFlag` and `withFor` in `internal/cli/mcp.go`; `TestMCPInstallFor` covers install, uninstall with arguments, the deprecation line and the help.
- Upgrade check on 2026-11-02, with a v1.2.0 binary built from the tag and a fixture source (two global skills, one always-on, and an always-on project skill): `status`, the MCP `instructions` and `list_skills` were the same before and after the first write, `registry.json` became version 2 with `state/projects.json`, and `doctor` reported no errors.
- The README and PROJECT.md already covered the phase; each task had updated them. The release notes are GoReleaser's changelog of the commits, as for v1.2.0.
- Website: the **Next release** asides and badges of the phase are gone, `latestRelease` is v1.3.0 with an empty `unreleased`, and the `quiver install` aside names v1.3.0. Three tests in `web/tests` that held the v1.2.0 state follow it.
- Tagged `v1.3.0` on 2026-11-02.
""")

task("TASKE4", "Release v1.5.0", "TASK39, TASK40, TASKB1", """
1. README and help text cover every change of this phase; PROJECT.md's roadmap lists them under v1.5.
2. The GitHub release notes list the changes since v1.4.0.
3. Tag `v1.5.0` only after the maintainer confirms. The release workflow publishes to GitHub and npm and installs on clean machines.
4. **Website.** The status marks follow the release (see `web/README.md`). Website changes are committed only after the maintainer confirms.

**Done when:** the release workflow is green.
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

**Notes:**
- `add` and `remove` share their flags through `target` in `internal/cli/assign.go`. Add `--always` there, and mark always-on skills in `printServed`.
- `quiver add --always` assigns and marks in one registry update; `quiver remove --always` only unmarks. The registry methods are `MarkAlwaysOn`, `UnmarkAlwaysOn`, their `Global` variants and `AlwaysOnServed`, in `internal/registry/assign.go`. Marking a skill that is not assigned in the same place is an error.
- `remove --always` in a project, for a skill that is always-on globally, prints a note to use `--global`. `printServed` adds an `always-on` column.
- The pointer is built by `pointer` in `internal/server/catalog.go`. `catalog` gets the budget left after it. Always-on IDs that are no longer in the library are left out, like in the catalog.
- The README does not exist yet. TASK30 must document the three limits above.

**Done when:**
- A harness test gets the pointer in `instructions`, ahead of the catalog.
- In Claude Code, a new session follows a global always-on caveman skill from the first message, with the caveman plugin disabled (checked in TASK18).
""")

# ---------------------------------------------------------------- Phase 6
phase("Phase 6: Session-start delivery (v1.4)")

task("TASKF1", "Spike: session-start hooks per assistant", "TASK38", """
Always-on skills reach the model through the MCP server's `instructions` (TASKA1). Only some assistants put those into the prompt: Claude Code and Cursor do; Codex loads them only after the model searches for Quiver's tools, Antigravity writes them to a file the model rarely reads, and Claude Desktop drops them (docs/spike-mcp-clients.md). In those three, always-on skills applied in 0 of 10 prompts.

A session-start hook solves this where an assistant has one: the assistant runs a command when a session starts and adds its output to the model's context, so the skill text is there without the model deciding to load it. This spike finds out, per assistant, whether such a hook exists and how it behaves. TASKF2 and TASKF4 are built on its results.

**Delivery levels.** Each assistant gets one:
- **hook:** the always-on skills' text is added at session start (TASKF2).
- **instructions:** the model gets Quiver's `instructions` up front and loads the always-on skills itself (Cursor, 10 of 10 in TASKD4).
- **rules file:** a pointer in a file the assistant always loads makes the model load Quiver (measured in TASKF1.4; no installed assistant needs it).
- **on request:** skills load only when the prompt asks for them (Claude Desktop).

**How to run.** Use the B01 fixture skills and prompts: `spike/b01/gui-setup.sh` builds a throwaway Quiver home whose always-on skill `reply-style` ends every reply with `[always-on]`, and the fitting skills end their answer with `[used:<id>]`; `spike/b01/prompts.txt` has 6 fitting and 4 other prompts. Put hooks into a throwaway workspace's project config where the assistant reads one, or into a throwaway config folder; never change the user's own config. The existing runners (`codex-run.sh`, `cursor-run.sh`, `antigravity-run.sh`, `run-real.sh`) show how each assistant is run headless.

**Output.** `docs/spike-hooks.md`, in the style of docs/spike-mcp-clients.md: one section per assistant with the version, the setup, the counts (always-on in n of 10, fitting skills in n of 6, other skills loaded in n of 4), and one summary table with a row per assistant and these columns: hook event, user and managed config file, input on stdin, output format, output reaches the model, size limit, after compaction, sub-agents, trust or approval prompt, command form on Windows, global rules file, delivery level.

**Done when:** every subtask is done and `docs/spike-hooks.md` gives each assistant a delivery level.
""")

task("TASKF1.1", "Claude Code hooks", "TASK38", """
Facts to confirm against the current Claude Code docs before measuring:
- Hooks are set in settings files: `~/.claude/settings.json` (or `$CLAUDE_CONFIG_DIR/settings.json`) for the user, `.claude/settings.json` in a project, and a managed settings file for companies.
- The entry has this shape:
  ```json
  {"hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": "/path/to/quiver hook session-start claude-code"}]}]}}
  ```
- The hook gets JSON on stdin with `session_id`, `cwd`, `hook_event_name` and `source` (`startup`, `resume`, `clear` or `compact`). For `SessionStart`, its stdout is added to the context, or the JSON `{"hookSpecificOutput": {"hookEventName": "SessionStart", "additionalContext": "..."}}`.

Steps, in a throwaway workspace with the hook in its `.claude/settings.json`:
1. The hook prints `reply-style` in full. Run the 10 prompts with `claude -p` and count `[always-on]` and `get_skill` calls for `reply-style` (expected: none).
2. Compare plain stdout with `additionalContext`. Find the size limit: print 5, 10, 20 and 50 KB and check whether the end of the text still reaches the model.
3. Check each `source`: is the text there after `/compact`, after `/clear`, and in a resumed session?
4. Sub-agents: does a sub-agent started with the Task tool see the text? Is there an event that can add context to a sub-agent?
5. Managed settings: find the file's path on macOS, Linux and Windows, whether its hooks run next to the user's, and whether a user can turn them off.
6. Time: how long may the hook run before Claude Code gives up, and does a slow hook delay the first reply?

**Done when:** the results are in `docs/spike-hooks.md`.

**Notes:**
- Results in docs/spike-hooks.md (Claude Code 2.1.308, `spike/b01/claude-hooks-run.sh`). Delivery level: hook. Always-on 10 of 10, fitting skills 6 of 6, other skills 0 of 4, with plain stdout and with `additionalContext` alike, and no `get_skill` for `reply-style`. With the server's pointer also active, the model still loaded it in 6 of 10.
- Limit: 10,000 characters per hook command, counted in characters. Above it the model gets a file path and a 2 KB preview, and a rule after the preview was followed in 0 of 6.
- Sources: `clear` needs the hook, or the rule is gone. On `resume` the earlier text stays and a new firing adds a second copy. The matcher `startup|clear|compact` gives every context the text once.
- `SessionStart` does not reach sub-agents (0 of 6); `SubagentStart` with `additionalContext` does (6 of 6).
- A failing hook never stops a session, and its text is dropped. A slow hook delays the first reply by its whole run time; the default timeout is 600 seconds.
- Besides the facts listed above, stdin has `transcript_path`, interactive sessions add `model` and `scratchpad_dir`, and `source` can also be `fork` (documented).
- Not measured: the user's own settings file, the managed settings file, Windows, automatic compaction, and what the terminal shows for a failed hook.
""")

task("TASKF1.2", "Codex hooks", "TASK38", """
Facts seen on 2026-11-01 with Codex 0.166:
- Hooks are read from `~/.codex/hooks.json` (or `$CODEX_HOME/hooks.json`), in the same shape as Claude Code's (`{"hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": "...", "timeout": 10}]}]}}`).
- The binary has the events `SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`, `PreCompact`, `PostCompact`, `SubagentStart` and `SubagentStop`, the output field `additionalContext`, a setting `additionalContextLimit`, and a managed hook folder (`managedDir`).
- Hooks need trust: `config.toml` keeps a `[hooks.state."<file>:<event>:<i>:<j>"]` table with `enabled` and a `trusted_hash` for each hook. `codex exec` has `--dangerously-bypass-hook-trust`.

Steps:
1. Use a throwaway `CODEX_HOME` with its own login (`codex login`); never copy or change `~/.codex`.
2. Run the checks of TASKF1.1 with `codex exec --json` and once in the interactive CLI.
3. Trust: what does the user see for a new hook, in the CLI and in the Codex app? Can `quiver install` make a hook trusted without editing `config.toml` by hand, for example through a `codex` command? What changes when the hook's command changes (a moved binary)?
4. Record the default and maximum of `additionalContextLimit`, and the path and rules of the managed hook folder.
5. Find out whether a project can have its own `hooks.json`.

**Done when:** the results are in `docs/spike-hooks.md`.

**Notes:**
- Results in docs/spike-hooks.md (Codex 0.166.0, `spike/b01/codex-hooks-run.sh`, source tag `rust-v0.166.0`). Delivery level: hook. Always-on 10 of 10 with plain stdout and with `additionalContext` alike, fitting skills 0 of 6 and other skills 0 of 4 as without the hook, since the model never searches for Quiver's tools. With the server's pointer also active, no `get_skill` for `reply-style`.
- The hook runs when the first prompt is sent. A failing hook never stops a session; `codex exec` reports nothing, the TUI prints `Hook failed`. Codex does not check `hookEventName`. The command runs as `$SHELL -lc` (`cmd.exe /C` on Windows, where `commandWindows` replaces `command`).
- Limit: `additionalContextLimit` per hook, default 2,500 tokens counted as 10,000 bytes, no maximum, `0` turns it off. Above it the model gets the start and the end and a file path; a rule in the middle was followed in 0 of 6.
- Sources as in Claude Code, plus `fork`. Compaction keeps no hook text, so `compact` needs the hook. The matcher `startup|clear|compact` gives every context the text once.
- Sub-agents fire `SubagentStart` (plain stdout or `additionalContext`, 6 of 6 each). `spawn_agent` starts them with the parent's history by default, and then they have the text already; with `fork_turns` `none` only `SubagentStart` brings it.
- Trust: every new or changed hook needs it; the TUI shows "Hooks need review", `codex exec` skips the hook silently. It is a hash per hook in the user's `config.toml`, so a moved binary needs it again. No `codex` command sets it; `codex app-server` (`hooks/list`, `config/batchWrite`) did, and so did a hash computed outside Codex.
- A project can have `.codex/hooks.json`; it runs once the user trusts the folder and the hook. Managed hooks (`[hooks]` in `requirements.toml`, or `/etc/codex/config.toml` and `/etc/codex/hooks.json`) need no trust; `managed_dir` only names their source.
- Not measured: the Codex app (not installed), managed hooks, Windows, automatic compaction.
""")

task("TASKF1.3", "VS Code Copilot", "TASK38", """
Facts seen on 2026-11-01 with VS Code 1.143:
- MCP servers are read from the user's `mcp.json` (`~/Library/Application Support/Code/User/mcp.json` on macOS) and a workspace's `.vscode/mcp.json`. The servers sit under `servers`, not `mcpServers`, and each has `"type": "stdio"`.
- `code --add-mcp '<json>'` adds a server to the user profile; there is no remove command.
- `code chat -m agent "<prompt>"` opens a chat with a prompt. VS Code has no headless mode, so the answers are read from the chat window or from the chat session files in the workspace storage.
- TASK38 did not measure Copilot.

Steps, with the fixture server in a throwaway workspace's `.vscode/mcp.json`:
1. Run the 10 prompts in Copilot Chat's agent mode, one new chat each. Does the model get `instructions` (always-on count), does a fitting prompt load the skill?
2. With the server in the user's `mcp.json` instead: which folder does it start in? If not the workspace folder, does `"cwd": "${workspaceFolder}"` fix it?
3. Does Copilot in VS Code have a session-start hook whose output reaches the model?
4. Does VS Code load a user-level instructions file in every workspace (for example an `.instructions.md` file in the user's prompts folder with `applyTo: "**"`)? Does a pointer there make the model load Quiver?
5. Where is the `mcp.json` on Windows and Linux, and for VS Code Insiders?

**Done when:** the results are in `docs/spike-hooks.md`.

**Notes:**
- Results in docs/spike-hooks.md (VS Code 1.143.1, Copilot Chat 0.71.0, `spike/b01/vscode-hooks-run.sh`). Delivery level: hook. A second VS Code with its own profile runs the prompts through a small driver extension, since `code chat` ignores `--user-data-dir`.
- Server only: always-on 4 of 10, fitting skills 6 of 6, other skills 0 of 4. With a `SessionStart` hook: 10 of 10, 6 of 6, 0 of 4, and no `get_skill` for `reply-style`; with the pointer also active, the model loaded it again in 6 of 10.
- VS Code has two harnesses: Local (the default) takes only `hookSpecificOutput.additionalContext`, the Copilot harness only a top-level `additionalContext`; plain stdout reaches neither. A command that prints both works in both, once. Both read `~/.copilot/hooks/*.json`; the Copilot CLI reads it too.
- No size limit up to 200,000 characters (Local). `SubagentStart` reaches sub-agents, `SessionStart` does not. A failing hook never stops a session. The command runs with `/bin/sh`; the default timeout is 30 seconds.
- `.claude/settings.json` files count only with `chat.useClaudeHooks` (off by default).
- From the user's `mcp.json` the server starts in the home folder; with `"cwd": "${workspaceFolder}"` it did not start. VS Code sends the workspace folder as an MCP root.
- A user instructions file with a pointer raised always-on to 8 of 10.
- Not measured: trust (workspace trust was off), managed hooks, compaction, Windows, Linux, Insiders.
""")

task("TASKF1.4", "Cursor, Antigravity and global rules files", "TASK38", """
Facts: in a workspace test on 2026-11-01, a one-line `AGENTS.md` telling the model to load Quiver's tools before the first reply raised always-on from 0 to 10 of 10 prompts in Codex and from 0 to 9 of 10 in Antigravity, and fitting skills from 0 and 1 to 6 of 6. That test put the file in the workspace only.

Steps:
1. Cursor: does it have a hooks file with an event at session start or before the first prompt whose output reaches the model? Cursor already gets `instructions`, so a hook only matters if it is more reliable.
2. Antigravity CLI (`agy` 1.4): does it have hooks, for example in its plugins (`agy plugin`)? Where does the Antigravity app, as opposed to the CLI, keep its MCP config?
3. Global rules files: does Codex load `~/.codex/AGENTS.md` (or `$CODEX_HOME/AGENTS.md`) in every project, and Antigravity `~/.gemini/GEMINI.md`? Repeat the workspace test with only the global file.
4. Does a project's own `AGENTS.md` add to the global one or replace it?

**Done when:** the results are in `docs/spike-hooks.md`.

**Notes:**
- Results in docs/spike-hooks.md (`cursor-hooks-run.sh`, `antigravity-hooks-run.sh`, `codex-rules-run.sh`). Delivery level: hook for both.
- Cursor CLI 2026.10.22: a `sessionStart` hook in `.cursor/hooks.json` printing `{"additional_context": ...}` gave always-on 10 of 10, fitting 6 of 6, other 0 of 4, no `get_skill` for `reply-style`. `beforeSubmitPrompt` did not run in the CLI, and `.claude/settings.json` was not read. User file: `~/.cursor/hooks.json`. The Cursor app was not measured.
- Antigravity `agy` 1.4.12: hooks live in `hooks.json` in a customization root, the global `~/.gemini/config/` or the workspace's `.agents/`, as `{"<name>": {"SessionStart": [{"type": "command", "command": "..."}]}}`. Output `{"injectSteps": [{"systemMessage": {"systemMessage": "..."}}]}` gave always-on 10 of 10 (fitting 0 of 6, as without the hook); plain stdout does nothing. The text is cut after about 48,900 bytes. Each sub-agent conversation runs `SessionStart` too. The global `hooks.json` worked. `.agents/mcp_config.json` in a workspace works, which TASK38 missed.
- Global rules files: a pointer in `$CODEX_HOME/AGENTS.md` gave 10 of 10 in Codex; in `~/.gemini/config/GEMINI.md` 7 of 10 in Antigravity, where 3 headless sessions stopped on a permission. `~/.gemini/config/AGENTS.md` and `~/.gemini/GEMINI.md` are read too. A project's `AGENTS.md` adds to the global file in both.
- The Antigravity app was not installed; by `agy`'s built-in docs it shares `~/.gemini/config/`.
""")

task("TASKF2", "`quiver hook session-start`", "TASKF1, TASKA1", """
A command that a session-start hook runs: it prints the always-on skills of the folder the session starts in, in the format the assistant expects. A second command does the same when a sub-agent starts, since the session-start text does not reach sub-agents (TASKF1.1). Installing the hooks is TASKF4.

1. **Command.** `quiver hook session-start <assistant>` in a new `internal/cli/hook.go`. `<assistant>` is a name from `quiver install` (`claude-code`, `codex`, `vscode`, `cursor`, `antigravity`) and selects the input and output format found in TASKF1: for `cursor` `{"additional_context": ...}`, for `antigravity` `{"injectSteps": [{"systemMessage": {"systemMessage": ...}}]}` with the input's `workspacePaths[0]` as the folder (TASKF1.4). For `vscode` the output carries the text twice, as `additionalContext` and as `hookSpecificOutput.additionalContext`, since VS Code's two harnesses each read one of them (TASKF1.3). It is listed in the help like `quiver mcp`, with a Long text that says assistants run it.
2. **Project.** Read the hook's JSON from stdin and take its `cwd`. Resolve the project as the MCP server does (`server.Resolve` in `internal/server/resolve.go`): `QUIVER_PROJECT` first, then that folder. A folder outside any project gets only the global skills.
3. **Skills.** The IDs come from `Registry.AlwaysOnServed` (`internal/registry/assign.go`): global ones first, then the project's, each in the order it was marked, each ID once. IDs no longer in the library are left out, as in the catalog. The text of each skill is what `get_skill` returns: move the code that builds it (`skillText` in `internal/server/skills.go`) where both can use it.
4. **Block.** Print one marked block, so the MCP server's pointer (TASKF3) can refer to it:
   ```
   <quiver-always-on>
   Always-on skills from Quiver. Follow each one for the whole conversation.

   ## caveman/caveman
   <the skill's SKILL.md body>

   ## reply-style
   <...>

   Extra files of these skills are loaded with the get_skill_file tool of the quiver MCP server.
   </quiver-always-on>
   ```
5. **Size.** Keep the whole block under 10,000 characters, Claude Code's limit (TASKF1.1). The Codex hook turns Codex's own limit off (TASKF4.3), so the same limit applies there; it also stays below Antigravity's cut at about 48,900 bytes (TASKF1.4). If the skills do not fit, print whole skills in order while they fit, and list the rest by ID with a line to load them with `get_skill` before the first reply.
6. **Sub-agents.** `quiver hook subagent-start <assistant>` prints the same block for the assistant's sub-agent event (`SubagentStart` in Claude Code, Codex and VS Code; Antigravity runs `SessionStart` for each sub-agent, and Cursor's `subagentStart` cannot add text). Its input has the same `cwd`. Claude Code takes a sub-agent's context only from `additionalContext`, and `hookEventName` must name the event: with a wrong name the hook fails (TASKF1.1).
7. **Never break a session.** No network. On any error, print what is known, write the error to `~/.quiver/logs/hook.log`, and exit 0. Log one line per run: time, assistant, project and skill IDs. The run must stay within the budget of TASKG6.3.

**Out of scope:** installing the hook (TASKF4), per-prompt hooks.

**Done when:** tests in `internal/cli` cover the order, an ID in both scopes, a removed skill, the block, the size limit, a folder outside any project, `QUIVER_PROJECT`, a broken `registry.json` (exit 0 and a log line), each output format and `subagent-start`.

**Notes:**
- `internal/cli/hook.go` and `server.AlwaysOnBlock` (`internal/server/alwayson.go`). `server.ResolveIn` resolves a given folder; `Resolve` now calls it with the working directory.
- Order: global skills first, then the project's, each ID once (`Registry.AlwaysOnInOrder`). Within each scope the order is the registry's, which is alphabetical: the registry keeps every list sorted, and sync merges lists as sorted sets, so the order of marking is not kept anywhere.
- The session's folder is the input's `cwd`, Cursor's `workspace_roots[0]` or Antigravity's `workspacePaths[0]`; without one, the hook's own working directory.
- With no always-on skill the hook prints nothing. `subagent-start` takes `claude-code`, `codex` and `vscode`; for the others, like a wrong assistant name, it prints nothing, says why on stderr, logs it and exits 0.
- The hook skips the root's auto-init and the update check, so it never writes the Quiver home or uses the network.
- Checked end to end: Claude Code with `quiver hook session-start claude-code` as its `SessionStart` hook followed the B01 always-on skill; the hook logged `ms=0`.
""")

task("TASKF3", "Always-on pointer becomes the fallback", "TASKF2", """
With the hook installed, the pointer in `instructions` (TASKA1) makes the model load each always-on skill a second time with `get_skill`: in 6 of 10 B01 prompts in Claude Code (TASKF1.1) and in VS Code (TASKF1.3), and in 0 of 10 in Codex (TASKF1.2). Without the hook it is still needed.

1. Change `pointer` in `internal/server/catalog.go` to:
   ```
   Always-on skills: <ids>. If the conversation already has a <quiver-always-on> block, these skills are loaded: follow them. Otherwise call the get_skill tool of this server for each before your first reply and follow it for the whole conversation.
   ```
2. The catalog and its budget (`maxInstructions`) stay as they are; check that the longer pointer still leaves room for the catalog.

**Done when:** the server tests check the new pointer, and a Claude Code session with the hook (TASKF4.2, or a hook set by hand) follows `reply-style` in 10 of 10 B01 prompts with no `get_skill` call for it. Record the counts in the task notes.
""")

task("TASKF4", "`quiver install`", "TASKF2, TASK38", """
Quiver now sets up more than an MCP server, so `quiver install` replaces `quiver mcp install`. Per assistant it sets up the MCP server and, where the assistant has one (TASKF1), the session-start hook, and it reports how always-on skills reach each assistant. `quiver mcp` stays: it is the command assistants run. `quiver mcp config` stays too.

The current code is in `internal/cli/mcp.go` (`mcpTargets`, `mcpInstall`, `mcpUninstall`, `installClaude`, `installCodex`, `installJSON`), `internal/assistant` (Claude Code and Codex through their commands, the JSON assistants through `jsonconfig.go`) and `internal/cli/doctor.go` (`checkMCP` and the checks after it).

**Done when:** every subtask is done.
""")

task("TASKF4.1", "Rename to `quiver install` and `quiver uninstall`", "TASK38", """
1. Move the commands from `internal/cli/mcp.go` to `internal/cli/install.go` as `quiver install [assistant...]` and `quiver uninstall [assistant...]`, with the same names, completion and output.
2. **Keep the old names.** Removing a command is a breaking change, which needs a new major version. `quiver mcp install` and `quiver mcp uninstall` stay as hidden, deprecated commands that run the new ones and print one line on stderr naming the new command; their deprecated `--for` (TASKE3) and `--print` keep working. They are removed in the next major version.
3. Update every caller and message: `guidedSetup` in `internal/cli/init.go`, the fix hints of `doctor` ("run: quiver mcp install ..."), the npm note in `mcpInstall`, and any other output a search for `mcp install` finds.
4. Update the docs: README (command table and the MCP server section), PROJECT.md (CLI and How It Works), CONTRIBUTING.md if it names the command, and the First Milestone of `tasks/README.md` in `scripts/gen_tasks.py`. The website is updated by the maintainer.
5. The GitHub release notes of v1.4.0 say that the command changed and that the old name still works.

**Done when:** the tests and completion use the new commands, the old ones still work and print the deprecation line, and a search for `mcp install` finds nothing outside the old commands, their tests and the files of finished tasks.
""")

task("TASKF4.2", "Claude Code hook", "TASKF1.1, TASKF2, TASKF4.1", """
1. **Where.** The user's Claude Code settings file found in TASKF1.1 (`~/.claude/settings.json`, or `$CLAUDE_CONFIG_DIR/settings.json`).
2. **What.** A `SessionStart` entry with the matcher `startup|clear|compact`, whose command is `<quiver> hook session-start claude-code`, and a `SubagentStart` entry with no matcher, whose command is `<quiver> hook subagent-start claude-code`. `<quiver>` is the path `command()` in `internal/cli/mcp.go` returns (Claude Code has the shell's `PATH`, so the npm launcher works), quoted with `assistant.ShellQuote`.
3. **How.** Generalize `internal/assistant/jsonconfig.go`: today `change` edits the `mcpServers` object; the hooks live in `hooks.SessionStart` and `hooks.SubagentStart`, arrays of groups that each hold a `hooks` array. Keep every other key, group and hook, the key order, the indentation and the file mode; save `<file>.quiver-backup` first; write atomically (`fsutil.WriteFileAtomic`); never write a file that is not valid JSON.
4. **Recognize Quiver's hooks** by their command: one whose arguments end with `hook session-start claude-code` or `hook subagent-start claude-code` and whose program is named `quiver` (or `quiver.exe`). `install` points it to the current binary (reported as updated), adds it if missing, and leaves it alone if it is current; `uninstall` removes only it, and the group if it is then empty.
5. Set each hook's `timeout` to 10 seconds. The default is 600, and a slow hook delays the first reply by its whole run time (TASKF1.1).

**Done when:** tests in `internal/assistant` cover a missing file, a file with other settings and hooks, an update after a move, removal, and a file that is not valid JSON; tests in `internal/cli` cover install and uninstall; and a new Claude Code session and a sub-agent it starts both follow an always-on fixture skill from the hooks.
""")

task("TASKF4.3", "Codex hook", "TASKF1.2, TASKF2, TASKF4.1", """
1. **Where.** Codex's `hooks.json` (`~/.codex/hooks.json`, or `$CODEX_HOME/hooks.json`). The Codex CLI and app share it.
2. **What.** A `SessionStart` entry with the matcher `startup|clear|compact`, whose command is `<quiver> hook session-start codex`, and a `SubagentStart` entry with no matcher, whose command is `<quiver> hook subagent-start codex`. `<quiver>` is the binary itself (`appCommand()`), since the Codex app has no shell `PATH`, quoted with `assistant.ShellQuote`. A sub-agent that starts with its parent's history, Codex's default, then has the text twice; one that starts without it gets the text only from `SubagentStart` (TASKF1.2).
3. **Fields.** Set each hook's `timeout` to 10 seconds; `additionalContextLimit` to 0, so Codex does not cut the block (TASKF2 keeps it within its limit); and `statusMessage` to `Quiver always-on skills`, the name Codex's review list shows.
4. **How.** The JSON editing of TASKF4.2, with the same rules for other hooks, backups and recognition.
5. **Trust.** Codex runs a new or changed hook only once it is trusted, and `codex exec` skips it without a message (TASKF1.2). After writing the file, trust Quiver's own hooks, and only those, through `codex app-server`: `hooks/list` for their keys and current hashes, then `config/batchWrite` of `hooks.state.<key>.trusted_hash`, the two requests Codex's own review dialog sends. `install` says that it trusted them. Every change to a hook's fields or command changes its hash, so trust again whenever `install` updates the hook. If the app server fails, print the step for the user: start `codex`, open "Review hooks" and trust the Quiver hooks. `doctor` reads the status through `hooks/list` and reports a Quiver hook that is new or changed, for example after the binary moved.
6. **Tests.** The fake codex (`assistanttest.InstallCodex`) covers the `codex` calls in `internal/cli`, the app server included; a test like `TestRealCodex` in `internal/assistant/codex_test.go` checks the real codex in a throwaway `CODEX_HOME` when it is installed, including that `codex exec` runs the hooks right after `install`.

**Done when:** the tests pass, and right after `install`, with no review dialog, a new Codex session and a sub-agent it starts with `fork_turns` `none` both follow an always-on fixture skill with no `get_skill` call for it.
""")

task("TASKF4.4", "Delivery report, doctor and status", "TASKF4.2", """
Users must see which assistants load always-on skills by themselves, so they do not take those skills for optional ones where they are not.

1. **Model.** Replace `JSONAssistant.OnRequest` with a delivery level on every assistant (the levels of TASKF1: hook, instructions, on request), set from `docs/spike-hooks.md`.
2. **Install report.** After the per-assistant lines, print one line per assistant with its level. For every assistant without a hook, say that loading skills there is up to the model, and when always-on skills are assigned, name them:
   ```
   Claude Desktop: no session-start hook. Quiver's skills are loaded when the model decides to.
     These always-on skills do not load by themselves there: security-review, reply-style.
   ```
   The names come from the global always-on skills and those of the project in the current folder.
3. **doctor.** For each hook: present, runs this binary, file readable, trusted (Codex). For each assistant without a hook while always-on skills are assigned: a warning that names them.
4. **status.** A short section that lists each installed assistant and its level.

**Done when:** tests cover every level, the warning with and without always-on skills, and the doctor checks.
""")

task("TASKF4.5", "Cursor hook", "TASKF1.4, TASKF2, TASKF4.1", """
1. **Where.** The user's `~/.cursor/hooks.json`, which the Cursor app and the CLI share; `{"version": 1, "hooks": {...}}`.
2. **What.** A `sessionStart` entry `{"command": "<quiver> hook session-start cursor", "timeout": 10}`, `<quiver>` being the binary itself (`appCommand()`) quoted with `assistant.ShellQuote`, since the app has no shell `PATH`. No sub-agent entry: Cursor's `subagentStart` cannot add text (TASKF1.4).
3. **How.** The JSON editing of TASKF4.2: keep every other hook and key, back up, write atomically, recognize Quiver's entry by its command, and remove only it on `uninstall`.
4. The MCP server's `instructions` stay the fallback (TASKF3); with the hook, Cursor's level is hook.

**Done when:** tests cover a new file, a file with other hooks, an update after a move and removal; a Cursor CLI session follows an always-on fixture skill from the hook with no `get_skill` call for it; and a session in the Cursor app does too, checked by hand.
""")

task("TASKF4.6", "Antigravity hook", "TASKF1.4, TASKF2, TASKF4.1", """
1. **Where.** `~/.gemini/config/hooks.json`, the global customization root that every Antigravity product reads (TASKF1.4).
2. **What.** A hook named `quiver`: `{"quiver": {"SessionStart": [{"type": "command", "command": "<quiver> hook session-start antigravity", "timeout": 10}]}}`, `<quiver>` being the binary itself (`appCommand()`) quoted with `assistant.ShellQuote`. `SessionStart` also runs for every sub-agent, so no second entry is needed.
3. **How.** Quiver owns the `quiver` key and keeps every other named hook and key; back up, write atomically, and remove only that key on `uninstall`.
4. **Doctor.** Reports the hook as missing, outdated or pointing to another binary.

**Done when:** tests cover a new file, a file with other hooks, an update and removal, and an `agy` session follows an always-on fixture skill from the hook with no `get_skill` call for it.
""")

task("TASKF5", "VS Code Copilot", "TASKF1.3, TASKF2, TASKF4.1", """
Add VS Code to `quiver install` with the delivery level found in TASKF1.3.

1. **MCP.** Add `quiver` to `servers` in VS Code's user `mcp.json`: `~/Library/Application Support/Code/User/mcp.json` on macOS, `%APPDATA%\\Code\\User\\mcp.json` on Windows, `~/.config/Code/User/mcp.json` on Linux. The entry has `"type": "stdio"` and no `cwd`: with `"cwd": "${workspaceFolder}"` the server did not start (TASKF1.3). Extend `JSONAssistant` with the name of the servers key (today always `mcpServers`) and the extra fields of an entry. Do not use `code --add-mcp`; it cannot remove an entry.
2. **Project.** VS Code starts the server in the home folder but sends the workspace folder as an MCP root when the server is initialized (TASKF1.3). When `QUIVER_PROJECT` is unset and the working directory is in no project, `server.Resolve` (`internal/server/resolve.go`) asks the client for its roots after initialization and serves the project of the first root that is in one. Clients that start the server in the project behave as before. Update the comment there that says roots are not used.
3. **Hook.** Quiver owns the file `~/.copilot/hooks/quiver.json`, which VS Code's Local and Copilot harnesses both read (and the Copilot CLI too): a `SessionStart` and a `SubagentStart` entry in the flat form, `{"type": "command", "command": "<quiver> hook session-start vscode", "timeoutSec": 10}` and the same with `subagent-start`, `<quiver>` being the binary itself (`appCommand()`), quoted with `assistant.ShellQuote` since the command runs with `/bin/sh`. `uninstall` removes the file. It is written whole, so no merging is needed.
4. **Installed** means its user folder exists. VS Code Insiders keeps its files under `Code - Insiders` in place of `Code` (VS Code's documented user folder; not measured).
5. The MCP command is the binary itself (`appCommand()`), as for the other desktop apps.

**Done when:** install, update, uninstall and doctor are tested for VS Code; a Copilot agent session in VS Code follows an always-on fixture skill from the hook, in the Local and in the Copilot harness; and a skill assigned to the open workspace's project is served.
""")

task("TASKF7", "Release v1.4.0", "TASKF1, TASKF2, TASKF3, TASKF4, TASKF5", """
1. **Old names still work.** `quiver mcp install` and `quiver mcp uninstall`, with `--for` and `--print`, work as in v1.3.0 and name the new commands (TASKF4.1).
2. **Upgrade check.** As in TASKE3, from a home made by the v1.3.0 binary: `quiver install` gives every installed assistant its hook, the same skills are served, and `doctor` is clean.
3. README and help text cover every change of this phase; PROJECT.md's roadmap lists them under v1.4.
4. The GitHub release notes say that `quiver install` replaces `quiver mcp install`, and which assistants get always-on skills from a hook.
5. Tag `v1.4.0` only after the maintainer confirms. The release workflow publishes to GitHub and npm and installs on clean machines.
6. **Website.** The status marks follow the release (see `web/README.md`). Website changes are committed only after the maintainer confirms.

**Done when:** the release workflow is green.
""")

# ---------------------------------------------------------------- Phase 8
phase("Phase 8: Test phase")

task("TASKG1", "Coverage baseline", "TASKF5", """
Package coverage today undercounts code that other packages' tests run: `internal/library` shows 39% and `internal/git` 41% because most of their code runs from `internal/cli`, and `internal/homesync` has no tests of its own. The e2e suite runs the built binary, whose coverage is not measured at all.

1. Measure all code together: unit tests with `-coverpkg=./...`, and the e2e binary built with `go build -cover` and run with `GOCOVERDIR`, merged with `go tool covdata`. Add a CI step that produces the merged report and keeps it as an artifact.
2. List the user-facing paths no test reaches, such as error branches of `install`, sync conflicts and update failures, in this task's notes.
3. Add tests for them, starting with `internal/homesync`.
4. Record the total in decisions.md as the baseline. CI fails when a change lowers it by more than 0.5 points.

**Done when:** the merged report is in CI and CI enforces the baseline.
""")

task("TASKG2", "End-to-end user journeys", "TASKF5, TASK29", """
The e2e suite (`e2e/`) builds the binary once (`buildAndRun` in `e2e/harness_test.go`) and covers six flows: new, add and remove; `create_skill`; import; source update; move, copy and split; `doctor --fix`. Its helpers run quiver and git in a throwaway home (`newEnv`, `run`, `quiver`, `git`), open an MCP session (`mcp`, `call`) and write files (`write`, `skillMD`).

The journeys below run whole user stories with the real binary, the fake assistants of `internal/assistant/assistanttest` on `PATH`, and local bare git repositories as remotes, without network. Each journey is one test in `e2e/journeys_test.go`; each step checks its output, not only its exit code.

**Done when:** every subtask is done and the journeys run on Linux, macOS and Windows.
""")

task("TASKG2.1", "Journey: setup and delivery", "TASKF5, TASK29", """
1. A fresh home: `quiver init` with its questions answered on stdin.
2. `quiver source add` of a local bare repository; `quiver add <id> --global --always`, and in a project folder `quiver add <id> --always`.
3. `quiver install` with fake Claude Code and Codex: both report the hook level; a JSON assistant without a hook reports its level and names the always-on skills.
4. `quiver hook session-start claude-code` with the project folder on stdin prints both skills, global first; with another folder only the global one.
5. An MCP session in the project: `instructions` has the fallback pointer and the catalog; `list_skills` and `get_skill` agree with the hook.
6. `quiver uninstall`: no Quiver entry or hook is left, and every other entry is.

**Done when:** the journey passes on all three systems.
""")

task("TASKG2.2", "Journey: library lifecycle", "TASKF5, TASK29", """
1. `source add`, then a new commit in the remote that changes an always-on skill and a script.
2. `quiver update`: the preview marks the always-on skill and flags the script; answering no changes nothing, answering yes applies.
3. `pin` the skill, change the remote again, `update` leaves it; `unpin`, `update` takes the change.
4. `fork` the skill, `edit` the copy; the copy replaces the original wherever it was assigned or always-on.
5. `source remove` is refused while a skill is assigned, then works with `--force`.

After each step, `quiver status`, the hook's output and the MCP server's `get_skill` agree.

**Done when:** the journey passes on all three systems.
""")

task("TASKG2.3", "Journey: two machines", "TASKF5, TASK29", """
Two homes (machine A and B) and one bare remote:
1. `quiver sync init <remote>` on both.
2. A adds a local skill and marks it always-on; B, after `quiver sync`, has the skill, and B's hook prints it.
3. A and B change assignments of the same project apart from each other; after both sync, both changes are kept (the three-way merge of `internal/jsonmerge`).
4. A adds a source; B's next `sync` clones it.
5. A removes a skill B still has assigned: B's `doctor` reports it.

**Done when:** the journey passes on all three systems.
""")

task("TASKG2.4", "Journey: failures", "TASKF5, TASK29", """
Each case ends with a clear message, no half-written file, and a `doctor` report that names the fix:
- an unreachable remote: `source add`, `update` and `sync` time out (the git timeouts) and say so;
- an assistant config that is not valid JSON, and a Codex `config.toml` the codex command rejects: `install` skips that assistant, sets up the others, and exits with an error;
- a `registry.json` that is not valid JSON, and one with a newer version: commands refuse it, and the hook prints nothing and exits 0;
- a skill folder deleted by hand;
- a read-only Quiver home;
- a quiver process killed in the middle of a write (the file is the old or the new one, never a mix);
- the lock held by another process: the command waits, then fails with a message that names the lock file.

**Done when:** every case passes on all three systems.
""")

task("TASKG3", "Windows", "TASKF5", """
CI runs the tests on Windows (without `-race`, which needs cgo). Six tests skip there because the fake `claude` and `codex` in `internal/assistant/assistanttest/fake.go` are shell scripts. The hook commands of TASKF4 add new Windows cases: quoting, paths and the shell each assistant uses. The Release workflow installs a release with `install.ps1` and npm on Windows, but only when a tag is pushed.

**Done when:** every subtask is done, and no test skips on Windows for lack of a fake.
""")

task("TASKG3.1", "Fake assistants in Go", "TASKF5", """
1. Write the fakes as one Go program (for example `internal/assistant/assistanttest/fakecmd`) that acts as `claude` or `codex` depending on the name it is run as, and keeps its state and call log in files next to it, as the scripts do now.
2. `Install` and `InstallCodex` build it once per test binary (in a temp folder, named `claude.exe` and `codex.exe` on Windows) and put it first on `PATH`.
3. Keep the `Fake` API (`SetUser`, `SetLocal`, `SetCodex`, `User`, `Calls`), so the tests do not change.
4. Remove the Windows skips that were only there for the fakes.

**Done when:** the six tests run and pass on the Windows runner.
""")

task("TASKG3.2", "Hooks and paths on Windows", "TASKG3.1", """
1. Quote hook commands for the shell each assistant runs them in on Windows (TASKF1). `assistant.ShellQuote` quotes for a POSIX shell only; add the Windows form.
2. Paths with spaces and non-ASCII user names, in the Quiver home, the binary's path and the project path.
3. The `%APPDATA%` locations of every assistant config.
4. CRLF line endings in skill files: parsing still works.

**Done when:** tests cover each point and pass on the Windows runner.
""")

task("TASKG3.3", "Installed binary runs the hook", "TASKG3.1", """
The release checks install the binary but do not run it with an assistant.

1. In the Release workflow, after each install (`install.sh`, `install.ps1`, npm), run `quiver install` with the fake assistants and run the installed hook command as the assistant would, from a project folder.
2. Add the same check to the `release-dry-run` job of CI, so it runs on every push and not only on a tag.

**Done when:** the checks pass on Linux, macOS and Windows.
""")

task("TASKG4", "Versioned data formats and migration", "TASKF5", """
Facts: `registry.json` is at version 2 and `registry.Load` upgrades version 1 (`internal/registry/registry.go`); the state file is at version 1 and a newer one is refused; `config.json` is at version 1; `.quiver.json` holds only `id`; the hook's block (TASKF2) has no version.

1. Every file Quiver writes in the Quiver home has a version, and a file from a newer Quiver is refused with a message that names the version needed, as the state file does.
2. `.quiver.json` is committed in users' repositories, so it gets a version only when its format changes; a file without one is version 1.
3. **Golden homes.** Keep a Quiver home as written by v1.0.0, v1.1.0, v1.2.0 and every later release under `testdata/formats/<version>/`, made by running that release's binary through the same script of commands (init, source add, add, add --always, a project). A test copies each, runs the current code on it, and checks the upgraded files and a clean `doctor`.
4. Describe each format version in data-model.md.
5. Add "add the release's golden home" to docs/releasing.md.

**Done when:** every golden home upgrades in the tests, and every file from a newer version is refused clearly.
""")

task("TASKG5", "Assistant behaviour benchmark", "TASKF5", """
Whether assistants load fitting skills and follow always-on skills was measured by hand, once per assistant (docs/spike-b01-results.md, docs/spike-mcp-clients.md, docs/spike-hooks.md). A new model or assistant version can change it without notice. The spike runners in `spike/b01` (`run-real.sh` and `analyze.py` for Claude Code, `cursor-run.sh`, `codex-run.sh`, `antigravity-run.sh`, `gemini-run.sh` and their analyzers) become one benchmark, run before every release and after an assistant update.

Each run calls paid models, so it runs by hand or on a schedule, not on every push.

**Done when:** every subtask is done, and a baseline run is committed.
""")

task("TASKG5.1", "Benchmark runner", "TASKF5", """
1. **Decide first:** Go (one toolchain, reuses Quiver's code) or Python (what the analyzers use now).
2. One tool in `bench/` that prepares the fixture home (as `spike/b01/gui-setup.sh` does), runs a prompt set through an adapter (TASKG5.2), and writes one JSON result per run: assistant and version, model, date, Quiver version, and per prompt the Quiver tool calls in order, the skill markers, the always-on marker and any error.
3. It replaces the runners and analyzers in `spike/b01`; `docs/spike-*.md` point to it.

**Done when:** the runner reproduces the Codex numbers in docs/spike-mcp-clients.md within one prompt.
""")

task("TASKG5.2", "Assistant adapters", "TASKG5.1", """
One adapter per assistant with a headless mode: Claude Code (`claude -p`), Codex (`codex exec --json`), Cursor (`cursor-agent -p`), Antigravity (`agy -p`), and Gemini CLI once its login works (its Google sign-in was refused on 2026-11-01).

Each adapter sets Quiver up only for the run and never changes the user's own config: a throwaway workspace with project config (Claude Code, Cursor, Antigravity's `.agents/` folder), per-run flags (Codex `-c mcp_servers...`), or a throwaway home where an assistant has no project config. Tool permissions are given per run, never with a flag that skips every permission.

**Done when:** each adapter runs the B01 prompt set and its result matches the last hand measurement.
""")

task("TASKG5.3", "Benchmark cases", "TASKG5.2", """
Prompt sets, each with the expected result:
- fitting and non-fitting prompts (B01);
- always-on through the hook, and through the MCP fallback with the hook removed;
- a project skill inside and outside its project;
- a catalog longer than the `instructions` limit (`FILLER` in `run-real.sh`);
- after compaction, where the assistant supports it;
- a sub-agent task.

**Done when:** every case runs on every adapter that supports it.
""")

task("TASKG5.4", "Baselines and reports", "TASKG5.3", """
1. A report prints, per assistant and case, the rates (fitting skill loaded, always-on followed, wrong skill loaded) and compares them with the last committed baseline. It fails when a rate drops by more than a threshold set per case.
2. Results are kept as `docs/bench/<date>-<assistant>.json`, and a short table in `docs/bench/README.md`.
3. docs/releasing.md requires a run before a release.

**Done when:** a baseline for every supported assistant is committed.
""")

task("TASKG6", "Robustness", "TASKF5", """
Hostile input, processes running at the same time, and large libraries.

**Done when:** every subtask is done.
""")

task("TASKG6.1", "Fuzzing", "TASKF5", """
CI fuzzes `skill.Parse` for 30 seconds (`fuzz` job in `.github/workflows/ci.yml`). Add fuzz targets for:
- the JSON config editor (`parseObject` and `change` in `internal/assistant/jsonconfig.go`), and the hooks editing of TASKF4.2: other content must come back unchanged;
- reading `registry.json`, the state file and `config.json`;
- the three-way merge in `internal/jsonmerge`;
- the hook's stdin (TASKF2).

Run each in the CI fuzz job.

**Done when:** every target runs in CI, and each crash found is fixed with a regression test.
""")

task("TASKG6.2", "Parallel processes", "TASKF5", """
The MCP server, CLI commands and the hook run at the same time: two sessions start while `quiver update` writes the registry. Writers take the lock (`fsutil.WithLock`, a flock on `~/.quiver/.lock`) and replace files atomically (`fsutil.WriteFileAtomic`); readers such as the hook take no lock.

1. Tests start real quiver processes in parallel: a writer loop (`add`, `remove`, `update`) and reader loops (`hook session-start`, `status`, an MCP session calling `list_skills`).
2. Check that no reader ever fails to parse a file or sees a mix of two versions, and that no write is lost.

**Done when:** the tests pass with `-race` on Linux and macOS and without it on Windows.
""")

task("TASKG6.3", "Performance budgets", "TASKF5", """
1. A generator in `testdata` (or a test helper) builds a library of 1000 skills, 50 of them always-on in various projects, and 200 projects.
2. Measure on it: the hook's output, MCP server start up to `instructions`, `quiver status` and `quiver list`.
3. Set budgets from the measurements, with room for slower machines. A starting point: the hook under 100 ms, the MCP server start under 300 ms. The hook's budget matters most: it delays every session start.
4. Write the budgets in decisions.md; a test fails when one is exceeded by more than 50%, so CI noise does not fail it.

**Done when:** the budgets are recorded and tested.
""")

task("TASKG7", "Security review", "TASKF5", """
A review of everything that can reach a session or run on the machine, written as a threat model in `docs/security.md`: for each risk, what can go wrong, the mitigation, and the test that shows it.
- Remote skill text that the hook puts into every session. Mitigations today: the update preview (TASK36) marks always-on skills and flags scripts, and pins (TASK35) keep a reviewed commit.
- The hook command that runs at every session start: which binary, from which path, and what happens if that path becomes writable by others.
- Scripts inside skills.
- Path traversal and symlinks in `get_skill_file`, in skill folders and in source repositories.
- File modes of the configs Quiver writes, of their backups and of the logs.
- Secrets in logs and in the hook's log line.
- Hooks written into other tools' files.

**Done when:** `docs/security.md` lists each risk with its mitigation and test, and none is open.
""")

task("TASKG8", "Test phase review", "TASKG1, TASKG2, TASKG3, TASKG4, TASKG5, TASKG6, TASKG7", """
The test phase ends when:
- every task of this phase is done;
- CI passes on Linux, macOS and Windows, with no test skipped for lack of a fake;
- coverage is at or above the baseline of TASKG1;
- a benchmark run (TASKG5) is committed for every supported assistant and meets its thresholds;
- `docs/security.md` has no open item.

Record the result in this task's notes. The enterprise phase starts after it.

**Done when:** every point above holds.
""")

# ---------------------------------------------------------------- Phase 9
phase("Phase 9: Enterprise")

task("TASKH1", "Enterprise", "TASKG8", """
Central skill management for organizations, as described in PROJECT.md (Direction). To be planned in detail after the test phase; this task only holds its place.
""")

# ---------------------------------------------------------------- renumber
ORDER = [
    ("Phase 0 — Foundation", ["TASK01", "TASK02", "TASK03", "TASK04", "TASK05", "TASK06", "TASK07"]),
    ("Phase 1 — Remote Sources", ["TASK08", "TASK23", "TASK24", "TASK25", "TASK26", "TASK27", "TASK10", "TASK28"]),
    ("Phase 2 — MCP Server (first usable version)", ["TASK15", "TASK16", "TASK17", "TASKA1", "TASK22", "TASK18"]),
    ("Phase 3 — Local Skills and Management", ["TASK11", "TASK09", "TASK19", "TASK20", "TASK21", "TASK12", "TASK13", "TASK14"]),
    ("Phase 4 — v1 Release", ["TASK29", "TASK30", "TASK31"]),
    ("Phase 4b — v1.1", ["TASKC1"]),
    ("Phase 4c — v1.2", ["TASKD1", "TASKD2", "TASKD3", "TASKD4", "TASKD5", "TASKD6", "TASKD7", "TASKD8"]),
    ("Phase 5 — v1.3", ["TASKE1", "TASKE2", "TASK32", "TASK33", "TASK34", "TASK35", "TASK36", "TASK37", "TASK38", "TASKE3"]),
    ("Phase 6: Session-start delivery (v1.4)", ["TASKF1", "TASKF1.1", "TASKF1.2", "TASKF1.3", "TASKF1.4", "TASKF2", "TASKF3", "TASKF4", "TASKF4.1", "TASKF4.2", "TASKF4.3", "TASKF4.4", "TASKF4.5", "TASKF4.6", "TASKF5", "TASKF7"]),
    ("Phase 7: TUI and slash commands (v1.5)", ["TASK39", "TASK40", "TASKB1", "TASKE4"]),
    ("Phase 8: Test phase", ["TASKG1", "TASKG2", "TASKG2.1", "TASKG2.2", "TASKG2.3", "TASKG2.4", "TASKG3", "TASKG3.1", "TASKG3.2", "TASKG3.3", "TASKG4", "TASKG5", "TASKG5.1", "TASKG5.2", "TASKG5.3", "TASKG5.4", "TASKG6", "TASKG6.1", "TASKG6.2", "TASKG6.3", "TASKG7", "TASKG8"]),
    ("Phase 9: Enterprise", ["TASKH1"]),
]
ALL = {t[0]: t for _, ts in PHASES for t in ts}
flat = [tid for _, ids in ORDER for tid in ids]
assert sorted(flat) == sorted(ALL), set(ALL) ^ set(flat)
# A subtask (TASKF1.2) takes its parent's final number (TASK54.2).
REMAP = {}
for old in flat:
    parent, dot, sub = old.partition(".")
    REMAP[old] = REMAP[parent] + dot + sub if dot else f"TASK{sum('.' not in t for t in REMAP) + 1:02d}"
remap = lambda s: re.sub(r"TASK(?:\d\d|[A-H]\d)(?:\.\d+)?", lambda m: REMAP.get(m.group(0)) or REMAP[m.group(0).split(".")[0]] + m.group(0)[len(m.group(0).split(".")[0]):], s)
MARK = {"x": "✅", "~": "🚧"}
STATUS = {"TASK01": "x", "TASK02": "x", "TASK03": "x", "TASK04": "x", "TASK05": "x", "TASK06": "x", "TASK07": "x", "TASK08": "x", "TASK09": "x", "TASK10": "x", "TASK11": "x", "TASK12": "x", "TASK13": "x", "TASK14": "x", "TASK15": "x", "TASK16": "x", "TASK17": "x", "TASK18": "x", "TASK19": "x", "TASK20": "x", "TASK21": "x", "TASK22": "x", "TASK23": "x", "TASK24": "x", "TASK25": "x", "TASK26": "x", "TASK27": "x", "TASK28": "x", "TASK29": "x", "TASK30": "x", "TASK31": "x", "TASK32": "x", "TASK33": "x", "TASK34": "x", "TASK35": "x", "TASK36": "x", "TASK37": "x", "TASK38": "x", "TASK39": "x", "TASK40": "x", "TASK41": "x", "TASK42": "x", "TASK43": "x", "TASK44": "x", "TASK45": "x", "TASK46": "x", "TASK47": "x", "TASK48": "x", "TASK49": "x", "TASK50": "x", "TASK51": "x", "TASK52.1": "x", "TASK52.2": "x", "TASK52.3": "x", "TASK52.4": "x", "TASK53": "x"}  # final IDs; x done, ~ in progress
EXTRA_DEPS = {"TASK29": ", TASK14"}  # e2e also waits for the last management task
PHASES = [
    (ptitle, [(REMAP[o], remap(ALL[o][1]), remap(ALL[o][2] + EXTRA_DEPS.get(o, "")), remap(ALL[o][3])) for o in ids])
    for ptitle, ids in ORDER
]

# ---------------------------------------------------------------- write
def slug(s):
    s = s.replace("`", "")
    return re.sub(r"[^a-z0-9]+", "-", s.lower()).strip("-")

FNAME = {tid: f"{tid}-{slug(title)}.md" for _, tasks in PHASES for tid, title, _, _ in tasks}
TITLE = {tid: title for _, tasks in PHASES for tid, title, _, _ in tasks}
CHILDREN = {}
for tid in FNAME:
    if "." in tid:
        CHILDREN.setdefault(tid.split(".")[0], []).append(tid)

def status_of(tid):
    """A parent is done when all its subtasks are, in progress when one is."""
    if tid not in CHILDREN:
        return STATUS.get(tid)
    sts = [status_of(c) for c in CHILDREN[tid]]
    if all(s == "x" for s in sts):
        return "x"
    return "~" if any(sts) else None

index = []
for ptitle, tasks in PHASES:
    rows = []
    for tid, title, dep, body in tasks:
        fname = FNAME[tid]
        st = status_of(tid)
        status = {"x": "✅ Done", "~": "🚧 In progress"}.get(st, "Todo")
        parent = tid.split(".")[0]
        part = f"**Part of:** [{parent}]({FNAME[parent]}): {TITLE[parent]}\n\n" if "." in tid else ""
        # Phases up to 5 keep their em dash; later ones use a colon.
        sep = " — " if " — " in ptitle else ": "
        subs = "".join(f"- [{c}]({FNAME[c]}): {TITLE[c]}\n" for c in CHILDREN.get(tid, []))
        subs = f"\n**Subtasks:**\n{subs}" if subs else ""
        (OUT / fname).write_text(f"# {tid}{sep}{title}\n\n**Phase:** {ptitle}\n\n{part}**Status:** {status}\n\n**Depends on:** {dep}\n\n{body}\n{subs}")
        link = f"↳ [{tid}]({fname})" if "." in tid else f"[{tid}]({fname})"
        rows.append(f"| {MARK.get(st, '')} | {link} | {title.replace('|', chr(92) + '|')} | {dep} |")
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
- **Subtasks:** a large task is split into subtasks (TASK54.1, TASK54.2, …), one file each, listed under their parent in the index. Each subtask is finished and committed on its own; the parent is done when all its subtasks are.
- **Done when:** acceptance criteria. A task is not done until every item passes and has a test.
- Every command needs unit tests. Every user-facing flow also needs an end-to-end test (see TASK30).
- MCP tasks are tested through the in-memory harness from TASK16.

## First Milestone

Phases 0–2 are the first usable version. After TASK21 you can run:

```bash
quiver source add JuliusBrussee/caveman --name caveman
quiver add caveman/caveman --global --always
quiver mcp install
```

and every new assistant session follows caveman from the first message. Use Quiver this way every day before you start Phase 3.

## Index

""" + "\n\n".join(index) + """

## Later (not yet planned in detail)

- Graphical app with the same features as the TUI.
"""
(OUT / "README.md").write_text(readme)
n = sum(len(t) for _, t in PHASES)
print(n - sum(map(len, CHILDREN.values())), "tasks,", sum(map(len, CHILDREN.values())), "subtasks")
