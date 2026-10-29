# TASK23 — `quiver new` and `quiver edit`

**Phase:** Phase 3 — Local Skills and Management

**Status:** ✅ Done

**Depends on:** TASK04, TASK07, TASK08

`new <name>` and the MCP tool `create_skill` (TASK24) share one function: `library.CreateLocal(name, description, body, files, scope, projectID)`.

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
3. For remote skills, refuses through `library.CheckEditable` and suggests `fork` (TASK46).

**Notes:**
- `library.CreateLocal` (`internal/library/create.go`) returns the validation warnings. It writes into a hidden `local/.new-<name>-*` folder, which the library skips, validates, then renames it into place, all under the Quiver lock. Nothing is left behind or assigned when it fails before the commit. With `scope: "project"` the project must already be in the registry; `new` registers the folder first, as `add` does.
- Extra file paths must pass `filepath.IsLocal` and may not contain a backslash, a `.git` part, or be `SKILL.md`.
- `skill.ValidateName` is the name rule for new skills; a name over 64 characters is an error here, not a warning.
- `new` checks the name is free before it registers a project, so a typo does not create a `.quiver.json`. Without `--description` the skill is created and committed with the template (`add <name>`), then opened; a change is committed as `edit <name>`. Both `new` and `edit` warn while the description still starts with `TODO`.
- `edit` opens a local skill from its folder, not through the library, so a skill that an earlier edit broke can still be fixed. An invalid edit stays on disk, uncommitted. The editor is `$VISUAL`, then `$EDITOR`, then `vi`, split on spaces so `code --wait` works.
- The read-only error already suggests `quiver fork`, which comes with TASK46.
- Tests replace `runEditor` in `internal/cli/new.go`.

**Done when:** Create, assign and commit are tested for all three scopes. Duplicate names and path traversal are rejected.
