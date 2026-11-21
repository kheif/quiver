# TASK70: Security review

**Phase:** Phase 8: Test phase

**Status:** Todo

**Depends on:** TASK56, TASK63.8

A review of everything that can reach a session or run on the machine, written as a threat model in `docs/security.md`, which TASK62.10 starts and TASK63.8 extends with the risks of their own releases, and which this task completes: for each risk, what can go wrong, the mitigation, and the test that shows it.
- Remote skill text that the hook puts into every session. Mitigations today: the update preview (TASK48) marks always-on skills and flags scripts, and pins (TASK47) keep a reviewed commit.
- The hook command that runs at every session start: which binary, from which path, and what happens if that path becomes writable by others.
- Scripts inside skills.
- Tool bundles (TASK63): the commands that run from a manifest after one confirmation, the entries and files written into assistants' configs and projects, a manifest that changes after it was confirmed, and the ledger that removal trusts.
- Path traversal and symlinks in `get_skill_file`, in skill folders and in source repositories.
- File modes of the configs Quiver writes, of their backups and of the logs.
- Secrets in logs and in the hook's log line.
- Hooks written into other tools' files.
- Skills that the assistant loads without asking (TASK62.3, TASK62.4): the bulk question when a source is added, what `quiver update` shows for them, and that `update_skill` cannot change them.
- The download of the local model file (TASK62.7: pinned URLs, redirects limited, hash decides, no command run), the loader and the model code that read that file (TASK62.6: a malformed weight file must neither crash nor use unbounded memory), and that a search makes no network call.
- Skills created or changed through the MCP server (`create_skill`, `update_skill`): an assistant that reads hostile text can plant a global skill that every later session loads (backlog B07). Decide the review step (an `origin` mark, a notice in `quiver status`, the hook and the panel until the skill is reviewed) and test it.

**Done when:** `docs/security.md` lists each risk with its mitigation and test, and none is open.
