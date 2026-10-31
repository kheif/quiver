# Releasing

A tag `v<major>.<minor>.<patch>` releases Quiver. `.github/workflows/release.yml` then:

1. runs the tests,
2. builds the archives with GoReleaser and publishes the GitHub release with `checksums.txt`,
3. publishes the npm packages (`@kheif/quiver` and one package per platform) with npm trusted publishing,
4. installs the release with `install.sh` on Linux and macOS, with `install.ps1` on Windows, and with npm on Linux, macOS and Windows.

A tag with a suffix, such as `v1.1.0-rc.1`, is a prerelease: GitHub marks it so, `releases/latest` skips it, and npm publishes it under the `next` tag.

Every push already runs the `release-dry-run` CI job, which builds the archives and the npm packages and installs both without publishing anything, and `release-dry-run-windows`, which installs the Windows archive with `install.ps1`.

## A normal release

```sh
git tag v1.0.1
git push origin v1.0.1
```

Then watch the Release workflow: `gh run watch`.

## The first npm release (once)

npm trusted publishing is set up per package, on the package's page, so the packages must exist first. The first version is published by hand from your machine, with your npm login and 2FA. No npm token goes into CI.

1. Tag the release locally, without pushing it:
   ```sh
   git tag v1.0.0
   ```
2. Build it with the release version:
   ```sh
   go run github.com/goreleaser/goreleaser/v2@v2.18.3 release --clean --skip=publish
   ```
3. Log in and publish the seven packages. npm may ask for a 2FA code for each one:
   ```sh
   npm login
   node scripts/npm-publish.mjs v1.0.0
   ```
4. Push the tag. The workflow publishes the GitHub release; it skips the npm versions that already exist.
   ```sh
   git push origin v1.0.0
   ```

## Trusted publishing

Set up on 2026-10-29 for all seven packages; v1.1.0 was the first release published by CI. Repeat this only for a new package.

On npmjs.com, for each of the seven packages (`@kheif/quiver`, `@kheif/quiver-darwin-arm64`, `-darwin-x64`, `-linux-arm64`, `-linux-x64`, `-win32-arm64`, `-win32-x64`): **Settings → Trusted publishing → GitHub Actions**, with:

- Organization or user: `kheif`
- Repository: `quiver`
- Workflow filename: `release.yml`
- Allowed actions: also allow `npm publish`

A new trusted publisher expires if it is not used for a publish within 2 days, so set it up right before you push the next tag.

A new version can take several minutes to show up on npm; the workflow's npm install check waits up to 15 minutes.
