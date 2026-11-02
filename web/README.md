# Quiver website

The home page and documentation of Quiver, published at <https://kheif.github.io/quiver/>. Built with [Astro](https://astro.build) and [Starlight](https://starlight.astro.build), as a static site with no runtime services, analytics or cookies.

## Develop

Needs Node.js 22 or later. Generating the CLI reference also needs Go.

```sh
cd web
npm ci
npm run dev       # http://localhost:4321/quiver/
```

| Command | What it does |
| --- | --- |
| `npm run dev` | Development server with live reload. |
| `npm run build` | Production build into `dist/`. |
| `npm run preview` | Serve `dist/` as it will be deployed. |
| `npm run check` | Type-check the Astro components and content. |
| `npm test` | Unit tests, and checks of the build in `dist/`: links, anchors and metadata. Run `npm run build` first. |
| `npm run cli-reference` | Regenerate the CLI reference from `quiver --help` (builds `../cmd/quiver`, or uses `$QUIVER_BIN`). |

## Layout

```text
web/
├── astro.config.mjs        site, base path, Starlight and the docs sidebar
├── public/                 favicon.svg and og.png, copied as they are
├── scripts/
│   ├── cli-reference.mjs   writes the CLI reference page
│   └── og.html             source of public/og.png
├── src/
│   ├── components/         the home page sections and shared pieces
│   ├── content/docs/docs/  the documentation, one Markdown file per page
│   ├── data/               install commands, links, unreleased features
│   ├── layouts/            the home page layout (head, header, footer)
│   ├── pages/              index.astro (home), robots.txt
│   ├── plugins/            rehype plugin that adds the base path to links
│   └── styles/             tokens.css (shared), base.css (home), starlight.css (docs)
└── tests/                  node:test files
```

## Edit the docs

- Each page is a Markdown (`.md`) or MDX (`.mdx`) file in `src/content/docs/docs/`. MDX is only needed for Starlight components such as tabs and steps.
- New pages must be added to `sidebar` in `astro.config.mjs`.
- Link to other pages by their site path, such as `/docs/quickstart/`. The base path (`/quiver`) is added at build time; do not write it.
- `src/content/docs/docs/reference/cli.md` is generated. Change the help text in `internal/cli` and run `npm run cli-reference`; the deploy workflow regenerates it on every build anyway.
- The repository README stays the short reference for the CLI. When behavior changes, update the README, the help text and the matching docs page together. `tests/install.test.mjs` fails when the install commands on the site are not in the README.

### Unreleased features

The docs follow the main branch. Features that are on main but not in the latest release are marked:

- on the home page, by the IDs in `unreleased` in `src/data/site.ts`, with `latestRelease` naming the latest release;
- in the docs, by a `:::note[Next release]` aside on the page and a `Next` sidebar badge in the page's frontmatter.

After a release, empty `unreleased`, update `latestRelease`, and remove the asides and badges: `grep -rn "Next release\|text: Next" src`.

### Status marks

The site describes Quiver as it will be when it is published. What is not built yet carries one of three marks. When a feature ships, remove its mark; nothing else on the site needs to change.

| Mark | Used for | Find it with |
| --- | --- | --- |
| **Next release** | On main, not in `latestRelease` yet | `grep -rn "Next release\|text: Next\|next release" src` and `unreleased` in `src/data/site.ts` |
| **coming soon** | Being built: the session-start hook, hosting for teams | `grep -rni "coming soon\|text: Soon\|coming-soon" src` |
| **Planned** | Not started: Quiver for teams, VS Code Copilot | `grep -rn "Planned\|planned" src` |

When the session-start hook ships, the `coming soon` marks on it go: the hero summary (`LayerPicker.astro`), the Try section (`TrySession.astro`), the assistant labels (`src/data/assistants.json`), and the `Always-on skills`, `MCP delivery` and `Connect an assistant` pages. The marks on hosting stay until it exists; it will get a page of its own.

## Deploy

`.github/workflows/pages.yml` builds the site on every push to `main` that touches `web/`, the CLI or the README, and deploys it to GitHub Pages. Pull requests build and test it without deploying. It can also be started by hand from the Actions tab (**Website → Run workflow**).

### Turn on GitHub Pages (once)

1. In the repository, open **Settings → Pages**.
2. Under **Build and deployment**, set **Source** to **GitHub Actions**.
3. Push to `main`, or run the **Website** workflow by hand. The URL appears in the workflow run and under **Settings → Pages**.

The workflow needs no secrets. Its deploy job uses the `github-pages` environment that GitHub creates; if the environment has protection rules, allow the `main` branch.

### When a build fails

Open the failed run in the **Actions** tab and look at the first red step:

- **npm ci**: `package-lock.json` is out of sync with `package.json`. Run `npm install` in `web/` and commit the lock file.
- **npm run cli-reference**: the Go build failed; the CLI itself does not compile.
- **npm run check** or **npm run build**: a component or Markdown error, such as a page missing from the sidebar or invalid frontmatter. Run the same command locally.
- **npm test**: a broken link or anchor, missing metadata, or an install command that is no longer in the README. The message names the file and the link.
- **deploy**: Pages is not set to **GitHub Actions**, or the environment's protection rules block `main`.

### Custom domain

The site is configured for the project URL. To move it to a domain such as `quiver.example`:

1. Add `public/CNAME` containing the domain, one line.
2. In `.github/workflows/pages.yml`, set the build's environment:
   ```yaml
   env:
     SITE_URL: https://quiver.example
     BASE_PATH: /
   ```
3. Point the domain's DNS at GitHub Pages and enter it under **Settings → Pages → Custom domain**.

Every link, canonical URL, the sitemap and `robots.txt` follow `SITE_URL` and `BASE_PATH`. Check a build locally first:

```sh
SITE_URL=https://quiver.example BASE_PATH=/ npm run build && BASE_PATH=/ npm test
```

## Social image

`public/og.png` (1200×630) is rendered from `scripts/og.html`. To change it, edit the HTML, open it in a browser at 1200×630 and save a screenshot over `public/og.png`.
