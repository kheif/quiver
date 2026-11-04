# TASK55.7: Hook text in parts

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK55](TASK55-quiver-install.md): `quiver install`

**Status:** ✅ Done

**Depends on:** TASK53, TASK55.2, TASK55.3, TASK55.5, TASK55.6, TASK56

Always-on skills must reach the model in full at session start, whatever their number and size, and every session must know the other skills served in its folder. An assistant passes only so much text from one hook command (Claude Code 10,000 characters), so the text goes out in parts, one hook command each.

1. **Text.** `server.BuildSessionContext` builds, for a project or for the global skills alone, the always-on skills in full (global first, then the project's) and then the ID and description of every other served skill, and splits it into parts under the assistant's limit (`hookLimits` in `internal/cli/hookparts.go`: Claude Code 10,000 characters, Codex 40,000 bytes, VS Code and Cursor 100,000 characters, Antigravity 40,000 bytes). A skill is kept whole when it fits a part and cut at line ends only when it is longer than one; nothing is dropped. Each part is tagged `part="i/n"` and says when it continues the previous one.
2. **Command.** `quiver hook session-start|subagent-start <assistant> [part]` prints part N (1 when omitted, so the command of an earlier Quiver is part 1) and nothing past the last part. Assistants that run a hook's commands at the same time (Claude Code, Codex, VS Code) get part N after a wait of 50 ms times N-1, so the parts mostly arrive in order.
3. **Install.** Every `Install*Hooks` takes the number of commands for each event and removes those beyond it. The number is the most any folder needs: the global text and every registered project's (`server.PartsNeeded`). `quiver install` sets it; every command that changes skills (`add`, `remove`, `new`, `edit`, `delete`, `import`, `fork`, `pin`, `unpin`, `update`, `source`, `sync`, `projects`) sets it again for the assistants whose hook runs this binary, and says so on stderr when it changes.
4. **Safety net.** When the last of an assistant's commands finds that the text takes more, it adds a note to its part, and adds the missing commands for the next session. `quiver doctor` warns when an assistant has fewer commands than the text takes.
5. **Instructions.** The MCP instructions no longer carry the catalog: they say how to use the tools, name the always-on skills for an assistant without a hook, and point to `list_skills` when there is no `<quiver-skills>` block.

**Done when:** unit tests cover the split (several parts, a skill longer than a part, bytes and characters, multibyte text, the largest project), the hook command, install, shrink and removal for each assistant, the sync after skill commands, and the doctor warning; and real sessions in Claude Code, Cursor and Antigravity received all always-on skills and the catalog in parts (docs/spike-hooks.md, "One text in several hook commands").
