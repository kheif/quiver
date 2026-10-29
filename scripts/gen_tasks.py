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

**Not done (decided 2026-10-28):** prefix-free commands (TASKB1) stay in v2. It is not measured yet whether the assistant reliably follows a pointer file to `get_skill`, so the MCP prompts stay as they are for now.

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
""")

task("TASKD4", "`quiver mcp install` for Cursor and Claude Desktop", "TASKC1", """
Today `mcp install` registers Quiver only with Claude Code (`internal/assistant/claude.go`). Pulls the first part of the v2 task "`mcp install` for more assistants" into v1.2.

1. **Spike first, per assistant.** The catalog and always-on skills reach the assistant through the MCP `instructions`. Check that Cursor and Claude Desktop pass `instructions` to the model, as in docs/spike-b01-results.md: does a fitting prompt load the skill, does an always-on skill apply from the first reply? Record the results in `docs/`. If an assistant ignores `instructions`, decide with the user before adding it.
2. **Shared JSON installer.** Most assistants keep MCP servers in a JSON file under `mcpServers`. One function adds, updates and removes the `quiver` entry, keeps every other key and the file's indentation, backs up the original, and writes atomically. A broken JSON file is an error, never overwritten.
   - Cursor: `~/.cursor/mcp.json`.
   - Claude Desktop: `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS, `%APPDATA%\\Claude\\claude_desktop_config.json` on Windows.
3. **CLI.** `quiver mcp install [--for claude-code|cursor|claude-desktop]`. Without `--for`, install for every assistant that is installed, and list what was done. `mcp uninstall` the same. `quiver mcp config` prints the JSON snippet for any other assistant.
4. The command is the absolute path of the binary (backlog B09), or the npm shim as today. Claude Desktop does not run in a project folder, so it gets only global skills: say so after installing.
5. `doctor` checks each installed assistant's entry.

**Done when:** tests cover install, update, uninstall, other keys kept and a broken file for each JSON assistant; the spike results are written down; the README lists the assistants.
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
""")

task("TASKD6", "Notice of a new Quiver version", "TASKC1", """
1. At most once a day, a command asks GitHub for the latest release (`api.github.com/repos/kheif/quiver/releases/latest`) with a short timeout, in the background, and stores the answer in `~/.quiver/state/version-check.json`.
2. When a newer version is known, commands end with one line on stderr: `Quiver 1.3.0 is out (you have 1.2.0): npm install -g @kheif/quiver` (or the curl line, or `go install`, depending on how it was installed; `QUIVER_NPM` tells npm).
3. Never in `quiver mcp` (stdout is the protocol), with `--json`, outside a terminal, for `dev` builds, or with `QUIVER_NO_UPDATE_CHECK=1`. A failed check is silent.
4. `quiver doctor` shows the latest version as an info line.

**Done when:** tests with a fake release endpoint cover newer, same, failed and disabled, and that the check runs at most once a day.
""")

task("TASKD7", "`install.ps1` for Windows", "TASKC1", """
1. `install.ps1`, used as `irm https://raw.githubusercontent.com/kheif/quiver/main/install.ps1 | iex`: the same steps as `install.sh` for the Windows zip (`quiver_windows_<arch>.zip`), with `QUIVER_VERSION`, `QUIVER_INSTALL_DIR` (default `%LOCALAPPDATA%\\Programs\\quiver`) and `QUIVER_DOWNLOAD_URL`.
2. Check the SHA-256 against `checksums.txt`, and add the folder to the user's `PATH` (user scope, no admin).
3. CI's `release-dry-run` and the release workflow's install checks run it on Windows.
4. README: the Windows install line; `install.sh`'s Windows error mentions it.

**Done when:** a clean Windows runner installs with the script and `quiver --version` works in a new shell.
""")

task("TASKD8", "Release v1.2.0", "TASKD1, TASKD2, TASKD3, TASKD4, TASKD5, TASKD6, TASKD7", """
1. README and help text cover every change of this phase; PROJECT.md's roadmap moves the pulled-in items from v2 to v1.2.
2. Mark the done items in `tasks/backlog.md` (B08, B09, B13).
3. Tag `v1.2.0` only after the maintainer confirms. The release workflow publishes to GitHub and npm and installs on clean machines.

