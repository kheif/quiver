---
title: Supported platforms
description: Operating systems, CPU architectures and requirements of Quiver.
---

## Release builds

Each release has a binary for:

| System | x86-64 (`amd64`) | ARM64 (`arm64`) | Archive |
| --- | --- | --- | --- |
| macOS | Yes | Yes | `quiver_darwin_<arch>.tar.gz` |
| Linux | Yes | Yes | `quiver_linux_<arch>.tar.gz` |
| Windows | Yes | Yes | `quiver_windows_<arch>.zip` |

The binaries are static builds without cgo. CI runs the test suite on Linux, macOS and Windows, and installs every release with each install method on clean machines.

## Requirements

- `git` on your `PATH`.
- For npm: Node.js 18 or later.
- For `go install` and builds from source: the Go version in `go.mod` or later.

## Assistants

See [Connect an assistant](/docs/assistants/) for the assistants `quiver install` sets up and what each supports.
