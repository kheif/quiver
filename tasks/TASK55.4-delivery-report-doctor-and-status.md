# TASK55.4: Delivery report, doctor and status

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK55](TASK55-quiver-install.md): `quiver install`

**Status:** ✅ Done

**Depends on:** TASK55.2

Users must see which assistants load always-on skills by themselves, so they do not take those skills for optional ones where they are not.

1. **Model.** Replace `JSONAssistant.OnRequest` with a delivery level on every assistant (the levels of TASK52: hook, instructions, on request), set from `docs/spike-hooks.md`.
2. **Install report.** After the per-assistant lines, print one line per assistant with its level. For every assistant without a hook, say that loading skills there is up to the model, and when always-on skills are assigned, name them:
   ```
   Claude Desktop: no session-start hook. Quiver's skills are loaded when the model decides to.
     These always-on skills do not load by themselves there: security-review, reply-style.
   ```
   The names come from the global always-on skills and those of the project in the current folder.
3. **doctor.** For each hook: present, runs this binary, file readable, trusted (Codex). For each assistant without a hook while always-on skills are assigned: a warning that names them.
4. **status.** A short section that lists each installed assistant and its level.

**Done when:** tests cover every level, the warning with and without always-on skills, and the doctor checks.

**Notes:**
- `internal/assistant/delivery.go` adds a `Delivery` level (`DeliveryHook`, `DeliveryInstructions`, `DeliveryOnRequest`) with `HasHook`, `Level` and `Describe`. `JSONAssistant.OnRequest` became `Delivery`: Cursor `DeliveryInstructions`, Claude Desktop and Antigravity `DeliveryOnRequest`. Claude Code and Codex are `DeliveryHook` (in `mcpTarget.delivery`).
- Install report: `deliveryReport` prints, after the per-assistant lines, one line per assistant with `Describe`, and for a non-hook assistant names the always-on skills (`assignedAlwaysOn`, the global ones plus the current project's). The old per-assistant on-request note was removed.
- doctor: `checkClaudeHook` reports the Claude Code session-start hook (present, runs this binary, readable) via `assistant.ClaudeHookInstalled`; the Codex trust check (TASK55.3) stays; and each installed assistant without a hook, when always-on skills are assigned, gets a warning that names them.
- status: a new "Assistants (how always-on skills reach them)" section lists each installed assistant and its level, in text and JSON (`statusAssistant`).
- Tests: `TestMCPInstallEveryAssistant` covers all three levels in the report with no always-on assigned; `TestDeliveryReportNamesAlwaysOn` covers the report line and the doctor warning with an always-on skill; `TestStatusAssistants` covers the status section and the three level names; the doctor hook check is exercised through `registered` (hook installed → ok) and `TestDoctorClaudeRunsOtherQuiver`. Status golden/JSON tests isolate `PATH` so the assistant list is empty and deterministic.
- README's MCP section updated: Codex moved from "on request" to the hook group, and the per-assistant delivery is described.
