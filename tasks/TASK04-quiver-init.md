# TASK04 — `quiver init`

**Phase:** Phase 0 — Foundation

**Status:** Todo

**Depends on:** TASK03

1. Create `~/.quiver/`, `local/`, `sources/`, `config.json` and `registry.json`.
2. Run `git init` in `local/` and commit a short README.
3. Make the command idempotent. Running it again only fills in what is missing and never overwrites anything.
4. Every other command, including `quiver mcp`, auto-runs `init` if `~/.quiver` does not exist.

**Done when:**
- Running `init` twice gives the same result.
- `local/` is a git repo with one commit.
