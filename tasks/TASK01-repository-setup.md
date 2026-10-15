# TASK01 — Repository setup

**Phase:** Phase 0 — Foundation

**Depends on:** —

1. Initialize the Go module: `go mod init github.com/<owner>/quiver`.
2. Create this layout:
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
5. Add GitHub Actions CI. It runs `go test ./...` and `golangci-lint` on macOS, Linux and Windows.

**Done when:**
- `make build` produces `./bin/quiver`.
- `quiver --version` prints the version.
- CI passes on all three OSes.
