---
title: Releases
description: Where to find Quiver releases, how versions are numbered, and how the docs relate to releases.
---

Releases are published on [GitHub](https://github.com/kheif/quiver/releases), each with archives for every platform, `checksums.txt` and a changelog. The same version is published to npm as [`@kheif/quiver`](https://www.npmjs.com/package/@kheif/quiver).

## Versions

Versions follow `v<major>.<minor>.<patch>`. A tag with a suffix, such as `v1.3.0-rc.1`, is a prerelease: GitHub marks it so, the install scripts' "latest" skips it, and npm publishes it under the `next` tag.

To install a given version:

```sh
curl -fsSL https://raw.githubusercontent.com/kheif/quiver/main/install.sh | QUIVER_VERSION=v1.2.0 sh
npm install -g @kheif/quiver@1.2.0
```

## The docs follow the main branch

This website is built from the main branch. Features that are on main but not in the latest release are marked **Next release**, with what differs in the latest release.

## How a release is made

A version tag starts the release workflow. It runs the tests and `govulncheck`, builds the archives with GoReleaser, publishes the GitHub release and the npm packages, then installs the release on clean Linux, macOS and Windows machines with each install method. The process is described in [docs/releasing.md](https://github.com/kheif/quiver/blob/main/docs/releasing.md).

## License

Quiver is licensed under the [Apache License 2.0](https://github.com/kheif/quiver/blob/main/LICENSE).
