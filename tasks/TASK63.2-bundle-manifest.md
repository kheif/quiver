# TASK63.2: Bundle manifest

**Phase:** Phase 7c: Tool bundles (v1.7)

**Part of:** [TASK63](TASK63-tool-bundles.md): Tool bundles

**Status:** Todo

**Depends on:** TASK63.1, TASK62.2

The data that describes a bundle, and the code that reads it. File name, syntax and where manifests live are fixed by TASK63.1; the fields below are the starting point.

1. **Fields.** `id`, `name`, `description`, `version`, the source (a git URL and a commit), an optional `kind`, `requires` (a runtime and a minimum version), `env` (names and what each is for), and the components of TASK63: `mcp`, `hooks`, `instructions`, `files`, `skills`, and `commands` (`install`, `upgrade`, `uninstall`, `status`). Every command is an argv array, never a string a shell parses. A component names the scopes it supports (global, project) and the assistants it applies to; the default is every assistant that `quiver install` supports.
2. **Strict reading.** An unknown field, an unknown component type, a command that is not an array, a path outside the home or the project, or a file over the size limit is an error that names the field. The manifest says its own format version, and a manifest of a newer version is refused with the version needed.
3. **Where bundles come from.** From a source added with `quiver source add`, like skills, so a bundle is pinned to a commit and updated with the preview of diffs that sources already have (`quiver update <source>` updates the clone as today and never runs a bundle command nor rewrites an installed bundle; it reports that an installed bundle has a newer manifest, and `quiver tool update` applies it, TASK63.5); manifests that ship in the binary only if TASK63.1 decides so. A local directory is also accepted (`quiver tool add ./dir`) so that a manifest can be written and tried; it is marked "local, not pinned" in the consent text.
4. **Library.** `internal/bundle` lists the bundles of all sources, with the same ID rules as skills (`<source>/<bundle>`), and a bundle whose ID clashes with a skill's is not ambiguous because they are listed apart.
5. Describe the manifest in data-model.md and in a new docs page; the format version follows the rule of TASK62.2.

**Done when:** tests cover every field, every refusal of item 2, a manifest from a newer version, and the listing across two sources.
