# TASK63: Coverage baseline

**Phase:** Phase 8: Test phase

**Status:** Todo

**Depends on:** TASK57

Package coverage today undercounts code that other packages' tests run: `internal/library` shows 39% and `internal/git` 41% because most of their code runs from `internal/cli`, and `internal/homesync` has no tests of its own. The e2e suite runs the built binary, whose coverage is not measured at all.

1. Measure all code together: unit tests with `-coverpkg=./...`, and the e2e binary built with `go build -cover` and run with `GOCOVERDIR`, merged with `go tool covdata`. Add a CI step that produces the merged report and keeps it as an artifact.
2. List the user-facing paths no test reaches, such as error branches of `install`, sync conflicts and update failures, in this task's notes.
3. Add tests for them, starting with `internal/homesync`.
4. Record the total in decisions.md as the baseline. CI fails when a change lowers it by more than 0.5 points.

**Done when:** the merged report is in CI and CI enforces the baseline.
