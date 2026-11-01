# TASK57.4: Delivery report, doctor and status

**Phase:** Phase 6: Session-start delivery

**Part of:** [TASK57](TASK57-quiver-install.md): `quiver install`

**Status:** Todo

**Depends on:** TASK57.2

Users must see which assistants load always-on skills by themselves, so they do not take those skills for optional ones where they are not.

1. **Model.** Replace `JSONAssistant.OnRequest` with a delivery level on every assistant (the levels of TASK54: hook, instructions, rules file, on request), set from `docs/spike-hooks.md`.
2. **Install report.** After the per-assistant lines, print one line per assistant with its level. For every assistant without a hook, say that loading skills there is up to the model, and when always-on skills are assigned, name them:
   ```
   Cursor: no session-start hook. Quiver's skills are loaded when the model decides to.
     These always-on skills do not load by themselves there: security-review, reply-style.
   ```
   The names come from the global always-on skills and those of the project in the current folder.
3. **doctor.** For each hook: present, runs this binary, file readable, trusted (Codex). For each assistant without a hook while always-on skills are assigned: a warning that names them.
4. **status.** A short section that lists each installed assistant and its level.

**Done when:** tests cover every level, the warning with and without always-on skills, and the doctor checks.
