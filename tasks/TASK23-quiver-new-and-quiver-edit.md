# TASK23 — `quiver new` and `quiver edit`

**Phase:** Phase 3 — Local Skills and Management

**Status:** Todo

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
3. For remote skills, refuses and suggests `fork` (TASK35).

**Done when:** Create, assign and commit are tested for all three scopes. Duplicate names and path traversal are rejected.
