# TASK32 — Release

**Phase:** Phase 4 — v1 Release

**Status:** Todo

**Depends on:** TASK31

Ship v1 through three channels: GitHub Releases, a curl install script and npm. No Homebrew tap (decided 2026-10-28).

1. Set up GoReleaser (the free version) for darwin/linux/windows on amd64 and arm64. A tag `v*` runs it in GitHub Actions and publishes a GitHub Release with the archives and a checksums file.
2. **curl:** add `install.sh` to the repository. It detects the OS and CPU, downloads the archive of the latest (or a given) release, checks it against the checksums file, and installs `quiver` into `~/.local/bin` (overridable). It prints how to add that folder to `PATH` when it is missing. Usage: `curl -fsSL https://raw.githubusercontent.com/kheif/quiver/main/install.sh | sh`. Windows users install with npm or from the release archive.
3. **npm:** GoReleaser's npm publishing is Pro only, so publish with a script in the release workflow. Use the esbuild layout: one package per platform (such as `@<scope>/quiver-darwin-arm64`, with `os` and `cpu` set) holding the binary, and a main package `@<scope>/quiver` that lists them as `optionalDependencies` and has a small `bin` script that runs the right one. No postinstall download: pnpm 10 skips install scripts by default, and installs keep working offline from a cache. `quiver` itself is taken on npm, so the package is scoped.
4. **npm auth:** publish from GitHub Actions with npm trusted publishing (OIDC, `id-token: write`), so no npm token is stored. Trusted publishing only works for packages that already exist, so the user publishes a first version of each package by hand, then adds the trusted publisher on npmjs.com.
5. **Stable path for `mcp install`:** an npm install lives under the Node version (nvm, fnm, volta), so the binary path changes when the user switches Node. When `quiver mcp install` runs from inside `node_modules`, register the path of the `quiver` command on `PATH` instead, and print a note that a Node switch needs `quiver mcp install` again. The curl install path (`~/.local/bin/quiver`) is stable.
6. Update the README's Install section: curl first, then npm, then `go install`.
7. Decide backlog B07 (approval for global `create_skill`) before tagging. A global skill's description reaches the instructions in every folder. The smallest fix: leave unapproved global skills out of the catalog.
8. Update the GitHub Actions to versions that run on Node 24 (CI warns about Node 20).
9. Tag `v1.0.0`. Pushing the tag publishes everything, so only after the user confirms.

**The user does by hand:** create the npm scope (the npm username or an organization), publish the first version of each package, and add the trusted publisher for `kheif/quiver` and the release workflow on npmjs.com.

**Done when:** on a clean macOS and Linux machine, the curl install and `npm install -g @<scope>/quiver` both give a working `quiver`, on Windows the npm install does, and the README quickstart runs.
