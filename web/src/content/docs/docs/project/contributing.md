---
title: Contributing
description: Build and test Quiver, open a pull request, and edit this website.
---

Quiver is developed on GitHub at [kheif/quiver](https://github.com/kheif/quiver). Bug reports, ideas and pull requests are welcome. The full guide is [CONTRIBUTING.md](https://github.com/kheif/quiver/blob/main/CONTRIBUTING.md).

## Build and test

Quiver needs Go (the version in `go.mod` or later) and git.

```sh
make build              # bin/quiver
make test               # go test ./..., including the end-to-end suite in e2e/
make lint               # golangci-lint
go test -race ./...     # as CI runs it on Linux and macOS
```

`QUIVER_HOME` points Quiver at another library folder, so you can try changes without touching `~/.quiver`:

```sh
QUIVER_HOME=/tmp/quiver-try ./bin/quiver init --yes
```

## Pull requests

- Keep one change per pull request, with tests.
- Every command has unit tests; a user-facing flow also has an end-to-end test in `e2e/`.
- Update the README and the help text when behavior changes.

The work is planned in [tasks/](https://github.com/kheif/quiver/tree/main/tasks): open questions are in `tasks/backlog.md`, technical decisions in `tasks/decisions.md`.

## Issues

- [Report a bug or ask for a feature](https://github.com/kheif/quiver/issues/new/choose). Include `quiver --version` and the output of `quiver doctor`.
- Security problems go through the private [vulnerability report](https://github.com/kheif/quiver/security/advisories/new), not an issue.

## Edit this website

The website is in [`web/`](https://github.com/kheif/quiver/tree/main/web), built with Astro and Starlight. Each docs page has an **Edit page** link at the bottom.

- Docs pages are Markdown files in `web/src/content/docs/docs/`. Link to other pages as `/docs/<page>/`; the base path is added at build time.
- The [CLI reference](/docs/reference/cli/) is generated from `quiver --help`. Change the help text in `internal/cli`, not the page.
- The README stays the short reference. When a change affects users, update the README, the help text and the matching docs page together.

```sh
cd web
npm ci
npm run dev    # http://localhost:4321/quiver/
```

See [`web/README.md`](https://github.com/kheif/quiver/blob/main/web/README.md) for the build, the tests and the deployment.
