# TASK69: Security review

**Phase:** Phase 8: Test phase

**Status:** Todo

**Depends on:** TASK57

A review of everything that can reach a session or run on the machine, written as a threat model in `docs/security.md`: for each risk, what can go wrong, the mitigation, and the test that shows it.
- Remote skill text that the hook puts into every session. Mitigations today: the update preview (TASK48) marks always-on skills and flags scripts, and pins (TASK47) keep a reviewed commit.
- The hook command that runs at every session start: which binary, from which path, and what happens if that path becomes writable by others.
- Scripts inside skills.
- Path traversal and symlinks in `get_skill_file`, in skill folders and in source repositories.
- File modes of the configs Quiver writes, of their backups and of the logs.
- Secrets in logs and in the hook's log line.
- The rules-file block and hooks written into other tools' files.

**Done when:** `docs/security.md` lists each risk with its mitigation and test, and none is open.
