# TASK63.8: Release v1.7.0

**Phase:** Phase 7c: Tool bundles (v1.7)

**Part of:** [TASK63](TASK63-tool-bundles.md): Tool bundles

**Status:** Todo

**Depends on:** TASK62.10, TASK63.2, TASK63.3, TASK63.4, TASK63.5, TASK63.6, TASK63.7

1. README and help text cover every change of this phase, including the `quiver tool` commands, the consent text and the Tools page; each subtask has already updated its own documentation, and this task checks that nothing is missing. PROJECT.md's roadmap lists the phase under v1.7, and its "Out of Scope" section words the two limits that bundles change (project files, running commands) so that skills keep them and bundles are the stated exception; the first sentence of tasks/data-model.md ("The only file Quiver puts into a project is `.quiver.json`") gets the same exception.
2. The GitHub release notes list the changes since v1.6.0, and say in their first lines that a bundle runs the commands of its manifest after the user confirmed them, and that v1.7 is a minor release because every v1.6 file is still read.
3. **Upgrade check.** From a home made by the v1.6.0 binary (its golden home was made in TASK62.10 from the released binary), the new binary runs with no `state/bundles.json`, `quiver doctor` is clean, and the ledger is created by the first `quiver tool add`.
4. **Clean machine.** On a machine without the two tools, `quiver tool add` and `quiver tool remove` of each first bundle run on Claude Code with the shipped build; the ledger is empty afterwards and the config files are as before.
5. Tag `v1.7.0` (annotated) only after the maintainer confirms. The release workflow publishes to GitHub and npm and installs on clean machines.
6. **Website.** The pages that the subtasks wrote carry the "Next release" aside and badge until the release, have their sidebar entries, and are committed only after the maintainer confirms; at the release the status marks follow it (see `web/README.md`).

7. **Security check.** Before the tag, the risks that this release adds are worked through in `docs/security.md` (started by TASK62.10; TASK70 completes it): the commands that run from a manifest after one confirmation, `--yes` and the pinned commit, the entries and files written into assistants' configs and projects, a manifest that changes after it was confirmed, the commands and the entries that the ledger holds and removal trusts, and a bundle hook that Codex has not trusted. Each risk has its mitigation and its test, and one that is open blocks the tag.
8. **Golden home.** After the tag, the golden home of v1.7.0 is made by running the released binary through the script of TASK62.2 and one `quiver tool add` of the fake tool of TASK63.7, and committed under `testdata/formats/v1.7.0/` (without `local/.git`, as TASK62.2 says); TASK67 upgrades from it.

**Done when:** the upgrade check, the clean-machine run, the security check and the golden home are recorded, and the release workflow is green.
