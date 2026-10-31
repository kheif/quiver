# Spike: Cursor and Claude Desktop over MCP

TASK37 adds Cursor and Claude Desktop to `quiver mcp install`. The skill catalog and the always-on skills reach the assistant only through the server's `instructions` (see [spike-b01-results.md](spike-b01-results.md)). An assistant that does not pass `instructions` to the model never sees the skills. This spike checks each assistant before Quiver lists it as supported.

**Status:** done on 2026-10-31. Cursor passes `instructions` to the model; Claude Desktop does not.

## Setup

1. Build Quiver: `make build`.
2. Prepare a throwaway Quiver home with the B01 fixture skills, all global:

   ```sh
   QUIVER=$PWD/bin/quiver spike/b01/gui-setup.sh /tmp/quiver-gui-spike
   ```

   It prints an `mcpServers` entry that runs this build with `QUIVER_HOME` set to the throwaway home, so your own library is not touched.
3. Add the entry to the assistant's MCP config, then restart the assistant:
   - Cursor: `~/.cursor/mcp.json`
   - Claude Desktop: `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS), `%APPDATA%\Claude\claude_desktop_config.json` (Windows)
4. Check that the server started: `/tmp/quiver-gui-spike/home/logs/mcp.log` has a `server started` line. In Cursor, the server also shows in Settings → MCP.

For Cursor, `PATH=$HOME/.local/bin:$PATH QUIVER=$PWD/bin/quiver spike/b01/cursor-run.sh /tmp/quiver-cursor-spike` does all of this and runs every prompt headless with the Cursor CLI (after `cursor-agent login`); `python3 spike/b01/cursor-analyze.py /tmp/quiver-cursor-spike/out` summarizes the sessions.

The fixture skills end each answer with a marker such as `[used:k8s-debug]`. The always-on skill `reply-style` ends every reply with `[always-on]` and writes headings in lowercase.

## Checks

Start a new chat for each prompt. In Cursor, use Agent mode. The prompts are in `spike/b01/prompts.txt`.

| # | Check | How |
|---|---|---|
| 1 | The server starts | The log has `server started`; the assistant lists Quiver's four tools. |
| 2 | A fitting prompt loads the skill | `fit-k8s1`, `fit-rel1` and `fit-sql1`: the assistant calls `get_skill` with the right ID, and the answer ends with the skill's marker. |
| 3 | A non-fitting prompt loads no skill | `none-1` and `none-4`: no `get_skill` call other than `reply-style`. |
| 4 | Always-on applies from the first reply | The first reply of every chat ends with `[always-on]`. |
| 5 | `get_skill_file` is used | `fit-sql2`: the assistant reads `references/checklist.md`. |
| 6 | The folder the server starts in | The `server started` line of the log names the project, or none. Open a folder that is a Quiver project (Cursor), and note whether `project=` is set. |
| 7 | Installed with npm | With Quiver installed by npm and Node from nvm or Homebrew, does the app start the `quiver` shim? Desktop apps do not get the shell's `PATH`, so `#!/usr/bin/env node` may fail. |

Check 2 and 4 decide the outcome. If an assistant ignores `instructions` (2 and 4 fail, but calling `list_skills` by hand works), decide before release whether `mcp install` still sets it up.

## Results

### Cursor

Measured on 2026-10-31 with the Cursor CLI (`cursor-agent` 2026.10.22-27dcf5f, model Auto), headless, with `spike/b01/cursor-run.sh` and `spike/b01/cursor-analyze.py`. The CLI reads the same `mcp.json` files as the Cursor app (3.27.22 was installed), but it is not the app. The workspace got a project-level `.cursor/mcp.json` and a `.cursor/cli.json` that allows only `Mcp(quiver:*)`. Without that allow rule, headless runs reject every MCP call ("User rejected MCP: quiver-get_skill").

Cursor passes the server's `instructions` to the model as the server's `namespaceDescription`. The model named the fitting skill in its first sentence, before any tool call.

| # | Check | Result |
|---|---|---|
| 1 | The server starts | Yes; the log has `server started` and the four tools are listed. |
| 2 | A fitting prompt loads the skill | **6 / 6**, each with the right skill and its marker |
| 3 | A non-fitting prompt loads no skill | **0 / 4** loaded a skill other than `reply-style` |
| 4 | Always-on applies from the first reply | **10 / 10** called `get_skill(reply-style)` first and ended with `[always-on]` |
| 5 | `get_skill_file` is used | **2 / 2** `sql-migration-review` prompts read `references/checklist.md` |
| 6 | The folder the server starts in | The workspace folder (`from="working directory"`). With the workspace registered as a project and `k8s-debug` assigned only to it, `fit-k8s2` loaded `k8s-debug`, so project skills work. |
| 7 | Installed with npm | Not testable with the CLI, which inherits the shell's `PATH`. Run with a desktop app's `PATH` (`/usr/bin:/bin:/usr/sbin:/sbin`), the npm launcher fails with `env: node: No such file or directory`. So `mcp install` gives Cursor and Claude Desktop the path of the binary itself, never the launcher. |

Not checked: the catalog that is too long for `instructions` (the `list_skills` pointer) and `/compact`.

### Claude Desktop

Measured on 2026-10-31 with Claude Desktop 4.4212.0 on macOS, in the Chat tab, model Sonnet 5.5. Claude Desktop has no headless mode, so the prompts were sent by hand. `quiver mcp install --for claude-desktop` wrote the entry into the real `claude_desktop_config.json`, which held about 3 KB of app preferences: every preference was kept, as was the file mode 0600, and the backup was byte-identical to the original. For the test, the entry got `env.QUIVER_HOME` pointing to the `gui-setup.sh` home.

| # | Check | Result |
|---|---|---|
| 1 | The server starts | Yes; Claude Desktop starts two servers, one for Chat (its traffic is in `~/Library/Logs/Claude/mcp-server-quiver.log`) and one through `localMcpBridge`. Both list the four tools. |
| 2 | A fitting prompt loads the skill | **No.** `fit-k8s1` got a plain answer: no Quiver tool call (no `tools/call` in the log), no marker. |
| 3 | A non-fitting prompt loads no skill | Not meaningful, since 2 fails. |
| 4 | Always-on applies from the first reply | **No.** No `get_skill(reply-style)` call, no `[always-on]`. |
| 5 | `get_skill_file` is used | Not run. |
| 6 | The folder the server starts in | `/`, so only global skills are served, as expected. |
| 7 | Installed with npm | Same as Cursor: a desktop app's `PATH` has no `node`. |

Asked whether its context held instructions from a server named quiver, the model answered: "No. The MCP server instructions in my context only cover `claude-in-chrome`." It said Quiver's tools were deferred, loaded `list_skills` on request, and listed the four skills correctly. So Claude Desktop connects and runs the tools, but drops the server's `instructions`: skills load only when the user names Quiver in the prompt.

The Code tab of Claude Desktop is Claude Code and uses Claude Code's own MCP config, not `claude_desktop_config.json`; spike B01 covers it.

## Decision

- **Cursor:** supported. `quiver mcp install` sets it up whenever `~/.cursor` exists.
- **Claude Desktop:** set up only with `quiver mcp install --for claude-desktop`, which prints that skills are used only when the prompt asks for Quiver's skills and that always-on skills do not apply. `doctor` checks its entry only when there is one. Check again when Claude Desktop passes server `instructions` to the model.
- **Both:** the entry runs the quiver binary itself, never the npm launcher.
