# TASK55.1: Rename to `quiver install` and `quiver uninstall`

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK55](TASK55-quiver-install.md): `quiver install`

**Status:** Todo

**Depends on:** TASK50

1. Move the commands from `internal/cli/mcp.go` to `internal/cli/install.go` as `quiver install [assistant...]` and `quiver uninstall [assistant...]`, with the same names, completion and output.
2. **Keep the old names.** Removing a command is a breaking change, which needs a new major version. `quiver mcp install` and `quiver mcp uninstall` stay as hidden, deprecated commands that run the new ones and print one line on stderr naming the new command; their deprecated `--for` (TASK51) and `--print` keep working. They are removed in the next major version.
3. Update every caller and message: `guidedSetup` in `internal/cli/init.go`, the fix hints of `doctor` ("run: quiver mcp install ..."), the npm note in `mcpInstall`, and any other output a search for `mcp install` finds.
4. Update the docs: README (command table and the MCP server section), PROJECT.md (CLI and How It Works), CONTRIBUTING.md if it names the command, and the First Milestone of `tasks/README.md` in `scripts/gen_tasks.py`. The website is updated by the maintainer.
5. The GitHub release notes of v1.4.0 say that the command changed and that the old name still works.

**Done when:** the tests and completion use the new commands, the old ones still work and print the deprecation line, and a search for `mcp install` finds nothing outside the old commands, their tests and the files of finished tasks.
