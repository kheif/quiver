# TASK01 — Repository setup

**Phase:** Phase 0 — Foundation

**Status:** ✅ Done

**Depends on:** —

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
5. CI is deferred until the project has a GitHub repository. Then add GitHub Actions that run `go test ./...` and `golangci-lint` on macOS, Linux and Windows. Done in TASK30.

**Done when:**
- `make build` produces `./bin/quiver`.
- `quiver --version` prints the version.
- `make test` and `make lint` pass.
