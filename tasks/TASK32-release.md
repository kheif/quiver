# TASK32 — Release

**Phase:** Phase 4 — v1 Release

**Status:** 🚧 In progress

**Depends on:** TASK31

Ship v1 through three channels: GitHub Releases, a curl install script and npm. No Homebrew tap (decided 2026-10-28).

**Before the release** (fixes from the user-perspective review on 2026-10-28; A to D done):

A. **Git worktrees are not copies.** Folders with the same project ID that are worktrees of one git repository (same `git rev-parse --git-common-dir`) are one project: no `PossibleCopy` warning, and `projects split` refuses there, because it would change the committed `.quiver.json`.
B. **`quiver delete <id>`** removes a local skill from the library and commits the removal in `local/`. It asks first (`--yes` skips). If the skill is assigned, it lists where and refuses unless `--force`, which unassigns it too. Source skills cannot be deleted; `quiver source remove` removes the whole source.
C. **Windows editor:** without `$VISUAL` and `$EDITOR`, `quiver new` and `quiver edit` open `notepad` on Windows instead of `vi`.
D. **README:** assignments are per user and machine (a teammate's clone gets `.quiver.json` but no skills; share skills through a source), how to uninstall, and that assistants other than Claude Code may not start the server in the project folder, so they get only global skills unless `QUIVER_PROJECT` is set.

A to D notes: `git.CommonDir` and `registry.SameRepository` compare worktrees; `SplitProject` returns `ErrWorktree`. `library.DeleteLocal` moves the folder aside before the registry write, and commits only when git tracked the folder. The editor default is `defaultEditor` in `internal/cli/new.go`.

**Release:**

1. Set up GoReleaser (the free version) for darwin/linux/windows on amd64 and arm64. A tag `v*` runs it in GitHub Actions and publishes a GitHub Release with the archives and a checksums file.
2. **curl:** add `install.sh` to the repository. It detects the OS and CPU, downloads the archive of the latest (or a given) release, checks it against the checksums file, and installs `quiver` into `~/.local/bin` (overridable). It prints how to add that folder to `PATH` when it is missing. Usage: `curl -fsSL https://raw.githubusercontent.com/kheif/quiver/main/install.sh | sh`. Windows users install with npm or from the release archive.
3. **npm:** GoReleaser's npm publishing is Pro only, so publish with a script in the release workflow. Use the esbuild layout: one package per platform (such as `@<scope>/quiver-darwin-arm64`, with `os` and `cpu` set) holding the binary, and a main package `@<scope>/quiver` that lists them as `optionalDependencies` and has a small `bin` script that runs the right one. No postinstall download: pnpm 10 skips install scripts by default, and installs keep working offline from a cache. `quiver` itself is taken on npm, so the package is scoped.
4. **npm auth:** publish from GitHub Actions with npm trusted publishing (OIDC, `id-token: write`), so no npm token is stored. Trusted publishing only works for packages that already exist, so the user publishes a first version of each package by hand, then adds the trusted publisher on npmjs.com.
5. **Stable path for `mcp install`:** an npm install lives under the Node version (nvm, fnm, volta), so the binary path changes when the user switches Node. When `quiver mcp install` runs from inside `node_modules`, register the path of the `quiver` command on `PATH` instead, and print a note that a Node switch needs `quiver mcp install` again. The curl install path (`~/.local/bin/quiver`) is stable.
6. Update the README's Install section: curl first, then npm, then `go install`.
7. Decided 2026-10-28: B07 stays in the backlog for v1. Decide backlog B07 (approval for global `create_skill`) before tagging. A global skill's description reaches the instructions in every folder. The smallest fix: leave unapproved global skills out of the catalog.
8. Update the GitHub Actions to versions that run on Node 24 (CI warns about Node 20).
9. Tag `v1.0.0`. Pushing the tag publishes everything, so only after the user confirms.

**Release setup notes (steps 1 to 6 and 8 done):** `.goreleaser.yaml` names archives `quiver_<os>_<arch>` without the version, so `install.sh` can use `releases/latest/download`. `install.sh` takes `QUIVER_DOWNLOAD_URL` for mirrors and tests. The npm launcher is `npm/quiver/bin/quiver.js`; `scripts/npm-publish.mjs` builds the seven packages from `dist/` and skips versions already published. The launcher sets `QUIVER_NPM=1`, and `mcp install` then registers the `quiver` on `PATH` (`command` in `internal/cli/mcp.go`). `.github/workflows/release.yml` publishes on a `v*` tag and installs both ways on clean machines; CI's `release-dry-run` job does the same without publishing on every push. A `TestMain` in `internal/cli` now points `HOME` and `QUIVER_HOME` at a temp folder, because some `mcp install` tests had used the real `~/.quiver`.

**License:** Apache-2.0 (decided 2026-10-28, with the enterprise track in mind). `LICENSE` is in the repository, the archives and every npm package.

**The user does by hand:** create the npm scope (the npm username or an organization), publish the first version of each package, and add the trusted publisher for `kheif/quiver` and the release workflow on npmjs.com.

**Done when:** on a clean macOS and Linux machine, the curl install and `npm install -g @<scope>/quiver` both give a working `quiver`, on Windows the npm install does, and the README quickstart runs.
