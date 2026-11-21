# TASK63.6: Tools page in the panel

**Phase:** Phase 7c: Tool bundles (v1.7)

**Part of:** [TASK63](TASK63-tool-bundles.md): Tool bundles

**Status:** Todo

**Depends on:** TASK58.1, TASK58.4, TASK63.5

A **Tools** page in the panel of TASK58, next to Skills, Projects, Sources and Health.

1. A list of the bundles of all sources with their state (not installed, installed in this folder, installed everywhere, changed, missing), filtered with `/` like the other pages.
2. A detail view with the components, the pinned commit and the scopes. Add and remove use the consent text of TASK63.5 in the same words, not a shorter one, and a decline changes nothing.
3. The Health page lists changed or missing entries of installed bundles and a missing runtime, with the command that repairs each.
4. Text that comes from a manifest or from a tool's status command goes through the safe display of TASK58.4.

**Done when:** tests drive the page with the panel's test harness for the list, the filter, add with a yes and with a no, remove, and the Health entries; a manifest with control characters in a field displays safely.
