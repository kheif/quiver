# Spike: other assistants over MCP

TASK37 adds Cursor and Claude Desktop to `quiver mcp install`; TASK50 adds Codex and Antigravity and checks Gemini CLI. VS Code is not measured yet. The skill catalog and the always-on skills reach the assistant only through the server's `instructions` (see [spike-b01-results.md](spike-b01-results.md)). An assistant that does not pass `instructions` to the model never sees the skills. This spike checks each assistant before Quiver lists it as supported.

**Status:** Cursor and Claude Desktop done on 2026-10-31, Codex, Antigravity and Gemini CLI on 2026-11-01. Cursor passes `instructions` to the model. Codex and Antigravity pass them only once the model looks for Quiver's tools. Claude Desktop drops them.

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

Codex and Antigravity have their own scripts, described in their results below. For Cursor, `PATH=$HOME/.local/bin:$PATH QUIVER=$PWD/bin/quiver spike/b01/cursor-run.sh /tmp/quiver-cursor-spike` does all of this and runs every prompt headless with the Cursor CLI (after `cursor-agent login`); `python3 spike/b01/cursor-analyze.py /tmp/quiver-cursor-spike/out` summarizes the sessions.

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

### Codex

Measured on 2026-11-01 with `codex-cli` 0.166.0 (model `gpt-6-astra`), headless, with `QUIVER=$PWD/bin/quiver spike/b01/codex-run.sh <dir>` and `python3 spike/b01/codex-analyze.py <dir>/out`. The script passes the server with `-c mcp_servers.quiver...`, so `~/.codex/config.toml` is not changed. `codex exec` cannot ask for approval, so the first run set `default_tools_approval_mode = "approve"`; once the read tools were marked read-only (`readOnlyHint`), Codex ran them without approval.

Codex always defers MCP tools (`tool_search_always_defer_mcp_tools`): the model sees no Quiver tool until it searches for one. The server's instructions come with the tools: asked for Quiver's skills, the model called `get_skill` with IDs that only the instructions name, `reply-style` first.

| # | Check | Result |
|---|---|---|
| 1 | The server starts | Yes; `server started` in the log, in the workspace folder. |
| 2 | A fitting prompt loads the skill | **0 / 6.** The model never searched for Quiver's tools. With Quiver named in the prompt, 4 / 4 loaded the right skill and ended with its marker. |
| 3 | A non-fitting prompt loads no skill | 0 / 4 loaded a skill. |
| 4 | Always-on applies from the first reply | **0 / 10.** With Quiver named in the prompt, 5 / 5. |
| 5 | `get_skill_file` is used | Yes, when Quiver was named (`sql-migration-review` read `references/checklist.md`). |
| 6 | The folder the server starts in | The working directory. With `release-notes` assigned only to a project, the project folder loaded it and another folder did not. |
| 7 | Installed with npm | Not tested; `mcp install` gives Codex the binary itself, since the Codex app reads the same config. |

`codex mcp add` replaces the whole entry, so `mcp install` passes an existing entry's `env` again. It keeps the comments and other settings of `config.toml`.

### Antigravity

Measured on 2026-11-01 with the Antigravity CLI `agy` 1.4.12, default model, headless, with `QUIVER=$PWD/bin/quiver spike/b01/antigravity-run.sh <dir>` and `python3 spike/b01/antigravity-analyze.py <dir>/out`. `agy` reads MCP servers from `~/.gemini/config/mcp_config.json`, so the script adds a `quiver` entry there and removes it at the end. A workspace's `.agents/mcp_config.json` works too, as [spike-hooks.md](spike-hooks.md) found later. Headless runs deny MCP calls unless `permissions.allow` has `mcp(quiver/*)`; the script passes it with `--settings` for each run.

Antigravity lists Quiver's tools directly, but writes the server's instructions to `~/.gemini/antigravity-cli/mcp/quiver/instructions.md` instead of the prompt. Asked for Quiver's skills, the model read that file and then called `get_skill(reply-style)`.

| # | Check | Result |
|---|---|---|
| 1 | The server starts | Yes, in the workspace folder. |
| 2 | A fitting prompt loads the skill | **1 / 6** (`fit-k8s1`, through `search_skills`, without reading the instructions). |
| 3 | A non-fitting prompt loads no skill | 0 / 4 loaded a skill. |
| 4 | Always-on applies from the first reply | **0 / 10.** Named in the prompt, the model read the instructions and loaded `reply-style`. |
| 5 | `get_skill_file` is used | Not run. |
| 6 | The folder the server starts in | The working directory, so project skills work. |
| 7 | Installed with npm | Not tested; `mcp install` gives Antigravity the binary itself. |

Not checked: the Antigravity app, which may keep its MCP config elsewhere.

### Gemini CLI

Gemini CLI 0.66.0 registers servers with `gemini mcp add` into `~/.gemini/settings.json`. Not measured: signing in with a Google account fails with "This client is no longer supported for Gemini Code Assist for individuals. To continue using Gemini, please migrate to the Antigravity suite of products". `spike/b01/gemini-run.sh` runs the prompts once a login (such as an API key) works.

## Decision

- **Every assistant `mcp install` knows is set up when it is installed** (decided 2026-11-01; until then Claude Desktop was set up only when named). Naming assistants sets up only those.
- **Cursor:** supported.
- **Codex, Claude Desktop and Antigravity:** `mcp install` prints that they use Quiver's skills only when the prompt asks for them. Check again when they pass server `instructions` to the model up front.
- **Gemini CLI:** not supported until it can be measured.
- **All but Claude Code:** the entry runs the quiver binary itself, never the npm launcher.