**Done when:** the release workflow is green.
""")

# ---------------------------------------------------------------- Phase 5
phase("Phase 5 — v2")

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
TASKD4 added Cursor and Claude Desktop, the shared JSON installer and `--for`.

1. Find out how Codex, Gemini CLI, VS Code and others register stdio MCP servers (config file path and format, or a CLI command). Codex uses TOML.
2. Run the TASKD4 spike for each: does it pass the MCP `instructions` to the model?
3. Add each one that does to `mcp install --for <name>`.
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

task("TASKB1", "Prefix-free commands in Claude Code", "TASK20, TASK10, TASK22", """
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
6. Opt in: `quiver mcp install` asks once (or `--commands`), saved in the config. `quiver mcp uninstall` removes every generated folder.

**Decide first:** the body could instead hold a copy of the skill. That needs no tool call and works when the server is down, but extra files (`references/…`) are then missing, so the pointer is the default.

**Done when:**
- Tests cover naming, name clashes, the marker rule, removal of stale folders and `.git/info/exclude`.
- In Claude Code, `/review ship version 1.2` loads the skill and gets the whole task.
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

# ---------------------------------------------------------------- renumber
ORDER = [
    ("Phase 0 — Foundation", ["TASK01", "TASK02", "TASK03", "TASK04", "TASK05", "TASK06", "TASK07"]),
    ("Phase 1 — Remote Sources", ["TASK08", "TASK23", "TASK24", "TASK25", "TASK26", "TASK27", "TASK10", "TASK28"]),
    ("Phase 2 — MCP Server (first usable version)", ["TASK15", "TASK16", "TASK17", "TASKA1", "TASK22", "TASK18"]),
    ("Phase 3 — Local Skills and Management", ["TASK11", "TASK09", "TASK19", "TASK20", "TASK21", "TASK12", "TASK13", "TASK14"]),
    ("Phase 4 — v1 Release", ["TASK29", "TASK30", "TASK31"]),
    ("Phase 4b — v1.1", ["TASKC1"]),
    ("Phase 4c — v1.2", ["TASKD1", "TASKD2", "TASKD3", "TASKD4", "TASKD5", "TASKD6", "TASKD7", "TASKD8"]),
    ("Phase 5 — v2", ["TASKE1", "TASKE2", "TASK32", "TASK33", "TASK34", "TASK35", "TASK36", "TASK37", "TASK38", "TASK39", "TASK40", "TASKB1"]),
]
ALL = {t[0]: t for _, ts in PHASES for t in ts}
flat = [tid for _, ids in ORDER for tid in ids]
assert sorted(flat) == sorted(ALL), set(ALL) ^ set(flat)
REMAP = {old: f"TASK{i:02d}" for i, old in enumerate(flat, 1)}
remap = lambda s: re.sub(r"TASK(?:\d\d|A1|B1|C1|D\d|E\d)", lambda m: REMAP[m.group(0)], s)
MARK = {"x": "✅", "~": "🚧"}
STATUS = {"TASK01": "x", "TASK02": "x", "TASK03": "x", "TASK04": "x", "TASK05": "x", "TASK06": "x", "TASK07": "x", "TASK08": "x", "TASK09": "x", "TASK10": "x", "TASK11": "x", "TASK12": "x", "TASK13": "x", "TASK14": "x", "TASK15": "x", "TASK16": "x", "TASK17": "x", "TASK18": "x", "TASK19": "x", "TASK20": "x", "TASK21": "x", "TASK22": "x", "TASK23": "x", "TASK24": "x", "TASK25": "x", "TASK26": "x", "TASK27": "x", "TASK28": "x", "TASK29": "x", "TASK30": "x", "TASK31": "x", "TASK32": "x", "TASK33": "x", "TASK34": "x"}  # final IDs; x done, ~ in progress
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
quiver source add JuliusBrussee/caveman --name caveman
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
